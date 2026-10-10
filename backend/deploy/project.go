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

	for _, setup := range []func(context.Context, string) error{
		func(ctx context.Context, ns string) error { return d.svc.CreateNetworkPolicy(ctx, ns, d.svc.ClusterIP) },
		d.svc.CreateResourceQuota,
		d.svc.CreateLimitRange,
	} {
		if err := setup(ctx, namespace); err != nil {
			if cleanupErr := d.svc.DeleteNamespace(ctx, namespace); cleanupErr != nil {
				slog.Error(cleanupErr.Error())
			}
			return err
		}
	}
	return nil
}

// ReconcileProjects re-applies the namespace labels (Pod Security Admission),
// ResourceQuota and LimitRange to every existing tysoncloud namespace, so
// namespaces created before those guards existed get them too. Errors are
// logged per namespace and don't stop the rest.
func (d *DeployService) ReconcileProjects(ctx context.Context) error {
	namespaces, err := d.svc.ListManagedNamespaces(ctx)
	if err != nil {
		return err
	}
	for _, namespace := range namespaces {
		for _, apply := range []func(context.Context, string) error{
			d.svc.CreateNamespace,
			d.svc.CreateResourceQuota,
			d.svc.CreateLimitRange,
		} {
			if err := apply(ctx, namespace); err != nil {
				slog.Error("reconcile project namespace failed", "namespace", namespace, "err", err)
			}
		}
	}
	slog.Info("reconciled project namespaces", "count", len(namespaces))
	return nil
}

func (d *DeployService) DeleteProject(ctx context.Context, namespace string) error {
	err := d.svc.DeleteNamespace(ctx, namespace)
	if err != nil {
		return err
	}

	return nil
}
