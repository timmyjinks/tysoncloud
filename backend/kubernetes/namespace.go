package kubernetes

import (
	"context"

	"github.com/timmyjinks/tysoncloud/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	appcorev1 "k8s.io/client-go/applyconfigurations/core/v1"
	appmetav1 "k8s.io/client-go/applyconfigurations/meta/v1"
)

func (d *KubernetesService) CreateNamespace(ctx context.Context, namespace string) error {
	_, err := d.clientset.CoreV1().Namespaces().Apply(ctx, &appcorev1.NamespaceApplyConfiguration{
		TypeMetaApplyConfiguration: appmetav1.TypeMetaApplyConfiguration{
			APIVersion: util.StringPtr("v1"),
			Kind:       util.StringPtr("Namespace"),
		},
		ObjectMetaApplyConfiguration: &appmetav1.ObjectMetaApplyConfiguration{
			Name: util.StringPtr(namespace),
			Labels: map[string]string{
				"managed-by": "tysoncloud",
				// Pod Security Admission: block privileged/host-level pods
				// in tenant namespaces, and surface anything that misses
				// the stricter "restricted" profile.
				"pod-security.kubernetes.io/enforce": "baseline",
				"pod-security.kubernetes.io/warn":    "restricted",
				"pod-security.kubernetes.io/audit":   "restricted",
			},
		},
	}, metav1.ApplyOptions{
		FieldManager: "tysoncloud",
	})
	if err != nil {
		return err
	}
	return nil
}

func (d *KubernetesService) DeleteNamespace(ctx context.Context, namespace string) error {
	return d.clientset.CoreV1().Namespaces().Delete(ctx, namespace, metav1.DeleteOptions{})
}

// ListManagedNamespaces returns every namespace created by tysoncloud
// (project and preview namespaces).
func (d *KubernetesService) ListManagedNamespaces(ctx context.Context) ([]string, error) {
	list, err := d.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{
		LabelSelector: "managed-by=tysoncloud",
	})
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(list.Items))
	for _, ns := range list.Items {
		names = append(names, ns.Name)
	}
	return names, nil
}
