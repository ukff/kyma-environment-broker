package storage

import (
	"bufio"
	"context"
	"fmt"
	"io"
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
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	testDockerUserNetwork = "testnetwork"
)

type ContainerCreateConfig struct {
	Port          string
	User          string
	Password      string
	Name          string
	Host          string
	ContainerName string
	Image         string
}

func dockerClient() (*client.Client, error) {
	return client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
}

func ExtractPortFromContainer(container types.Container) (string, error) {
	if container.Ports == nil || len(container.Ports) == 0 {
		return "", fmt.Errorf("no ports: %w", nil)
	}

	port := container.Ports[0].PublicPort
	if port == 0 {
		return "", fmt.Errorf("port is 0 %w", nil)
	}

	return fmt.Sprint(container.Ports[0].PublicPort), nil
}

func CreateDBContainer(config ContainerCreateConfig) (func(), *types.Container, error) {
	cli, err := dockerClient()
	if err != nil {
		return nil, &types.Container{}, fmt.Errorf("while creating docker client: %w", err)
	}

	filterBy := filters.NewArgs()
	filterBy.Add("name", config.Image)
	image, err := cli.ImageList(context.Background(), types.ImageListOptions{Filters: filterBy})

	if image == nil || err != nil {
		log.Print("Image not found... pulling...")
		reader, err := cli.ImagePull(context.Background(), config.Image, types.ImagePullOptions{})
		if err != nil {
			return nil, &types.Container{}, fmt.Errorf("while pulling dbImage: %w", err)
		}
		defer reader.Close()
		_, err = io.Copy(os.Stdout, reader)
		if err != nil {
			return nil, &types.Container{}, fmt.Errorf("while handling dbImage: %w", err)
		}
	}

	_, parsedPortSpecs, err := nat.ParsePortSpecs([]string{config.Port})
	if err != nil {
		return nil, &types.Container{}, fmt.Errorf("while parsing ports specs: %w", err)
	}

	body, err := cli.ContainerCreate(context.Background(),
		&container.Config{
			Image: config.Image,
			Env: []string{
				fmt.Sprintf("POSTGRES_USER=%s", config.User),
				fmt.Sprintf("POSTGRES_PASSWORD=%s", config.Password),
				fmt.Sprintf("POSTGRES_DB=%s", config.Name),
			},
		},
		&container.HostConfig{
			NetworkMode:     "default",
			PublishAllPorts: false,
			PortBindings:    parsedPortSpecs,
		},
		&network.NetworkingConfig{
			EndpointsConfig: map[string]*network.EndpointSettings{
				testDockerUserNetwork: {
					Aliases: []string{
						config.Host,
					},
				},
			},
		},
		&v1.Platform{},
		config.ContainerName)

	if err != nil {
		return nil, &types.Container{}, fmt.Errorf("during container creation: %w", err)
	}

	cleanupFunc := func() {
		err := cli.ContainerRemove(context.Background(), body.ID, types.ContainerRemoveOptions{RemoveVolumes: true, RemoveLinks: false, Force: true})
		if err != nil {
			panic(fmt.Errorf("during container removal: %w", err))
		}
	}

	if err := cli.ContainerStart(context.Background(), body.ID, types.ContainerStartOptions{}); err != nil {
		return cleanupFunc, &types.Container{}, fmt.Errorf("during container startup: %w", err)
	}

	err = waitForContainer(cli, body.ID, "database system is ready to accept connections")
	if err != nil {
		log.Printf("Failed to query container's logs: %s", err)
		return cleanupFunc, &types.Container{}, fmt.Errorf("while waiting for DB readiness: %w", err)
	}

	filterBy = filters.NewArgs()
	filterBy.Add("id", body.ID)
	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{Filters: filterBy})

	if err != nil || len(containers) == 0 {
		log.Printf("no containers found: %s", err)
		return cleanupFunc, &types.Container{}, fmt.Errorf("while loading containers: %w", err)
	}

	var created = &containers[0]
	if created == nil {
		log.Printf("no container found: %s", err)
		return cleanupFunc, &types.Container{}, fmt.Errorf("while searching for a container: %w", err)
	}

	return cleanupFunc, created, nil
}

func isDockerTestNetworkPresent() (bool, error) {
	cli, err := dockerClient()
	if err != nil {
		return false, fmt.Errorf("while creating docker client: %w", err)
	}

	filterBy := filters.NewArgs()
	filterBy.Add("name", testDockerUserNetwork)
	filterBy.Add("driver", "bridge")
	list, err := cli.NetworkList(context.Background(), types.NetworkListOptions{Filters: filterBy})

	if err == nil {
		return len(list) == 1, nil
	}

	return false, fmt.Errorf("while testing network availbility: %w", err)
}

func createTestNetworkForDB() (*types.NetworkResource, error) {
	cli, err := dockerClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create a Docker client: %w", err)
	}

	createdNetworkResponse, err := cli.NetworkCreate(context.Background(), testDockerUserNetwork, types.NetworkCreate{Driver: "bridge"})
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
	exec.Command("systemctl start docker.service")

	networkPresent, err := isDockerTestNetworkPresent()
	if networkPresent && err == nil {
		return func() {}, nil
	}

	createdNetwork, err := createTestNetworkForDB()

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

func SetupTestNetworkForDB() (cleanupFunc func(), err error) {
	exec.Command("systemctl start docker.service")

	networkPresent, err := isDockerTestNetworkPresent()
	if networkPresent && err == nil {
		return func() {}, nil
	}

	createdNetwork, err := createTestNetworkForDB()

	if err != nil {
		return func() {}, fmt.Errorf("while creating test network: %w", err)
	}

	cli, err := dockerClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create a Docker client: %w", err)
	}
	cleanupFunc = func() {
		err = cli.NetworkRemove(context.Background(), createdNetwork.ID)
		if err != nil {
			err = fmt.Errorf("failed to remove docker network: %w + %s", err, testDockerUserNetwork)
		}
		time.Sleep(1 * time.Second)
	}

	if err != nil {
		return cleanupFunc, fmt.Errorf("while DB setup: %w", err)
	} else {
		return cleanupFunc, nil
	}
}

func waitForContainer(cli *client.Client, containerId string, text string) error {
	return wait.PollImmediate(1*time.Second, 10*time.Second, func() (done bool, err error) {
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
