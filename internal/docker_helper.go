package internal

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
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
	response, err := d.client.ContainerCreate(context.Background(),
		&container.Config{
			Image: config.Image,
			Env: []string{
				fmt.Sprintf("POSTGRES_USER=%s", config.User),
				fmt.Sprintf("POSTGRES_PASSWORD=%s", config.Password),
				fmt.Sprintf("POSTGRES_DB=%s", config.Name),
			},
		},
		nil,
		nil,
		nil,
		config.ContainerName)
	log.Printf("container started with ID: %s", response.ID)
	if err != nil {
		return nil, fmt.Errorf("during container creation: %w", err)
	}
	
	cleanupFunc := func() error {
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
	
	/*_, errCh := d.client.ContainerWait(context.Background(), response.ID, container.WaitConditionNotRunning)
	if err := <-errCh; err != nil {
		return cleanupFunc, nil
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
