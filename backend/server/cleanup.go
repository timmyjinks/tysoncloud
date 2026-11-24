package server

import (
	"context"
	"log/slog"

	"github.com/timmyjinks/tysoncloud/deploy"
	"github.com/timmyjinks/tysoncloud/store"
)

type configRollback struct {
	projectId         string
	userId            string
	serviceTables     []store.ServicesTable
	databaseTables    []store.DatabasesTable
	deployedServices  map[int]struct{}
	volumesCreated    map[int]struct{}
	volumesAttached   map[int]struct{}
	deployedDatabases map[int]struct{}
	cfRecords         map[int]struct{}
	cfRoutes          map[int]struct{}
}

func newConfigRollback(projectId, userId string) *configRollback {
	return &configRollback{
		projectId:         projectId,
		userId:            userId,
		deployedServices:  map[int]struct{}{},
		volumesCreated:    map[int]struct{}{},
		volumesAttached:   map[int]struct{}{},
		deployedDatabases: map[int]struct{}{},
		cfRecords:         map[int]struct{}{},
		cfRoutes:          map[int]struct{}{},
	}
}

func (app *Application) rollbackProjectConfig(rb *configRollback) {
	ctx := context.Background()
	namespace := "proj-" + rb.projectId

	for j := range rb.deployedDatabases {
		database := rb.databaseTables[j]
		if err := app.Deploy.DeleteDatabase(ctx, deploy.Database{
			Namespace: namespace,
			Name:      database.ResourceName,
			Engine:    database.Engine,
		}); err != nil {
			slog.Error("rollback: failed to delete database", "project_id", rb.projectId, "database_id", database.Id, "err", err)
		}
	}

	for i := range rb.volumesAttached {
		service := rb.serviceTables[i]
		if err := app.Deploy.DetachVolume(ctx, deploy.Service{
			Namespace: namespace,
			Name:      service.ResourceName,
		}); err != nil {
			slog.Error("rollback: failed to detach volume", "project_id", rb.projectId, "service_id", service.Id, "err", err)
		}
	}

	for i := range rb.volumesCreated {
		service := rb.serviceTables[i]
		if err := app.Supabase.DeleteVolume(service.Id, rb.userId); err != nil {
			slog.Error("rollback: failed to delete volume", "project_id", rb.projectId, "service_id", service.Id, "err", err)
		}
	}

	for i := range rb.deployedServices {
		service := rb.serviceTables[i]
		if err := app.Deploy.DeleteService(ctx, deploy.Service{
			Namespace: namespace,
			Name:      service.ResourceName,
		}); err != nil {
			slog.Error("rollback: failed to delete service", "project_id", rb.projectId, "service_id", service.Id, "err", err)
		}
	}

	for j := range rb.databaseTables {
		database := rb.databaseTables[j]
		if err := app.Supabase.DeleteDatabase(database.Id, rb.userId); err != nil {
			slog.Error("rollback: failed to delete database", "project_id", rb.projectId, "database_id", database.Id, "err", err)
		}
	}

	for i := range rb.serviceTables {
		service := rb.serviceTables[i]
		if err := app.Supabase.DeleteService(service.Id, rb.userId); err != nil {
			slog.Error("rollback: failed to delete service", "project_id", rb.projectId, "service_id", service.Id, "err", err)
		}
	}
}
