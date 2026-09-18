package database

import (
	"github.com/moby/moby/client"
)

type DatabaseManager struct {
	docker *client.Client
}

func NewDatabaseManager() (*DatabaseManager, error) {
	cli, err := client.New(client.FromEnv)

	if err != nil {
		return nil, err
	}

	dbManager := DatabaseManager{
		docker: cli,
	}

	return &dbManager, nil
}

func (d *DatabaseManager) Close() error {
	if d == nil || d.docker == nil {
		return nil
	}

	return d.docker.Close()
}
