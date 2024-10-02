package main

import (
	"github.com/kyma-project/kyma-environment-broker/internal/globalaccounts"
)

func main() {
	globalaccounts.Run(globalaccounts.Config{})
}
