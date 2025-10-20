# Orca

A web UI for [SST OpenCode](https://opencode.sh) that runs on desktop, primarily targeting mobile devices for remote code editing and AI-assisted development.

## Overview

Orca provides a mobile-friendly web interface to OpenCode, allowing you to interact with Claude for software development tasks from any device on your local network.

**Architecture:**
- **Backend**: Go server with WebSocket support for real-time OpenCode communication
- **Frontend**: React + TypeScript with Vite, styled with Tailwind CSS v4

## Prerequisites

- [Go](https://go.dev/) 1.25+
- [pnpm](https://pnpm.io/)
- [just](https://github.com/casey/just) (command runner)
- [air](https://github.com/air-verse/air) (for Go hot reload in development)

## Development

Run the frontend and backend in separate terminals:

```bash
just frontend
```

```bash
just backend
```

The frontend dev server runs on `http://localhost:5173` and the backend proxies requests to it during development.

## Production Build

Build and run the production bundle:

```bash
just prod
```

This builds the frontend, copies assets to the backend's public directory, and starts the server.

## Available Commands

- `just frontend` - Start frontend dev server
- `just backend` - Start backend with hot reload
- `just fmt` - Format Go and TypeScript code
- `just prod` - Build and run production server
- `just clean` - Remove build artifacts

## Tech Stack

**Backend:**
- Go with native `net/http`
- Gorilla WebSocket
- [OpenCode Go SDK](https://github.com/sst/opencode-sdk-go)

**Frontend:**
- React 19
- TypeScript
- Vite 7
- Tailwind CSS v4
- TanStack React Query
- [OpenCode SDK](https://github.com/opencode-ai/sdk)
- Shiki (syntax highlighting)

## License

See [LICENSE](LICENSE)
