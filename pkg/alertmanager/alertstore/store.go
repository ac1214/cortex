package alertstore

import (
	"context"
	"fmt"

	"github.com/go-kit/kit/log"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/cortexproject/cortex/pkg/alertmanager/alertspb"
	"github.com/cortexproject/cortex/pkg/alertmanager/alertstore/bucketclient"
	"github.com/cortexproject/cortex/pkg/alertmanager/alertstore/configdb"
	"github.com/cortexproject/cortex/pkg/alertmanager/alertstore/local"
	"github.com/cortexproject/cortex/pkg/alertmanager/alertstore/objectclient"
	"github.com/cortexproject/cortex/pkg/chunk"
	"github.com/cortexproject/cortex/pkg/chunk/aws"
	"github.com/cortexproject/cortex/pkg/chunk/azure"
	"github.com/cortexproject/cortex/pkg/chunk/gcp"
	"github.com/cortexproject/cortex/pkg/configs/client"
	"github.com/cortexproject/cortex/pkg/storage/bucket"
)

// AlertStore stores and configures users rule configs
type AlertStore interface {
	// ListAlertConfigs loads and returns the alertmanager configuration for all users.
	ListAlertConfigs(ctx context.Context) (map[string]alertspb.AlertConfigDesc, error)

	// GetAlertConfig loads and returns the alertmanager configuration for the given user.
	GetAlertConfig(ctx context.Context, user string) (alertspb.AlertConfigDesc, error)

	// SetAlertConfig stores the alertmanager configuration for an user.
	SetAlertConfig(ctx context.Context, cfg alertspb.AlertConfigDesc) error

	// DeleteAlertConfig deletes the alertmanager configuration for an user.
	DeleteAlertConfig(ctx context.Context, user string) error
}

// NewLegacyAlertStore returns a new alertmanager storage backend poller and store
func NewLegacyAlertStore(cfg LegacyConfig, logger log.Logger) (AlertStore, error) {
	if cfg.Type == configdb.Name {
		c, err := client.New(cfg.ConfigDB)
		if err != nil {
			return nil, err
		}
		return configdb.NewStore(c), nil
	}

	if cfg.Type == local.Name {
		return local.NewStore(cfg.Local)
	}

	// Create the object store client.
	var client chunk.ObjectClient
	var err error
	switch cfg.Type {
	case "azure":
		client, err = azure.NewBlobStorage(&cfg.Azure)
	case "gcs":
		client, err = gcp.NewGCSObjectClient(context.Background(), cfg.GCS)
	case "s3":
		client, err = aws.NewS3ObjectClient(cfg.S3)
	default:
		return nil, fmt.Errorf("unrecognized alertmanager storage backend %v, choose one of: azure, configdb, gcs, local, s3", cfg.Type)
	}
	if err != nil {
		return nil, err
	}

	return objectclient.NewAlertStore(client, logger), nil
}

// NewAlertStore returns a alertmanager store backend client based on the provided cfg.
func NewAlertStore(ctx context.Context, cfg Config, cfgProvider bucket.TenantConfigProvider, logger log.Logger, reg prometheus.Registerer) (AlertStore, error) {
	if cfg.Backend == configdb.Name {
		c, err := client.New(cfg.ConfigDB)
		if err != nil {
			return nil, err
		}
		return configdb.NewStore(c), nil
	}

	if cfg.Backend == local.Name {
		return local.NewStore(cfg.Local)
	}

	bucketClient, err := bucket.NewClient(ctx, cfg.Config, "alertmanager-storage", logger, reg)
	if err != nil {
		return nil, err
	}

	return bucketclient.NewBucketAlertStore(bucketClient, cfgProvider, logger), nil
}
