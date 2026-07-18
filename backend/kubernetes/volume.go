package kubernetes

import (
	"context"
	"fmt"

	"github.com/timmyjinks/tysoncloud/util"
	corev1 "k8s.io/api/core/v1"
	resourcev1 "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	appcorev1 "k8s.io/client-go/applyconfigurations/core/v1"
	appmetav1 "k8s.io/client-go/applyconfigurations/meta/v1"
)

func (d *KubernetesService) CreatePVC(ctx context.Context, resource Resource) error {
	_, err := d.clientset.CoreV1().PersistentVolumeClaims(resource.Namespace).Apply(ctx, &appcorev1.PersistentVolumeClaimApplyConfiguration{
		TypeMetaApplyConfiguration: appmetav1.TypeMetaApplyConfiguration{
			Kind:       util.StringPtr("PersistentVolumeClaim"),
			APIVersion: util.StringPtr("v1"),
		},
		ObjectMetaApplyConfiguration: &appmetav1.ObjectMetaApplyConfiguration{
			Name:      &resource.Name,
			Namespace: &resource.Namespace,
		},
		Spec: &appcorev1.PersistentVolumeClaimSpecApplyConfiguration{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			Resources: &appcorev1.VolumeResourceRequirementsApplyConfiguration{
				Requests: &corev1.ResourceList{
					corev1.ResourceStorage: resourcev1.MustParse(fmt.Sprintf("%dGi", resource.StorageGB)),
				},
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

func (d *KubernetesService) DeletePVC(ctx context.Context, resource Resource) error {
	return d.clientset.CoreV1().PersistentVolumeClaims(resource.Namespace).Delete(ctx, resource.Name, metav1.DeleteOptions{})
}
