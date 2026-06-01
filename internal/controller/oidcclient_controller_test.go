package controller

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	authv1alpha1 "github.com/kettleofketchup/AuthentikOperator/api/v1alpha1"
)

var _ = Describe("OIDCClient Controller", func() {
	const (
		timeout  = time.Second * 10
		interval = time.Millisecond * 250
	)

	BeforeEach(func() {
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: "test-target"},
		}
		_ = k8sClient.Create(ctx, ns)
	})

	Context("When creating an OIDCClient with grafana profile", func() {
		It("Should create a secret in the target namespace", func() {
			cr := &authv1alpha1.OIDCClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-grafana",
					Namespace: "default",
				},
				Spec: authv1alpha1.OIDCClientSpec{
					Authentik: authv1alpha1.AuthentikSource{
						ApplicationSlug: "grafana",
					},
					Target: authv1alpha1.SecretTarget{
						Namespace:  "test-target",
						SecretName: "grafana-oauth",
					},
					SecretProfile: "grafana",
				},
			}
			Expect(k8sClient.Create(ctx, cr)).Should(Succeed())

			secretKey := types.NamespacedName{
				Name:      "grafana-oauth",
				Namespace: "test-target",
			}
			createdSecret := &corev1.Secret{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, secretKey, createdSecret)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(createdSecret.Data).To(HaveKey("GF_AUTH_GENERIC_OAUTH_CLIENT_ID"))
			Expect(createdSecret.Data).To(HaveKey("GF_AUTH_GENERIC_OAUTH_CLIENT_SECRET"))
			Expect(createdSecret.Data).To(HaveKey("GF_AUTH_GENERIC_OAUTH_AUTH_URL"))
			Expect(createdSecret.Data).To(HaveKey("GF_AUTH_GENERIC_OAUTH_ENABLED"))

			Expect(createdSecret.Labels).To(HaveKeyWithValue(
				"auth.kettleofketchup/managed-by", "authentik-operator"))

			fetchedCR := &authv1alpha1.OIDCClient{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name: "test-grafana", Namespace: "default",
				}, fetchedCR)
				if err != nil {
					return false
				}
				for _, c := range fetchedCR.Status.Conditions {
					if c.Type == "SecretSynced" && c.Status == metav1.ConditionTrue {
						return true
					}
				}
				return false
			}, timeout, interval).Should(BeTrue())

			Expect(fetchedCR.Status.SecretHash).ToNot(BeEmpty())
			Expect(fetchedCR.Status.LastSyncTime).ToNot(BeNil())
		})
	})

	Context("When Authentik provider is not found", func() {
		It("Should set AuthentikProviderFound condition to False", func() {
			cr := &authv1alpha1.OIDCClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-missing",
					Namespace: "default",
				},
				Spec: authv1alpha1.OIDCClientSpec{
					Authentik: authv1alpha1.AuthentikSource{
						ApplicationSlug: "nonexistent-app",
					},
					Target: authv1alpha1.SecretTarget{
						Namespace:  "test-target",
						SecretName: "missing-oauth",
					},
					SecretProfile: "generic",
				},
			}
			Expect(k8sClient.Create(ctx, cr)).Should(Succeed())

			fetchedCR := &authv1alpha1.OIDCClient{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name: "test-missing", Namespace: "default",
				}, fetchedCR)
				if err != nil {
					return false
				}
				for _, c := range fetchedCR.Status.Conditions {
					if c.Type == "AuthentikProviderFound" && c.Status == metav1.ConditionFalse {
						return true
					}
				}
				return false
			}, timeout, interval).Should(BeTrue())
		})
	})

	Context("When secretOverrides are specified", func() {
		It("Should merge overrides into the secret", func() {
			cr := &authv1alpha1.OIDCClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-overrides",
					Namespace: "default",
				},
				Spec: authv1alpha1.OIDCClientSpec{
					Authentik: authv1alpha1.AuthentikSource{
						ApplicationSlug: "grafana",
					},
					Target: authv1alpha1.SecretTarget{
						Namespace:  "test-target",
						SecretName: "grafana-oauth-overrides",
					},
					SecretProfile: "grafana",
					SecretOverrides: map[string]string{
						"GF_AUTH_GENERIC_OAUTH_ROLE_ATTRIBUTE_PATH": "contains(groups, 'admins') && 'Admin'",
						"CUSTOM_KEY": "custom-value",
					},
				},
			}
			Expect(k8sClient.Create(ctx, cr)).Should(Succeed())

			secretKey := types.NamespacedName{
				Name:      "grafana-oauth-overrides",
				Namespace: "test-target",
			}
			createdSecret := &corev1.Secret{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, secretKey, createdSecret)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(string(createdSecret.Data["GF_AUTH_GENERIC_OAUTH_ROLE_ATTRIBUTE_PATH"])).To(
				Equal("contains(groups, 'admins') && 'Admin'"))
			Expect(string(createdSecret.Data["CUSTOM_KEY"])).To(Equal("custom-value"))
		})
	})

	// Regression: once a secret reaches steady state (hash unchanged), a
	// stale AuthentikProviderFound=False condition (e.g. written during a
	// transient Authentik outage) must self-heal back to True on the next
	// reconcile. The original code returned early on the hash match WITHOUT
	// persisting the recomputed condition, so the status stayed False forever.
	Context("When the secret is already in sync but the provider-found condition is stale", func() {
		It("Should re-persist AuthentikProviderFound=True on the next reconcile", func() {
			crKey := types.NamespacedName{Name: "test-stale-cond", Namespace: "default"}
			cr := &authv1alpha1.OIDCClient{
				ObjectMeta: metav1.ObjectMeta{Name: crKey.Name, Namespace: crKey.Namespace},
				Spec: authv1alpha1.OIDCClientSpec{
					Authentik:     authv1alpha1.AuthentikSource{ApplicationSlug: "grafana"},
					Target:        authv1alpha1.SecretTarget{Namespace: "test-target", SecretName: "stale-cond-oauth"},
					SecretProfile: "grafana",
				},
			}
			Expect(k8sClient.Create(ctx, cr)).Should(Succeed())

			// Wait for steady state: secret synced + SecretHash recorded.
			fetched := &authv1alpha1.OIDCClient{}
			Eventually(func() bool {
				if err := k8sClient.Get(ctx, crKey, fetched); err != nil {
					return false
				}
				return fetched.Status.SecretHash != ""
			}, timeout, interval).Should(BeTrue())

			// Simulate a stale condition from a past transient failure:
			// force AuthentikProviderFound=False while the secret (and hash)
			// stay in sync.
			meta.SetStatusCondition(&fetched.Status.Conditions, metav1.Condition{
				Type:    ConditionAuthentikProviderFound,
				Status:  metav1.ConditionFalse,
				Reason:  "ProviderNotFound",
				Message: "stale failure from a past outage",
			})
			Expect(k8sClient.Status().Update(ctx, fetched)).Should(Succeed())

			// Trigger a reconcile without changing the secret: a metadata
			// touch is enough (the hash will still match on reconcile).
			Eventually(func() error {
				if err := k8sClient.Get(ctx, crKey, fetched); err != nil {
					return err
				}
				if fetched.Annotations == nil {
					fetched.Annotations = map[string]string{}
				}
				fetched.Annotations["test.kettleofketchup/touch"] = "1"
				return k8sClient.Update(ctx, fetched)
			}, timeout, interval).Should(Succeed())

			// The condition must self-heal back to True even though the
			// secret hash is unchanged.
			Eventually(func() bool {
				if err := k8sClient.Get(ctx, crKey, fetched); err != nil {
					return false
				}
				c := meta.FindStatusCondition(fetched.Status.Conditions, ConditionAuthentikProviderFound)
				return c != nil && c.Status == metav1.ConditionTrue
			}, timeout, interval).Should(BeTrue(),
				"AuthentikProviderFound should return to True on a no-secret-change reconcile")
		})
	})
})
