package kubernetes

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/timmyjinks/tysoncloud/util"
	corev1 "k8s.io/api/core/v1"
	resourcev1 "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	appsv1apply "k8s.io/client-go/applyconfigurations/apps/v1"
	appcorev1 "k8s.io/client-go/applyconfigurations/core/v1"
	appmetav1 "k8s.io/client-go/applyconfigurations/meta/v1"
)

func (d *KubernetesService) CreateDeployment(ctx context.Context, resource Resource) error {
	pullPolicy := corev1.PullIfNotPresent
	if strings.HasSuffix(resource.Image, ":latest") {
		pullPolicy = corev1.PullAlways
	}
	container := []appcorev1.ContainerApplyConfiguration{
		{
			Name:            &resource.Name,
			Image:           &resource.Image,
			ImagePullPolicy: &pullPolicy,
			Resources: &appcorev1.ResourceRequirementsApplyConfiguration{
				Limits: &corev1.ResourceList{
					corev1.ResourceCPU:    resourcev1.MustParse("500m"),
					corev1.ResourceMemory: resourcev1.MustParse("1Gi"),
				},
				Requests: &corev1.ResourceList{
					corev1.ResourceCPU:    resourcev1.MustParse("100m"),
					corev1.ResourceMemory: resourcev1.MustParse("100Mi"),
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

func (d *KubernetesService) GetDeploymentLogs(ctx context.Context, resource Resource, lines chan string) error {
	pods, err := d.clientset.CoreV1().Pods(resource.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app=%s", resource.Name),
	})
	if err != nil {
		return err
	}
	if len(pods.Items) == 0 {
		return fmt.Errorf("no pods found for %s", resource.Name)
	}

	var tail int64 = 200
	req := d.clientset.CoreV1().Pods(resource.Namespace).GetLogs(pods.Items[0].Name, &corev1.PodLogOptions{
		Container: resource.Name,
		Follow:    true,
		TailLines: &tail,
	})
	stream, err := req.Stream(ctx)
	if err != nil {
		return err
	}
	defer stream.Close()

	scanner := bufio.NewScanner(stream)
	for scanner.Scan() {
		select {
		case lines <- scanner.Text():
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return scanner.Err()
}

func (d *KubernetesService) GetDeploymentDiagnosticLogs(ctx context.Context, resource Resource, lines chan string) error {
	send := func(msg string) bool {
		select {
		case lines <- msg:
			return true
		case <-ctx.Done():
			return false
		}
	}

	pods, err := d.clientset.CoreV1().Pods(resource.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app=%s", resource.Name),
	})

	if err != nil {
		_ = send(fmt.Sprintf("unable to list pods for %s: %v", resource.Name, err))
		return err
	}

	if len(pods.Items) == 0 {
		_ = send(fmt.Sprintf("no pods found for %s — deployment may still be scheduling or failed to create pods", resource.Name))
		return errors.New("no pods found")
	}

	for _, pod := range pods.Items {
		for _, cs := range pod.Status.ContainerStatuses {
			if cs.Name != resource.Name {
				continue
			}

			if cs.State.Waiting != nil && cs.State.Waiting.Reason != "" {
				_ = send(fmt.Sprintf("container waiting: %s — %s", cs.State.Waiting.Reason, cs.State.Waiting.Message))
			}

			if cs.State.Terminated != nil && cs.State.Terminated.Reason != "" {
				_ = send(fmt.Sprintf("container terminated: %s (exit %d) — %s", cs.State.Terminated.Reason, cs.State.Terminated.ExitCode, cs.State.Terminated.Message))
			}

			if cs.LastTerminationState.Terminated != nil && cs.LastTerminationState.Terminated.Reason != "" {
				lt := cs.LastTerminationState.Terminated
				_ = send(fmt.Sprintf("last termination: %s (exit %d) — %s", lt.Reason, lt.ExitCode, lt.Message))
			}
		}

		for _, cs := range pod.Status.InitContainerStatuses {
			if cs.State.Waiting != nil && cs.State.Waiting.Reason != "" {
				_ = send(fmt.Sprintf("init container %s waiting: %s — %s", cs.Name, cs.State.Waiting.Reason, cs.State.Waiting.Message))
			}

			if cs.State.Terminated != nil && cs.State.Terminated.ExitCode != 0 {
				_ = send(fmt.Sprintf("init container %s terminated: %s (exit %d) — %s", cs.Name, cs.State.Terminated.Reason, cs.State.Terminated.ExitCode, cs.State.Terminated.Message))
			}
		}
	}

	<-ctx.Done()
	return ctx.Err()
}

func (d *KubernetesService) DeleteDeployment(ctx context.Context, resource Resource) error {
	return d.clientset.AppsV1().Deployments(resource.Namespace).Delete(ctx, resource.Name, metav1.DeleteOptions{})
}
