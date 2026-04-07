<div align="center">

# 🌊 deer.sh

### The AI Sys-Admin for Enterprise

[![Commit Activity](https://img.shields.io/github/commit-activity/m/aspectrr/deer.sh?color=blue)](https://github.com/aspectrr/deer.sh/commits/main)
[![License](https://img.shields.io/github/license/aspectrr/deer.sh?color=blue)](https://github.com/aspectrr/deer.sh/blob/main/LICENSE)
[![Discord](https://img.shields.io/discord/1465124928650215710?label=discord)](https://discord.gg/4WGGXJWm8J)
[![GitHub stars](https://img.shields.io/github/stars/aspectrr/deer.sh)](https://github.com/aspectrr/deer.sh)

Fluid is an AI agent built for the core steps of debugging and managing Linux servers. Read-Only mode for getting context, Create a sandbox and make edits to test changes. Create an Ansible Playbook to recreate on prod.

[Features](#features) | [Quick Start](#quick-start) | [Demo](#demo) | [Docs](https://deer.sh/docs/quickstart)

</div>

---

## Problem

AI agents can install packages, configure services, write scripts - autonomously. But one mistake on production and you're getting paged at 3 AM. So we limit agents to chatbots instead of letting them do real work.

## Solution

**deer.sh** gives agents direct read-only SSH access to your servers for context gathering, then full root access in isolated VM sandboxes for testing changes. When done, a human reviews the diff and approves an auto-generated Ansible playbook before anything touches production.

```
                    Read-Only (direct SSH)
Agent Task  -->  Source Host (inspect)  -->  Sandbox VM (autonomous)  -->  Human Approval  -->  Production
                  - View logs                  - Full root access            - Review diff
                  - Check configs              - Install packages            - Approve Ansible
                  - Query services             - Edit configs                - One-click apply
                  - Read files                 - Run services
```

## Demo

[![CLI Agent Demo](https://img.youtube.com/vi/ZSUBGXNTz34/0.jpg)](https://www.youtube.com/watch?v=ZSUBGXNTz34)

## Features

| Feature | Description |
|---------|-------------|
| **Autonomous Execution** | Agents run commands, install packages, edit configs - no hand-holding |
| **Full VM Isolation** | Each agent gets a dedicated microVM with root access |
| **Interactive TUI** | Natural language interface - just type what you want done |
| **Human-in-the-Loop** | Blocking approval workflow before any production changes |
| **Ansible Export** | Auto-generate playbooks from agent work for production apply |
| **MCP Integration** | Use deer tools from Claude Code, Cursor, Windsurf |
| **Read-Only Mode** | Inspect source VMs safely without risk of modification |
| **Multi-Host** | Scale across hosts with the daemon + control plane |

## Read-Only Mode

The CLI connects directly to your source hosts over SSH - no daemon required. A dedicated `deer-readonly` user with a restricted shell ensures agents can only run read-only commands.

**What agents can do:**
- Read files, logs, and configs (`cat`, `journalctl`, `tail`, etc.)
- Inspect processes and services (`ps`, `systemctl status`, `top`)
- Query system state (`df`, `free`, `ip`, `ss`, `uname`)
- Run diagnostic commands (`dig`, `ping`, `lsblk`)

**What agents cannot do:**
- Write, modify, or delete files
- Install or remove packages
- Start, stop, or restart services
- Execute arbitrary scripts or interpreters

Commands are validated twice: client-side against an allowlist in the CLI, and server-side by the restricted shell on the host. You can extend the default allowlist with `extra_allowed_commands` in your config.

### Preparing a Host

Before deer can read from a host, you need to prepare it. This creates the `deer-readonly` user with a restricted shell and deploys an SSH key.

**Prerequisites:** The host must be accessible via SSH using your existing `~/.ssh/config` (any ProxyJump, port, or user settings are respected).

```bash
deer source prepare <hostname>
```

This runs 4 steps on the remote host:

1. Installs a restricted shell script at `/usr/local/bin/deer-readonly-shell`
2. Creates a `deer-readonly` system user with that shell
3. Deploys deer's SSH public key to the user's `authorized_keys`
4. Restarts sshd

After prepare, the host appears in `/hosts` as prepared. The CLI generates an ed25519 key pair at `~/.config/deer/keys/` on first run and reuses it for all hosts.

```bash
# List prepared hosts
deer source list
```

## Sensitive Data Redaction

All tool output is scanned for sensitive data before it reaches the AI agent. This prevents accidental exposure of credentials through commands like `cat /etc/ssl/private/server.key` or `kubectl get secret -o yaml`.

**What gets redacted:**

| Type | Examples |
|------|---------|
| PEM private keys | RSA, EC, ED25519, OPENSSH private key blocks |
| Base64-encoded keys | Output of `cat key.pem \| base64`, Kubernetes secret values |
| Kubernetes secrets | `tls.key`, `ssh-privatekey`, `private_key`, `secret_key` fields |
| API keys & tokens | `sk-...`, `key-...`, Bearer tokens, AWS access keys (`AKIA...`) |
| Connection strings | `postgres://`, `mysql://`, `mongodb://`, `redis://` URIs |
| IP addresses | IPv4 and IPv6 addresses |

Redaction runs at two layers: inline when each tool returns results, and again before the full conversation is sent to the LLM. The agent sees `[REDACTED: ...]` placeholders instead of the actual values.

## Quick Start

### Install

```bash
curl -fsSL https://deer.sh/install.sh | bash
```

Or with Go:

```bash
go install github.com/aspectrr/deer.sh/deer-cli/cmd/deer@latest
```

### Launch the TUI

```bash
deer
```

On first run, onboarding walks you through host setup, and LLM API key configuration.

### Architecture

```
                          Direct SSH (read-only)
deer (TUI/MCP)  -------------------------------->  Source Hosts
       |                                              - deer-readonly user
       |                                              - restricted shell
       |                                              - command allowlist
       |
       +--- gRPC :9091 --->  deer-daemon  --->  QEMU microVMs (sandboxes)
                                   |
                                   +--- control-plane (optional, multi-host)
                                   |
                                   +--- web dashboard
```

- **deer-cli**: Interactive TUI agent + MCP server. Connects directly to source hosts via SSH for read-only inspection, and to the daemon via gRPC for sandbox operations.
- **deer-daemon**: Background service managing microVM sandboxes
- **control-plane (api)**: Multi-host orchestration, REST API, web dashboard
- **web**: React dashboard for monitoring and approval

### MCP Integration

Connect Claude Code, Codex, or Cursor to deer via MCP:

```json
{
  "mcpServers": {
    "deer": {
      "command": "deer",
      "args": ["mcp"]
    }
  }
}
```

17 tools available: `create_sandbox`, `destroy_sandbox`, `run_command`, `edit_file`, `read_file`, `create_playbook`, and more. See the [full reference](https://deer.sh/docs/cli-reference).

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
- **QEMU/KVM** - See [local setup docs](https://deer.sh/docs/local-setup)

### 30-Second Start

```bash
git clone https://github.com/aspectrr/deer.sh.git
cd deer.sh
mprocs
```

Services:
- Web UI: http://localhost:5173
- API: http://localhost:8080

### Project Structure

```
deer-cli/        # Go - Interactive TUI agent + MCP server
deer-daemon/     # Go - Background microVM sandbox management daemon
api/              # Go - Control plane REST API + gRPC
sdk/              # Python - SDK for the API
web/              # React - Dashboard UI
proto/            # Protobuf definitions
```

### Running Tests

```bash
cd deer-cli && make test
cd deer-daemon && make test
cd api && make test
cd web && bun run build
```

### Downloading MicroVM Guest Assets

The live Redpanda guest integration test in `deer-daemon/internal/provider/microvm/redpanda_integration_test.go` needs three guest assets on the host:

- a base Ubuntu cloud image
- a matching kernel
- a matching initrd

The daemon expects a QCOW2 backing image. Ubuntu publishes the current Noble cloud image as `ubuntu-24.04-server-cloudimg-amd64.img`, which is suitable for QEMU microVM use even though the filename ends in `.img`. A simple symlink to a `.qcow2` name keeps the local workflow consistent with deer's image store.

As of March 26, 2026, the current official Ubuntu 24.04 release pages list:

- Base image: [Ubuntu Noble release image listing](https://cloud-images.ubuntu.com/releases/noble/release/)
- Kernel/initrd: [Ubuntu Noble unpacked kernel/initrd listing](https://cloud-images.ubuntu.com/releases/noble/release/unpacked/)

Use the helper script from the repository root:

```bash
./scripts/download-microvm-assets.sh
```

That script downloads the image, kernel, initrd, and `SHA256SUMS`, verifies the checksums, creates the `.qcow2` symlink, and prints the `DEER_E2E_*` environment variables you can use for the live guest test. It defaults to Ubuntu Noble on `amd64` and supports `--arch arm64` for Apple Silicon and other ARM64 hosts.

To inspect exactly what it would do without downloading anything:

```bash
./scripts/download-microvm-assets.sh --dry-run
```

To place the assets somewhere else:

```bash
./scripts/download-microvm-assets.sh --output-dir /absolute/path/to/microvm-assets
```

Run the live guest integration test with:

```bash
cd deer-daemon

sudo env \
  DEER_E2E_MICROVM=1 \
  DEER_E2E_BASE_IMAGE="$PWD/../.cache/deer/e2e/noble-amd64/ubuntu-24.04-server-cloudimg-amd64.qcow2" \
  DEER_E2E_KERNEL="$PWD/../.cache/deer/e2e/noble-amd64/ubuntu-24.04-server-cloudimg-amd64-vmlinuz-generic" \
  DEER_E2E_INITRD="$PWD/../.cache/deer/e2e/noble-amd64/ubuntu-24.04-server-cloudimg-amd64-initrd-generic" \
  DEER_E2E_BRIDGE=br0 \
  DEER_E2E_ACCEL=tcg \
  GOCACHE=/tmp/deer-daemon-go-build \
  go test -v ./internal/provider/microvm -run TestProviderIntegration_RedpandaStartsInGuest
```

Notes:

- `DEER_E2E_BRIDGE` must be a working host bridge with outbound network access so the guest can install Redpanda packages during cloud-init.
- `DEER_E2E_ACCEL=kvm` is faster if `/dev/kvm` is available; `tcg` is slower but works on more hosts.
- These guest assets are large and should stay out of git.

### Running The Live Guest Test With Lima On macOS

On macOS, the most reliable way to run `TestProviderIntegration_RedpandaStartsInGuest` is inside a Linux Lima VM. The test uses Linux TAP + bridge networking, and macOS host TAP creation is not available on every machine.

The shortest host-side path is:

```bash
brew install lima
bash ./scripts/run-redpanda-e2e-lima-host.sh --repo-root "$PWD"
```

From `deer-daemon/`, there is also a make target:

```bash
cd deer-daemon
make redpanda-e2e-lima
```

To inspect the exact host-side and guest-side commands without running them:

```bash
bash ./scripts/run-redpanda-e2e-lima-host.sh --repo-root "$PWD" --dry-run
```

If you want to do it manually, the wrapper performs these steps.

Create and enter a Lima VM:

```bash
brew install lima
limactl start --name deer-e2e template://ubuntu
limactl shell deer-e2e
```

Inside the Lima guest, install the runtime dependencies and enable libvirt:

```bash
sudo apt-get update
sudo apt-get install -y qemu-system qemu-utils libvirt-daemon-system libvirt-clients iproute2 openssh-client golang-go
sudo systemctl enable --now libvirtd
sudo virsh net-autostart default
sudo virsh net-start default || true
ip -4 addr show virbr0
```

Lima usually mounts your macOS home directory into the Linux guest, so the repository is often available at the same absolute path. From inside the Lima guest:

```bash
REPO_ROOT="$HOME/GitHub/deer.sh"
cd "$REPO_ROOT"

if [ "$(uname -m)" = "aarch64" ]; then
  ./scripts/download-microvm-assets.sh --arch arm64
else
  ./scripts/download-microvm-assets.sh --arch amd64
fi
```

Then run the live guest test from inside the Lima guest using libvirt's default network and lease-file IP discovery:

```bash
REPO_ROOT="$HOME/GitHub/deer.sh"
"$REPO_ROOT/scripts/run-redpanda-e2e-lima.sh" --repo-root "$REPO_ROOT"
```

Notes:

- `DEER_E2E_BRIDGE=virbr0` matches libvirt's default network inside the Lima guest.
- `DEER_E2E_DHCP_MODE=libvirt` makes the test read libvirt lease files instead of depending on ARP discovery.
- `DEER_E2E_ACCEL=tcg` is the safe default inside Lima because nested KVM is usually unavailable.
- `DEER_E2E_ROOT_DEVICE=/dev/vda1` is used by the helper because the Ubuntu cloud images downloaded here boot from the first virtio block partition, not the whole disk.
- `golang-go` only bootstraps the toolchain. The test run exports `GOTOOLCHAIN=auto`, so Go downloads the exact `go1.24.4` toolchain declared by the module when needed.
- If your repository is not mounted at `$HOME/GitHub/deer.sh` inside Lima, clone it inside the guest or adjust `REPO_ROOT`.
- To inspect the exact `sudo env ... go test` command without running it from inside the guest, use:

```bash
REPO_ROOT="$HOME/GitHub/deer.sh"
"$REPO_ROOT/scripts/run-redpanda-e2e-lima.sh" --repo-root "$REPO_ROOT" --dry-run
```

## Enterprise

For teams with security and compliance requirements, deer.sh supports:

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

[![Star History Chart](https://api.star-history.com/svg?repos=aspectrr/deer.sh&type=date&legend=top-left)](https://www.star-history.com/#aspectrr/deer.sh&type=date&legend=top-left)

<div align="center">

Made with ❤️ by Collin, Claude & [Contributors](https://github.com/aspectrr/deer.sh/graphs/contributors)

</div>
