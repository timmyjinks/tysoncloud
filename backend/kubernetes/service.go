package kubernetes

import (
	"context"

	"github.com/timmyjinks/tysoncloud/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	appcorev1 "k8s.io/client-go/applyconfigurations/core/v1"
	appmetav1 "k8s.io/client-go/applyconfigurations/meta/v1"
)

func (d *KubernetesService) CreateService(ctx context.Context, resource Resource) error {
	_, err := d.clientset.CoreV1().Services(resource.Namespace).Apply(ctx, &appcorev1.ServiceApplyConfiguration{
		TypeMetaApplyConfiguration: appmetav1.TypeMetaApplyConfiguration{
			Kind:       util.StringPtr("Service"),
			APIVersion: util.StringPtr("v1"),
		},
		ObjectMetaApplyConfiguration: &appmetav1.ObjectMetaApplyConfiguration{
			Name: &resource.Name,
			Labels: map[string]string{
				"app": resource.Name,
			},
		},
		Spec: &appcorev1.ServiceSpecApplyConfiguration{
			Ports: []appcorev1.ServicePortApplyConfiguration{
				{
					Protocol:   (*corev1.Protocol)(util.StringPtr(string(corev1.ProtocolTCP))),
					Port:       util.IntPtr(80),
					TargetPort: &intstr.IntOrString{IntVal: resource.Port},
				},
			},
			Selector: map[string]string{
				"app": resource.Name,
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

func (d *KubernetesService) DeleteService(ctx context.Context, resource Resource) error {
	return d.clientset.CoreV1().Services(resource.Namespace).Delete(ctx, resource.Name, metav1.DeleteOptions{})
}
