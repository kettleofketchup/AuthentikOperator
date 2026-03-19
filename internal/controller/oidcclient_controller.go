package controller

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	authv1alpha1 "github.com/kettleofketchup/AuthentikOperator/api/v1alpha1"
	"github.com/kettleofketchup/AuthentikOperator/internal/authentik"
	"github.com/kettleofketchup/AuthentikOperator/internal/hash"
	"github.com/kettleofketchup/AuthentikOperator/internal/profiles"
	"github.com/kettleofketchup/AuthentikOperator/internal/rollout"
)

// OIDCClientReconciler reconciles an OIDCClient object
type OIDCClientReconciler struct {
	client.Client
	Scheme            *runtime.Scheme
	AuthentikClient   *authentik.Client
	AuthentikURL      string
	ReconcileInterval time.Duration
}

// +kubebuilder:rbac:groups=auth.kettleofketchup,resources=oidcclients,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=auth.kettleofketchup,resources=oidcclients/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=auth.kettleofketchup,resources=oidcclients/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=apps,resources=deployments;statefulsets,verbs=get;update;patch
// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;watch;create;update;patch;delete

func (r *OIDCClientReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	requeueInterval := r.ReconcileInterval
	if requeueInterval == 0 {
		requeueInterval = 5 * time.Minute
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
		logger.Error(err, "failed to fetch Authentik provider", "slug", slug)
		meta.SetStatusCondition(&oidcClient.Status.Conditions, metav1.Condition{
			Type:               "AuthentikProviderFound",
			Status:             metav1.ConditionFalse,
			Reason:             "ProviderNotFound",
			Message:            fmt.Sprintf("Failed to fetch provider for slug %q: %v", slug, err),
			ObservedGeneration: oidcClient.Generation,
		})
		_ = r.Status().Update(ctx, oidcClient)
		return ctrl.Result{RequeueAfter: requeueInterval}, nil
	}

	// Provider found
	meta.SetStatusCondition(&oidcClient.Status.Conditions, metav1.Condition{
		Type:               "AuthentikProviderFound",
		Status:             metav1.ConditionTrue,
		Reason:             "Found",
		Message:            fmt.Sprintf("Provider %q found for slug %q", provider.Name, slug),
		ObservedGeneration: oidcClient.Generation,
	})

	// 3. Build OIDC data and apply profile
	oidcData := profiles.BuildOIDCData(r.AuthentikURL, slug, provider.ClientID, provider.ClientSecret)
	secretData := profiles.Apply(oidcClient.Spec.SecretProfile, oidcData, oidcClient.Spec.SecretOverrides)

	// 4. Compute hash and compare
	newHash := hash.ComputeSecretHash(secretData)
	if newHash == oidcClient.Status.SecretHash {
		return ctrl.Result{RequeueAfter: requeueInterval}, nil
	}

	// 5. Create or update the target secret
	targetNS := oidcClient.Spec.Target.Namespace
	targetName := oidcClient.Spec.Target.SecretName

	secret := &corev1.Secret{}
	err = r.Get(ctx, types.NamespacedName{Name: targetName, Namespace: targetNS}, secret)
	if errors.IsNotFound(err) {
		secret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      targetName,
				Namespace: targetNS,
				Labels: map[string]string{
					"auth.kettleofketchup/managed-by":  "authentik-operator",
					"auth.kettleofketchup/oidc-client": oidcClient.Name,
				},
			},
			Type: corev1.SecretTypeOpaque,
			Data: toByteMap(secretData),
		}
		if err := r.Create(ctx, secret); err != nil {
			logger.Error(err, "failed to create secret", "name", targetName, "namespace", targetNS)
			meta.SetStatusCondition(&oidcClient.Status.Conditions, metav1.Condition{
				Type:               "SecretSynced",
				Status:             metav1.ConditionFalse,
				Reason:             "CreateFailed",
				Message:            fmt.Sprintf("Failed to create secret: %v", err),
				ObservedGeneration: oidcClient.Generation,
			})
			_ = r.Status().Update(ctx, oidcClient)
			return ctrl.Result{RequeueAfter: requeueInterval}, nil
		}
	} else if err != nil {
		logger.Error(err, "failed to get secret", "name", targetName, "namespace", targetNS)
		return ctrl.Result{RequeueAfter: requeueInterval}, nil
	} else {
		secret.Data = toByteMap(secretData)
		if secret.Labels == nil {
			secret.Labels = make(map[string]string)
		}
		secret.Labels["auth.kettleofketchup/managed-by"] = "authentik-operator"
		secret.Labels["auth.kettleofketchup/oidc-client"] = oidcClient.Name
		if err := r.Update(ctx, secret); err != nil {
			logger.Error(err, "failed to update secret", "name", targetName, "namespace", targetNS)
			meta.SetStatusCondition(&oidcClient.Status.Conditions, metav1.Condition{
				Type:               "SecretSynced",
				Status:             metav1.ConditionFalse,
				Reason:             "UpdateFailed",
				Message:            fmt.Sprintf("Failed to update secret: %v", err),
				ObservedGeneration: oidcClient.Generation,
			})
			_ = r.Status().Update(ctx, oidcClient)
			return ctrl.Result{RequeueAfter: requeueInterval}, nil
		}
	}

	logger.Info("secret synced", "name", targetName, "namespace", targetNS)

	// 6. Trigger rollout restart if enabled and secret changed
	if oidcClient.Spec.RolloutRestart != nil &&
		oidcClient.Spec.RolloutRestart.Enabled &&
		oidcClient.Spec.RolloutRestart.TargetRef != nil {

		ref := oidcClient.Spec.RolloutRestart.TargetRef
		if err := rollout.TriggerRollout(ctx, r.Client, ref.Kind, ref.Name, ref.Namespace, newHash); err != nil {
			logger.Error(err, "failed to trigger rollout restart",
				"kind", ref.Kind, "name", ref.Name, "namespace", ref.Namespace)
			meta.SetStatusCondition(&oidcClient.Status.Conditions, metav1.Condition{
				Type:               "RolloutTriggered",
				Status:             metav1.ConditionFalse,
				Reason:             "RolloutFailed",
				Message:            fmt.Sprintf("Failed to trigger rollout: %v", err),
				ObservedGeneration: oidcClient.Generation,
			})
		} else {
			meta.SetStatusCondition(&oidcClient.Status.Conditions, metav1.Condition{
				Type:               "RolloutTriggered",
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
		Type:               "SecretSynced",
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

func toByteMap(data map[string]string) map[string][]byte {
	result := make(map[string][]byte, len(data))
	for k, v := range data {
		result[k] = []byte(v)
	}
	return result
}

// SetupWithManager sets up the controller with the Manager
func (r *OIDCClientReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&authv1alpha1.OIDCClient{}).
		Complete(r)
}
