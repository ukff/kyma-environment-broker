package storage

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/gocraft/dbr"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/postsql"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	TestDbHostname        = "localhost"
	TestDbUser            = "test_user"
	TestDbPass            = "testpass"
	TestDbName            = "broker"
	TestDbPort            = "5432"
	TestDockerUserNetwork = "test_network"
)

var (
	testDbConfig Config
)

func SetTestDbConfig(value Config) {
	testDbConfig = value
}

func GetTestDBContainer() (Config, error) {
	return testDbConfig, nil
}

func MakeTestDbConfig(hostname string, port string) *Config {
	return &Config{
		Host:            hostname,
		User:            TestDbUser,
		Password:        TestDbPass,
		Port:            port,
		Name:            TestDbName,
		SSLMode:         "disable",
		SecretKey:       "$C&F)H@McQfTjWnZr4u7x!A%D*G-KaNd",
		MaxOpenConns:    1,
		MaxIdleConns:    1,
		ConnMaxLifetime: time.Minute,
	}
}

func CloseDatabase(t *testing.T, connection *dbr.Connection) {

}

func CreateDBContainer(log func(format string, args ...interface{}), ctx context.Context) (func(), Config, error) {
	start := time.Now()
	defer func() {
		fmt.Printf("InitTestDBContainer took -> %s\n", time.Since(start))
	}()

	cli, err := dockerClient()
	if err != nil {
		return nil, Config{}, fmt.Errorf("while creating docker client: %w", err)
	}

	dbImage := "postgres:11"

	filterBy := filters.NewArgs()
	filterBy.Add("name", dbImage)
	image, err := cli.ImageList(context.Background(), types.ImageListOptions{Filters: filterBy})

	if image == nil || err != nil {
		log("Image not found... pulling...")
		reader, err := cli.ImagePull(context.Background(), dbImage, types.ImagePullOptions{})
		io.Copy(os.Stdout, reader)
		defer reader.Close()

		if err != nil {
			return nil, Config{}, fmt.Errorf("while pulling dbImage: %w", err)
		}
	}

	_, parsedPortSpecs, err := nat.ParsePortSpecs([]string{TestDbPort})
	if err != nil {
		return nil, Config{}, fmt.Errorf("while parsing ports specs: %w", err)
	}

	body, err := cli.ContainerCreate(context.Background(),
		&container.Config{
			Image: dbImage,
			Env: []string{
				fmt.Sprintf("POSTGRES_USER=%s", TestDbUser),
				fmt.Sprintf("POSTGRES_PASSWORD=%s", TestDbPass),
				fmt.Sprintf("POSTGRES_DB=%s", TestDbName),
			},
		},
		&container.HostConfig{
			NetworkMode:     "default",
			PublishAllPorts: false,
			PortBindings:    parsedPortSpecs,
		},
		&network.NetworkingConfig{
			EndpointsConfig: map[string]*network.EndpointSettings{
				TestDockerUserNetwork: {
					Aliases: []string{
						TestDockerUserNetwork,
					},
				},
			},
		},
		&v1.Platform{},
		"")

	if err != nil {
		return nil, Config{}, fmt.Errorf("during container creation: %w", err)
	}

	cleanupFunc := func() {
		err := cli.ContainerRemove(context.Background(), body.ID, types.ContainerRemoveOptions{RemoveVolumes: true, RemoveLinks: false, Force: true})
		if err != nil {
			panic(fmt.Errorf("during container removal: %w", err))
		}
	}

	if err := cli.ContainerStart(context.Background(), body.ID, types.ContainerStartOptions{}); err != nil {
		return cleanupFunc, Config{}, fmt.Errorf("during container startup: %w", err)
	}

	err = waitFor(cli, body.ID, "database system is ready to accept connections")
	if err != nil {
		log("Failed to query container's logs: %s", err)
		return cleanupFunc, Config{}, fmt.Errorf("while waiting for DB readiness: %w", err)
	}

	filterBy = filters.NewArgs()
	filterBy.Add("id", body.ID)
	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{Filters: filterBy})

	if err != nil || len(containers) == 0 {
		log("no containers found: %s", err)
		return cleanupFunc, Config{}, fmt.Errorf("while loading containers: %w", err)
	}

	var container = &containers[0]

	if container == nil {
		log("no container found: %s", err)
		return cleanupFunc, Config{}, fmt.Errorf("while searching for a container: %w", err)
	}

	ports := container.Ports

	dbCfg := MakeTestDbConfig(TestDbHostname, fmt.Sprint(ports[0].PublicPort))
	SetTestDbConfig(*dbCfg)
	return cleanupFunc, *dbCfg, nil
}

func InitTestDBTables(t *testing.T, connectionURL string) (func(), error) {
	start := time.Now()
	defer func() {
		fmt.Printf("InitTestDBTables (total) -> %s\n", time.Since(start))
	}()
	fmt.Printf("InitTestDBTables start (wait for db access) to -> %s : %s \n", connectionURL, time.Since(start))
	connection, err := postsql.WaitForDatabaseAccess(connectionURL, 1000, 10*time.Millisecond, logrus.New())
	if err != nil {
		t.Logf("Cannot connect to database with URL - reload test 2 - %s", connectionURL)
		return nil, fmt.Errorf("while waiting for database access: %w", err)
	}
	fmt.Printf("InitTestDBTables (connected after) -> %s\n", time.Since(start))

	cleanupFunc := func() {
		_, err = connection.Exec(clearDBQuery())
		if err != nil {
			err = fmt.Errorf("failed to clear DB tables: %w", err)
		}
	}

	initialized, err := postsql.CheckIfDatabaseInitialized(connection)
	fmt.Printf("InitTestDBTables took after check if initialized -> %s\n", time.Since(start))
	if err != nil {
		if connection != nil {
			err := connection.Close()
			assert.Nil(t, err, "Failed to close db connection")
		}
		return nil, fmt.Errorf("while checking DB initialization: %w", err)
	} else if initialized {
		return cleanupFunc, nil
	}

	dirPath := "./../../../../resources/keb/migrations/"
	files, err := ioutil.ReadDir(dirPath)
	if err != nil {
		log.Printf("Cannot read files from directory %s", dirPath)
		return nil, fmt.Errorf("while reading migration data: %w", err)
	}
	fmt.Printf("InitTestDBTables took before migration -> %s\n", time.Since(start))

	for _, file := range files {
		if strings.HasSuffix(file.Name(), "up.sql") {
			v, err := ioutil.ReadFile(dirPath + file.Name())
			if err != nil {
				log.Printf("Cannot read file %s", file.Name())
			}
			if _, err = connection.Exec(string(v)); err != nil {
				log.Printf("Cannot apply file %s", file.Name())
				return nil, fmt.Errorf("while applying migration files: %w", err)
			}
		}
	}
	log.Printf("Files applied to database")
	fmt.Printf("InitTestDBTables took after migration and at the end -> %s\n", time.Since(start))

	return cleanupFunc, nil
}

func dockerClient() (*client.Client, error) {
	start := time.Now()
	defer func() {
		fmt.Printf("dockerClient took -> %s\n", time.Since(start))
	}()
	return client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
}

func isDockerTestNetworkPresent(ctx context.Context) (bool, error) {
	start := time.Now()
	defer func() {
		fmt.Printf("isDockerTestNetworkPresent took -> %s\n", time.Since(start))
	}()
	cli, err := dockerClient()
	if err != nil {
		return false, fmt.Errorf("while creating docker client: %w", err)
	}

	filterBy := filters.NewArgs()
	filterBy.Add("name", TestDockerUserNetwork)
	filterBy.Add("driver", "bridge")
	list, err := cli.NetworkList(context.Background(), types.NetworkListOptions{Filters: filterBy})

	if err == nil {
		return len(list) == 1, nil
	}

	return false, fmt.Errorf("while testing network availbility: %w", err)
}

func createTestNetworkForDB(ctx context.Context) (*types.NetworkResource, error) {
	start := time.Now()
	defer func() {
		fmt.Printf("createTestNetworkForDB took -> %s\n", time.Since(start))
	}()
	cli, err := dockerClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create a Docker client: %w", err)
	}

	createdNetworkResponse, err := cli.NetworkCreate(context.Background(), TestDockerUserNetwork, types.NetworkCreate{Driver: "bridge"})
	if err != nil {
		return nil, fmt.Errorf("failed to create docker user network: %w", err)
	}

	filterBy := filters.NewArgs()
	filterBy.Add("id", createdNetworkResponse.ID)
	list, err := cli.NetworkList(context.Background(), types.NetworkListOptions{Filters: filterBy})

	if err != nil || len(list) != 1 {
		return nil, fmt.Errorf("network not found or not created: %w", err)
	}

	return &list[0], nil
}

func EnsureTestNetworkForDB(t *testing.T, ctx context.Context) (func(), error) {
	start := time.Now()
	defer func() {
		fmt.Printf("EnsureTestNetworkForDB took -> %s\n", time.Since(start))
	}()
	exec.Command("systemctl start docker.service")

	networkPresent, err := isDockerTestNetworkPresent(ctx)
	if networkPresent && err == nil {
		return func() {}, nil
	}

	createdNetwork, err := createTestNetworkForDB(ctx)

	if err != nil {
		return func() {}, fmt.Errorf("while creating test network: %w", err)
	}

	cli, err := dockerClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create a Docker client: %w", err)
	}

	cleanupFunc := func() {
		err = cli.NetworkRemove(ctx, createdNetwork.ID)
		assert.NoError(t, err)
		time.Sleep(1 * time.Second)
	}

	return cleanupFunc, nil
}

func SetupTestNetworkForDB(ctx context.Context) (cleanupFunc func(), err error) {
	start := time.Now()
	defer func() {
		fmt.Printf("SetupTestNetworkForDB took -> %s\n", time.Since(start))
	}()
	exec.Command("systemctl start docker.service")

	networkPresent, err := isDockerTestNetworkPresent(ctx)
	if networkPresent && err == nil {
		return func() {}, nil
	}

	createdNetwork, err := createTestNetworkForDB(ctx)

	if err != nil {
		return func() {}, fmt.Errorf("while creating test network: %w", err)
	}

	cli, err := dockerClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create a Docker client: %w", err)
	}
	cleanupFunc = func() {
		err = cli.NetworkRemove(ctx, createdNetwork.ID)
		if err != nil {
			err = fmt.Errorf("failed to remove docker network: %w + %s", err, TestDockerUserNetwork)
		}
		time.Sleep(1 * time.Second)
	}

	if err != nil {
		return cleanupFunc, fmt.Errorf("while DB setup: %w", err)
	} else {
		return cleanupFunc, nil
	}
}

func clearDBQuery() string {
	start := time.Now()
	defer func() {
		fmt.Printf("clearDBQuery took -> %s\n", time.Since(start))
	}()
	return fmt.Sprintf("TRUNCATE TABLE %s, %s, %s, %s RESTART IDENTITY CASCADE",
		postsql.InstancesTableName,
		postsql.OperationTableName,
		postsql.OrchestrationTableName,
		postsql.RuntimeStateTableName,
	)
}

func waitFor(cli *client.Client, containerId string, text string) error {
	return wait.PollImmediate(1*time.Millisecond, 10*time.Second, func() (done bool, err error) {
		out, err := cli.ContainerLogs(context.Background(), containerId, types.ContainerLogsOptions{ShowStdout: true})
		if err != nil {
			return true, fmt.Errorf("while loading logs: %w", err)
		}

		bufReader := bufio.NewReader(out)
		defer out.Close()

		for line, isPrefix, err := bufReader.ReadLine(); err == nil; line, isPrefix, err = bufReader.ReadLine() {
			if !isPrefix && strings.Contains(string(line), text) {
				return true, nil
			}
		}

		if err != nil {
			return false, fmt.Errorf("while waiting for a container: %w", err)
		}

		return true, nil
	})
}
