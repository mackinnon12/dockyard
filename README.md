# Dockyard

Dockyard is a desktop application for managing local MySQL Docker containers. It provides a simple Vue and PrimeVue interface backed by Go and the Wails framework.

## Features

- Create a MySQL 8.4 container with a persistent Docker volume.
- Configure the container name, database, root password, and host port.
- View MySQL containers, status, public address, and mapped port.
- Start, stop, restart, and delete containers.
- See image-pull and creation progress in the frontend.
- Switch between light and dark mode.

## Requirements

- Go 1.25 or newer
- Node.js and npm
- Wails v2
- Docker Desktop or a running Docker Engine

Docker does not need to be running for the application window to open, but container operations require an available Docker daemon.

## Development

Install the frontend dependencies and start the Wails development app:

```bash
cd frontend
npm install
cd ..
wails dev
```

The frontend can also be run independently with Vite:

```bash
cd frontend
npm run dev
```

## Production Build

Build the desktop application with:

```bash
wails build
```

The generated application is written to `build/bin`.

## Project Structure

- `app.go` — Wails bindings exposed to the frontend.
- `internal/database` — Docker client, MySQL creation flow, and container operations.
- `frontend/src/views/BlankCanvas.vue` — the main application interface.
- `frontend/wailsjs` — generated JavaScript and TypeScript bindings.

