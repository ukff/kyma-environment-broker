package globalaccounts

import (
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
)

type Config struct {
	Database             storage.Config
	Broker               broker.ClientConfig
	DryRun               bool   `envconfig:"default=true"`
	AccountServiceID     string `envconfig:"default=account-service-id"`
	AccountServiceSecret string `envconfig:"default=account-service-secret"`
	AccountServiceAuth   string `envconfig:"default=url"`
	AccountServiceURL    string `envconfig:"default=url"`
}
