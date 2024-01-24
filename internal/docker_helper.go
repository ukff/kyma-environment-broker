package internal

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

type DockerHelper struct {
	client *client.Client
}

func NewDockerHandler() (*DockerHelper, error) {
	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	fmt.Println(fmt.Sprintf("host is -> %s", dockerClient.DaemonHost()))

	return &DockerHelper{
		client: dockerClient,
	}, nil
}

type ContainerCreateRequest struct {
	Port          string
	User          string
	Password      string
	Name          string
	Host          string
	ContainerName string
	Image         string
	Envs          []string
}

func (d *DockerHelper) CreateDBContainer(config ContainerCreateRequest) (func() error, error) {
	_, err := d.client.Ping(context.Background())
	if err != nil {
		return nil, fmt.Errorf("ping docker failed with: %w", err)
	}

	filterBy := filters.NewArgs()
	filterBy.Add("name", config.Image)
	image, err := d.client.ImageList(context.Background(), types.ImageListOptions{Filters: filterBy})

	if image == nil || err != nil {
		log.Print(fmt.Sprintf("Image %s not found... pulling...", config.Image))
		reader, err := d.client.ImagePull(context.Background(), config.Image, types.ImagePullOptions{})
		if err != nil {
			return nil, fmt.Errorf("while pulling dbImage: %w of %s", err, config.Image)
		}
		defer reader.Close()
		_, err = io.Copy(os.Stdout, reader)
		if err != nil {
			return nil, fmt.Errorf("while handling dbImage: %w of %s", err, config.Name)
		}
	}

	/*portMapping := make(map[nat.Port][]nat.PortBinding, 1)
	portMapping["5432"] = []nat.PortBinding{{
		HostIP:   config.Host,
		HostPort: config.Port,
	}}*/
	_, parsedPortSpecs, err := nat.ParsePortSpecs([]string{config.Port})
	if err != nil {
		return nil, fmt.Errorf("while parsing ports specs: %w", err)
	}

	body, err := d.client.ContainerCreate(context.Background(),
		&container.Config{
			Image: config.Image,
			Env:   config.Envs,
		},
		&container.HostConfig{
			NetworkMode:     "default",
			PublishAllPorts: false,
			AutoRemove:      true,
			PortBindings:    parsedPortSpecs,
		},
		nil,
		nil,
		config.ContainerName)

	if err != nil {
		return nil, fmt.Errorf("during container creation: %w", err)
	}

	cleanupFunc := func() error {
		err := d.client.ContainerRemove(context.Background(), body.ID, types.ContainerRemoveOptions{RemoveVolumes: true, RemoveLinks: false, Force: true})
		if err != nil {
			return fmt.Errorf("during container removal: %w", err)
		}
		return nil
	}

	if err := d.client.ContainerStart(context.Background(), body.ID, types.ContainerStartOptions{}); err != nil {
		return cleanupFunc, fmt.Errorf("during container startup: %w", err)
	}

	statusCh, errCh := d.client.ContainerWait(context.Background(), body.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return cleanupFunc, err
		}
	case <-statusCh:
	}

	return cleanupFunc, nil
}

func (d *DockerHelper) CloseDockerClient() error {
	if d.client == nil {
		return fmt.Errorf("docker client is nil")
	}

	err := d.client.Close()
	if err != nil {
		return fmt.Errorf("while closing docker client: %s", err.Error())
	}
	return nil
}

func (d *DockerHelper) GetContainerLogs(containerName string) (string, error) {
	reader, err := d.client.ContainerLogs(context.Background(), containerName, types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true})
	if err != nil {
		return "", fmt.Errorf("while getting container logs: %w", err)
	}
	defer reader.Close()
	buf := new(strings.Builder)
	_, err = io.Copy(buf, reader)
	if err != nil {
		return "", fmt.Errorf("while reading container logs: %w", err)
	}
	return buf.String(), nil
}
