package bootstrap

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/kettleofketchup/AuthentikOperator/internal/authentik"
)

func TestBootstrap(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Verify Bearer auth (NOT Basic auth)
		auth := r.Header.Get("Authorization")
		if auth != "Bearer bootstrap-token-123" {
			t.Errorf("expected Bearer bootstrap-token-123, got %s", auth)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		switch {
		case r.URL.Path == "/api/v3/core/tokens/" && r.Method == http.MethodPost:
			resp := authentik.TokenCreateResponse{
				PK:         "uuid-1",
				Identifier: "authentik-operator",
			}
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(resp)

		case r.URL.Path == "/api/v3/core/tokens/authentik-operator/view_key/" && r.Method == http.MethodGet:
			resp := authentik.TokenViewKeyResponse{Key: "new-api-key-123"}
			_ = json.NewEncoder(w).Encode(resp)

		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "operator-ns"}}
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(ns).Build()

	testLog := zap.New(zap.UseDevMode(true))
	ctx := ctrl.LoggerInto(context.Background(), testLog)
	err := Run(ctx, fakeClient, Config{
		AuthentikURL:    server.URL,
		BootstrapToken:  "bootstrap-token-123",
		TokenIdentifier: "authentik-operator",
		TokenSecretName: "authentik-operator-token",
		Namespace:       "operator-ns",
	})
	if err != nil {
		t.Fatalf("bootstrap failed: %v", err)
	}

	secret := &corev1.Secret{}
	err = fakeClient.Get(context.Background(), types.NamespacedName{
		Name: "authentik-operator-token", Namespace: "operator-ns",
	}, secret)
	if err != nil {
		t.Fatalf("secret not found: %v", err)
	}

	if string(secret.Data["token"]) != "new-api-key-123" {
		t.Errorf("expected token new-api-key-123, got %s", string(secret.Data["token"]))
	}
}

func TestBootstrap_SecretAlreadyExists(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	existingSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "authentik-operator-token",
			Namespace: "operator-ns",
		},
		Data: map[string][]byte{"token": []byte("existing-token")},
	}
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "operator-ns"}}
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(ns, existingSecret).Build()

	testLog := zap.New(zap.UseDevMode(true))
	ctx := ctrl.LoggerInto(context.Background(), testLog)
	err := Run(ctx, fakeClient, Config{
		AuthentikURL:    "http://unused",
		BootstrapToken:  "unused",
		TokenIdentifier: "authentik-operator",
		TokenSecretName: "authentik-operator-token",
		Namespace:       "operator-ns",
	})
	if err != nil {
		t.Fatalf("bootstrap should succeed (skip) when secret exists: %v", err)
	}
}
