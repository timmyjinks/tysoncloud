package deploy

import (
	"context"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

func (d *DeployService) GetServiceLogs(ctx context.Context, service Service, lines chan string) error {
	return d.svc.GetDeploymentLogs(ctx, ServiceToResource(service), lines)
}

func (d *DeployService) CreateService(ctx context.Context, service Service) error {
	if err := d.svc.CreateService(ctx, ServiceToResource(service)); err != nil {
		return err
	}

	if len(service.Env) != 0 {
		if err := d.svc.CreateSecret(ctx, ServiceToResource(service)); err != nil {
			return err
		}
	} else {
		if err := d.svc.DeleteSecret(ctx, ServiceToResource(service)); err != nil && !apierrors.IsNotFound(err) {
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

func (d *DeployService) BatchCreateServices(ctx context.Context, services []Service) error {
	for _, service := range services {
		if err := d.svc.CreateService(ctx, ServiceToResource(service)); err != nil {
			return err
		}

		if len(service.Env) != 0 {
			if err := d.svc.CreateSecret(ctx, ServiceToResource(service)); err != nil {
				return err
			}
		} else {
			if err := d.svc.DeleteSecret(ctx, ServiceToResource(service)); err != nil && !apierrors.IsNotFound(err) {
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
	}

	return nil
}

func (d *DeployService) GetServiceEnv(ctx context.Context, service Service) (map[string]string, error) {
	return d.svc.GetSecret(ctx, ServiceToResource(service))
}

func (d *DeployService) DeleteService(ctx context.Context, service Service) error {
	if err := d.svc.DeleteSecret(ctx, ServiceToResource(service)); err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	if err := d.svc.DeletePVC(ctx, ServiceToResource(service)); err != nil && !apierrors.IsNotFound(err) {
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
