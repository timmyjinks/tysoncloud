package kubernetes

import (
	"context"

	"github.com/timmyjinks/tysoncloud/util"
	corev1 "k8s.io/api/core/v1"
	resourcev1 "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	appcorev1 "k8s.io/client-go/applyconfigurations/core/v1"
	appmetav1 "k8s.io/client-go/applyconfigurations/meta/v1"
)

const tenantLimitsName = "tysoncloud-tenant-limits"

// tenantQuota caps what a single project namespace can consume so one tenant
// can't exhaust the cluster.
var tenantQuota = corev1.ResourceList{
	corev1.ResourceLimitsCPU:              resourcev1.MustParse("8"),
	corev1.ResourceLimitsMemory:           resourcev1.MustParse("16Gi"),
	corev1.ResourceRequestsCPU:            resourcev1.MustParse("4"),
	corev1.ResourceRequestsMemory:         resourcev1.MustParse("8Gi"),
	corev1.ResourceRequestsStorage:        resourcev1.MustParse("100Gi"),
	corev1.ResourcePods:                   resourcev1.MustParse("30"),
	corev1.ResourcePersistentVolumeClaims: resourcev1.MustParse("20"),
	corev1.ResourceServices:               resourcev1.MustParse("30"),
	corev1.ResourceSecrets:                resourcev1.MustParse("100"),
}

func (d *KubernetesService) CreateResourceQuota(ctx context.Context, namespace string) error {
	_, err := d.clientset.CoreV1().ResourceQuotas(namespace).Apply(ctx, &appcorev1.ResourceQuotaApplyConfiguration{
		TypeMetaApplyConfiguration: appmetav1.TypeMetaApplyConfiguration{
			APIVersion: util.StringPtr("v1"),
			Kind:       util.StringPtr("ResourceQuota"),
		},
		ObjectMetaApplyConfiguration: &appmetav1.ObjectMetaApplyConfiguration{
			Name:      util.StringPtr(tenantLimitsName),
			Namespace: &namespace,
		},
		Spec: &appcorev1.ResourceQuotaSpecApplyConfiguration{
			Hard: &tenantQuota,
		},
	}, metav1.ApplyOptions{
		FieldManager: "tysoncloud",
	})
	return err
}

// CreateLimitRange gives containers without explicit resources (e.g. CNPG
// instances) defaults, which the quota above requires.
func (d *KubernetesService) CreateLimitRange(ctx context.Context, namespace string) error {
	containerType := corev1.LimitTypeContainer
	_, err := d.clientset.CoreV1().LimitRanges(namespace).Apply(ctx, &appcorev1.LimitRangeApplyConfiguration{
		TypeMetaApplyConfiguration: appmetav1.TypeMetaApplyConfiguration{
			APIVersion: util.StringPtr("v1"),
			Kind:       util.StringPtr("LimitRange"),
		},
		ObjectMetaApplyConfiguration: &appmetav1.ObjectMetaApplyConfiguration{
			Name:      util.StringPtr(tenantLimitsName),
			Namespace: &namespace,
		},
		Spec: &appcorev1.LimitRangeSpecApplyConfiguration{
			Limits: []appcorev1.LimitRangeItemApplyConfiguration{
				{
					Type: &containerType,
					Default: &corev1.ResourceList{
						corev1.ResourceCPU:    resourcev1.MustParse("500m"),
						corev1.ResourceMemory: resourcev1.MustParse("1Gi"),
					},
					DefaultRequest: &corev1.ResourceList{
						corev1.ResourceCPU:    resourcev1.MustParse("100m"),
						corev1.ResourceMemory: resourcev1.MustParse("128Mi"),
					},
					Max: &corev1.ResourceList{
						corev1.ResourceCPU:    resourcev1.MustParse("2"),
						corev1.ResourceMemory: resourcev1.MustParse("4Gi"),
					},
				},
			},
		},
	}, metav1.ApplyOptions{
		FieldManager: "tysoncloud",
	})
	return err
}
