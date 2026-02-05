<div align="center">

# fluid.sh

### Autonomous AI Agents for Infrastructure

**Make Infrastructure Safe for AI**

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://go.dev)
[![Python](https://img.shields.io/badge/Python-3.11+-3776AB?logo=python&logoColor=white)](https://python.org)
[![React](https://img.shields.io/badge/React-18+-61DAFB?logo=react&logoColor=black)](https://react.dev)

[Features](#features) * [Quick Start](#quick-start) * [Demo](#demo) * [Documentation](#documentation)

</div>

---

## Demo

<a href="https://www.youtube.com/watch?v=nAlqRMhZxP0">
  <img src="https://img.youtube.com/vi/nAlqRMhZxP0/maxresdefault.jpg" alt="Fluid.sh Demo" width="600">
</a>

## Problem

AI agents are ready to do infrastructure work, but they can't touch prod:

- Agents can install packages, configure services, write scripts--autonomously
- But one mistake on production and you're getting paged at 3 AM to fix it
- So we limit agents to chatbots instead of letting them *do the work*

## Solution

**fluid.sh** lets AI agents work autonomously in isolated VMs, then a human approves before anything touches production:

```
+-----------------------------------------------------------------------+
|                    Autonomous AI Sysadmin Workflow                      |
|                                                                         |
|  +---------+     +-----------------+     +----------+     +----------+  |
|  |  Agent  |---->|  Sandbox VM     |---->|  Human   |---->|Production|  |
|  |  Task   |     |  (autonomous)   |     | Approval |     |  Server  |  |
|  +---------+     +-----------------+     +----------+     +----------+  |
|                         |                      |                        |
|                    - Full root access     - Review diff                 |
|                    - Install packages     - Approve Ansible             |
|                    - Edit configs         - One-click apply             |
|                    - Run services                                       |
|                    - Snapshot/restore                                   |
+-----------------------------------------------------------------------+
```

## Features

| Feature | Description |
|---------|-------------|
|  **Autonomous Execution** | Agents run commands, install packages, edit configs--no hand-holding |
|  **Full VM Isolation** | Each agent gets a dedicated KVM virtual machine with root access |
|  **Snapshot & Restore** | Checkpoint progress, rollback mistakes, branch experiments |
|  **Human-in-the-Loop** | Blocking approval workflow before any production changes |
|  **Diff & Audit Trail** | See exactly what changed, every action logged |
|  **Ansible Export** | Auto-generate playbooks from agent work for production apply |
|  **Tmux Integration** | Watch agent work in real-time, intervene if needed |
|  **Python SDK** | First-class SDK for building autonomous agents |

## SDK Example

```python
from virsh_sandbox import VirshSandbox

client = VirshSandbox("http://localhost:8080")
sandbox = None

try:
    # Agent gets its own VM with full root access
    sandbox = client.sandbox.create_sandbox(
        source_vm_name="ubuntu-base",
        agent_id="nginx-setup-agent",
        auto_start=True,
        wait_for_ip=True
    ).sandbox

    run_agent("Install nginx and configure TLS, create an Ansible playbook to recreate the task.", sandbox.id)

    # NOW the human reviews:
    # - Diff between snapshots shows exactly what changed
    # - Auto-generated Ansible playbook ready to apply
    # - Human approves -> playbook runs on production
    # - Human rejects -> nothing happens, agent tries again

finally:
    if(sandbox):
        # Clean up sandbox
        client.sandbox.destroy_sandbox(sandbox.id)
```

## Quick Start

### Prerequisites

`fluid` is setup to be ran on a control plane on the same network as the VM hosts it needs to connect with. It will also need a postgres instance running on the control plane to keep track of commands run, sandboxes, and other auditing.

If you need another way of accessing VMs, open an issue and we will get back to you.

### Installation

The recommended deployment model is a **single control node** running the `fluid` API and PostgreSQL, with SSH access to one or more libvirt/KVM hosts.

---

## Architecture Overview

```
+--------------------+        SSH        +------------------+
| Control Node       |----------------->| KVM / libvirt    |
|                    |                  | Hosts            |
| - fluid            |                  |                  |
| - PostgreSQL       |                  | - libvirtd       |
+--------------------+                  +------------------+
```

The control node:

* Runs the `fluid` API
* Stores audit logs and metadata in PostgreSQL
* Connects to hosts over SSH to execute libvirt operations

The hypervisor hosts:

* Run KVM + libvirt only
* Do not run agents or additional services

---

## Requirements

### Control Node

* Linux (x86_64)
* systemd
* PostgreSQL 14+
* SSH client

### Hypervisor Hosts

* Linux
* KVM enabled
* libvirt installed and running
* SSH access from control node

### Network

* Private management network between control node and hosts
* Public or tenant-facing network configured on hosts for VMs

---

## Production Installation (Recommended)

This method installs a **static binary** and runs it as a systemd service. No container runtime is required.

### 1. Import the GPG public key
```bash
# Import from keyserver
gpg --keyserver keys.openpgp.org --recv-keys B27DED65CFB30427EE85F8209DD0911D6CB0B643

# OR import from file
curl https://raw.githubusercontent.com/aspectrr/fluid.sh/main/public-key.asc | gpg --import
```

### 2. Download release assets
```bash
VERSION=0.0.4-beta
wget https://github.com/aspectrr/fluid.sh/releases/download/v${VERSION}/fluid_${VERSION}_linux_amd64.tar.gz
wget https://github.com/aspectrr/fluid.sh/releases/download/v${VERSION}/checksums.txt
wget https://github.com/aspectrr/fluid.sh/releases/download/v${VERSION}/checksums.txt.sig
```

### 3. Verify signature and checksum
```bash
# Verify GPG signature
gpg --verify checksums.txt.sig checksums.txt

# Verify file checksum
sha256sum -c checksums.txt --ignore-missing
```

### 4. Extract and install
```bash
tar -xzf fluid_${VERSION}_linux_amd64.tar.gz
sudo install -m 755 fluid /usr/local/bin/
```

## System User and Directories

Create a dedicated system user and required directories:

```bash
useradd --system --home /var/lib/fluid --shell /usr/sbin/nologin fluid

mkdir -p /etc/fluid \
         /var/lib/fluid \
         /var/log/fluid

chown -R fluid:fluid \
  /var/lib/fluid \
  /var/log/fluid
```

Filesystem layout:

```
/usr/local/bin/fluid
/etc/fluid/config.yaml
/var/lib/fluid/
/var/log/fluid/
```

---

## PostgreSQL Setup

PostgreSQL runs **locally on the control node** and is bound to localhost only.

### Create Database and User

```bash
sudo -u postgres psql
```

```sql
CREATE DATABASE fluid;
CREATE USER fluid WITH PASSWORD 'strong-password';
GRANT ALL PRIVILEGES ON DATABASE fluid TO fluid;
```

Ensure PostgreSQL is listening only on localhost:

```conf
listen_addresses = '127.0.0.1'
```

---

## Configuration

Create the main configuration file:

```bash
vim /etc/fluid/config.yaml
```

Example:

```yaml
server:
  listen: 127.0.0.1:8080

database:
  host: 127.0.0.1
  port: 5432
  name: fluid
  user: fluid
  password: strong-password

hosts:
  - name: kvm-01
    address: 10.0.0.11
  - name: kvm-02
    address: 10.0.0.12
```

---

## SSH Access to Hosts

The control node requires SSH access to each libvirt host.

Recommended approach:

* Generate a dedicated SSH key for `fluid`
* Grant limited sudo or libvirt access on hosts

```bash
sudo -u fluid ssh-keygen -t ed25519
```

On each host, allow execution of `virsh` via sudo or libvirt permissions.

---

## systemd Service

Create the service unit:

```bash
vim /etc/systemd/system/fluid.service
```

```ini
[Unit]
Description=fluid.sh control plane
After=network.target postgresql.service

[Service]
User=fluid
Group=fluid
ExecStart=/usr/local/bin/fluid \
  --config /etc/fluid/config.yaml
Restart=on-failure
RestartSec=5
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
systemctl daemon-reload
systemctl enable fluid
systemctl start fluid
```

---

## Verifying the Installation

Check service status:

```bash
systemctl status fluid
```

Basic health checks:

```bash
fluid status
fluid hosts list
```

---

## Upgrade Strategy

* Download the new binary
* Verify checksum
* Replace `/usr/local/bin/fluid`
* Restart the systemd service

PostgreSQL migrations are handled automatically on startup.

---

## Uninstallation

```bash
systemctl stop fluid
systemctl disable fluid
rm /usr/local/bin/fluid
rm /etc/systemd/system/fluid.service
```

(Optional) Remove data and user:

```bash
userdel fluid
rm -rf /etc/fluid /var/lib/fluid /var/log/fluid
```

## Contributing Quickstart

### Prerequisites

- **mprocs** - For local dev
- **Docker & Docker Compose** - For containerized deployment in production
- **libvirt/KVM** - For virtual machine management
- **macOS**:
  - **qemu** - `brew install qemu` (the hypervisor)
  - **libvirt** - `brew install libvirt` (VM management daemon)
  - **socket_vmnet** - `brew install socket_vmnet` (VM networking)
  - **cdrtools** - `brew install cdrtools` (provides `mkisofs` for cloud-init)

### 30-Second Start

```bash
# Clone and start
git clone https://github.com/aspectrr/fluid.sh.git
cd fluid.sh
mprocs

# Services available at:
# API:      http://localhost:8080
# Web UI:   http://localhost:5173
```

---

## Platform Setup

<details>
<summary><b>Mac</b></summary>

You will need to install qemu, libvirt, socket_vmnet, and cdrtools on Mac:

```bash
# Install qemu, libvirt, socket_vmnet, and cdrtools
brew install qemu libvirt socket_vmnet cdrtools

# Start the libvirt daemon
brew services start libvirt

# Create image directories
sudo mkdir -p /var/lib/libvirt/images/{base,jobs}
sudo chown -R $(whoami) /var/lib/libvirt/images/{base,jobs}

# Verify libvirt is running
virsh -c qemu:///session list --all

# Set up SSH CA (Needed for Sandbox VMs)
cd fluid.sh
./scripts/setup-ssh-ca.sh --dir .ssh-ca

# Set up libvirt VM (ARM64 Ubuntu)
SSH_CA_PUB_PATH=.ssh-ca/ssh_ca.pub SSH_CA_KEY_PATH=.ssh-ca/ssh_ca ./scripts/reset-libvirt-macos.sh

# Start services
mprocs
```

**What happens:**
1. A SSH CA is generated and then is used to build the golden VM
2. libvirt runs on the machine and is queried by the fluid API
4. Test VMs run on your local machine

**Architecture:**
```
+---------------------------------------------------------------------+
|                     Apple Silicon Mac                                 |
|  +-----------------+                                                 |
|  | fluid           |                                                 |
|  | API + Web UI    |---->  +----------------------------------+      |
|  |                 |       |     libvirt/QEMU (ARM64)         |      |
|  | LIBVIRT_URI=    |       |  +----------+  +----------+      |      |
|  | qemu:///session |       |  | sandbox  |  | sandbox  | ...  |      |
|  |                 |       |  | VM (arm) |  | VM (arm) |      |      |
|  +-----------------+       |  +----------+  +----------+      |      |
|                            +----------------------------------+      |
|                                                                      |
+----------------------------------------------------------------------+
```

**Create ARM64 test VMs:**
```bash
./scripts/reset-libvirt-macos.sh
```

**Default test VM credentials:**
- Username: `testuser` / Password: `testpassword`
- Username: `root` / Password: `rootpassword`

</details>

<details>
<summary><b>Linux x86_64 (On-Prem / Bare Metal)</b></summary>

Direct libvirt access for best performance:

```bash
# Install libvirt and dependencies (Ubuntu/Debian)
sudo apt update
sudo apt install -y \
    qemu-kvm qemu-utils libvirt-daemon-system \
    libvirt-clients virtinst bridge-utils ovmf \
    cpu-checker cloud-image-utils genisoimage

# Or on Fedora/RHEL
sudo dnf install -y \
    qemu-kvm qemu-img libvirt libvirt-client \
    virt-install bridge-utils edk2-ovmf \
    cloud-utils genisoimage

# Enable and start libvirtd
sudo systemctl enable --now libvirtd

# Add your user to libvirt group
sudo usermod -aG libvirt,kvm $(whoami)
newgrp libvirt  # or log out and back in

# Verify KVM is available
kvm-ok

# Create image directories
sudo mkdir -p /var/lib/libvirt/images/{base,jobs}

# Create environment file
cat > .env << 'EOF'
LIBVIRT_URI=qemu:///system
LIBVIRT_NETWORK=default
DATABASE_URL=postgresql://fluid:fluid@localhost:5432/fluid
BASE_IMAGE_DIR=/var/lib/libvirt/images/base
SANDBOX_WORKDIR=/var/lib/libvirt/images/jobs
EOF

# Start the default network
sudo virsh net-autostart default
sudo virsh net-start default

# Verify
virsh -c qemu:///system list --all

# Start services
docker-compose up --build
```

**Architecture:**
```
+---------------------------------------------------------------------+
|                    Linux x86_64 Host                                 |
|                                                                      |
|  +-----------------+  +-----------------+  +---------------------+   |
|  | fluid           |  |   PostgreSQL    |  |    Web UI           |   |
|  | API (Go)        |  |   (Docker)      |  |    (React)          |   |
|  | :8080           |  |   :5432         |  |    :5173            |   |
|  +--------+--------+  +-----------------+  +---------------------+   |
|           |                                                          |
|           | LIBVIRT_URI=qemu:///system                               |
|           v                                                          |
|  +--------------------------------------------------------------+   |
|  |                    libvirt/KVM (native)                       |   |
|  |                                                               |   |
|  |   +--------------+  +--------------+  +--------------+        |   |
|  |   |  sandbox-1   |  |  sandbox-2   |  |  sandbox-N   |  ...  |   |
|  |   |  (x86_64)    |  |  (x86_64)    |  |  (x86_64)    |       |   |
|  |   +--------------+  +--------------+  +--------------+        |   |
|  +--------------------------------------------------------------+   |
+----------------------------------------------------------------------+
```

**Create a base VM image:**
```bash
# Download Ubuntu cloud image
cd /var/lib/libvirt/images/base
sudo wget https://cloud-images.ubuntu.com/jammy/current/jammy-server-cloudimg-amd64.img

# Create test VM using the provided script
./scripts/setup-ssh-ca.sh --dir [ssh-ca-dir]
./scripts/reset-libvirt-macos.sh [vm-name] [ca-pub-path] [ca-key-path]
```

**Default test VM credentials:**
- Username: `testuser` / Password: `testpassword`
- Username: `root` / Password: `rootpassword`

</details>

<details>
<summary><b>Linux ARM64 (Ampere, Graviton, Raspberry Pi)</b></summary>

Native ARM64 Linux with libvirt:

```bash
# Install libvirt and dependencies (Ubuntu/Debian ARM64)
sudo apt update
sudo apt install -y \
    qemu-kvm qemu-utils qemu-efi-aarch64 \
    libvirt-daemon-system libvirt-clients \
    virtinst bridge-utils cloud-image-utils genisoimage

# Enable and start libvirtd
sudo systemctl enable --now libvirtd

# Add your user to libvirt group
sudo usermod -aG libvirt,kvm $(whoami)
newgrp libvirt

# Create environment file
cat > .env << 'EOF'
LIBVIRT_URI=qemu:///system
LIBVIRT_NETWORK=default
DATABASE_URL=postgresql://fluid:fluid@localhost:5432/fluid
BASE_IMAGE_DIR=/var/lib/libvirt/images/base
SANDBOX_WORKDIR=/var/lib/libvirt/images/jobs
EOF

# Start the default network
sudo virsh net-autostart default
sudo virsh net-start default

# Start services
docker-compose up --build
```

**Download ARM64 cloud images:**
```bash
cd /var/lib/libvirt/images/base
sudo wget https://cloud-images.ubuntu.com/jammy/current/jammy-server-cloudimg-arm64.img
```

**Architecture is the same as x86_64 but with ARM64 VMs.**

**Default test VM credentials:**
- Username: `testuser` / Password: `testpassword`
- Username: `root` / Password: `rootpassword`

</details>

<details>
<summary><b>Remote libvirt Server</b></summary>

Connect to a remote libvirt host over SSH or TCP:

```bash
# SSH connection (recommended - secure)
export LIBVIRT_URI="qemu+ssh://user@remote-host/system"

# Or with specific SSH key
export LIBVIRT_URI="qemu+ssh://user@remote-host/system?keyfile=/path/to/key"

# TCP connection (less secure - ensure network is trusted)
export LIBVIRT_URI="qemu+tcp://remote-host:16509/system"

# Test connection
virsh -c "$LIBVIRT_URI" list --all

# Create .env file
cat > .env << EOF
LIBVIRT_URI=${LIBVIRT_URI}
LIBVIRT_NETWORK=default
DATABASE_URL=postgresql://fluid:fluid@localhost:5432/fluid
EOF

# Start services
docker-compose up --build
```

**Remote server setup (on the libvirt host):**
```bash
# For SSH access, ensure SSH is enabled and user has libvirt access
sudo usermod -aG libvirt remote-user

# For TCP access (development only!), configure /etc/libvirt/libvirtd.conf:
#   listen_tls = 0
#   listen_tcp = 1
#   auth_tcp = "none"  # WARNING: No authentication!
# Then restart: sudo systemctl restart libvirtd
```

</details>

---

## Project Structure

```
fluid.sh/
├── fluid/                 #    Main API server & CLI (Go)
│   ├── cmd/fluid/         #    Entry point
│   ├── internal/          #    Business logic
├── scripts/               #    Setup & utility scripts
├── web/                   #    React frontend
│   └── src/               #    Components, hooks, routes
├── sdk/                   #    Python SDK
│   └── fluid-sdk-py/      #    Auto-generated client
├── examples/              #    Example implementations
│   └── agent-example/     #    AI agent with OpenAI
└── docker-compose.yml     #    Container orchestration
```

## API Reference


### Sandbox Lifecycle

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/v1/sandboxes` | Create a new sandbox |
| `GET` | `/v1/sandboxes/{id}` | Get sandbox details |
| `POST` | `/v1/sandboxes/{id}/start` | Start a sandbox |
| `POST` | `/v1/sandboxes/{id}/stop` | Stop a sandbox |
| `DELETE` | `/v1/sandboxes/{id}` | Destroy a sandbox |

### Command Execution

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/v1/sandboxes/{id}/command` | Run SSH command |
| `POST` | `/api/v1/tmux/panes/send-keys` | Send keystrokes to tmux |
| `POST` | `/api/v1/tmux/panes/read` | Read tmux pane content |

### Snapshots

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/v1/sandboxes/{id}/snapshots` | Create snapshot |
| `GET` | `/v1/sandboxes/{id}/snapshots` | List snapshots |
| `POST` | `/v1/sandboxes/{id}/snapshots/{name}/restore` | Restore snapshot |

### Human Approval

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/api/v1/human/ask` | Request approval (blocking) |

## Security Model

### Isolation Layers

1. **VM Isolation** - Each sandbox is a separate KVM virtual machine
2. **Network Isolation** - VMs run on isolated virtual networks
3. **SSH Certificates** - Ephemeral credentials that auto-expire (1-10 minutes)
4. **Human Approval** - Gate sensitive operations

### Safety Features

-  Command allowlists/denylists
-  Path restrictions for file access
-  Timeout limits on all operations
-  Output size limits
-  Full audit trail
-  Snapshot rollback

### SSH Host Key Verification

The control node connects to hypervisor hosts via SSH. You **must** configure proper host key verification to prevent man-in-the-middle attacks.

**Required: Configure `~/.ssh/config` on the control node:**

```ssh-config
# /home/fluid/.ssh/config (for the fluid user)

# Global defaults - strict verification
Host *
    StrictHostKeyChecking yes
    UserKnownHostsFile ~/.ssh/known_hosts

# Hypervisor hosts - explicitly trusted
Host kvm-01
    HostName 10.0.0.11
    User root
    IdentityFile ~/.ssh/id_ed25519

Host kvm-02
    HostName 10.0.0.12
    User root
    IdentityFile ~/.ssh/id_ed25519
```

**Pre-populate known_hosts before first use:**

```bash
# As the fluid user, add each host's key
sudo -u fluid ssh-keyscan -H 10.0.0.11 >> /home/fluid/.ssh/known_hosts
sudo -u fluid ssh-keyscan -H 10.0.0.12 >> /home/fluid/.ssh/known_hosts

# Verify the fingerprints match your hosts
sudo -u fluid ssh-keygen -lf /home/fluid/.ssh/known_hosts
```

**Warning:** Never use `StrictHostKeyChecking=no` in production. This disables host verification and exposes you to MITM attacks.

##  Documentation

- [Docs from Previous Issues](./docs/) - Documentation on common issues working with the project
- [Scripts Reference](./scripts/README.md) - Setup and utility scripts
- [SSH Certificates](./scripts/README.md#ssh-certificate-based-access) - Ephemeral credential system
- [Agent Connection Flow](./docs/agent-connection-flow.md) - How agents connect to sandboxes
- [Examples](./examples/) - Working examples

## Development

To run the API locally, first build the `fluid` binary:

```bash
# Build the API binary
cd fluid && make build
```

Then, use `mprocs` to run all the services together for local development.

```bash
# Install mprocs for multi-service development
brew install mprocs  # macOS
cargo install mprocs # Linux

# Start all services with hot-reload
mprocs

# Or run individual services
cd fluid && make run
cd web && bun run dev
```

### Running Tests

```bash
# Go services
(cd fluid && make test)

# Python SDK
(cd sdk/fluid-sdk-py && pytest)

# All checks
(cd fluid && make check)
```

##  Contributing

1. Fork the repository
2. Create a feature branch
3. Make changes with tests
4. Run `make check`
5. Submit a pull request

All contributions must maintain the security model and include appropriate tests.

## License

MIT License - see [LICENSE](LICENSE) for details.


<div align="center">

Made with <3

</div>
