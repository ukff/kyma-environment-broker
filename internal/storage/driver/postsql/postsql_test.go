package postsql_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/gocraft/dbr"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
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

	containerCleanupFunc, dbCfg, err := storage.CreateDBContainer(log.Printf, ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer containerCleanupFunc()

	connection, err := dbr.Open("postgres", dbCfg.ConnectionURL(), nil)
	if err != nil {
		log.Fatal(err)
	}

	defer func(c *dbr.Connection) {
		fmt.Println("closing connection")
		err = c.Close()
		if err != nil {
			err = fmt.Errorf("failed to close database connection: %w", err)
		}
	}(connection)

	fmt.Println(fmt.Sprintf("connection created to -> : %v", dbCfg.ConnectionURL()))
	exitVal = m.Run()
}
