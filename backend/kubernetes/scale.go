package kubernetes

import (
	"context"

	"github.com/timmyjinks/tysoncloud/util"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v2 "k8s.io/client-go/applyconfigurations/autoscaling/v2"
	appmetav1 "k8s.io/client-go/applyconfigurations/meta/v1"
)

func (d *KubernetesService) CreateHPA(ctx context.Context, resource Resource) error {
	_, err := d.clientset.AutoscalingV2().HorizontalPodAutoscalers(resource.Namespace).Apply(ctx, &v2.HorizontalPodAutoscalerApplyConfiguration{
		TypeMetaApplyConfiguration: appmetav1.TypeMetaApplyConfiguration{
			Kind:       util.StringPtr("HorizontalPodAutoscaler"),
			APIVersion: util.StringPtr("autoscaling/v2"),
		},
		ObjectMetaApplyConfiguration: &appmetav1.ObjectMetaApplyConfiguration{
			Name: &resource.Name,
			Labels: map[string]string{
				"app.kubernetes.io/component": "service",
			},
		},
		Spec: &v2.HorizontalPodAutoscalerSpecApplyConfiguration{
			ScaleTargetRef: &v2.CrossVersionObjectReferenceApplyConfiguration{
				Kind:       util.StringPtr("Deployment"),
				APIVersion: util.StringPtr("apps/v1"),
				Name:       &resource.Name,
			},
			MinReplicas: util.IntPtr(1),
			MaxReplicas: util.IntPtr(10),
			Metrics: []v2.MetricSpecApplyConfiguration{
				{
					Type: (*autoscalingv2.MetricSourceType)(util.StringPtr(string(autoscalingv2.ResourceMetricSourceType))),
					Resource: &v2.ResourceMetricSourceApplyConfiguration{
						Name: (*corev1.ResourceName)(util.StringPtr(string(corev1.ResourceCPU))),
						Target: &v2.MetricTargetApplyConfiguration{
							Type:               (*autoscalingv2.MetricTargetType)(util.StringPtr(string(autoscalingv2.UtilizationMetricType))),
							AverageUtilization: util.IntPtr(50),
						},
					},
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

func (d *KubernetesService) DeleteHPA(ctx context.Context, resource Resource) error {
	return d.clientset.AutoscalingV1().HorizontalPodAutoscalers(resource.Namespace).Delete(ctx, resource.Name, metav1.DeleteOptions{})
}
