package deploy

import "github.com/timmyjinks/tysoncloud/kubernetes"

func ServiceToResource(service Service) kubernetes.Resource {
	return kubernetes.Resource{
		Namespace: service.Namespace,
		Name:      service.Name,
		Hostname:  service.Hostname,
		Env:       service.Env,
		Image:     service.Image,
		Port:      service.Port,
		StorageGB: service.Volume.StorageGB,
		MountPath: service.Volume.MountPath,
	}
}
