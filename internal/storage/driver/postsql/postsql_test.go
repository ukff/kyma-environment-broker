package postsql_test

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/postsql"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

var (
	kebStorageTestConfig storage.Config
)

const (
	kebStorageTestHostname  = "testnetwork"
	kebStorageTestDbUser    = "testuser"
	kebStorageTestDbPass    = "testpass"
	kebStorageTestDbName    = "testbroker"
	kebStorageTestDbPort    = "5432"
	kebStorageTestSecretKey = "################################"
)

func TestMain(m *testing.M) {
	exitVal := 0
	defer func() {
		os.Exit(exitVal)
	}()

	dockerHandler, err := internal.NewDockerHandler()
	if err != nil {
		log.Fatal(err)
	}

	cleanupNetwork, err := dockerHandler.SetupTestNetworkForDB()
	if err != nil {
		log.Fatal(err)
	}
	defer cleanupNetwork()

	kebStorageTestConfig = testConfig()

	cleanupContainer, container, err := dockerHandler.CreateDBContainer(internal.ContainerCreateConfig{
		Port:          kebStorageTestConfig.Port,
		User:          kebStorageTestConfig.User,
		Password:      kebStorageTestConfig.Password,
		Name:          kebStorageTestConfig.Name,
		Host:          kebStorageTestConfig.Host,
		ContainerName: "keb-storage-tests",
		Image:         "postgres:11",
	})

	if err != nil || container == nil {
		log.Fatal(err)
	}
	defer cleanupContainer()

	port, err := internal.ExtractPortFromContainer(*container)
	if err != nil {
		log.Fatal(err)
	}
	kebStorageTestConfig.Port = port

	exitVal = m.Run()
}

// dont set defaults
func emptyConfig() storage.Config {
	return storage.Config{
		User:            "",
		Password:        "",
		Host:            "",
		Port:            "",
		Name:            "",
		SSLMode:         "",
		SSLRootCert:     "",
		SecretKey:       "",
		MaxOpenConns:    0,
		MaxIdleConns:    0,
		ConnMaxLifetime: 0,
	}
}

func testConfig() storage.Config {
	return storage.Config{
		Host:            kebStorageTestHostname,
		User:            kebStorageTestDbUser,
		Password:        kebStorageTestDbPass,
		Port:            kebStorageTestDbPort,
		Name:            kebStorageTestDbName,
		SSLMode:         "disable",
		SecretKey:       kebStorageTestSecretKey,
		MaxOpenConns:    1,
		MaxIdleConns:    1,
		ConnMaxLifetime: time.Minute,
	}
}

func prepareStorageTestEnvironment(t *testing.T) (func(), storage.Config, error) {
	testDbConnection, err := postsql.WaitForDatabaseAccess(kebStorageTestConfig.ConnectionURL(), 1000, 10*time.Millisecond, logrus.New())

	cleanupFunc := func() {
		_, err := testDbConnection.Exec(fmt.Sprintf("TRUNCATE TABLE %s, %s, %s, %s RESTART IDENTITY CASCADE",
			postsql.InstancesTableName,
			postsql.OperationTableName,
			postsql.OrchestrationTableName,
			postsql.RuntimeStateTableName,
		))
		if err != nil {
			err = fmt.Errorf("failed to clear DB tables: %w", err)
			assert.NoError(t, err)
		}
	}

	initialized, err := postsql.CheckIfDatabaseInitialized(testDbConnection)
	if err != nil {
		if testDbConnection != nil {
			err := testDbConnection.Close()
			assert.Nil(t, err, "Failed to close db connection")
		}
		return nil, emptyConfig(), fmt.Errorf("while checking DB initialization: %w", err)
	} else if initialized {
		return cleanupFunc, kebStorageTestConfig, nil
	}

	dirPath := "./../../../../resources/keb/migrations/"
	files, err := ioutil.ReadDir(dirPath)
	if err != nil {
		log.Printf("cannot read files from keb migrations directory %s", dirPath)
		return nil, emptyConfig(), fmt.Errorf("while reading migration data: %w", err)
	}

	for _, file := range files {
		if strings.HasSuffix(file.Name(), "up.sql") {
			v, err := ioutil.ReadFile(dirPath + file.Name())
			if err != nil {
				log.Printf("cannot read migration file %s", file.Name())
			}
			if _, err = testDbConnection.Exec(string(v)); err != nil {
				log.Printf("cannot apply migration file %s", file.Name())
				return nil, emptyConfig(), fmt.Errorf("while applying migration files: %w", err)
			}
		}
	}
	log.Printf("migration applied to database")

	return cleanupFunc, kebStorageTestConfig, nil
}
