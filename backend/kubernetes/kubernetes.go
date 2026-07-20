package kubernetes

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	gatewayclient "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"
)

type KubernetesService struct {
	ClusterIP     string
	clientset     *kubernetes.Clientset
	gatewayClient *gatewayclient.Clientset
	dynamicClient *dynamic.DynamicClient
}

func NewKubernetesService(kubeconfigPath string) (*KubernetesService, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, err
	}

	clientset := kubernetes.NewForConfigOrDie(config)
	gatewayClient := gatewayclient.NewForConfigOrDie(config)
	dynamicClient := dynamic.NewForConfigOrDie(config)

	svc, err := clientset.CoreV1().
		Services("default").
		Get(context.Background(), "kubernetes", metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	ip := svc.Spec.ClusterIP

	return &KubernetesService{
		ClusterIP:     ip,
		clientset:     clientset,
		gatewayClient: gatewayClient,
		dynamicClient: dynamicClient,
	}, nil
}
