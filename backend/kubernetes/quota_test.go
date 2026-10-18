package kubernetes

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestTenantQuotaCoversComputeAndStorage(t *testing.T) {
	for _, name := range []corev1.ResourceName{
		corev1.ResourceLimitsCPU,
		corev1.ResourceLimitsMemory,
		corev1.ResourceRequestsStorage,
		corev1.ResourcePods,
		corev1.ResourcePersistentVolumeClaims,
	} {
		if q, ok := tenantQuota[name]; !ok || q.IsZero() {
			t.Errorf("tenant quota missing %s", name)
		}
	}
}
