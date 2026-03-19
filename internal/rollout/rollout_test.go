package rollout

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestTriggerRollout_Deployment(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = appsv1.AddToScheme(scheme)

	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deploy",
			Namespace: "default",
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "test"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": "test"},
				},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(deploy).Build()

	err := TriggerRollout(context.Background(), fakeClient, "Deployment", "test-deploy", "default", "sha256:abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updatedDeploy := &appsv1.Deployment{}
	err = fakeClient.Get(context.Background(), keyFor("test-deploy", "default"), updatedDeploy)
	if err != nil {
		t.Fatalf("failed to get deployment: %v", err)
	}

	ann := updatedDeploy.Spec.Template.Annotations
	if ann == nil {
		t.Fatal("expected annotations on pod template")
	}
	if ann["auth.kettleofketchup/secret-hash"] != "sha256:abc123" {
		t.Errorf("expected hash annotation, got %v", ann)
	}
}

func TestTriggerRollout_NotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = appsv1.AddToScheme(scheme)

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	err := TriggerRollout(context.Background(), fakeClient, "Deployment", "missing", "default", "sha256:abc")
	if err == nil {
		t.Fatal("expected error for missing deployment")
	}
}
