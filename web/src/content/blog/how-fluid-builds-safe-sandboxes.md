---
title: 'How Fluid Builds Safe Sandboxes for AI Agents'
pubDate: 2026-02-10
description: "A technical deep-dive into how Fluid's CLI creates isolated VM sandboxes from golden images -- fast linked clones, ephemeral SSH certificates, and cleanup that leaves no trace."
author: 'Collin @ Fluid.sh'
authorImage: '/images/skeleton_smoking_cigarette.jpg'
authorEmail: 'cpfeifer@madcactus.org'
authorPhone: '+3179955114'
authorDiscord: 'https://discordapp.com/users/301068417685913600'
---

## The Problem

AI agents need somewhere safe to work. When an agent installs packages, edits config files, or runs arbitrary shell commands, those changes can't touch production infrastructure. The agent needs a disposable environment that looks and acts like a real machine but can be thrown away the moment the task is done.

The naive approach is spinning up a fresh VM from scratch every time. That works, but it's slow. A full OS install takes minutes. Downloading a base image takes longer. And if the agent needs a specific software stack -- Python 3.12, nginx, PostgreSQL -- you're either baking custom images for every combination or waiting for the agent to install everything from scratch each time.

Fluid solves this differently. It takes existing "golden" VMs that already have the right software stack and creates lightweight clones in seconds. The clone gets its own identity, its own network address, its own SSH credentials -- but it shares the base disk with the original. When the agent is done, everything gets cleaned up: the VM, the disk, the credentials, the DHCP lease. Nothing lingers.

## Linked Clones: Fast Without Copying

The core trick is QCOW2 linked clones. Instead of duplicating a 10GB disk image, Fluid creates a copy-on-write overlay that starts at zero bytes and only grows as the sandbox writes data. All reads that don't hit the overlay fall through to the source disk.

When `fluid create --source-vm=ubuntu-base` runs, here's what actually happens inside `CloneFromVM`:

**1. Find the source disk.** The CLI runs `virsh domblklist` to locate the source VM's QCOW2 image -- something like `/var/lib/libvirt/images/base/ubuntu-base.qcow2`.

**2. Create a workspace.** Each sandbox gets its own directory:

```
/var/lib/libvirt/images/sandboxes/sbx-abc123/
├── disk-overlay.qcow2    # Copy-on-write overlay
├── cloud-init.iso         # Unique identity seed
└── domain.xml             # Libvirt domain definition
```

**3. Create the overlay.**

```bash
qemu-img create -f qcow2 -F qcow2 -b <base-disk-path> <overlay-path>
```

This is the fast part. The overlay is a thin file that references the source disk as its backing store. No data is copied. The sandbox can read anything the source VM had, but every write goes to the overlay.

**4. Give the clone its own identity.** This is where it gets subtle. Without a unique `instance-id`, cloud-init inside the clone detects that it already ran (from the source VM's state) and skips network initialization. The sandbox comes up with no IP address, which is useless.

Fluid generates a cloud-init NoCloud ISO with a fresh identity:

```yaml
# meta-data
instance-id: sbx-abc123
local-hostname: sbx-abc123

# user-data
#cloud-config
network:
  version: 2
  ethernets:
    id0:
      match:
        driver: virtio*
      dhcp4: true
```

ISO generation is best-effort -- if tools like `cloud-localds` or `genisoimage` aren't available, the CLI logs a warning and continues. The clone might still work if the source VM didn't rely on cloud-init for networking, but there's a risk it comes up without a fresh IP.

**5. Modify the domain XML.** The source VM's libvirt XML is dumped and rewritten:

- **Name**: set to the sandbox name
- **UUID**: removed (libvirt assigns a new one)
- **Disk path**: pointed at the overlay instead of the source
- **MAC address**: regenerated so the clone gets its own DHCP lease
- **Cloud-init CDROM**: attached with the new ISO
- **PCI addresses**: stripped from NICs to avoid slot conflicts

**6. Boot it.** `virsh define` and `virsh start`. The CLI then polls for an IP address via DHCP lease data and verifies SSH connectivity before returning the sandbox as ready.

The full orchestration in the VM service follows this sequence: create the clone, persist metadata to SQLite, start the VM, wait for an IP, verify SSH, mark it `RUNNING`. If any step fails, the partial state is cleaned up.

## SSH Certificates: No Keys on the VM

Traditional VM access means scattering SSH public keys across `authorized_keys` files. That creates a management problem -- who has access to what, when do keys expire, how do you revoke them?

Fluid runs its own SSH Certificate Authority instead. On `fluid init`, an Ed25519 CA keypair is generated at `~/.fluid/ssh-ca/`. The CA public key gets installed on VMs so `sshd` can verify certificates signed by it.

When the CLI needs to run a command in a sandbox, the credential flow is:

1. **Check cache.** If valid credentials exist for this sandbox (with a 30-second refresh margin before expiry), reuse them.
2. **Generate an ephemeral keypair.** A fresh Ed25519 key, unique to this sandbox.
3. **Sign it.** The CA issues a certificate:

```bash
ssh-keygen -s ~/.fluid/ssh-ca/ssh-ca \
    -I "user:<agent-id>-vm:<vm-id>-sbx:<sandbox-id>-cert:<cert-id>" \
    -n sandbox \
    -V +30m \
    -z <serial> \
    -O no-port-forwarding \
    -O no-agent-forwarding \
    -O no-X11-forwarding \
    <user_key.pub>
```

4. **Store locally.** The private key and certificate land in `~/.fluid/sandbox-keys/SBX-abc123/`.

Every certificate embeds the agent ID, VM ID, sandbox ID, and a unique cert ID in the identity string. Certificates expire after 30 minutes by default. They're automatically renewed 30 seconds before expiry. No port forwarding, no agent forwarding, no X11 -- the certificate is scoped to running commands and nothing else.

The security properties here are worth spelling out:

- **Short-lived**: 30-minute TTL limits exposure if a key leaks
- **Per-sandbox isolation**: compromising one sandbox's credentials doesn't grant access to another
- **Audit trail**: the identity string makes every SSH session traceable back to a specific agent, sandbox, and cert
- **Permission enforcement**: the CA private key must be `0600` or `0400` -- the CLI refuses to start if permissions are too open

## Command Execution: SSH With Retries

Running a command is straightforward. The CLI re-discovers the sandbox IP from DHCP leases (never cached -- prevents stale routing), validates that no other running sandbox shares the same IP, obtains credentials, and executes:

```bash
ssh -i <private-key> \
    -o CertificateFile=<cert-path> \
    -o StrictHostKeyChecking=no \
    -o UserKnownHostsFile=/dev/null \
    -o ConnectTimeout=15 \
    sandbox@<ip> \
    -- <command>
```

Every command, its exit code, stdout, and stderr get persisted to SQLite. This creates a full audit trail -- every command an agent ran in every sandbox, with complete output, available for replay and debugging.

SSH connection failures (exit code 255) trigger retries with exponential backoff: 2s, 4s, 8s, 16s, 30s (capped), up to 5 attempts. Only transient failures -- connection refused, timeout, DNS errors -- get retried. If the command itself fails (non-zero exit code from the actual program), that's returned immediately.

For sandboxes on remote KVM hosts, the CLI chains through a ProxyJump:

```bash
ssh -J root@<host-address> sandbox@<sandbox-ip> -- <command>
```

## Cleanup: Leave No Trace

When `fluid destroy` is called, everything gets cleaned up. Not just the VM -- everything associated with it.

The cleanup sequence hits five layers:

**1. SSH credentials.** Cached credentials are purged from memory. The key directory (`~/.fluid/sandbox-keys/{id}/`) is deleted from disk. The per-sandbox mutex is removed from the lock map.

**2. The VM itself.** `virsh destroy` force-stops the VM process. `virsh undefine --remove-all-storage` removes the domain definition and deletes associated storage volumes. Falls back to `virsh undefine` without the storage flag for older libvirt versions.

**3. DHCP lease.** The VM's MAC address is removed from the dnsmasq lease file. This prevents IP address conflicts when future sandboxes are created -- without this, a new VM could collide with a stale lease.

**4. Workspace.** The entire working directory -- disk overlay, cloud-init ISO, domain XML, external snapshots -- is deleted.

**5. Database.** Sandbox records are soft-deleted (a `deleted_at` timestamp rather than physical removal). Related records -- commands, snapshots, diffs -- are retained for audit.

| Resource                 | Method                       | Location                      |
| ------------------------ | ---------------------------- | ----------------------------- |
| VM process               | `virsh destroy`              | libvirt                       |
| Domain definition        | `virsh undefine`             | libvirt                       |
| Disk overlay + ISO + XML | `os.RemoveAll`               | `{WorkDir}/{vm-name}/`        |
| DHCP lease               | Lease file rewrite           | `/var/lib/libvirt/dnsmasq/`   |
| SSH keys + certificate   | `os.RemoveAll`               | `~/.fluid/sandbox-keys/{id}/` |
| In-memory credentials    | `delete(m.credentials, key)` | KeyManager                    |
| Database record          | Soft delete                  | `~/.fluid/state.db`           |

The interactive TUI takes this further. It tracks every sandbox created during a session, and on exit -- including `Ctrl+C` -- a deferred cleanup function destroys them all. A "leave no trace" policy for the whole session.

## Snapshots and Diffs

Sometimes an agent needs to checkpoint its work. Fluid supports two snapshot types:

**Internal snapshots** are managed by QEMU inside the VM's QCOW2 file. Faster to create, but the VM must be in a consistent state. Created with `virsh snapshot-create-as`.

**External snapshots** create a new QCOW2 overlay at each snapshot point. They can be taken while the VM is running and produce separate files at `{WorkDir}/{vm}/snap-{name}.qcow2`.

Diffing snapshots gives the agent a way to see what changed between two checkpoints. For external snapshots, Fluid provides instructions for mounting both QCOW2 files via `qemu-nbd` and diffing the filesystem trees. Command history between snapshot points is also available from the SQLite audit trail.

## Why This Architecture

The design choices come back to a single constraint: AI agents are untrusted code that needs real machine access.

Linked clones give agents fast, cheap environments without duplicating disk. The SSH CA gives them scoped, short-lived credentials without key management overhead. Full command auditing means every action is traceable. And aggressive cleanup means a destroyed sandbox truly disappears -- no orphaned disks, no stale leases, no lingering credentials.

The result is an agent that can `fluid create`, install software, run tests, take snapshots, and `fluid destroy` -- all in a self-contained loop where the blast radius of any mistake is exactly one disposable VM.
