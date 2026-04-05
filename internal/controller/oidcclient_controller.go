package controller

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	applyconfigscorev1 "k8s.io/client-go/applyconfigurations/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/log"

	authv1alpha1 "github.com/kettleofketchup/AuthentikOperator/api/v1alpha1"
	"github.com/kettleofketchup/AuthentikOperator/internal/authentik"
	"github.com/kettleofketchup/AuthentikOperator/internal/bootstrap"
	"github.com/kettleofketchup/AuthentikOperator/internal/hash"
	"github.com/kettleofketchup/AuthentikOperator/internal/profiles"
	"github.com/kettleofketchup/AuthentikOperator/internal/rollout"
)

const (
	ConditionAuthentikProviderFound = "AuthentikProviderFound"
	ConditionSecretSynced           = "SecretSynced"
	ConditionRolloutTriggered       = "RolloutTriggered"
)

// OIDCClientReconciler reconciles an OIDCClient object
type OIDCClientReconciler struct {
	client.Client
	Scheme               *runtime.Scheme
	AuthentikClient      *authentik.Client
	AuthentikURL         string
	ReconcileInterval    time.Duration
	TokenSecretName      string
	TokenSecretNamespace string
	// Bootstrap config for automatic token refresh on 403
	BootstrapSecretName string // K8s secret name containing bootstrap token (e.g. "authentik-bootstrap")
	BootstrapSecretKey  string // Key within that secret (e.g. "bootstrap_token")
	BootstrapClientOpts []authentik.ClientOption
}

// +kubebuilder:rbac:groups=auth.kettleofketchup,resources=oidcclients,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=auth.kettleofketchup,resources=oidcclients/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=auth.kettleofketchup,resources=oidcclients/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=apps,resources=deployments;statefulsets,verbs=get;update;patch
// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;watch;create;update;patch;delete

func (r *OIDCClientReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	requeueInterval := r.ReconcileInterval
	if requeueInterval == 0 {
		requeueInterval = 5 * time.Minute
	}

	// 0. Ensure we have an Authentik API token (may not exist at startup if bootstrap job hasn't run yet)
	if !r.AuthentikClient.HasToken() && r.TokenSecretName != "" {
		tokenSecret := &corev1.Secret{}
		if err := r.Get(ctx, types.NamespacedName{
			Name:      r.TokenSecretName,
			Namespace: r.TokenSecretNamespace,
		}, tokenSecret); err == nil {
			if tok := string(tokenSecret.Data["token"]); tok != "" {
				r.AuthentikClient.SetToken(tok)
				logger.Info("loaded Authentik API token from secret", "secret", r.TokenSecretName)
			}
		}
		if !r.AuthentikClient.HasToken() {
			logger.Info("Authentik API token not yet available, will retry", "secret", r.TokenSecretName)
			return ctrl.Result{RequeueAfter: requeueInterval}, nil
		}
	}

	// 1. Get the CR
	oidcClient := &authv1alpha1.OIDCClient{}
	if err := r.Get(ctx, req.NamespacedName, oidcClient); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// 2. Fetch provider from Authentik (two-step: app by slug, then provider by app PK)
	slug := oidcClient.Spec.Authentik.ApplicationSlug
	provider, err := r.AuthentikClient.GetOAuth2ProviderBySlug(ctx, slug)
	if err != nil {
		// Auto-refresh on expired token
		if stderrors.Is(err, authentik.ErrTokenExpired) {
			logger.Info("Authentik API token expired, attempting refresh")
			if refreshErr := r.refreshToken(ctx); refreshErr != nil {
				logger.Error(refreshErr, "failed to refresh Authentik API token")
			} else {
				// Retry immediately with new token
				return ctrl.Result{Requeue: true}, nil
			}
		}

		logger.Error(err, "failed to fetch Authentik provider", "slug", slug)
		meta.SetStatusCondition(&oidcClient.Status.Conditions, metav1.Condition{
			Type:               ConditionAuthentikProviderFound,
			Status:             metav1.ConditionFalse,
			Reason:             "ProviderNotFound",
			Message:            fmt.Sprintf("Failed to fetch provider for slug %q: %v", slug, err),
			ObservedGeneration: oidcClient.Generation,
		})
		if statusErr := r.Status().Update(ctx, oidcClient); statusErr != nil {
			logger.Error(statusErr, "failed to update status")
		}
		return ctrl.Result{RequeueAfter: requeueInterval}, nil
	}

	// Provider found
	meta.SetStatusCondition(&oidcClient.Status.Conditions, metav1.Condition{
		Type:               ConditionAuthentikProviderFound,
		Status:             metav1.ConditionTrue,
		Reason:             "Found",
		Message:            fmt.Sprintf("Provider %q found for slug %q", provider.Name, slug),
		ObservedGeneration: oidcClient.Generation,
	})

	// Fetch signing certificate if configured on provider
	var signingCert *profiles.SigningCert
	if provider.SigningKey != nil && *provider.SigningKey != "" {
		certKP, err := r.AuthentikClient.GetCertificateByID(ctx, *provider.SigningKey)
		if err != nil {
			logger.Error(err, "failed to fetch signing certificate", "id", *provider.SigningKey)
			// Non-fatal: continue without signing cert
		} else {
			signingCert = &profiles.SigningCert{
				CertificatePEM:    certKP.CertificateData,
				FingerprintSHA256: certKP.FingerprintSHA256,
			}
		}
	}

	// 3. Build OIDC data and apply profile
	redirectURIs := parseRedirectURIs(provider.RedirectURIs)
	oidcData := profiles.BuildOIDCData(r.AuthentikURL, slug, provider.ClientID, provider.ClientSecret, redirectURIs)
	secretData := profiles.ApplyWithCert(oidcClient.Spec.SecretProfile, oidcData, oidcClient.Spec.SecretOverrides, signingCert)

	// 4. Compute hash and compare
	newHash := hash.ComputeSecretHash(secretData)
	if newHash == oidcClient.Status.SecretHash {
		return ctrl.Result{RequeueAfter: requeueInterval}, nil
	}

	// 5. Apply the target secret using server-side apply
	targetNS := oidcClient.Spec.Target.Namespace
	targetName := oidcClient.Spec.Target.SecretName

	secret := applyconfigscorev1.Secret(targetName, targetNS).
		WithLabels(map[string]string{
			"auth.kettleofketchup/managed-by":  "authentik-operator",
			"auth.kettleofketchup/oidc-client": oidcClient.Name,
		}).
		WithAnnotations(map[string]string{
			"argocd.argoproj.io/compare-options": "IgnoreExtraneous",
		}).
		WithData(toByteMap(secretData))

	if err := r.Apply(ctx, secret, client.FieldOwner("authentik-operator"), client.ForceOwnership); err != nil {
		logger.Error(err, "failed to apply secret", "name", targetName, "namespace", targetNS)
		meta.SetStatusCondition(&oidcClient.Status.Conditions, metav1.Condition{
			Type:               ConditionSecretSynced,
			Status:             metav1.ConditionFalse,
			Reason:             "ApplyFailed",
			Message:            fmt.Sprintf("Failed to apply secret: %v", err),
			ObservedGeneration: oidcClient.Generation,
		})
		if statusErr := r.Status().Update(ctx, oidcClient); statusErr != nil {
			logger.Error(statusErr, "failed to update status")
		}
		return ctrl.Result{RequeueAfter: requeueInterval}, nil
	}

	logger.Info("secret synced", "name", targetName, "namespace", targetNS)

	// 6. Patch ConfigMap if configured
	if oidcClient.Spec.ConfigMapTarget != nil {
		cmt := oidcClient.Spec.ConfigMapTarget
		sourceValue, ok := secretData[cmt.SourceKey]
		if !ok {
			logger.Error(fmt.Errorf("source key %q not found in profile output", cmt.SourceKey), "configmap patch skipped")
		} else {
			cm := &corev1.ConfigMap{}
			cmKey := types.NamespacedName{Name: cmt.Name, Namespace: cmt.Namespace}
			if err := r.Get(ctx, cmKey, cm); err != nil {
				logger.Error(err, "failed to get configmap", "name", cmt.Name, "namespace", cmt.Namespace)
				meta.SetStatusCondition(&oidcClient.Status.Conditions, metav1.Condition{
					Type:               "ConfigMapSynced",
					Status:             metav1.ConditionFalse,
					Reason:             "GetFailed",
					Message:            fmt.Sprintf("Failed to get ConfigMap %s/%s: %v", cmt.Namespace, cmt.Name, err),
					ObservedGeneration: oidcClient.Generation,
				})
			} else {
				if cm.Data == nil {
					cm.Data = make(map[string]string)
				}
				cm.Data[cmt.DataKey] = sourceValue
				if err := r.Update(ctx, cm); err != nil {
					logger.Error(err, "failed to patch configmap", "name", cmt.Name, "namespace", cmt.Namespace)
					meta.SetStatusCondition(&oidcClient.Status.Conditions, metav1.Condition{
						Type:               "ConfigMapSynced",
						Status:             metav1.ConditionFalse,
						Reason:             "UpdateFailed",
						Message:            fmt.Sprintf("Failed to update ConfigMap: %v", err),
						ObservedGeneration: oidcClient.Generation,
					})
				} else {
					logger.Info("configmap patched", "name", cmt.Name, "namespace", cmt.Namespace, "key", cmt.DataKey)
					meta.SetStatusCondition(&oidcClient.Status.Conditions, metav1.Condition{
						Type:               "ConfigMapSynced",
						Status:             metav1.ConditionTrue,
						Reason:             "Synced",
						Message:            fmt.Sprintf("ConfigMap %s/%s key %q synced", cmt.Namespace, cmt.Name, cmt.DataKey),
						ObservedGeneration: oidcClient.Generation,
					})
				}
			}
		}
	}

	// 7. Trigger rollout restart if enabled and secret changed
	if oidcClient.Spec.RolloutRestart != nil &&
		oidcClient.Spec.RolloutRestart.Enabled &&
		oidcClient.Spec.RolloutRestart.TargetRef != nil {

		ref := oidcClient.Spec.RolloutRestart.TargetRef
		if err := rollout.TriggerRollout(ctx, r.Client, ref.Kind, ref.Name, ref.Namespace, newHash); err != nil {
			logger.Error(err, "failed to trigger rollout restart",
				"kind", ref.Kind, "name", ref.Name, "namespace", ref.Namespace)
			meta.SetStatusCondition(&oidcClient.Status.Conditions, metav1.Condition{
				Type:               ConditionRolloutTriggered,
				Status:             metav1.ConditionFalse,
				Reason:             "RolloutFailed",
				Message:            fmt.Sprintf("Failed to trigger rollout: %v", err),
				ObservedGeneration: oidcClient.Generation,
			})
		} else {
			meta.SetStatusCondition(&oidcClient.Status.Conditions, metav1.Condition{
				Type:               ConditionRolloutTriggered,
				Status:             metav1.ConditionTrue,
				Reason:             "Triggered",
				Message:            fmt.Sprintf("Rollout triggered for %s/%s", ref.Kind, ref.Name),
				ObservedGeneration: oidcClient.Generation,
			})
		}
	}

	// 7. Update status
	now := metav1.Now()
	oidcClient.Status.SecretHash = newHash
	oidcClient.Status.LastSyncTime = &now
	meta.SetStatusCondition(&oidcClient.Status.Conditions, metav1.Condition{
		Type:               ConditionSecretSynced,
		Status:             metav1.ConditionTrue,
		Reason:             "Synced",
		Message:            "Secret successfully synced",
		ObservedGeneration: oidcClient.Generation,
	})
	if err := r.Status().Update(ctx, oidcClient); err != nil {
		logger.Error(err, "failed to update status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: requeueInterval}, nil
}

// refreshToken re-bootstraps the API token using the bootstrap secret.
// Called when the current token is expired/invalid (403).
func (r *OIDCClientReconciler) refreshToken(ctx context.Context) error {
	logger := log.FromContext(ctx)

	if r.BootstrapSecretName == "" {
		return fmt.Errorf("bootstrap secret not configured, cannot auto-refresh token")
	}

	// Read bootstrap token from K8s secret
	bsSecret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{
		Name:      r.BootstrapSecretName,
		Namespace: r.TokenSecretNamespace,
	}, bsSecret); err != nil {
		return fmt.Errorf("reading bootstrap secret %q: %w", r.BootstrapSecretName, err)
	}

	bootstrapToken := string(bsSecret.Data[r.BootstrapSecretKey])
	if bootstrapToken == "" {
		return fmt.Errorf("bootstrap token empty in secret %q key %q", r.BootstrapSecretName, r.BootstrapSecretKey)
	}

	logger.Info("Token expired, re-bootstrapping from bootstrap secret")

	// Clear the stale token
	r.AuthentikClient.ClearToken()

	// Run bootstrap with ForceRefresh to replace the stale secret
	cfg := bootstrap.Config{
		AuthentikURL:    r.AuthentikURL,
		BootstrapToken:  bootstrapToken,
		TokenIdentifier: "authentik-operator",
		TokenSecretName: r.TokenSecretName,
		Namespace:       r.TokenSecretNamespace,
		ClientOpts:      r.BootstrapClientOpts,
		ForceRefresh:    true,
	}

	if err := bootstrap.Run(ctx, r.Client, cfg); err != nil {
		return fmt.Errorf("re-bootstrap failed: %w", err)
	}

	// Reload the new token
	tokenSecret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{
		Name:      r.TokenSecretName,
		Namespace: r.TokenSecretNamespace,
	}, tokenSecret); err != nil {
		return fmt.Errorf("reading refreshed token secret: %w", err)
	}

	newToken := string(tokenSecret.Data["token"])
	if newToken == "" {
		return fmt.Errorf("refreshed token secret is empty")
	}

	r.AuthentikClient.SetToken(newToken)
	logger.Info("Token refreshed successfully")
	return nil
}

func toByteMap(data map[string]string) map[string][]byte {
	result := make(map[string][]byte, len(data))
	for k, v := range data {
		result[k] = []byte(v)
	}
	return result
}

// parseRedirectURIs extracts URL strings from the Authentik provider's redirect_uris field.
// Authentik returns redirect URIs as a JSON array of objects with "url" and "matching_mode" keys.
func parseRedirectURIs(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return nil
	}
	// Try as array of objects: [{"url": "...", "matching_mode": "..."}]
	var structured []struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(raw, &structured); err == nil && len(structured) > 0 {
		uris := make([]string, 0, len(structured))
		for _, s := range structured {
			if s.URL != "" {
				uris = append(uris, s.URL)
			}
		}
		return uris
	}
	// Fallback: try as plain string array
	var plain []string
	if err := json.Unmarshal(raw, &plain); err == nil {
		return plain
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager
func (r *OIDCClientReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&authv1alpha1.OIDCClient{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 5}).
		Complete(r)
}
