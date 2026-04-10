# Fast Sandbox Booting

This guide explains how to optimize sandbox boot times, especially when running on macOS with QEMU TCG emulation.

## Problem: Slow Boot Times on macOS

On macOS, QEMU uses TCG (tiny code generator) for CPU emulation instead of KVM hardware virtualization. This can result in sandbox boot times of 9-10 minutes.

## Important: Deer Always Uses Source VMs

**Note:** Deer.sh is designed to clone from **existing source VMs only**, not from base QCOW2 images directly.

```bash
# Current workflow - only source VMs
deer create sandbox my-vm-prod-ubuntu my-sandbox  # Clone from running VM
```

The daemon supports `base_image` in the proto, but the CLI only exposes `--source_vm`. This is intentional - all sandbox creation flows through cloning source VMs, which is why snapshot caching exists.

If you need image-based sandbox creation, that would be a separate feature request.

## Quick Setup for Lima VM (Demo)

If you're using the demo setup with Lima, there's a helper script to set up Redpanda cache:

```bash
# From deer-daemon directory
cd deer-daemon
./setup-redpanda-cache.sh
```

This script:
1. **Downloads Redpanda** to `~/Downloads/deer-cache/` (detects ARM64 vs Intel)
2. **Extracts** to `.tar` format
3. **Shows you commands** to copy to Lima VM
4. **Shows config** to add to `/etc/deer-daemon/daemon.yaml`

Then run the commands shown by the script to copy to Lima VM and update config.

## Optimization 1: Skip Redpanda Download

If you don't need Kafka in your sandboxes, ensure you're not requesting it. Redpanda installation (download + extract + start) adds several minutes to boot time.

**Check**: Verify your sandbox creation isn't requesting Kafka data sources:
```yaml
# deer-daemon config
microvm:
  redpanda_cache_path: ""  # Leave empty unless caching
  disable_cloudinit: false
```

## Optimization 2: Cache Redpanda Locally

**Recommended**: Use the setup script for Lima VM (see "Quick Setup" above).

For sandboxes that need Redpanda (Kafka), cache the Redpanda tarball locally to avoid downloading from S3 on every boot.

The `./scripts/setup-redpanda-cache.sh` script handles this:
1. Detects your architecture (ARM64 vs Intel)
2. Downloads to `~/Downloads/deer-cache/`
3. Extracts `.tar.gz` → `.tar` (cloud-init needs `.tar`)
4. Shows commands to copy to Lima VM

### Manual Setup (If Script Fails)

If you prefer manual setup or the script fails:

```bash
# On macOS host - download appropriate version
mkdir -p ~/Downloads/deer-cache
cd ~/Downloads/deer-cache

# For ARM64 (Apple Silicon)
curl -L -o redpanda-arm64.tar.gz https://vectorized-public.s3.us-west-2.amazonaws.com/releases/redpanda/25.2.7/redpanda-25.2.7-arm64.tar.gz

# For Intel
# curl -L -o redpanda-amd64.tar.gz https://vectorized-public.s3.us-west-2.amazonaws.com/releases/redpanda/25.2.7/redpanda-25.2.7-amd64.tar.gz

# Extract (cloud-init expects .tar, not .tar.gz)
gunzip redpanda-arm64.tar.gz
# Result: redpanda-arm64.tar (3.2GB)
```

### Copy to Lima VM

```bash
# Copy tarball (extracted .tar) into Lima VM
limactl cp ~/Downloads/deer-cache/redpanda-arm64.tar default:/var/lib/deer-daemon/

# Verify it's there
limactl shell default -- ls -la /var/lib/deer-daemon/redpanda*
```

### Configure Cache Path

Inside Lima VM, edit the daemon config:

```bash
limactl shell default
sudo nano /etc/deer-daemon/daemon.yaml
```

Add the cache path (note: no `.gz` extension):

```yaml
microvm:
  redpanda_cache_path: "/var/lib/deer-daemon/redpanda-arm64.tar"
```

**Important**: Use the `.tar` file (not `.tar.gz`). The setup script already extracts it for you.

### Restart Daemon

```bash
limactl shell default -- sudo systemctl restart deer-daemon
```

### Verify It Works

```bash
# Check logs - should say "Redpanda cache configured"
limactl shell default -- sudo journalctl -u deer-daemon -n 50 | grep -i redpanda

# Check config
limactl shell default -- sudo cat /etc/deer-daemon/daemon.yaml | grep redpanda
```

## Optimization 3: Pre-Bake Base Images (Fastest)

For maximum speed, pre-bake all configuration into your base QCOW2 images. This skips cloud-init entirely.

### Prerequisites

1. Boot a sandbox normally with cloud-init
2. Make all configuration changes you want permanent
3. Ensure SSH CA keys, users, and services are installed
4. Shutdown the sandbox cleanly

### Create Pre-Baked Image

```bash
# Convert overlay to base image
cd /var/lib/deer-daemon/overlays
qemu-img convert -f qcow2 -O qcow2 <sandbox-id>/root.qcow2 /var/lib/deer-daemon/images/my-prebaked-image.qcow2
```

### Disable Cloud-Init

```yaml
# ~/.deer/daemon.yaml
microvm:
  disable_cloudinit: true  # Skip cloud-init ISO entirely
```

**Result**: Sandbox boots in ~30-60 seconds (just kernel boot + network DHCP), no cloud-init overhead.

## Optimization 4: Reduce Network Discovery Timeout

If your network is reliable, reduce the IP discovery timeout:

```yaml
# ~/.deer/daemon.yaml
microvm:
  ip_discovery_timeout: 30s  # Default 2m
  readiness_timeout: 3m       # Default 15m
```

## Optimization 5: Use Faster Kernel

If you're using a distribution kernel with initrd, try building a minimal kernel with virtio drivers built-in:

```bash
# Minimal kernel build
# virtio-blk, virtio-net, etc. as =y instead of modules
```

Then in config:
```yaml
microvm:
  initrd_path: ""  # No initramfs needed
```

## Expected Boot Times

| Configuration | Boot Time (macOS TCG) |
|--------------|------------------------|
| Default with Redpanda download | 9-10 min |
| With Redpanda cache | 6-7 min |
| Skip cloud-init | 30-60 sec |
| Production (Linux KVM) | 2-5 min |

## Verification

To check if cloud-init is disabled:

```bash
# Check QEMU command line (should have no -drive with cidata.iso)
ps aux | grep qemu-system
```

To verify base image is pre-baked:

```bash
# SSH into sandbox and check
ls /etc/ssh/deer_ca.pub  # Should exist
ls /usr/local/bin/   # Should have your custom scripts
```

## Troubleshooting

**Sandbox still takes 5+ min with disable_cloudinit: true?**
- Check base image has all required files
- Verify SSH CA key is installed in `/etc/ssh/deer_ca.pub`
- Ensure network is configured (DHCP working)

**Redpanda cache not working?**
- Verify file path is correct and readable by daemon user
- Check daemon logs for "Redpanda cache configured" message
- Ensure `file://` prefix is added automatically in code

**Still slow with cache?**
- TCG emulation is inherently slow (main bottleneck)
- Consider using Apple Virtualization Framework (VZ) for dev
- On Linux, ensure KVM is working (`-enable-kvm` in QEMU args)
