package database

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/netip"
	"regexp"
	"time"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
)

type MySQLConfig struct {
	Name         string `json:"name"`
	Version      string `json:"version"`
	Port         int    `json:"port"`
	RootPassword string `json:"rootPassword"`
	Database     string `json:"database"`
}

type PullProgress struct {
	ID          string `json:"id"`
	Status      string `json:"status"`
	Progress    string `json:"progress"`
	Stage       string `json:"stage"`
	Message     string `json:"message"`
	ContainerID string `json:"containerId,omitempty"`
}

var ALLOWED_MYSQL_VERSIONS = []string{"8.4"}

func IsAllowedVersion(version string) bool {
	for _, allowedVersion := range ALLOWED_MYSQL_VERSIONS {
		if allowedVersion == version {
			return true
		}
	}
	return false
}

func waitForMySQL(ctx context.Context, port int) error {
	address := fmt.Sprintf("127.0.0.1:%d", port)
	timeout := time.NewTimer(30 * time.Second)
	ticker := time.NewTicker(500 * time.Millisecond)

	defer timeout.Stop()
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout.C:
			return fmt.Errorf("timed out waiting for MySQL on %s", address)

		case <-ticker.C:
			conn, err := net.DialTimeout("tcp", address, time.Second)

			if err != nil {
				continue
			}

			conn.Close()
			return nil
		}
	}
}

func validateMySQLConfig(config MySQLConfig) error {
	if !IsAllowedVersion(config.Version) {
		return fmt.Errorf("invalid version %q; allowed versions: %v", config.Version, ALLOWED_MYSQL_VERSIONS)
	}

	if !regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,62}$`).MatchString(config.Name) {
		return fmt.Errorf("name must start with a letter or number and contain only letters, numbers, hyphens, or underscores")
	}

	if config.Port < 1 || config.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}

	if config.RootPassword == "" {
		return fmt.Errorf("root password is required")
	}

	if !regexp.MustCompile(`^[a-zA-Z0-9_$-]+$`).MatchString(config.Database) {
		return fmt.Errorf("database must contain only letters, numbers, underscores, dollar signs, or hyphens")
	}

	return nil
}

func (d *DatabaseManager) CreateMySQL(ctx context.Context, config MySQLConfig, onProgress func(PullProgress)) error {
	if err := validateMySQLConfig(config); err != nil {
		return err
	}

	if ctx == nil {
		ctx = context.Background()
	}

	image := fmt.Sprintf("mysql:%s", config.Version)

	report := func(progress PullProgress) {
		if onProgress != nil {
			onProgress(progress)
		}
	}

	report(PullProgress{Stage: "pull", Status: "starting", Message: fmt.Sprintf("Pulling %s", image)})

	response, err := d.docker.ImagePull(ctx, image, client.ImagePullOptions{})

	if err != nil {
		return err
	}

	defer response.Close()

	decoder := json.NewDecoder(response)

	for {
		var progress PullProgress

		err := decoder.Decode(&progress)

		if err == io.EOF {
			break
		}

		if err != nil {
			return err
		}

		progress.Stage = "pull"
		report(progress)
	}

	volumeName := fmt.Sprintf("dockyard-%s-data", config.Name)
	report(PullProgress{Stage: "volume", Status: "creating", Message: fmt.Sprintf("Creating volume %s", volumeName)})

	_, err = d.docker.VolumeCreate(ctx, client.VolumeCreateOptions{
		Name: volumeName,
	})

	if err != nil {
		return err
	}

	env := []string{
		fmt.Sprintf("MYSQL_ROOT_PASSWORD=%s", config.RootPassword),
		fmt.Sprintf("MYSQL_DATABASE=%s", config.Database),
	}

	portBindings := network.PortMap{
		network.MustParsePort("3306/tcp"): []network.PortBinding{
			{
				HostIP:   netip.MustParseAddr("127.0.0.1"),
				HostPort: fmt.Sprintf("%d", config.Port),
			},
		},
	}

	containerConfig := &container.Config{
		Image: image,
		Env:   env,
		ExposedPorts: network.PortSet{
			network.MustParsePort("3306/tcp"): struct{}{},
		},
	}

	hostConfig := &container.HostConfig{
		PortBindings: portBindings,
		Binds: []string{
			fmt.Sprintf("%s:/var/lib/mysql", volumeName),
		},
	}

	containerName := fmt.Sprintf("dockyard-%s", config.Name)
	report(PullProgress{Stage: "container", Status: "creating", Message: fmt.Sprintf("Creating container %s", containerName)})

	created, err := d.docker.ContainerCreate(ctx, client.ContainerCreateOptions{
		Config:     containerConfig,
		HostConfig: hostConfig,
		Name:       containerName,
	})

	if err != nil {
		return err
	}

	report(PullProgress{ID: created.ID, Stage: "container", Status: "created", ContainerID: created.ID, Message: fmt.Sprintf("Created container %s", containerName)})

	_, err = d.docker.ContainerStart(
		ctx,
		created.ID,
		client.ContainerStartOptions{},
	)

	if err != nil {
		return err
	}

	report(PullProgress{ID: created.ID, Stage: "ready", Status: "waiting", ContainerID: created.ID, Message: fmt.Sprintf("Waiting for MySQL on 127.0.0.1:%d", config.Port)})

	if err := waitForMySQL(ctx, config.Port); err != nil {
		return err
	}

	report(PullProgress{ID: created.ID, Stage: "ready", Status: "complete", Progress: "100%", ContainerID: created.ID, Message: fmt.Sprintf("MySQL is ready on 127.0.0.1:%d", config.Port)})

	return nil
}
