package deploy

import (
	"context"
	"log/slog"
)

func (d *DeployService) CreateProject(ctx context.Context, namespace string) error {
	err := d.svc.CreateNamespace(ctx, namespace)
	if err != nil {
		return err
	}

	if err := d.svc.CreateNetworkPolicy(ctx, namespace, d.svc.ClusterIP); err != nil {
		if cleanupErr := d.svc.DeleteNamespace(ctx, namespace); cleanupErr != nil {
			slog.Error(cleanupErr.Error())
		}
		return err
	}
	return nil
}

func (d *DeployService) DeleteProject(ctx context.Context, namespace string) error {
	err := d.svc.DeleteNamespace(ctx, namespace)
	if err != nil {
		return err
	}

	return nil
}
