package bootstrap

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kettleofketchup/AuthentikOperator/internal/authentik"
)

// Config holds bootstrap configuration
type Config struct {
	AuthentikURL    string
	BootstrapToken  string // Bearer token (AUTHENTIK_BOOTSTRAP_TOKEN)
	TokenIdentifier string
	TokenSecretName string
	Namespace       string
	ClientOpts      []authentik.ClientOption
}

// Run executes the bootstrap flow:
// 1. Check if K8s secret already exists (skip if so)
// 2. Create API token in Authentik using Bearer auth
// 3. Retrieve the token key via view_key endpoint
// 4. Write token to K8s Secret
func Run(ctx context.Context, c client.Client, cfg Config) error {
	log := ctrl.LoggerFrom(ctx)
	// Check if secret already exists
	existing := &corev1.Secret{}
	err := c.Get(ctx, types.NamespacedName{Name: cfg.TokenSecretName, Namespace: cfg.Namespace}, existing)
	if err == nil {
		log.Info("Token secret already exists, skipping bootstrap")
		return nil
	}
	if !errors.IsNotFound(err) {
		return fmt.Errorf("checking existing secret: %w", err)
	}

	// Create API token in Authentik using Bearer auth (NOT Basic — Authentik does not support Basic)
	authentikClient := authentik.NewClient(cfg.AuthentikURL, cfg.BootstrapToken, cfg.ClientOpts...)
	tokenKey, err := authentikClient.CreateAPIToken(ctx, cfg.TokenIdentifier)
	if err != nil {
		return fmt.Errorf("creating Authentik API token: %w", err)
	}

	// Write token to K8s Secret
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cfg.TokenSecretName,
			Namespace: cfg.Namespace,
			Labels: map[string]string{
				"auth.kettleofketchup/managed-by": "authentik-operator",
				"auth.kettleofketchup/component":  "bootstrap",
			},
			Annotations: map[string]string{
				// Prevent ArgoCD from pruning this out-of-band secret
				"argocd.argoproj.io/compare-options": "IgnoreExtraneous",
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"token": []byte(tokenKey),
		},
	}

	if err := c.Create(ctx, secret); err != nil {
		return fmt.Errorf("creating token secret: %w", err)
	}

	log.Info("Bootstrap complete", "secret", cfg.Namespace+"/"+cfg.TokenSecretName)
	return nil
}
