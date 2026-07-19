package deploy

import (
	"github.com/timmyjinks/tysoncloud/kubernetes"
)

type DeployService struct {
	svc *kubernetes.KubernetesService
}

func NewDeployService(svc *kubernetes.KubernetesService) *DeployService {
	return &DeployService{
		svc: svc,
	}
}
