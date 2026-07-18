package deploy

import (
	"context"

	"github.com/timmyjinks/tysoncloud/kubernetes"
)

func (d *DeployService) CreateService(ctx context.Context, service Service) error {
	if err := d.svc.CreateService(ctx, kubernetes.Resource{}); err != nil {
		return err
	}

	if len(service.Env) != 0 {
		if err := d.svc.CreateSecret(ctx, ServiceToResource(service)); err != nil {
			return err
		}
	}

	if service.Volume != nil {
		if err := d.svc.CreatePVC(ctx, ServiceToResource(service)); err != nil {
			return err
		}
	}

	if err := d.svc.CreateDeployment(ctx, ServiceToResource(service)); err != nil {
		return err
	}

	if err := d.svc.CreateHPA(ctx, ServiceToResource(service)); err != nil {
		return err
	}

	if err := d.svc.CreateHTTPRoute(ctx, ServiceToResource(service)); err != nil {
		return err
	}

	return nil
}

func (d *DeployService) DeleteService(ctx context.Context, service Service) error {
	if err := d.svc.DeletePVC(ctx, ServiceToResource(service)); err != nil {
		return err
	}

	if err := d.svc.DeleteHPA(ctx, ServiceToResource(service)); err != nil {
		return err
	}

	if err := d.svc.DeleteDeployment(ctx, ServiceToResource(service)); err != nil {
		return err
	}

	err := d.svc.DeleteService(ctx, ServiceToResource(service))
	if err != nil {
		return err
	}

	if err := d.svc.DeleteHTTPRoute(ctx, ServiceToResource(service)); err != nil {
		return err
	}

	return nil
}
