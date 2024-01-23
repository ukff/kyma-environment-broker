package postsql_test

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/kyma-project/kyma-environment-broker/internal/storage"
)

func TestMain(m *testing.M) {
	exitVal := 0
	defer func() {
		os.Exit(exitVal)
	}()

	ctx := context.Background()

	cleanupNetwork, err := storage.SetupTestNetworkForDB(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer cleanupNetwork()

	containerCleanupFunc, err := storage.CreateDBContainer(log.Printf)
	if err != nil {
		log.Fatal(err)
	}
	defer containerCleanupFunc()

	exitVal = m.Run()
}
