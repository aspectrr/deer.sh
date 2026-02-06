# fluid.sh

Autonomous AI agents for infrastructure -- with human approval.

## What This Is

fluid.sh lets AI agents do infrastructure work (provision servers, configure services, set up networking) in isolated VM sandboxes. The agent works autonomously. A human reviews and approves before production.

## Architecture

```
Agent Task -> Sandbox VM (autonomous) -> Human Approval -> Production
```

- **fluid/** - Go CLI & API server. Manages VMs via libvirt/KVM.
- **web/** - React frontend. Monitor sandboxes, approve actions.
- **sdk/** - Python SDK. Build agents that talk to the API.
- **examples/** - Working agent implementations.

## Quick Start

```bash
docker-compose up --build

# API:      http://localhost:8080
# Web UI:   http://localhost:5173
```

## Project Rules

### Testing Required

Every code change needs tests. No exceptions.

- Go: `*_test.go` files
- Python: `test/test_client.py`
- Web: Component tests as needed

### Building

Use docker-compose:

```bash
docker-compose up fluid            # API server
docker-compose up web              # Frontend
docker-compose up postgres         # Database
```

### Project-Specific Docs

- @fluid/AGENTS.md - API server details
- @sdk/AGENTS.md - Python SDK details
- @web/AGENTS.md - Frontend details
- @examples/agent-example/AGENTS.md - Agent example

## Service Ports

| Service | Port | Purpose |
|---------|------|---------|
| fluid | 8080 | REST API for VM management |
| web | 5173 | React UI |
| PostgreSQL | 5432 | State persistence |

## Key Commands

```bash
# Go services
cd fluid && make test && make check

# Python SDK
cd sdk/fluid-sdk-py && pytest

# Frontend
cd web && bun run lint && bun run build
```

## macOS Setup

```bash
brew install libvirt socket_vmnet cdrtools
brew services start libvirt
./scripts/setup-ssh-ca.sh --dir .ssh-ca
./scripts/reset-libvirt-macos.sh
```

## Environment Variables

```bash
LIBVIRT_URI=qemu:///system          # libvirt connection
DATABASE_URL=postgresql://...        # postgres connection
API_HTTP_ADDR=:8080                  # API listen address
```

See `.env.example` for full list.
