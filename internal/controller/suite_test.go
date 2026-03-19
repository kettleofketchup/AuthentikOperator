package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	authv1alpha1 "github.com/kettleofketchup/AuthentikOperator/api/v1alpha1"
	"github.com/kettleofketchup/AuthentikOperator/internal/authentik"
)

var (
	cfg                 *rest.Config
	k8sClient           client.Client
	testEnv             *envtest.Environment
	ctx                 context.Context
	cancel              context.CancelFunc
	mockAuthentikServer *httptest.Server
)

func TestControllers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	ctx, cancel = context.WithCancel(context.TODO())

	// Mock Authentik API with two-step lookup
	mockAuthentikServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/v3/core/applications/grafana/":
			// Step 1: Return application with PK
			resp := authentik.ApplicationResponse{PK: "a1b2c3d4-0001-0000-0000-000000000001", Slug: "grafana", Name: "Grafana"}
			_ = json.NewEncoder(w).Encode(resp)

		case "/api/v3/providers/oauth2/":
			// Step 2: Return providers filtered by application PK
			if r.URL.Query().Get("application") == "a1b2c3d4-0001-0000-0000-000000000001" {
				resp := authentik.ProviderListResponse{
					Pagination: authentik.Pagination{Count: 1},
					Results: []authentik.OAuth2Provider{
						{
							PK:                      1,
							Name:                    "grafana-oidc",
							ClientID:                "mock-client-id",
							ClientSecret:            "mock-client-secret",
							ClientType:              "confidential",
							AssignedApplicationSlug: "grafana",
							AssignedApplicationName: "Grafana",
						},
					},
				}
				_ = json.NewEncoder(w).Encode(resp)
			} else {
				resp := authentik.ProviderListResponse{
					Pagination: authentik.Pagination{Count: 0},
					Results:    []authentik.OAuth2Provider{},
				}
				_ = json.NewEncoder(w).Encode(resp)
			}

		default:
			// Unknown applications return 404
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"detail":"Not found."}`))
		}
	}))

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
		BinaryAssetsDirectory: filepath.Join("..", "..", "bin", "k8s",
			fmt.Sprintf("1.31.0-%s-%s", runtime.GOOS, runtime.GOARCH)),
	}

	var err error
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = authv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	Expect(err).ToNot(HaveOccurred())

	authentikClient := authentik.NewClient(mockAuthentikServer.URL, "mock-token")

	err = (&OIDCClientReconciler{
		Client:          k8sManager.GetClient(),
		Scheme:          k8sManager.GetScheme(),
		AuthentikClient: authentikClient,
		AuthentikURL:    mockAuthentikServer.URL,
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	go func() {
		defer GinkgoRecover()
		err = k8sManager.Start(ctx)
		Expect(err).ToNot(HaveOccurred(), "failed to run manager")
	}()
})

var _ = AfterSuite(func() {
	cancel()
	if mockAuthentikServer != nil {
		mockAuthentikServer.Close()
	}
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})
