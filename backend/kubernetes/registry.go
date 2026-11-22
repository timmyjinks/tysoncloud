package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/timmyjinks/tysoncloud/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	appcorev1 "k8s.io/client-go/applyconfigurations/core/v1"
	appmetav1 "k8s.io/client-go/applyconfigurations/meta/v1"
)

const RegistryPullSecretName = "tc-registry-pull"

func (d *KubernetesService) ConfigureRegistryPull(registryURL string, dockerConfigJSON []byte) error {
	host := util.RegistryHost(registryURL)
	var cfg struct {
		Auths map[string]json.RawMessage `json:"auths"`
	}
	if err := json.Unmarshal(dockerConfigJSON, &cfg); err != nil {
		return fmt.Errorf("parse registry docker config: %w", err)
	}
	filtered := map[string]json.RawMessage{}
	for k, v := range cfg.Auths {
		if util.RegistryHost(k) == host {
			filtered[k] = v
		}
	}
	if len(filtered) == 0 {
		return fmt.Errorf("registry docker config has no auths entry for %s", host)
	}
	out, err := json.Marshal(map[string]any{"auths": filtered})
	if err != nil {
		return err
	}
	d.registryURL = registryURL
	d.registryPullConfig = out
	return nil
}

func (d *KubernetesService) ensureRegistryPullSecret(ctx context.Context, namespace, image string) (string, error) {
	if len(d.registryPullConfig) == 0 || !util.ImageUsesRegistry(image, d.registryURL) {
		return "", nil
	}
	if err := d.applyRegistryPullSecret(ctx, namespace); err != nil {
		return "", err
	}
	return RegistryPullSecretName, nil
}

func (d *KubernetesService) applyRegistryPullSecret(ctx context.Context, namespace string) error {
	secretType := corev1.SecretTypeDockerConfigJson
	_, err := d.clientset.CoreV1().Secrets(namespace).Apply(ctx, &appcorev1.SecretApplyConfiguration{
		TypeMetaApplyConfiguration: appmetav1.TypeMetaApplyConfiguration{
			Kind:       util.StringPtr("Secret"),
			APIVersion: util.StringPtr("v1"),
		},
		ObjectMetaApplyConfiguration: &appmetav1.ObjectMetaApplyConfiguration{
			Namespace: &namespace,
			Name:      util.StringPtr(RegistryPullSecretName),
		},
		Type: &secretType,
		Data: map[string][]byte{corev1.DockerConfigJsonKey: d.registryPullConfig},
	}, metav1.ApplyOptions{FieldManager: "tysoncloud"})
	if err != nil {
		return fmt.Errorf("apply registry pull secret: %w", err)
	}
	return nil
}

func (d *KubernetesService) ReconcileRegistryPullSecret(ctx context.Context, namespace string) error {
	if len(d.registryPullConfig) == 0 {
		return nil
	}
	list, err := d.clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, dep := range list.Items {
		if !podUsesRegistry(dep.Spec.Template.Spec, d.registryURL) || hasPullSecret(dep.Spec.Template.Spec, RegistryPullSecretName) {
			continue
		}
		if err := d.applyRegistryPullSecret(ctx, namespace); err != nil {
			return err
		}
		patch := []byte(`{"spec":{"template":{"spec":{"imagePullSecrets":[{"name":"` + RegistryPullSecretName + `"}]}}}}`)
		if _, err := d.clientset.AppsV1().Deployments(namespace).Patch(ctx, dep.Name, types.StrategicMergePatchType, patch, metav1.PatchOptions{FieldManager: "tysoncloud"}); err != nil {
			return fmt.Errorf("patch %s/%s pull secret: %w", namespace, dep.Name, err)
		}
	}
	return nil
}

func podUsesRegistry(spec corev1.PodSpec, registryURL string) bool {
	for _, c := range append(append([]corev1.Container{}, spec.InitContainers...), spec.Containers...) {
		if util.ImageUsesRegistry(c.Image, registryURL) {
			return true
		}
	}
	return false
}

func hasPullSecret(spec corev1.PodSpec, name string) bool {
	for _, s := range spec.ImagePullSecrets {
		if s.Name == name {
			return true
		}
	}
	return false
}
