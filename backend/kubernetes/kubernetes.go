package kubernetes

import (
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	gatewayclient "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"
)

type KubernetesService struct {
	ClusterIP     string
	clientset     kubernetes.Interface
	gatewayClient gatewayclient.Interface
	dynamicClient dynamic.Interface
}

func NewKubernetesService(kubeconfigPath string, clusterIp string) (*KubernetesService, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	clientset := kubernetes.NewForConfigOrDie(config)
	gatewayClient := gatewayclient.NewForConfigOrDie(config)
	dynamicClient := dynamic.NewForConfigOrDie(config)

	return NewKubernetesServiceWithClients(clusterIp, clientset, gatewayClient, dynamicClient), nil
}

func NewKubernetesServiceWithClients(clusterIp string, clientset kubernetes.Interface, gatewayClient gatewayclient.Interface, dynamicClient dynamic.Interface) *KubernetesService {
	return &KubernetesService{
		ClusterIP:     clusterIp,
		clientset:     clientset,
		gatewayClient: gatewayClient,
		dynamicClient: dynamicClient,
	}
}
