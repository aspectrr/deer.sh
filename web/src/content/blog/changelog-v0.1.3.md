---
title: 'Changelog #003 v0.1.3'
pubDate: 2026-02-23
description: 'Fluid Daemon, MCP server with 17 tools, Proxmox support, control plane, and XDG directories'
author: 'Collin @ Fluid.sh'
authorImage: '/images/skeleton_smoking_cigarette.jpg'
authorEmail: 'cpfeifer@madcactus.org'
authorPhone: '+3179955114'
authorDiscord: 'https://discordapp.com/users/301068417685913600'
---

## Changelog #003

Hi everyone!

This release brings a major rewrite to what Fluid can do and where it can operate. Fluid MCP gets added, proxmox support is added, and remote execution via the Fluid daemon was added!

## Update to v0.1.3

Update your CLI agent with `go install`

```bash
go install github.com/aspectrr/fluid.sh/fluid/cmd/fluid@latest
```

## Fluid Daemon

By far the biggest change comes in the form of a Fluid architecture rebuild.

Previously, the CLI talked directly to libvirt via local calls. That meant the CLI had to run on the same machine as your VMs, or you had to forward a libvirt socket over SSH. It worked, but it didn't scale and it made remote access painful.

Now, the CLI connects to `fluid-daemon` over gRPC on port 9091. The daemon runs on your sandbox host and handles the entire sandbox lifecycle: creating, starting, stopping, destroying, snapshotting, and running commands. State is persisted to a local SQLite database, so sandboxes survive daemon restarts.

A janitor runs in the background to enforce TTLs. Every sandbox gets a default 24-hour TTL, and the janitor checks every minute and cleans up anything expired. No more orphaned VMs eating disk space.

The daemon also introduces a provider abstraction. The default backend is QEMU microVMs, but there's also an LXC provider for Proxmox environments. Configure the provider in your daemon config and the rest of the system doesn't need to change.

One daemon runs per sandbox host, but each daemon can reach multiple source VM hosts over SSH for image pulling and read-only access. For environments with multiple data centers or heavily NATed networks, you'd run a daemon on each sandbox host.

Optionally, the daemon can connect upstream to the control plane (more on that below) for multi-host orchestration.

Here's what a minimal daemon config looks like at `~/.config/fluid/daemon.yaml`:

```yaml
provider: microvm

daemon:
  listen_addr: ':9091'
  enabled: true

microvm:
  work_dir: /var/lib/fluid-daemon/overlays
  default_vcpus: 2
  default_memory_mb: 2048
  command_timeout: 5m

network:
  default_bridge: virbr0
  dhcp_mode: arp

image:
  base_dir: /var/lib/fluid-daemon/images

ssh:
  default_user: sandbox
  cert_ttl: 30m

state:
  db_path: ~/.fluid/sandbox-host.db

janitor:
  interval: 1m
  default_ttl: 24h
```

## MCP Server

Fluid now ships an MCP (Model Context Protocol) server. Run `fluid mcp` and it starts on stdio, ready to be wired into any MCP-compatible AI agent: Claude Code, Cursor, Windsurf, etc.

The MCP server exposes 17 tools that give AI agents full sandbox management capabilities:

| Tool                                    | Description                                       |
| --------------------------------------- | ------------------------------------------------- |
| `list_sandboxes`                        | List all sandboxes with state and IPs             |
| `create_sandbox`                        | Clone a source VM into a new sandbox              |
| `destroy_sandbox`                       | Destroy a sandbox and remove its storage          |
| `run_command`                           | Execute a shell command inside a sandbox via SSH  |
| `start_sandbox` / `stop_sandbox`        | Start or stop a sandbox                           |
| `get_sandbox`                           | Get detailed info about a specific sandbox        |
| `list_vms`                              | List available source VMs for cloning             |
| `create_snapshot`                       | Snapshot the current sandbox state                |
| `edit_file`                             | Edit or create a file inside a sandbox            |
| `read_file`                             | Read a file from a sandbox                        |
| `create_playbook` / `add_playbook_task` | Create and build Ansible playbooks                |
| `list_playbooks` / `get_playbook`       | List and inspect playbooks                        |
| `run_source_command`                    | Run read-only diagnostic commands on source hosts |
| `read_source_file`                      | Read files from source hosts                      |
| `list_hosts`                            | List configured source hosts and their status     |

To wire it up with Claude Code, add this to your MCP config:

```json
{
  "mcpServers": {
    "fluid": {
      "command": "fluid",
      "args": ["mcp"]
    }
  }
}
```

Security is built in at the tool level: path traversal protection on file operations, shell injection prevention on commands, base64 encoding for binary file content, and structured error responses so the agent can recover gracefully.

## Control Plane

A new `api/` server provides centralized orchestration for multi-host deployments. It runs a REST API on `:8080` and a gRPC server on `:9090` for daemon connections.

Multiple daemons can connect to a single control plane, and the control plane dispatches sandbox creation across connected hosts. PostgreSQL backs the shared state. A web dashboard (the `web/` frontend) connects to the REST API for monitoring sandboxes, viewing commands, and approving playbooks.

Daemons authenticate to the control plane using host tokens. The connection is a bidirectional gRPC stream with automatic reconnection, so daemons can sit behind NAT and still be reachable.

## XDG Base Directories

Config and data paths now follow the XDG Base Directory spec:

- **Config**: `~/.config/fluid/` (was `~/.fluid/`)
- **Data**: `~/.local/share/fluid/` (state.db, history)

The `$XDG_CONFIG_HOME` and `$XDG_DATA_HOME` environment variables are respected if set. Existing `~/.fluid/` configs are auto-migrated on first run.

## Container Images

The API server and web dashboard now ship as container images, built automatically via GitHub Actions on every release and pushed to GHCR. Run the full stack with `docker-compose up` or pull individual images from `ghcr.io/aspectrr/fluid.sh/api` and `ghcr.io/aspectrr/fluid.sh/web`.

## Come Hang Out

Questions? Join us on [Discord](https://discord.gg/4WGGXJWm8J)

Found a bug? Open an issue on [GitHub](https://github.com/aspectrr/fluid.sh/issues)
