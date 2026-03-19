package rollout

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const annotationKey = "auth.kettleofketchup/secret-hash"

// TriggerRollout patches the pod template annotation on a Deployment or StatefulSet
// to trigger a rolling restart.
func TriggerRollout(ctx context.Context, c client.Client, kind, name, namespace, secretHash string) error {
	key := keyFor(name, namespace)

	switch kind {
	case "Deployment":
		deploy := &appsv1.Deployment{}
		if err := c.Get(ctx, key, deploy); err != nil {
			return fmt.Errorf("getting deployment %s/%s: %w", namespace, name, err)
		}
		if deploy.Spec.Template.Annotations == nil {
			deploy.Spec.Template.Annotations = make(map[string]string)
		}
		deploy.Spec.Template.Annotations[annotationKey] = secretHash
		if err := c.Update(ctx, deploy); err != nil {
			return fmt.Errorf("updating deployment %s/%s: %w", namespace, name, err)
		}
		return nil

	case "StatefulSet":
		sts := &appsv1.StatefulSet{}
		if err := c.Get(ctx, key, sts); err != nil {
			return fmt.Errorf("getting statefulset %s/%s: %w", namespace, name, err)
		}
		if sts.Spec.Template.Annotations == nil {
			sts.Spec.Template.Annotations = make(map[string]string)
		}
		sts.Spec.Template.Annotations[annotationKey] = secretHash
		if err := c.Update(ctx, sts); err != nil {
			return fmt.Errorf("updating statefulset %s/%s: %w", namespace, name, err)
		}
		return nil

	default:
		return fmt.Errorf("unsupported kind %q, must be Deployment or StatefulSet", kind)
	}
}

func keyFor(name, namespace string) types.NamespacedName {
	return types.NamespacedName{Name: name, Namespace: namespace}
}
