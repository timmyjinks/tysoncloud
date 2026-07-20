package deploy

import (
	"context"
)

func (d *DeployService) AttachVolume(ctx context.Context, service Service, volume Volume) error {
	resource := ServiceToResource(service)
	resource.MountPath = volume.MountPath
	resource.StorageGB = volume.StorageGB

	err := d.svc.CreatePVC(ctx, resource)
	if err != nil {
		return err
	}

	if err := d.svc.AttachPVCToDeployment(ctx, resource); err != nil {
		return err
	}

	return nil
}

func (d *DeployService) DetachVolume(ctx context.Context, service Service) error {
	resource := ServiceToResource(service)

	err := d.svc.DetachPVCToDeployment(ctx, resource)
	if err != nil {
		return err
	}

	if err := d.svc.DeletePVC(ctx, resource); err != nil {
		return err
	}

	return nil
}
