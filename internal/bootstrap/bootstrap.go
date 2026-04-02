package bootstrap

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	applyconfigscorev1 "k8s.io/client-go/applyconfigurations/core/v1"
	k8sscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kettleofketchup/AuthentikOperator/internal/authentik"
)

// Config holds bootstrap configuration.
type Config struct {
	AuthentikURL        string
	BootstrapToken      string // Bearer token (AUTHENTIK_BOOTSTRAP_TOKEN)
	TokenIdentifier     string
	TokenSecretName     string
	Namespace           string
	ClientOpts          []authentik.ClientOption
	ForceRefresh        bool   // Delete existing secret before re-creating
	RestConfig          *rest.Config
	AuthentikNamespace  string // Namespace where Authentik server pods run (default "authentik")
}

// readinessTimeout is the maximum time Run will wait for Authentik to become
// ready before attempting token creation.
const readinessTimeout = 5 * time.Minute

// authentikPodSelector is the label selector for Authentik server pods.
const authentikPodSelector = "app.kubernetes.io/name=authentik,app.kubernetes.io/component=server"

// akShellCommand is the Python one-liner executed via `ak shell -c` to
// idempotently create or retrieve an API token for the operator.
const akShellCommand = `from authentik.core.models import Token, TokenIntents; from django.contrib.auth import get_user_model; User = get_user_model(); user = User.objects.get(username='akadmin'); token, created = Token.objects.get_or_create(identifier='authentik-operator', defaults={'user': user, 'intent': TokenIntents.INTENT_API, 'expiring': False}); print(token.key)`

// execGetToken execs into a running Authentik server pod and retrieves or
// creates the operator API token using the `ak shell` Django management command.
// This bypasses the HTTP API entirely, avoiding bootstrap-token/DB mismatch issues.
func execGetToken(ctx context.Context, restCfg *rest.Config, authentikNamespace string) (string, error) {
	log := ctrl.LoggerFrom(ctx)

	clientset, err := kubernetes.NewForConfig(restCfg)
	if err != nil {
		return "", fmt.Errorf("creating kubernetes clientset: %w", err)
	}

	// List running Authentik server pods.
	pods, err := clientset.CoreV1().Pods(authentikNamespace).List(ctx, metav1.ListOptions{
		LabelSelector: authentikPodSelector,
	})
	if err != nil {
		return "", fmt.Errorf("listing authentik pods in namespace %q: %w", authentikNamespace, err)
	}

	// Find the first Running pod.
	var targetPod string
	for _, pod := range pods.Items {
		if pod.Status.Phase == corev1.PodRunning {
			targetPod = pod.Name
			break
		}
	}
	if targetPod == "" {
		return "", fmt.Errorf("no running authentik server pods found in namespace %q (selector: %s)", authentikNamespace, authentikPodSelector)
	}

	log.Info("Executing ak shell in authentik pod", "pod", targetPod, "namespace", authentikNamespace)

	// Build the exec request.
	req := clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(targetPod).
		Namespace(authentikNamespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Command: []string{"ak", "shell", "-c", akShellCommand},
			Stdout:  true,
			Stderr:  true,
		}, k8sscheme.ParameterCodec)

	executor, err := remotecommand.NewSPDYExecutor(restCfg, "POST", req.URL())
	if err != nil {
		return "", fmt.Errorf("creating SPDY executor: %w", err)
	}

	var stdout, stderr bytes.Buffer
	if err := executor.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdout: &stdout,
		Stderr: &stderr,
	}); err != nil {
		return "", fmt.Errorf("exec stream error (stderr: %s): %w", strings.TrimSpace(stderr.String()), err)
	}

	tokenKey := strings.TrimSpace(stdout.String())
	if tokenKey == "" {
		return "", fmt.Errorf("ak shell returned empty output (stderr: %s)", strings.TrimSpace(stderr.String()))
	}

	log.Info("Token retrieved via ak shell exec")
	return tokenKey, nil
}

// Run executes the bootstrap flow:
//  1. Check if K8s secret already exists (skip if so)
//  2. Wait for Authentik to be ready (polls /api/v3/root/config/)
//  3. Try exec-based token creation via `ak shell` (primary path)
//  4. Fall back to HTTP API token creation if exec fails
//  5. Write token to K8s Secret
func Run(ctx context.Context, c client.Client, cfg Config) error {
	log := ctrl.LoggerFrom(ctx)

	// Check if secret already exists — skip early without waiting for Authentik.
	existing := &corev1.Secret{}
	err := c.Get(ctx, types.NamespacedName{Name: cfg.TokenSecretName, Namespace: cfg.Namespace}, existing)
	if err == nil {
		if !cfg.ForceRefresh {
			log.Info("Token secret already exists, skipping bootstrap")
			return nil
		}
		log.Info("Force refresh: deleting stale token secret", "secret", cfg.TokenSecretName)
		if err := c.Delete(ctx, existing); err != nil {
			return fmt.Errorf("deleting stale token secret: %w", err)
		}
	} else if !errors.IsNotFound(err) {
		return fmt.Errorf("checking existing secret: %w", err)
	}

	// Wait for Authentik to finish initializing before making any calls.
	// /api/v3/root/config/ is a public endpoint that returns 200 only when
	// Authentik is fully up. Without this wait, bootstrap hits 403/503 during
	// early startup.
	authentikClient := authentik.NewClient(cfg.AuthentikURL, cfg.BootstrapToken, cfg.ClientOpts...)
	log.Info("Waiting for Authentik to be ready", "timeout", readinessTimeout.String())
	if err := authentikClient.WaitForReady(ctx, readinessTimeout); err != nil {
		return fmt.Errorf("waiting for Authentik readiness: %w", err)
	}
	log.Info("Authentik is ready, proceeding with bootstrap")

	// Resolve the authentik namespace, defaulting to "authentik".
	authentikNS := cfg.AuthentikNamespace
	if authentikNS == "" {
		authentikNS = "authentik"
	}

	// Primary path: exec into Authentik pod and run ak shell.
	// This is idempotent (get_or_create) and bypasses HTTP API auth entirely.
	var tokenKey string
	if cfg.RestConfig != nil {
		tokenKey, err = execGetToken(ctx, cfg.RestConfig, authentikNS)
		if err != nil {
			log.Info("exec-based token creation failed, falling back to HTTP API", "error", err.Error())
		}
	} else {
		log.Info("RestConfig not set, skipping exec path and using HTTP API")
	}

	// Fallback path: HTTP API with Bearer auth.
	if tokenKey == "" {
		tokenKey, err = authentikClient.CreateAPIToken(ctx, cfg.TokenIdentifier)
		if err != nil {
			return fmt.Errorf("creating Authentik API token via HTTP: %w", err)
		}
	}

	// Write token to K8s Secret using server-side apply.
	secret := applyconfigscorev1.Secret(cfg.TokenSecretName, cfg.Namespace).
		WithLabels(map[string]string{
			"auth.kettleofketchup/managed-by": "authentik-operator",
			"auth.kettleofketchup/component":  "bootstrap",
		}).
		WithAnnotations(map[string]string{
			"argocd.argoproj.io/compare-options": "IgnoreExtraneous",
		}).
		WithData(map[string][]byte{
			"token": []byte(tokenKey),
		})

	if err := c.Apply(ctx, secret, client.FieldOwner("authentik-operator"), client.ForceOwnership); err != nil {
		return fmt.Errorf("applying token secret: %w", err)
	}

	log.Info("Bootstrap complete", "secret", cfg.Namespace+"/"+cfg.TokenSecretName)
	return nil
}
