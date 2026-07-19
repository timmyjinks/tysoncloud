package kubernetes

import (
	"context"

	"github.com/timmyjinks/tysoncloud/util"
	corev1 "k8s.io/api/core/v1"
	resourcev1 "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	appsv1apply "k8s.io/client-go/applyconfigurations/apps/v1"
	appcorev1 "k8s.io/client-go/applyconfigurations/core/v1"
	appmetav1 "k8s.io/client-go/applyconfigurations/meta/v1"
)

func (d *KubernetesService) CreateDeployment(ctx context.Context, resource Resource) error {
	container := []appcorev1.ContainerApplyConfiguration{
		{
			Name:  &resource.Name,
			Image: &resource.Image,
			Resources: &appcorev1.ResourceRequirementsApplyConfiguration{
				Limits: &corev1.ResourceList{
					corev1.ResourceCPU:    resourcev1.MustParse("500m"),
					corev1.ResourceMemory: resourcev1.MustParse("256Mi"),
				},
				Requests: &corev1.ResourceList{
					corev1.ResourceCPU:    resourcev1.MustParse("100m"),
					corev1.ResourceMemory: resourcev1.MustParse("1Mi"),
				},
			},
			Ports: []appcorev1.ContainerPortApplyConfiguration{
				{
					Protocol:      (*corev1.Protocol)(util.StringPtr(string(corev1.ProtocolTCP))),
					ContainerPort: &resource.Port,
				},
			},
		},
	}

	if len(resource.Env) != 0 {
		container[0].EnvFrom = []appcorev1.EnvFromSourceApplyConfiguration{
			{
				SecretRef: &appcorev1.SecretEnvSourceApplyConfiguration{
					LocalObjectReferenceApplyConfiguration: appcorev1.LocalObjectReferenceApplyConfiguration{
						Name: &resource.Name,
					},
				},
			},
		}
	}

	spec := &appsv1apply.DeploymentSpecApplyConfiguration{
		Selector: &appmetav1.LabelSelectorApplyConfiguration{
			MatchLabels: map[string]string{
				"app": resource.Name,
			},
		},
		Template: &appcorev1.PodTemplateSpecApplyConfiguration{
			ObjectMetaApplyConfiguration: &appmetav1.ObjectMetaApplyConfiguration{
				Name: &resource.Name,
				Labels: map[string]string{
					"app": resource.Name,
				},
			},
			Spec: &appcorev1.PodSpecApplyConfiguration{
				Containers: container,
			},
		},
	}

	_, err := d.clientset.AppsV1().Deployments(resource.Namespace).Apply(ctx, &appsv1apply.DeploymentApplyConfiguration{
		TypeMetaApplyConfiguration: appmetav1.TypeMetaApplyConfiguration{
			Kind:       util.StringPtr("Deployment"),
			APIVersion: util.StringPtr("apps/v1"),
		},
		ObjectMetaApplyConfiguration: &appmetav1.ObjectMetaApplyConfiguration{
			Name: &resource.Name,
			Labels: map[string]string{
				"app": resource.Name,
			},
			Annotations: map[string]string{
				"reloader.stakater.com/auto": "true",
			},
		},
		Spec: spec,
	}, metav1.ApplyOptions{
		FieldManager: "tysoncloud",
	})
	if err != nil {
		return err
	}
	return nil
}

func (d *KubernetesService) DeleteDeployment(ctx context.Context, resource Resource) error {
	return d.clientset.AppsV1().Deployments(resource.Namespace).Delete(ctx, resource.Name, metav1.DeleteOptions{})
}

func (d *KubernetesService) attachPVCToDeployment(ctx context.Context, resource Resource) error {
	deployment, err := d.clientset.AppsV1().
		Deployments(resource.Namespace).
		Get(ctx, resource.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	volumeName := "vol-" + resource.Name

	deployment.Spec.Template.Spec.Volumes = []corev1.Volume{
		{
			Name: volumeName,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: resource.Name,
				},
			},
		},
	}

	deployment.Spec.Template.Spec.Containers[0].VolumeMounts = []corev1.VolumeMount{
		{
			Name:      volumeName,
			MountPath: resource.MountPath,
		},
	}

	_, err = d.clientset.AppsV1().
		Deployments(resource.Namespace).
		Update(ctx, deployment, metav1.UpdateOptions{})
	return err
}

func (d *KubernetesService) detachPVCToDeployment(ctx context.Context, resource Resource) error {
	deployment, err := d.clientset.AppsV1().
		Deployments(resource.Namespace).
		Get(ctx, resource.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	deployment.Spec.Template.Spec.Volumes = nil
	deployment.Spec.Template.Spec.Containers[0].VolumeMounts = nil

	d.DeletePVC(ctx, resource)

	_, err = d.clientset.AppsV1().
		Deployments(resource.Namespace).
		Update(ctx, deployment, metav1.UpdateOptions{})
	return err
}
