# API (Control Plane) - Development Guide

The control plane server for fluid.sh. Provides REST API, gRPC streaming to daemons, multi-host orchestration, web dashboard backend, and agent execution.

## Architecture

```
Web Dashboard / SDK
  |
  v (REST API)
api server (:8080)
  |
  +--- PostgreSQL (state)
  |
  v (gRPC stream)
fluid-daemon (per host)
```

## Tech Stack

- **Language**: Go
- **REST**: Standard library HTTP + custom router
- **gRPC**: Bidirectional streaming to daemons
- **Database**: PostgreSQL
- **Auth**: OAuth, password, host token authentication

## Project Structure

```
api/
  cmd/server/main.go       # Entry point
  internal/
    agent/                  # LLM agent executor + tools
    auth/                   # OAuth, password, host auth, middleware
    config/                 # Configuration loading
    error/                  # Error response helpers
    grpc/                   # gRPC server for daemon connections
    json/                   # JSON encode/decode helpers
    orchestrator/           # Multi-host sandbox orchestration
    registry/               # Source VM registry
    rest/                   # REST API handlers
    store/                  # PostgreSQL store
  Makefile
```

## Quick Start

```bash
# Build
make build

# Run (requires PostgreSQL)
./bin/api

# Run tests
make test
```

## Makefile Targets

| Target | Description |
|--------|-------------|
| `all` | Run fmt, vet, test, and build (default) |
| `build` | Build the API binary |
| `run` | Build and run the API |
| `clean` | Clean build artifacts |
| `fmt` | Format code |
| `vet` | Run go vet |
| `test` | Run tests |
| `test-coverage` | Run tests with coverage |
| `check` | Run all code quality checks |
| `deps` | Download dependencies |
| `tidy` | Tidy and verify dependencies |
| `install` | Install to GOPATH/bin |

## Database Setup

```bash
# Create database
sudo -u postgres psql -c "CREATE DATABASE fluid;"
sudo -u postgres psql -c "CREATE USER fluid WITH PASSWORD 'fluid';"
sudo -u postgres psql -c "GRANT ALL PRIVILEGES ON DATABASE fluid TO fluid;"

# Schema is auto-migrated on startup via GORM AutoMigrate
```

## Development

### Prerequisites

- Go 1.24+
- PostgreSQL 14+

### Testing

```bash
make test           # Run all tests
make test-coverage  # Tests with coverage report
make check          # Run all quality checks
```
