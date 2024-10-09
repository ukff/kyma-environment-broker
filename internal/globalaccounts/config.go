package globalaccounts

import (
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
)

type Config struct {
	Database      storage.Config
	DryRun        bool   `envconfig:"default=true"`
	ServiceID     string `envconfig:"default=account-service-id"`
	ServiceSecret string `envconfig:"default=account-service-secret"`
	ServiceAuth   string `envconfig:"default=url"`
	ServiceURL    string `envconfig:"default=url"`
}
