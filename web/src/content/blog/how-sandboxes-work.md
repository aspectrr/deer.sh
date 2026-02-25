---
title: 'How Fluid.sh Sandboxes Work'
pubDate: 2026-02-25
description: 'A deep dive into QEMU microVMs, copy-on-write overlays, ephemeral SSH certificates, and the infrastructure behind Fluid sandboxes.'
author: 'Collin @ Fluid.sh'
authorImage: '/images/skeleton_smoking_cigarette.jpg'
authorEmail: 'cpfeifer@madcactus.org'
authorPhone: '+3179955114'
authorDiscord: 'https://discordapp.com/users/301068417685913600'
---

## Intro

When you ask Fluid to spin up a sandbox, you are not waiting for a full OS installation or a slow VM clone. A QEMU microVM boots from a shared base image in milliseconds - no bootloader, no BIOS, no wasted disk. Each sandbox gets a copy-on-write overlay that starts at a few kilobytes, a TAP device on the host's network bridge, and an ephemeral SSH certificate that expires in 30 minutes.

The stack: QEMU microVM + CoW qcow2 overlay + TAP networking + SSH Certificate Authority + gRPC control. Here is how all of it works.

<div class="sbx-diagram-container">
  <div class="sbx-diagram-header">Traditional VMs vs Fluid MicroVM Sandboxes</div>
  <div class="sbx-diagram-content">
    <div class="sbx-diagram-side">
      <div class="sbx-side-title">TRADITIONAL: 4 Full VM Clones</div>
      <div class="sbx-vm-grid">
        <div class="sbx-vm-box sbx-vm-traditional">
          <div class="sbx-vm-name">VM-1</div>
          <div class="sbx-vm-stats">
            <span>CPU: 2 cores</span>
            <span>RAM: 4 GB</span>
            <span>Disk: 20 GB</span>
          </div>
          <div class="sbx-disk-bar sbx-disk-full"></div>
        </div>
        <div class="sbx-vm-box sbx-vm-traditional">
          <div class="sbx-vm-name">VM-2</div>
          <div class="sbx-vm-stats">
            <span>CPU: 2 cores</span>
            <span>RAM: 4 GB</span>
            <span>Disk: 20 GB</span>
          </div>
          <div class="sbx-disk-bar sbx-disk-full"></div>
        </div>
        <div class="sbx-vm-box sbx-vm-traditional">
          <div class="sbx-vm-name">VM-3</div>
          <div class="sbx-vm-stats">
            <span>CPU: 2 cores</span>
            <span>RAM: 4 GB</span>
            <span>Disk: 20 GB</span>
          </div>
          <div class="sbx-disk-bar sbx-disk-full"></div>
        </div>
        <div class="sbx-vm-box sbx-vm-traditional">
          <div class="sbx-vm-name">VM-4</div>
          <div class="sbx-vm-stats">
            <span>CPU: 2 cores</span>
            <span>RAM: 4 GB</span>
            <span>Disk: 20 GB</span>
          </div>
          <div class="sbx-disk-bar sbx-disk-full"></div>
        </div>
      </div>
      <div class="sbx-totals sbx-totals-bad">
        <span>TOTAL DISK: 80 GB</span>
        <span>Creation: ~2-5 min each</span>
      </div>
    </div>
    <div class="sbx-diagram-side">
      <div class="sbx-side-title sbx-side-title-good">FLUID: MicroVM Sandboxes (CoW)</div>
      <div class="sbx-vm-grid">
        <div class="sbx-vm-box sbx-vm-sandbox">
          <div class="sbx-vm-name">SBX-1</div>
          <div class="sbx-vm-stats">
            <span>CPU: 2 cores</span>
            <span>RAM: 2 GB</span>
            <span class="sbx-disk-tiny">Disk: 128 KB</span>
          </div>
          <div class="sbx-disk-bar sbx-disk-tiny-bar"></div>
          <div class="sbx-connector"></div>
        </div>
        <div class="sbx-vm-box sbx-vm-sandbox">
          <div class="sbx-vm-name">SBX-2</div>
          <div class="sbx-vm-stats">
            <span>CPU: 2 cores</span>
            <span>RAM: 2 GB</span>
            <span class="sbx-disk-tiny">Disk: 256 KB</span>
          </div>
          <div class="sbx-disk-bar sbx-disk-tiny-bar"></div>
          <div class="sbx-connector"></div>
        </div>
        <div class="sbx-vm-box sbx-vm-sandbox">
          <div class="sbx-vm-name">SBX-3</div>
          <div class="sbx-vm-stats">
            <span>CPU: 2 cores</span>
            <span>RAM: 2 GB</span>
            <span class="sbx-disk-tiny">Disk: 64 KB</span>
          </div>
          <div class="sbx-disk-bar sbx-disk-tiny-bar"></div>
          <div class="sbx-connector"></div>
        </div>
        <div class="sbx-vm-box sbx-vm-sandbox">
          <div class="sbx-vm-name">SBX-4</div>
          <div class="sbx-vm-stats">
            <span>CPU: 2 cores</span>
            <span>RAM: 2 GB</span>
            <span class="sbx-disk-tiny">Disk: 512 KB</span>
          </div>
          <div class="sbx-disk-bar sbx-disk-tiny-bar"></div>
          <div class="sbx-connector"></div>
        </div>
      </div>
      <div class="sbx-base-image">
        <div class="sbx-base-label">BASE IMAGE (Read-Only)</div>
        <div class="sbx-base-stats">Disk: 20 GB</div>
        <div class="sbx-disk-bar sbx-disk-full sbx-disk-base"></div>
      </div>
      <div class="sbx-totals sbx-totals-good">
        <span>TOTAL DISK: ~20 GB</span>
        <span>Creation: ~50ms each</span>
      </div>
    </div>
  </div>
</div>

## The MicroVM: No Bootloader, No BIOS

Traditional VMs boot through BIOS/UEFI, a bootloader (GRUB), and then the kernel. Fluid skips all of that. Each sandbox is a QEMU microVM - a minimal virtual machine type designed for fast, lightweight workloads.

The daemon launches each sandbox with:

```
qemu-system-x86_64 \
  -M microvm -enable-kvm -cpu host \
  -m 2048 -smp 2 \
  -kernel /path/to/extracted-vmlinux \
  -append "console=ttyS0 root=/dev/vda rw quiet" \
  -drive id=root,file=overlay.qcow2,format=qcow2,if=none \
  -device virtio-blk-device,drive=root \
  -netdev tap,id=net0,ifname=fl-abc123def,script=no,downscript=no \
  -device virtio-net-device,netdev=net0,mac=52:54:00:xx:xx:xx \
  -serial stdio -nographic -nodefaults \
  -daemonize -pidfile qemu.pid
```

Key points:

- **`-M microvm`**: The microvm machine type strips away legacy hardware emulation. No PCI bus, no ACPI tables, no USB controllers. Just VirtIO devices.
- **`-kernel`**: Direct kernel boot. The kernel is extracted from the base image ahead of time (via libguestfs `virt-cat` or NBD mount), decompressed, and passed directly to QEMU. No bootloader involved.
- **`-append "console=ttyS0 root=/dev/vda rw quiet"`**: Kernel command line. The root filesystem lives on a VirtIO block device.
- **`-daemonize -pidfile`**: The QEMU process forks into the background. The daemon tracks it by PID and can recover state on restart by scanning PID files.

Defaults: 2 vCPUs, 2048 MB RAM. Each sandbox runs as an independent QEMU process on the host.

<div class="sbx-diagram-container">
  <div class="sbx-diagram-header">MicroVM Boot Sequence</div>
  <div class="sbx-flow-vertical">
    <div class="sbx-flow-card">
      <div class="sbx-flow-card-title">1. Base Image</div>
      <div class="sbx-flow-card-body">QCOW2 cloud image (e.g. Ubuntu 24.04)<br>Stored read-only in image cache</div>
    </div>
    <div class="sbx-flow-arrow">&#8595;</div>
    <div class="sbx-flow-card">
      <div class="sbx-flow-card-title">2. Kernel Extraction</div>
      <div class="sbx-flow-card-body">virt-cat /boot/vmlinuz-* from image<br>Decompress to .vmlinux (cached)</div>
    </div>
    <div class="sbx-flow-arrow">&#8595;</div>
    <div class="sbx-flow-card">
      <div class="sbx-flow-card-title">3. CoW Overlay</div>
      <div class="sbx-flow-card-body">qemu-img create -f qcow2 -b base.qcow2<br>Starts at ~128 KB</div>
    </div>
    <div class="sbx-flow-arrow">&#8595;</div>
    <div class="sbx-flow-card">
      <div class="sbx-flow-card-title">4. TAP + MAC</div>
      <div class="sbx-flow-card-body">Create TAP device, attach to bridge<br>Generate random 52:54:00:xx:xx:xx MAC</div>
    </div>
    <div class="sbx-flow-arrow">&#8595;</div>
    <div class="sbx-flow-card sbx-flow-card-highlight">
      <div class="sbx-flow-card-title">5. QEMU Launch</div>
      <div class="sbx-flow-card-body">-M microvm -enable-kvm -cpu host<br>Direct kernel boot, daemonized with PID tracking</div>
    </div>
    <div class="sbx-flow-arrow">&#8595;</div>
    <div class="sbx-flow-card">
      <div class="sbx-flow-card-title">6. IP Discovery</div>
      <div class="sbx-flow-card-body">Poll ARP table / lease files for MAC<br>Sandbox ready when IP resolves</div>
    </div>
  </div>
</div>

## Copy-on-Write Overlays

Every sandbox gets its own overlay disk, but they all share the same base image. The overlay records only the blocks that change:

```
qemu-img create -f qcow2 -b /var/lib/fluid/images/ubuntu-24.04.qcow2 -F qcow2 \
  /var/lib/fluid/overlays/<sandbox-id>/disk.qcow2
```

The overlay lives at `<workDir>/<sandboxID>/disk.qcow2` alongside a `metadata.json` file that stores the sandbox's TAP device, MAC address, bridge name, and resource allocation. This metadata enables the daemon to recover state on restart.

Why this matters:

- **Storage efficiency**: 10 sandboxes from the same base share 1 copy of the OS. Each overlay starts at a few KB.
- **Instant creation**: `qemu-img create` is effectively a metadata operation. No data copying.
- **Clean teardown**: Destroying a sandbox is `rm -rf <sandboxID>/`. The overlay directory and everything in it (disk, PID file, metadata) disappears.
- **Base image safety**: The base is read-only. A sandbox cannot corrupt the image used by other sandboxes.

## TAP Networking

Each sandbox gets a TAP device on the host, attached to a network bridge. This puts the sandbox on the same L2 network as the host - it gets a real IP via DHCP, reachable directly from the host and from other sandboxes on the same bridge.

TAP device naming follows a strict format to stay within the Linux 15-character interface name limit:

```
fl-<first 9 chars of sandbox ID>
```

Creation sequence:

```
ip tuntap add dev fl-abc123def mode tap
ip link set fl-abc123def master fluid0
ip link set fl-abc123def up
```

**Bridge resolution** follows a priority chain: explicit network request > source VM's network (queried via `virsh domiflist`) > default bridge from config.

**MAC address generation** uses the QEMU/KVM prefix `52:54:00` followed by 3 random bytes, giving each sandbox a unique identity on the network.

**IP discovery** supports three modes depending on your network setup:

- **ARP**: Polls `ip neigh show dev <bridge>` for the MAC address. Default mode.
- **libvirt**: Reads dnsmasq lease files from `/var/lib/libvirt/dnsmasq/`.
- **dnsmasq**: Reads lease files from `/var/lib/fluid/dnsmasq/`.

<div class="sbx-diagram-container">
  <div class="sbx-diagram-header">Network Architecture</div>
  <div class="sbx-net-layout">
    <div class="sbx-net-bridge">
      <div class="sbx-net-bridge-label">Bridge: fluid0 (10.0.0.0/24)</div>
      <div class="sbx-net-bridge-bar"></div>
    </div>
    <div class="sbx-net-taps">
      <div class="sbx-net-tap-group">
        <div class="sbx-net-tap-line"></div>
        <div class="sbx-net-tap-label">fl-abc123def</div>
        <div class="sbx-vm-box sbx-vm-sandbox sbx-net-vm">
          <div class="sbx-vm-name">SBX-1</div>
          <div class="sbx-vm-stats">
            <span>52:54:00:a1:b2:c3</span>
            <span>10.0.0.101</span>
          </div>
        </div>
      </div>
      <div class="sbx-net-tap-group">
        <div class="sbx-net-tap-line"></div>
        <div class="sbx-net-tap-label">fl-def456abc</div>
        <div class="sbx-vm-box sbx-vm-sandbox sbx-net-vm">
          <div class="sbx-vm-name">SBX-2</div>
          <div class="sbx-vm-stats">
            <span>52:54:00:d4:e5:f6</span>
            <span>10.0.0.102</span>
          </div>
        </div>
      </div>
      <div class="sbx-net-tap-group">
        <div class="sbx-net-tap-line"></div>
        <div class="sbx-net-tap-label">fl-789ghijkl</div>
        <div class="sbx-vm-box sbx-vm-sandbox sbx-net-vm">
          <div class="sbx-vm-name">SBX-3</div>
          <div class="sbx-vm-stats">
            <span>52:54:00:78:9a:bc</span>
            <span>10.0.0.103</span>
          </div>
        </div>
      </div>
    </div>
    <div class="sbx-net-host">
      <div class="sbx-net-host-label">HOST (DHCP, routing, internet)</div>
    </div>
  </div>
</div>

## Ephemeral SSH: The Certificate Authority

Fluid never drops a static SSH key onto a sandbox. Instead, the daemon runs an internal SSH Certificate Authority that issues short-lived, scoped certificates.

**Setup**: The daemon generates an Ed25519 CA keypair on first run (`ssh-keygen -t ed25519`). The CA public key is baked into base images so sandboxes trust it from boot. The private key stays on the daemon host at `/etc/fluid/ssh_ca` with `0600` permissions.

**Per-sandbox flow**:

1. When a sandbox is created, the daemon generates a fresh Ed25519 keypair for that sandbox.
2. The sandbox's public key is signed by the CA with `ssh-keygen -s`:
   - Identity: `user:<userID>-vm:<vmID>-sbx:<sandboxID>-cert:<certID>`
   - Principals: `["sandbox"]` (the user allowed to connect)
   - Default TTL: 30 minutes, maximum 60 minutes
   - Restrictions: `no-port-forwarding`, `no-agent-forwarding`, `no-X11-forwarding`
   - PTY: enabled (required for interactive use and tmux)
3. The daemon caches the private key, public key, and certificate. Credentials are refreshed automatically before expiry.
4. On sandbox destroy, all credentials are purged.

For **source VM** access (read-only commands against production VMs), the same CA issues certificates but the target VM is configured with a restricted shell that only allows read operations.

<div class="sbx-diagram-container">
  <div class="sbx-diagram-header">SSH Certificate Flow</div>
  <div class="sbx-flow-horizontal">
    <div class="sbx-flow-card sbx-hflow-card">
      <div class="sbx-flow-card-title">1. CA Init</div>
      <div class="sbx-flow-card-body">Ed25519 keypair<br>/etc/fluid/ssh_ca<br>Pub key in VM images</div>
    </div>
    <div class="sbx-flow-arrow-h">&#8594;</div>
    <div class="sbx-flow-card sbx-hflow-card">
      <div class="sbx-flow-card-title">2. Key Gen</div>
      <div class="sbx-flow-card-body">Per-sandbox Ed25519<br>keypair generated<br>on create</div>
    </div>
    <div class="sbx-flow-arrow-h">&#8594;</div>
    <div class="sbx-flow-card sbx-hflow-card sbx-flow-card-highlight">
      <div class="sbx-flow-card-title">3. Sign</div>
      <div class="sbx-flow-card-body">ssh-keygen -s CA_KEY<br>TTL: 30min<br>no-port-forwarding</div>
    </div>
    <div class="sbx-flow-arrow-h">&#8594;</div>
    <div class="sbx-flow-card sbx-hflow-card">
      <div class="sbx-flow-card-title">4. Connect</div>
      <div class="sbx-flow-card-body">ssh -i key -o CertificateFile=cert<br>sandbox@10.0.0.x<br>Auto-refresh before expiry</div>
    </div>
  </div>
</div>

## Command Execution Pipeline

When you run a command in a sandbox, here is the full path it takes:

1. **CLI -> Daemon (gRPC)**: The CLI sends a `RunCommand` RPC to the daemon on port 9091. TLS 1.2+ is enforced for transport security.
2. **Daemon -> Sandbox (SSH)**: The daemon looks up the sandbox's IP address, retrieves cached SSH credentials (private key + certificate), and opens an SSH connection to `sandbox@<ip>`.
3. **Execution**: The command runs inside the sandbox. Stdout, stderr, and the exit code are captured. Default timeout: 5 minutes.
4. **Persistence**: The command result is recorded in the daemon's SQLite state store - command text, stdout, stderr, exit code, duration in milliseconds, timestamps.
5. **Daemon -> CLI (gRPC)**: The result is returned via the gRPC response.

<div class="sbx-diagram-container">
  <div class="sbx-diagram-header">Command Execution Pipeline</div>
  <div class="sbx-flow-vertical">
    <div class="sbx-flow-card">
      <div class="sbx-flow-card-title">CLI</div>
      <div class="sbx-flow-card-body">RunCommand(sandbox_id, "apt update")</div>
    </div>
    <div class="sbx-flow-arrow">&#8595; gRPC :9091 (TLS 1.2+)</div>
    <div class="sbx-flow-card">
      <div class="sbx-flow-card-title">Daemon</div>
      <div class="sbx-flow-card-body">Lookup sandbox IP + SSH credentials<br>Open SSH session to sandbox@10.0.0.x</div>
    </div>
    <div class="sbx-flow-arrow">&#8595; SSH (certificate auth)</div>
    <div class="sbx-flow-card sbx-flow-card-highlight">
      <div class="sbx-flow-card-title">Sandbox MicroVM</div>
      <div class="sbx-flow-card-body">Execute command, capture stdout/stderr<br>Exit code + duration tracked</div>
    </div>
    <div class="sbx-flow-arrow">&#8593; stdout, stderr, exit_code</div>
    <div class="sbx-flow-card">
      <div class="sbx-flow-card-title">Daemon</div>
      <div class="sbx-flow-card-body">Persist to SQLite state store<br>Return result via gRPC</div>
    </div>
    <div class="sbx-flow-arrow">&#8593; gRPC response</div>
    <div class="sbx-flow-card">
      <div class="sbx-flow-card-title">CLI</div>
      <div class="sbx-flow-card-body">Display result to user</div>
    </div>
  </div>
</div>

## Control Plane (Optional)

For multi-host deployments, daemons can connect to a central control plane. The connection is designed to be NAT-friendly: the daemon connects _out_ to the control plane, not the other way around.

- **Protocol**: Bidirectional gRPC streaming. The daemon opens one long-lived stream and both sides send messages on it.
- **Authentication**: Bearer token per-RPC + optional mTLS with client certificates and custom CA.
- **Registration**: On connect, the daemon sends a registration message with its host ID, hostname, version, available CPUs, and source VM list. The control plane can reassign host IDs.
- **Heartbeat**: Every 30 seconds the daemon sends available CPU count, active sandbox count, and source VM count.
- **Reconnection**: Exponential backoff starting at 1 second, doubling up to 60 seconds max. Backoff resets to 1 second after a connection stays stable for 5 minutes.
- **Command dispatch**: The control plane sends sandbox operations (create, destroy, run command, etc.) over the stream. The daemon processes up to 64 concurrent command handlers.

## The Janitor

Sandboxes are not meant to live forever. The Janitor is a background goroutine that enforces TTL-based cleanup.

- Default TTL: 24 hours.
- Check interval: 1 minute.
- On expiry: full destroy pipeline - SIGKILL the QEMU process, delete the overlay directory, remove the TAP device, purge SSH credentials, delete from state store.

The Janitor runs a cleanup pass immediately on startup, then ticks every minute. If a sandbox has no custom TTL, the default applies. This prevents forgotten sandboxes from accumulating resources on the host.

## Wrapping Up

Fluid sandboxes are production infrastructure: QEMU microVMs with direct kernel boot, copy-on-write overlays, TAP networking on the host bridge, ephemeral SSH certificates from an internal CA, SQLite state persistence with daemon restart recovery, optional multi-host orchestration through the control plane, and automatic cleanup via the Janitor. Every sandbox is isolated at the hypervisor level and disposable by design.

<style>
.sbx-diagram-container {
  background: linear-gradient(135deg, #0a0a0a 0%, #0c1929 100%);
  border: 1px solid #1e3a5f;
  border-radius: 0.75rem;
  padding: 1.5rem;
  margin: 2rem 0;
  font-family: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace;
  box-shadow: 0 0 30px rgba(96, 165, 250, 0.1);
}
.sbx-diagram-header {
  text-align: center;
  color: #60a5fa;
  font-size: 0.875rem;
  font-weight: 600;
  letter-spacing: 0.1em;
  padding-bottom: 1rem;
  border-bottom: 1px solid #1e3a5f;
  margin-bottom: 1.5rem;
}
.sbx-diagram-content {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 2rem;
}
@media (max-width: 640px) {
  .sbx-diagram-content { grid-template-columns: 1fr; }
  .sbx-flow-horizontal { flex-direction: column; align-items: center; }
  .sbx-flow-arrow-h { transform: rotate(90deg); }
  .sbx-hflow-card { min-width: auto !important; width: 100% !important; }
  .sbx-net-taps { flex-direction: column; }
}
.sbx-diagram-side {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}
.sbx-side-title {
  color: #a3a3a3;
  font-size: 0.75rem;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  padding-bottom: 0.5rem;
  border-bottom: 1px dashed #374151;
}
.sbx-side-title-good { color: #60a5fa; }
.sbx-vm-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.75rem;
}
.sbx-vm-box {
  background: #111827;
  border: 1px solid #374151;
  border-radius: 0.5rem;
  padding: 0.75rem;
  position: relative;
}
.sbx-vm-traditional {
  border-color: #525252;
  box-shadow: 0 0 10px rgba(82, 82, 82, 0.2);
}
.sbx-vm-sandbox {
  border-color: #60a5fa;
  box-shadow: 0 0 10px rgba(96, 165, 250, 0.2);
}
.sbx-vm-name {
  color: #e5e5e5;
  font-size: 0.75rem;
  font-weight: 600;
  margin-bottom: 0.5rem;
}
.sbx-vm-stats {
  display: flex;
  flex-direction: column;
  gap: 0.125rem;
  font-size: 0.625rem;
  color: #737373;
}
.sbx-disk-tiny { color: #e5e5e5 !important; }
.sbx-disk-bar {
  height: 4px;
  border-radius: 2px;
  margin-top: 0.5rem;
  background: #1f2937;
}
.sbx-disk-full {
  background: linear-gradient(90deg, #a3a3a3 0%, #d4d4d4 100%);
}
.sbx-disk-tiny-bar {
  background: linear-gradient(90deg, #60a5fa 0%, #60a5fa 5%, #1f2937 5%);
}
.sbx-disk-base {
  background: linear-gradient(90deg, #60a5fa 0%, #93c5fd 100%);
}
.sbx-connector {
  position: absolute;
  bottom: -12px;
  left: 50%;
  width: 1px;
  height: 12px;
  background: #60a5fa;
}
.sbx-base-image {
  background: linear-gradient(135deg, #0c1929 0%, #1e3a5f 100%);
  border: 2px solid #60a5fa;
  border-radius: 0.5rem;
  padding: 0.75rem;
  text-align: center;
  box-shadow: 0 0 20px rgba(96, 165, 250, 0.3);
  margin-top: 0.5rem;
}
.sbx-base-label {
  color: #60a5fa;
  font-size: 0.7rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}
.sbx-base-stats {
  color: #a3a3a3;
  font-size: 0.625rem;
  margin-top: 0.25rem;
}
.sbx-totals {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  font-size: 0.7rem;
  padding-top: 0.75rem;
  border-top: 1px dashed #374151;
}
.sbx-totals-bad { color: #e5e5e5; }
.sbx-totals-good { color: #e5e5e5; }
/* Vertical flow diagram */
.sbx-flow-vertical {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.25rem;
}
.sbx-flow-card {
  background: #111827;
  border: 1px solid #374151;
  border-radius: 0.5rem;
  padding: 0.75rem 1.25rem;
  width: 100%;
  max-width: 420px;
  text-align: center;
}
.sbx-flow-card-highlight {
  border-color: #60a5fa;
  box-shadow: 0 0 15px rgba(96, 165, 250, 0.2);
}
.sbx-flow-card-title {
  color: #60a5fa;
  font-size: 0.75rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  margin-bottom: 0.375rem;
}
.sbx-flow-card-body {
  color: #a3a3a3;
  font-size: 0.675rem;
  line-height: 1.5;
}
.sbx-flow-arrow {
  color: #60a5fa;
  font-size: 0.8rem;
  padding: 0.125rem 0;
  text-align: center;
}
/* Horizontal flow diagram */
.sbx-flow-horizontal {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.25rem;
  flex-wrap: nowrap;
  overflow-x: auto;
}
.sbx-hflow-card {
  min-width: 140px;
  width: auto;
}
.sbx-flow-arrow-h {
  color: #60a5fa;
  font-size: 1.25rem;
  flex-shrink: 0;
  padding: 0 0.125rem;
}
/* Network diagram */
.sbx-net-layout {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 1rem;
}
.sbx-net-bridge {
  width: 100%;
  text-align: center;
}
.sbx-net-bridge-label {
  color: #60a5fa;
  font-size: 0.75rem;
  font-weight: 600;
  margin-bottom: 0.5rem;
}
.sbx-net-bridge-bar {
  height: 4px;
  background: linear-gradient(90deg, #60a5fa 0%, #93c5fd 50%, #60a5fa 100%);
  border-radius: 2px;
}
.sbx-net-taps {
  display: flex;
  justify-content: center;
  gap: 1.5rem;
  width: 100%;
}
.sbx-net-tap-group {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.25rem;
}
.sbx-net-tap-line {
  width: 2px;
  height: 20px;
  background: #374151;
}
.sbx-net-tap-label {
  color: #737373;
  font-size: 0.6rem;
}
.sbx-net-vm {
  min-width: 130px;
}
.sbx-net-host {
  width: 100%;
  text-align: center;
  padding-top: 0.75rem;
  border-top: 1px dashed #374151;
}
.sbx-net-host-label {
  color: #737373;
  font-size: 0.675rem;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}
</style>
