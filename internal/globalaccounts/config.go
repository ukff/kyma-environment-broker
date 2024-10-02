package globalaccounts

import (
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
)

type Config struct {
	Database             storage.Config
	Broker               broker.ClientConfig
	DryRun               bool          `envconfig:"default=true"`
	ExpirationPeriod     time.Duration `envconfig:"default=720h"` // 30 days
	TestRun              bool          `envconfig:"default=false"`
	TestSubaccountID     string        `envconfig:"default=prow-keb-trial-suspension"`
	PlanID               string        `envconfig:"default=7d55d31d-35ae-4438-bf13-6ffdfa107d9f"`
	AccountServiceID     string        `envconfig:"default=account-service-id"`
	AccountServiceSecret string        `envconfig:"default=account-service-secret"`
	AccountServiceAuth   string        `envconfig:"default=url"`
	AccountServiceURL    string        `envconfig:"default=url"`
}

// Auth -> https://management-plane.authentication.stagingaws.hanavlab.ondemand.com/oauth/token
// URL -> https://accounts-service.cfapps.stagingaws.hanavlab.ondemand.com/accounts/v1/technical/subaccounts/%s
