# Deer Daemon - Development Guide

## Communication Style

Always use the caveman skill (`/caveman`) for all responses.



Background service that manages microVM sandboxes on a sandbox host. One daemon runs per sandbox host. Multiple daemons are typically needed for heavily NATed enterprise networks or separate data centers. Exposes a gRPC API for the CLI and optionally connects upstream to the control plane.

## Architecture

```
deer CLI (TUI/MCP)
  |
  v (gRPC :9091)
deer-daemon
  |
  +--- QEMU microVMs (sandboxes)
  +--- SQLite (local state)
  +--- SSH CA (ephemeral certs)
  +--- Janitor (TTL cleanup)
  |
  v (optional gRPC stream)
control-plane
```

## Tech Stack

- **Language**: Go
- **VM Backend**: QEMU microVMs
- **State**: SQLite
- **Networking**: Bridge + TAP devices
- **SSH**: Internal CA with ephemeral certificates

## Project Structure

```
deer-daemon/
  cmd/deer-daemon/main.go   # Entry point
  internal/
    agent/                    # Control plane gRPC client + reconnect
    config/                   # Configuration loading
    daemon/                   # Main daemon orchestration
    image/                    # Image extraction and caching
    janitor/                  # TTL-based sandbox cleanup
    microvm/                  # MicroVM manager (overlay, boot)
    network/                  # Bridge + TAP device management
    provider/                 # VM provider abstraction
    readonly/                 # Read-only source VM access
    sourcevm/                 # Source VM manager
    sshca/                    # SSH Certificate Authority
    sshkeys/                  # SSH key management
    state/                    # SQLite state store
  Makefile
```

## Quick Start

```bash
# Build
go build -o bin/deer-daemon ./cmd/deer-daemon

# Run
sudo ./bin/deer-daemon serve

# Run with systemd
sudo systemctl enable --now deer-daemon
```

## Configuration

Default config: `~/.config/deer/daemon.yaml`

```yaml
listen:
  grpc: ":9091"

backend: qemu

storage:
  images: /var/lib/deer/images
  overlays: /var/lib/deer/overlays
  state: /var/lib/deer/state.db

network:
  bridge: deer0
  subnet: 10.0.0.0/24

# Optional: connect to control plane
# control_plane:
#   address: "cp.deer.sh:9090"
#   token: "your-host-token"
```

## Development

### Prerequisites

- Go 1.24+
- QEMU/KVM
- Root access (for network/VM management)

### Testing

```bash
go test ./... -v
go test ./... -coverprofile=coverage.out
```

### Build

```bash
go build -o bin/deer-daemon ./cmd/deer-daemon
```
