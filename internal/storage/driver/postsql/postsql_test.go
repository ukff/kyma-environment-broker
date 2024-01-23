package postsql_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/postsql"
	"github.com/sirupsen/logrus"
)

func TestMain(m *testing.M) {
	fmt.Println("===============================TestMain START===============================")
	exitVal := 0
	defer func() {
		fmt.Println("===============================TestMain END===============================")
		os.Exit(exitVal)
	}()

	ctx := context.Background()

	cleanupNetwork, err := storage.SetupTestNetworkForDB(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer cleanupNetwork()

	containerCleanupFunc, cfg, err := storage.InitTestDBContainer(log.Printf, ctx, "test_DB_1")
	if err != nil {
		log.Fatal(err)
	}
	defer containerCleanupFunc()

	_, err = postsql.WaitForDatabaseAccess(cfg.ConnectionURL(), 10, 1*time.Second, logrus.New())
	if err != nil {
		log.Fatal(err)
	}

	exitVal = m.Run()
}
