package kubernetes

import (
	"context"
	"fmt"

	cnpgv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func (d *KubernetesService) CreatePostgresDatabase(ctx context.Context, resource Resource) error {
	cluster := &cnpgv1.Cluster{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "postgresql.cnpg.io/v1",
			Kind:       "Cluster",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      resource.Name,
			Namespace: resource.Namespace,
		},
		Spec: cnpgv1.ClusterSpec{
			InheritedMetadata: &cnpgv1.EmbeddedObjectMetadata{
				Labels: map[string]string{
					"app.kubernetes.io/component": "resource",
				},
			},
			Instances: 1,

			Bootstrap: &cnpgv1.BootstrapConfiguration{
				InitDB: &cnpgv1.BootstrapInitDB{
					Database: "app",
					Owner:    "app",
				},
			},
		},
	}

	if resource.StorageGB > 0 {
		cluster.Spec.StorageConfiguration = cnpgv1.StorageConfiguration{
			Size: fmt.Sprintf("%dGi", resource.StorageGB),
		}
	}

	storageSize := resource.StorageGB
	if storageSize <= 0 {
		storageSize = 5
	}

	cluster.Spec.StorageConfiguration = cnpgv1.StorageConfiguration{
		Size: fmt.Sprintf("%dGi", storageSize),
	}

	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(cluster)
	if err != nil {
		return err
	}

	gvr := schema.GroupVersionResource{
		Group:    "postgresql.cnpg.io",
		Version:  "v1",
		Resource: "clusters",
	}

	_, err = d.dynamicClient.Resource(gvr).
		Namespace(resource.Namespace).
		Apply(
			ctx,
			resource.Name,
			&unstructured.Unstructured{Object: obj},
			metav1.ApplyOptions{
				FieldManager: "tysoncloud",
			},
		)
	if err != nil {
		return err
	}
	return nil
}

func (d *KubernetesService) DeletePostgresDatabase(resource Resource) error {
	gvr := schema.GroupVersionResource{
		Group:    "postgresql.cnpg.io",
		Version:  "v1",
		Resource: "clusters",
	}
	return d.dynamicClient.Resource(gvr).Namespace(resource.Namespace).Delete(context.Background(), resource.Name, metav1.DeleteOptions{})
}
