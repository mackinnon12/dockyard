package main

import (
	"context"

	"dockyard/internal/database"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx             context.Context
	databaseManager *database.DatabaseManager
}

// NewApp creates a new App application struct
func NewApp(databaseManager *database.DatabaseManager) *App {
	return &App{databaseManager: databaseManager}
}

func (a *App) ListContainers() ([]database.Container, error) {
	return a.databaseManager.Containers()
}

func (a *App) StartContainer(id string) error {
	return a.databaseManager.Start(id)
}

func (a *App) StopContainer(id string) error {
	return a.databaseManager.Stop(id)
}

func (a *App) RestartContainer(id string) error {
	return a.databaseManager.Restart(id)
}

func (a *App) DeleteContainer(id string) error {
	return a.databaseManager.Delete(id)
}

func (a *App) CreateMySQL(config database.MySQLConfig) error {
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}

	return a.databaseManager.CreateMySQL(ctx, config, func(progress database.PullProgress) {
		runtime.EventsEmit(ctx, "mysql:creation-progress", progress)
	})
}

// startup is called at application startup
func (a *App) startup(ctx context.Context) {
	// Perform your setup here
	a.ctx = ctx
}

// domReady is called after front-end resources have been loaded
func (a App) domReady(ctx context.Context) {
	// Add your action here
}

// beforeClose is called when the application is about to quit,
// either by clicking the window close button or calling runtime.Quit.
// Returning true will cause the application to continue, false will continue shutdown as normal.
func (a *App) beforeClose(ctx context.Context) (prevent bool) {
	return false
}

// shutdown is called at application termination
func (a *App) shutdown(ctx context.Context) {
	if err := a.databaseManager.Close(); err != nil {
		println("failed to close Docker client: " + err.Error())
	}
}
