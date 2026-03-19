package controller

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
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
})
