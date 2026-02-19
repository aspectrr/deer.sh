# Fluid Daemon - Development Guide

Background service that manages VM sandboxes on a sandbox host. One daemon runs per sandbox host, but each daemon can connect to multiple libvirt hosts over SSH for source VM access. Multiple daemons are typically needed for heavily NATed enterprise networks or separate data centers. Exposes a gRPC API for the CLI and optionally connects upstream to the control plane.

## Architecture

```
fluid CLI (TUI/MCP)
  |
  v (gRPC :9091)
fluid-daemon
  |
  +--- libvirt/KVM (sandbox VMs)
  +--- SQLite (local state)
  +--- SSH CA (ephemeral certs)
  +--- Janitor (TTL cleanup)
  |
  v (optional gRPC stream)
control-plane
```

## Tech Stack

- **Language**: Go
- **VM Backend**: QEMU microVMs via libvirt
- **State**: SQLite
- **Networking**: Bridge + TAP devices
- **SSH**: Internal CA with ephemeral certificates

## Project Structure

```
fluid-daemon/
  cmd/fluid-daemon/main.go   # Entry point
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
go build -o bin/fluid-daemon ./cmd/fluid-daemon

# Run
sudo ./bin/fluid-daemon serve

# Run with systemd
sudo systemctl enable --now fluid-daemon
```

## Configuration

Default config: `~/.config/fluid/daemon.yaml`

```yaml
listen:
  grpc: ":9091"

backend: qemu

storage:
  images: /var/lib/fluid/images
  overlays: /var/lib/fluid/overlays
  state: /var/lib/fluid/state.db

network:
  bridge: fluid0
  subnet: 10.0.0.0/24

# Optional: connect to control plane
# control_plane:
#   address: "cp.fluid.sh:9090"
#   token: "your-host-token"
```

## Development

### Prerequisites

- Go 1.24+
- libvirt/KVM
- Root access (for network/VM management)

### Testing

```bash
go test ./... -v
go test ./... -coverprofile=coverage.out
```

### Build

```bash
go build -o bin/fluid-daemon ./cmd/fluid-daemon
```
