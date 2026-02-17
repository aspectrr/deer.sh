<div align="center">

# 🌊 fluid.sh

### Claude Code for Debugging VMs

[![Commit Activity](https://img.shields.io/github/commit-activity/m/aspectrr/fluid.sh?color=blue)](https://github.com/aspectrr/fluid.sh/commits/main)
[![License](https://img.shields.io/github/license/aspectrr/fluid.sh?color=blue)](https://github.com/aspectrr/fluid.sh/blob/main/LICENSE)
[![Discord](https://img.shields.io/discord/1465124928650215710?label=discord)](https://discord.gg/4WGGXJWm8J)
[![GitHub stars](https://img.shields.io/github/stars/aspectrr/fluid.sh)](https://github.com/aspectrr/fluid.sh)

Fluid is an AI agent built for the core steps of debugging infrastructure. Read-Only mode for getting context, Create a sandbox and make edits to test changes. Create an Ansible Playbook to recreate on prod.

[Features](#features) | [Quick Start](#quick-start) | [Demo](#demo) | [Docs](https://fluid.sh/docs/quickstart)

</div>

---

## Problem

AI agents can install packages, configure services, write scripts - autonomously. But one mistake on production and you're getting paged at 3 AM. So we limit agents to chatbots instead of letting them do real work.

## Solution

**fluid.sh** gives agents full root access in isolated VM sandboxes. They work autonomously. When done, a human reviews the diff and approves an auto-generated Ansible playbook before anything touches production.

```
Agent Task  -->  Sandbox VM (autonomous)  -->  Human Approval  -->  Production
                  - Full root access            - Review diff
                  - Install packages            - Approve Ansible
                  - Edit configs                - One-click apply
                  - Run services
```

## Demo

[![CLI Agent Demo](https://img.youtube.com/vi/ZSUBGXNTz34/0.jpg)](https://www.youtube.com/watch?v=ZSUBGXNTz34)

## Features

| Feature | Description |
|---------|-------------|
| **Autonomous Execution** | Agents run commands, install packages, edit configs - no hand-holding |
| **Full VM Isolation** | Each agent gets a dedicated KVM virtual machine with root access |
| **Interactive TUI** | Natural language interface - just type what you want done |
| **Human-in-the-Loop** | Blocking approval workflow before any production changes |
| **Ansible Export** | Auto-generate playbooks from agent work for production apply |
| **MCP Integration** | Use fluid tools from Claude Code, Cursor, Windsurf |
| **Read-Only Mode** | Inspect source VMs safely without risk of modification |
| **Multi-Host** | Scale across hosts with the daemon + control plane |

## Quick Start

### Install

```bash
curl -fsSL https://fluid.sh/install.sh | bash
```

Or with Go:

```bash
go install github.com/aspectrr/fluid.sh/fluid-cli/cmd/fluid-cli@latest
```

### Launch the TUI

```bash
fluid
```

On first run, onboarding walks you through host setup, SSH CA generation, and LLM API key configuration.

### Architecture

```
fluid (TUI/MCP)  --->  fluid-daemon (gRPC :9091)  --->  libvirt/KVM
                            |
                            +--- control-plane (optional, multi-host)
                            |
                            +--- web dashboard
```

- **fluid-cli**: Interactive TUI agent + MCP server
- **fluid-daemon**: Background service managing sandboxes via libvirt
- **control-plane (api)**: Multi-host orchestration, REST API, web dashboard
- **web**: React dashboard for monitoring and approval

### MCP Integration

Connect Claude Code, Codex, or Cursor to fluid via MCP:

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

17 tools available: `create_sandbox`, `destroy_sandbox`, `run_command`, `edit_file`, `read_file`, `create_playbook`, and more. See the [full reference](https://fluid.sh/docs/cli-reference).

### TUI Slash Commands

| Command | Description |
|---------|-------------|
| `/vms` | List available VMs |
| `/sandboxes` | List active sandboxes |
| `/hosts` | List configured hosts |
| `/playbooks` | List Ansible playbooks |
| `/settings` | Open configuration |
| `/compact` | Compact conversation |
| `/context` | Show token usage |
| `/clear` | Clear history |
| `/help` | Show help |

Toggle between edit and read-only mode with `Shift+Tab`.

Copy text by dragging and holding `Shift`.

## Development

### Prerequisites

- **mprocs** - Multi-process runner for local dev
- **Go 1.24+**
- **libvirt/KVM** - See [local setup docs](https://fluid.sh/docs/local-setup)

### 30-Second Start

```bash
git clone https://github.com/aspectrr/fluid.sh.git
cd fluid.sh
mprocs
```

Services:
- Web UI: http://localhost:5173
- API: http://localhost:8080

### Project Structure

```
fluid-cli/        # Go - Interactive TUI agent + MCP server
fluid-daemon/     # Go - Background sandbox management daemon
api/              # Go - Control plane REST API + gRPC
web/              # React - Dashboard UI
demo-server/      # Go - WebSocket demo server
proto/            # Protobuf definitions
```

### Running Tests

```bash
cd fluid-cli && make test
cd fluid-daemon && make test
cd api && make test
cd web && bun run build
```

## Enterprise

For teams with security and compliance requirements, fluid.sh supports:

- **Encrypted snapshots at rest** - Source images encrypted on sandbox hosts with configurable TTL and secure wipe on eviction
- **Network isolation** - Sandboxes boot into isolated networks with no route to production by default, explicit allowlists for service access
- **RBAC** - Control which users and teams can create sandboxes from which source VMs
- **Audit logging** - Full trail of every snapshot pull, sandbox creation, and destruction
- **Secrets scrubbing** - Configurable per source VM: scrub credentials before sandbox creation or keep exact replica for auth debugging
- **Scoped daemon credentials** - Read-only snapshot capability on production hosts, nothing else

If you need these, reach out to [Collin](mailto:cpfeifer@madcactus.org) to learn more about an enterprise plan.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make changes with tests
4. Submit a pull request

All contributions must maintain the security model and include appropriate tests.

Reach out on [Discord](https://discord.gg/4WGGXJWm8J) with questions or for access to test VMs.

## License

MIT License - see [LICENSE](LICENSE) for details.

## Star History

[![Star History Chart](https://api.star-history.com/svg?repos=aspectrr/fluid.sh&type=date&legend=top-left)](https://www.star-history.com/#aspectrr/fluid.sh&type=date&legend=top-left)

<div align="center">

Made with ❤️ by Collin, Claude & [Contributors](https://github.com/aspectrr/fluid.sh/graphs/contributors)

</div>
