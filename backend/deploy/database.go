package deploy

import (
	"context"
	"errors"

	"github.com/timmyjinks/tysoncloud/kubernetes"
)

func (d *DeployService) CreateDatabase(ctx context.Context, database Database) error {
	switch database.Engine {
	case "postgres":
		return d.svc.CreatePostgresDatabase(ctx, kubernetes.Resource{
			Namespace: database.Namespace,
			Name:      database.Name,
			Engine:    database.Engine,
			StorageGB: database.StorageGB,
		})
	default:
		return errors.New("DB engine not found")
	}
}

func (d *DeployService) DeleteDatabase(ctx context.Context, database Database) error {
	switch database.Engine {
	case "postgres":
		return d.svc.DeletePostgresDatabase(ctx, kubernetes.Resource{
			Namespace: database.Namespace,
			Name:      database.Name,
		})
	default:
		return errors.New("DB engine not found")
	}
}
