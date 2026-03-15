# Fluid CLI - Development Guide

The interactive TUI agent and MCP server for fluid.sh. Connects directly to source hosts via SSH for read-only operations, and to fluid-daemon over gRPC to manage microVM sandboxes.

## Architecture

```
User
  |
  v
fluid CLI (TUI / MCP)
  |
  +--- Direct SSH -------> Source Hosts (read-only)
  |
  +--- gRPC :9091 -------> fluid-daemon
                              |
                              v
                            QEMU microVMs
```

## Quick Start

```bash
# Build the CLI
make build

# Launch the TUI
./bin/fluid

# Start MCP server on stdio
./bin/fluid mcp
```

## TUI Slash Commands

| Command | Description |
|---------|-------------|
| `/vms` | List available VMs for cloning |
| `/sandboxes` | List active sandboxes |
| `/hosts` | List configured remote hosts |
| `/playbooks` | List generated Ansible playbooks |
| `/connect` | Connect to a fluid daemon |
| `/prepare` | Prepare a source VM for sandbox cloning |
| `/compact` | Summarize and compact conversation history |
| `/context` | Show current context token usage |
| `/settings` | Open configuration settings |
| `/clear` | Clear conversation history |
| `/help` | Show available commands |

## TUI Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `Enter` | Send message |
| `Shift+Tab` | Toggle edit / read-only mode |
| `PgUp/PgDn` | Scroll conversation history |
| `Ctrl+R` | Reset conversation |
| `Ctrl+C` | Quit |

## MCP Tools

17 tools exposed via `fluid mcp`:

| Tool | Parameters | Description |
|------|-----------|-------------|
| `list_sandboxes` | (none) | List all sandboxes with state and IPs |
| `create_sandbox` | `source_vm` (required), `cpu`, `memory_mb` | Create a sandbox by cloning a source VM |
| `destroy_sandbox` | `sandbox_id` (required) | Destroy a sandbox and remove storage |
| `run_command` | `sandbox_id` (required), `command` (required), `timeout_seconds` | Execute a shell command via SSH |
| `start_sandbox` | `sandbox_id` (required) | Start a stopped sandbox |
| `stop_sandbox` | `sandbox_id` (required) | Stop a running sandbox |
| `get_sandbox` | `sandbox_id` (required) | Get detailed sandbox info |
| `list_vms` | (none) | List available VMs for cloning |
| `create_snapshot` | `sandbox_id` (required), `name` | Snapshot current sandbox state |
| `create_playbook` | `name` (required), `hosts`, `become` | Create an Ansible playbook |
| `add_playbook_task` | `playbook_id` (required), `name` (required), `module` (required), `params` | Add a task to a playbook |
| `edit_file` | `sandbox_id` (required), `path` (required), `new_str` (required), `old_str`, `replace_all` | Edit or create a file in a sandbox |
| `read_file` | `sandbox_id` (required), `path` (required) | Read a file from a sandbox |
| `list_playbooks` | (none) | List all created playbooks |
| `get_playbook` | `playbook_id` (required) | Get playbook definition and YAML |
| `run_source_command` | `source_vm` (required), `command` (required), `timeout_seconds` | Run read-only command on a source VM |
| `read_source_file` | `source_vm` (required), `path` (required) | Read a file from a source VM |

## Configuration

Default config location: `~/.fluid/config.yaml`

```yaml
libvirt:
  uri: qemu:///system
  network: default
  base_image_dir: /var/lib/libvirt/images/base
  work_dir: /var/lib/libvirt/images/sandboxes
  ssh_key_inject_method: virt-customize

vm:
  default_vcpus: 2
  default_memory_mb: 2048
  command_timeout: 5m
  ip_discovery_timeout: 2m

ssh:
  proxy_jump: ""
  default_user: sandbox
```

## Development

### Prerequisites

- Go 1.24+
- QEMU/KVM installed and running

### Build

```bash
make build          # Build bin/fluid
make build-dev      # Build with telemetry key
make clean          # Clean build artifacts
```

### Testing

```bash
make test           # Run all tests
make test-coverage  # Tests with coverage report
```

### Code Quality

```bash
make fmt            # Format with gofumpt
make vet            # Run go vet
make lint           # Run golangci-lint
make check          # Run all checks (fmt, vet, lint)
```

### Dependencies

```bash
make deps           # Download dependencies
make tidy           # Tidy and verify
make install-tools  # Install gofumpt, golangci-lint, swag
```

### CLI Subcommands

| Command | Description |
|---------|-------------|
| `fluid` | Launch the interactive TUI agent (default) |
| `fluid connect <address>` | Connect to a fluid daemon and save config |
| `fluid mcp` | Start MCP server on stdio |
| `fluid doctor` | Check daemon setup on a host |
| `fluid source prepare <host>` | Prepare a host for read-only access |
| `fluid source list` | List configured source hosts |
| `fluid update` | Self-update to the latest release |

## Makefile Targets

| Target | Description |
|--------|-------------|
| `all` | Run fmt, vet, test, and build (default) |
| `build` | Build the fluid CLI binary |
| `build-dev` | Build with PostHog telemetry key |
| `run` | Build and run the CLI |
| `clean` | Clean build artifacts |
| `fmt` | Format code with gofumpt |
| `lint` | Run golangci-lint |
| `vet` | Run go vet |
| `test` | Run tests |
| `test-coverage` | Run tests with coverage |
| `check` | Run all code quality checks |
| `deps` | Download dependencies |
| `tidy` | Tidy and verify dependencies |
| `install` | Install fluid to GOPATH/bin |
| `install-tools` | Install dev tools |

## Data Storage

State is stored in SQLite at `~/.fluid/state.db`:
- Sandboxes, Snapshots, Commands, Diffs

The database is auto-migrated on first run.

If you remove a parameter from a function, don't just pass in nil/null/empty string in a different layer, make sure to remove the extra parameter from every place.

## Security: Sensitive Data Redaction

All tool output (command results, file contents) is scanned for sensitive data before being sent to the LLM. Redaction is applied in two layers:

### Inline redaction (agent.go)
Applied at tool execution time via `redactContent()`. Uses the same `Redactor` from `internal/redact/` as the pre-LLM layer, ensuring TUI live output shows the same redacted tokens the LLM receives.

Runs on every `readFile`, `readSourceFile`, `runCommand`, and `runSourceCommand` result.

### Pattern-based redaction (redact package)
Applied to all messages before LLM API calls. Uses the `Redactor` from `internal/redact/`.

**Built-in detectors:**
- SSH/PEM private key blocks
- Base64-encoded PEM private keys
- Kubernetes secret data fields (`tls.key`, `ssh-privatekey`, `private_key`, `secret_key`, `ca.key`, `server.key`, `client.key`)
- IPv4 and IPv6 addresses
- API keys (`sk-...`, `key-...`, Bearer tokens)
- AWS access keys (`AKIA...`)
- Connection strings (postgres://, mysql://, etc.)
- Configured host names, addresses, and key paths

Note: `tls.crt` is intentionally not redacted - it is a public certificate, not secret material.

### Coverage

| Tool | Inline | Pre-LLM | Notes |
|------|--------|---------|-------|
| `run_command` | stdout, stderr | all fields | |
| `run_source_command` | stdout, stderr | all fields | |
| `read_file` | file content | all fields | |
| `read_source_file` | file content | all fields | |
| Other tools | - | all fields | Pre-LLM layer covers all tool results |

### Limitations
- Base64-PEM detection requires the PEM block to be at the start of the encoded content (standard for key files)
- Kubernetes secret detection uses a fixed list of known field names
- Hex-encoded keys and non-standard key formats are not detected
- Keys embedded inside nested encodings (e.g., base64 of JSON of base64) are not detected
