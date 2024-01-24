package postsql_test

import (
	"fmt"
	"log"
	"os"
	"strings"
	"testing"
	"time"
	
	"github.com/gocraft/dbr"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/events"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/dbmodel"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/postsql"
	"github.com/sirupsen/logrus"
	"golang.org/x/exp/slices"
)

type migrationOrder int

const (
	Up migrationOrder = iota
	Down
)

func brokerStorageTestConfig() storage.Config {
	const (
		kebStorageTestHostname  = "localhost"
		kebStorageTestDbUser    = "test"
		kebStorageTestDbPass    = "test"
		kebStorageTestDbName    = "testbroker"
		kebStorageTestDbPort    = "5430"
		kebStorageTestSecretKey = "################################"
	)
	
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

func TestMain(m *testing.M) {
	exitVal := 0
	defer func() {
		os.Exit(exitVal)
	}()
	
	config := brokerStorageTestConfig()
	fmt.Println(fmt.Sprintf("connection URL -> %s", config.ConnectionURL()))
	
	docker, err := internal.NewDockerHandler()
	if err != nil {
		log.Fatal(err)
	}
	defer docker.CloseDockerClient()
	
	cleanupContainer, err := docker.CreateDBContainer(internal.ContainerCreateRequest{
		Port:          config.Port,
		User:          config.User,
		Password:      config.Password,
		Name:          config.Name,
		Host:          config.Host,
		ContainerName: "keb-storage-tests",
		Image:         "postgres:11",
	})
	
	log.Print("container started")
	defer func() {
		log.Println("cleaning up")
		if cleanupContainer != nil {
			err := cleanupContainer()
			if err != nil {
				log.Fatal(err)
			}
		}
	}()
	if err != nil {
		log.Println("error while starting container")
		log.Fatal(err)
	}
	
	fmt.Println(fmt.Sprintf("connection URL -> %s", config.ConnectionURL()))
	
	exitVal = m.Run()
}

func GetStorageForTests() (func() error, storage.BrokerStorage, error) {
	config := brokerStorageTestConfig()
	fmt.Println(fmt.Sprintf("connection URL -> %s", config.ConnectionURL()))
	storage, connection, err := storage.NewFromConfig(config, events.Config{}, storage.NewEncrypter(config.SecretKey), logrus.StandardLogger())
	if err != nil {
		return nil, nil, fmt.Errorf("while creating storage: %w", err)
	}
	if connection == nil {
		return nil, nil, fmt.Errorf("connection is nil")
	}
	if storage == nil {
		return nil, nil, fmt.Errorf("storage is nil")
	}
	fmt.Println(fmt.Sprintf("connected to URL -> %s", config.ConnectionURL()))
	failOnIncorrectDB(connection, config)
	// failOnNotEmptyDb(connection, storage)
	
	err = runMigrations(connection, Up)
	if err != nil {
		return nil, nil, fmt.Errorf("while applying migration files: %w", err)
	}
	fmt.Println("migration files applied")
	
	cleanup := func() error {
		failOnIncorrectDB(connection, config)
		fmt.Println("cleaning up")
		err := runMigrations(connection, Down)
		if err != nil {
			return fmt.Errorf("failed to clear DB tables: %w", err)
		}
		fmt.Println("cleaned up")
		return nil
	}
	
	return cleanup, storage, nil
}

func runMigrations(connection *dbr.Connection, order migrationOrder) error {
	if order != Up && order != Down {
		return fmt.Errorf("unknown migration order")
	}
	
	migrations := "./../../../../resources/keb/migrations/"
	files, err := os.ReadDir(migrations)
	if err != nil {
		return fmt.Errorf("while reading migration data: %w in directory :%s", err, migrations)
	}
	
	if order == Down {
		slices.Reverse(files)
	}
	
	for _, file := range files {
		if strings.HasSuffix(file.Name(), "up.sql") {
			content, err := os.ReadFile(migrations + file.Name())
			if err != nil {
				return fmt.Errorf("while reading migration files: %w file: %s", err, file.Name())
			}
			if _, err = connection.Exec(string(content)); err != nil {
				return fmt.Errorf("while applying migration files: %w file: %s", err, file.Name())
			}
		}
	}
	
	return nil
}

func failOnIncorrectDB(db *dbr.Connection, config storage.Config) {
	if db == nil {
		return
	}
	row := db.QueryRow("SELECT CURRENT_DATABASE();")
	var result string
	err := row.Scan(&result)
	if err != nil {
		panic("cannot check if db has test prefix")
	}
	has := strings.HasPrefix(result, "test")
	if !has {
		panic("db has not test prefix")
	}
	equal := strings.EqualFold(result, config.Name)
	if !equal {
		panic(fmt.Sprintf("db: %s is not equal to config: %s", result, config.Name))
	}
}

func failOnNotEmptyDb(db *dbr.Connection, storage storage.BrokerStorage) {
	/*rowsExists := func(table string) bool {
			var rowResult string
			result := db.QueryRow(fmt.Sprintf(`SELECT CASE
	         WHEN EXISTS (SELECT * FROM %s LIMIT 1) THEN 1
	         ELSE 0
	       END`, table))
			result.Scan(rowResult)
			return strings.EqualFold(rowResult, "1")
		}*/
	
	_, len1, len2, err := storage.Instances().List(dbmodel.InstanceFilter{})
	if err != nil {
		panic(fmt.Sprintf("while checking len data for: %s , %s", postsql.InstancesTableName, err.Error()))
	}
	if len1 > 0 || len2 > 0 {
		panic(fmt.Sprintf("storage for: %s is not empty", postsql.InstancesTableName))
	}
	
	_, len1, len2, err = storage.Operations().ListOperations(dbmodel.OperationFilter{})
	if err != nil {
		panic(fmt.Sprintf("while checking len data for: %s , %s", postsql.OperationTableName, err.Error()))
	}
	if len1 > 0 || len2 > 0 {
		panic(fmt.Sprintf("storage for: %s is not empty", postsql.OperationTableName))
	}
	
	/*
		if rowsExists(postsql.InstancesTableName) {
			panic(fmt.Sprintf("in db, data for %s are in table", postsql.InstancesTableName))
		}
	
		if rowsExists(postsql.InstancesTableName) {
			panic(fmt.Sprintf("in db, data for %s are in table", postsql.InstancesTableName))
		}*/
}
