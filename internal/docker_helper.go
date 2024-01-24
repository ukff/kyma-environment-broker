package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"

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

	log.Println("creating container...")
	containerPortBinding := nat.PortMap{
		nat.Port("5432" + "/tcp"): []nat.PortBinding{{
			HostIP:   "0.0.0.0",
			HostPort: config.Port,
		}},
	}

	response, err := d.client.ContainerCreate(context.Background(),
		&container.Config{
			ExposedPorts: map[nat.Port]struct{}{
				"5432" + "/tcp": {},
			},
			Image: config.Image,
			Env: []string{
				fmt.Sprintf("POSTGRES_USER=%s", config.User),
				fmt.Sprintf("POSTGRES_PASSWORD=%s", config.Password),
				fmt.Sprintf("POSTGRES_DB=%s", config.Name),
			},
		},
		&container.HostConfig{
			PortBindings: containerPortBinding,
		},
		nil,
		nil,
		"")
	log.Printf("container started with ID: %s", response.ID)
	log.Printf("container started with name: %v", response.Warnings)
	if err != nil {
		return nil, fmt.Errorf("during container creation: %w", err)
	}

	cleanupFunc := func() error {
		log.Println("starting cleanUp function...")
		err := d.client.ContainerRemove(context.Background(), response.ID, types.ContainerRemoveOptions{RemoveVolumes: true, RemoveLinks: false, Force: true})
		if err != nil {
			return fmt.Errorf("during container removal: %w", err)
		}
		return nil
	}
	log.Println("starting cleanUp function...")
	log.Println("starting container function...")

	if err := d.client.ContainerStart(context.Background(), response.ID, types.ContainerStartOptions{}); err != nil {
		return cleanupFunc, fmt.Errorf("during container startup: %w", err)
	}
	log.Println("container started...")

	j, err := d.client.ContainerInspect(context.Background(), response.ID)
	if err != nil {
		return cleanupFunc, fmt.Errorf("during container inspect: %w", err)
	}
	log.Printf("container inspect: %v", j)
	res2B, _ := json.Marshal(j)
	fmt.Println(string(res2B))

	/*statusCh, errCh := d.client.ContainerWait(context.Background(), response.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			panic(err)
		}
	case <-statusCh:
	}*/

	log.Println("container created OK..")

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
