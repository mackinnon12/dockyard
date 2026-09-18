package database

import (
	"context"
	"fmt"
	"strings"

	"github.com/moby/moby/client"
)

var DATABASE_IMAGES = []string{"mysql:"}

type Container struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Image         string `json:"image"`
	State         string `json:"state"`
	Status        string `json:"status"`
	PublicAddress string `json:"publicAddress"`
	PublicPort    string `json:"publicPort"`
}

func IsDatabaseImage(image string) bool {
	for _, allowedImage := range DATABASE_IMAGES {
		if strings.HasPrefix(image, allowedImage) {
			return true
		}
	}
	return false
}

func (d *DatabaseManager) Start(id string) error {

	ctx := context.Background()
	_, err := d.docker.ContainerStart(ctx, id, client.ContainerStartOptions{})

	if err != nil {
		return err
	}
	return nil
}

func (d *DatabaseManager) Stop(id string) error {
	ctx := context.Background()
	_, err := d.docker.ContainerStop(ctx, id, client.ContainerStopOptions{})

	if err != nil {
		return err
	}
	return nil
}

func (d *DatabaseManager) Restart(id string) error {
	ctx := context.Background()
	_, err := d.docker.ContainerRestart(ctx, id, client.ContainerRestartOptions{})

	if err != nil {
		return err
	}
	return nil
}

func (d *DatabaseManager) Delete(id string) error {
	ctx := context.Background()
	_, err := d.docker.ContainerRemove(ctx, id, client.ContainerRemoveOptions{})

	if err != nil {
		return err
	}
	return nil
}

func (d *DatabaseManager) Containers() ([]Container, error) {

	ctx := context.Background()

	allContainers, err := d.docker.ContainerList(ctx, client.ContainerListOptions{
		All: true,
	})

	if err != nil {
		return nil, err
	}

	containers := make([]Container, 0, len(allContainers.Items))

	for _, container := range allContainers.Items {
		name := container.ID[:min(12, len(container.ID))]
		if len(container.Names) > 0 {
			name = strings.TrimPrefix(container.Names[0], "/")
		}

		if IsDatabaseImage(container.Image) {
			publicAddress := ""
			publicPort := ""
			for _, port := range container.Ports {
				if port.PublicPort == 0 {
					continue
				}

				if port.IP.IsValid() {
					publicAddress = port.IP.String()
				}
				publicPort = fmt.Sprintf("%d", port.PublicPort)
				break
			}

			containers = append(containers, Container{
				ID:            container.ID,
				Name:          name,
				Image:         container.Image,
				State:         string(container.State),
				Status:        container.Status,
				PublicAddress: publicAddress,
				PublicPort:    publicPort,
			})
		}
	}

	return containers, nil
}
