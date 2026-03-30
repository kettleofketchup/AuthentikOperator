package bootstrap

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	applyconfigscorev1 "k8s.io/client-go/applyconfigurations/core/v1"
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
	ForceRefresh    bool // Delete existing secret before re-creating
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

	// Create API token in Authentik using Bearer auth (NOT Basic — Authentik does not support Basic)
	authentikClient := authentik.NewClient(cfg.AuthentikURL, cfg.BootstrapToken, cfg.ClientOpts...)
	tokenKey, err := authentikClient.CreateAPIToken(ctx, cfg.TokenIdentifier)
	if err != nil {
		return fmt.Errorf("creating Authentik API token: %w", err)
	}

	// Write token to K8s Secret using server-side apply
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
