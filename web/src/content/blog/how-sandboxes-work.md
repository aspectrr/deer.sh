---
title: 'How Fluid.sh Sandboxes Work'
pubDate: 2026-01-24
description: 'Have you ever wondered how AI Sandboxes work?'
author: 'Collin @ Fluid.sh'
authorImage: '/images/skeleton_smoking_cigarette.jpg'
authorEmail: 'cpfeifer@madcactus.org'
authorPhone: '+3179955114'
authorDiscord: 'https://discordapp.com/users/301068417685913600'
---

## Intro

When you ask Fluid to spin up a new sandbox, you aren't waiting for a full OS installation. Instead, we use a Linked Clone mechanism that provisions a
fresh, isolated environment in milliseconds. Here is a deep dive into how it works, why it's safe, and what's next.

<div class="diagram-container">
  <div class="diagram-header">Traditional VMs vs Fluid Sandboxes</div>
  <div class="diagram-content">
    <!-- Traditional VMs Side -->
    <div class="diagram-side">
      <div class="side-title">TRADITIONAL: 4 Full VM Clones</div>
      <div class="vm-grid">
        <div class="vm-box vm-traditional">
          <div class="vm-name">VM-1</div>
          <div class="vm-stats">
            <span>CPU: 2 cores</span>
            <span>RAM: 4 GB</span>
            <span>Disk: 20 GB</span>
          </div>
          <div class="disk-bar disk-full"></div>
        </div>
        <div class="vm-box vm-traditional">
          <div class="vm-name">VM-2</div>
          <div class="vm-stats">
            <span>CPU: 2 cores</span>
            <span>RAM: 4 GB</span>
            <span>Disk: 20 GB</span>
          </div>
          <div class="disk-bar disk-full"></div>
        </div>
        <div class="vm-box vm-traditional">
          <div class="vm-name">VM-3</div>
          <div class="vm-stats">
            <span>CPU: 2 cores</span>
            <span>RAM: 4 GB</span>
            <span>Disk: 20 GB</span>
          </div>
          <div class="disk-bar disk-full"></div>
        </div>
        <div class="vm-box vm-traditional">
          <div class="vm-name">VM-4</div>
          <div class="vm-stats">
            <span>CPU: 2 cores</span>
            <span>RAM: 4 GB</span>
            <span>Disk: 20 GB</span>
          </div>
          <div class="disk-bar disk-full"></div>
        </div>
      </div>
      <div class="totals totals-bad">
        <span>TOTAL DISK: 80 GB</span>
        <span>Creation: ~2-5 min each</span>
      </div>
    </div>
    <!-- Fluid CoW Side -->
    <div class="diagram-side">
      <div class="side-title side-title-good">FLUID: Copy-on-Write Sandboxes</div>
      <div class="vm-grid">
        <div class="vm-box vm-sandbox">
          <div class="vm-name">SBX-1</div>
          <div class="vm-stats">
            <span>CPU: 2 cores</span>
            <span>RAM: 4 GB</span>
            <span class="disk-tiny">Disk: 128 KB</span>
          </div>
          <div class="disk-bar disk-tiny-bar"></div>
          <div class="connector"></div>
        </div>
        <div class="vm-box vm-sandbox">
          <div class="vm-name">SBX-2</div>
          <div class="vm-stats">
            <span>CPU: 2 cores</span>
            <span>RAM: 4 GB</span>
            <span class="disk-tiny">Disk: 256 KB</span>
          </div>
          <div class="disk-bar disk-tiny-bar"></div>
          <div class="connector"></div>
        </div>
        <div class="vm-box vm-sandbox">
          <div class="vm-name">SBX-3</div>
          <div class="vm-stats">
            <span>CPU: 2 cores</span>
            <span>RAM: 4 GB</span>
            <span class="disk-tiny">Disk: 64 KB</span>
          </div>
          <div class="disk-bar disk-tiny-bar"></div>
          <div class="connector"></div>
        </div>
        <div class="vm-box vm-sandbox">
          <div class="vm-name">SBX-4</div>
          <div class="vm-stats">
            <span>CPU: 2 cores</span>
            <span>RAM: 4 GB</span>
            <span class="disk-tiny">Disk: 512 KB</span>
          </div>
          <div class="disk-bar disk-tiny-bar"></div>
          <div class="connector"></div>
        </div>
      </div>
      <div class="base-image">
        <div class="base-label">BASE IMAGE (Read-Only)</div>
        <div class="base-stats">Disk: 20 GB</div>
        <div class="disk-bar disk-full disk-base"></div>
      </div>
      <div class="totals totals-good">
        <span>TOTAL DISK: ~20 GB</span>
        <span>Creation: ~50ms each</span>
      </div>
    </div>
  </div>
</div>

<style>
.diagram-container {
  background: linear-gradient(135deg, #0a0a0a 0%, #0c1929 100%);
  border: 1px solid #1e3a5f;
  border-radius: 0.75rem;
  padding: 1.5rem;
  margin: 2rem 0;
  font-family: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace;
  box-shadow: 0 0 30px rgba(96, 165, 250, 0.1);
}
.diagram-header {
  text-align: center;
  color: #60a5fa;
  font-size: 0.875rem;
  font-weight: 600;
  letter-spacing: 0.1em;
  padding-bottom: 1rem;
  border-bottom: 1px solid #1e3a5f;
  margin-bottom: 1.5rem;
}
.diagram-content {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 2rem;
}
@media (max-width: 640px) {
  .diagram-content {
    grid-template-columns: 1fr;
  }
}
.diagram-side {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}
.side-title {
  color: #a3a3a3;
  font-size: 0.75rem;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  padding-bottom: 0.5rem;
  border-bottom: 1px dashed #374151;
}
.side-title-good {
  color: #60a5fa;
}
.vm-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.75rem;
}
.vm-box {
  background: #111827;
  border: 1px solid #374151;
  border-radius: 0.5rem;
  padding: 0.75rem;
  position: relative;
}
.vm-traditional {
  border-color: #525252;
  box-shadow: 0 0 10px rgba(82, 82, 82, 0.2);
}
.vm-sandbox {
  border-color: #60a5fa;
  box-shadow: 0 0 10px rgba(96, 165, 250, 0.2);
}
.vm-name {
  color: #e5e5e5;
  font-size: 0.75rem;
  font-weight: 600;
  margin-bottom: 0.5rem;
}
.vm-stats {
  display: flex;
  flex-direction: column;
  gap: 0.125rem;
  font-size: 0.625rem;
  color: #737373;
}
.disk-tiny {
  color: #e5e5e5 !important;
}
.disk-bar {
  height: 4px;
  border-radius: 2px;
  margin-top: 0.5rem;
  background: #1f2937;
}
.disk-full {
  background: linear-gradient(90deg, #a3a3a3 0%, #d4d4d4 100%);
}
.disk-tiny-bar {
  background: linear-gradient(90deg, #60a5fa 0%, #60a5fa 5%, #1f2937 5%);
}
.disk-base {
  background: linear-gradient(90deg, #60a5fa 0%, #93c5fd 100%);
}
.connector {
  position: absolute;
  bottom: -12px;
  left: 50%;
  width: 1px;
  height: 12px;
  background: #60a5fa;
}
.base-image {
  background: linear-gradient(135deg, #0c1929 0%, #1e3a5f 100%);
  border: 2px solid #60a5fa;
  border-radius: 0.5rem;
  padding: 0.75rem;
  text-align: center;
  box-shadow: 0 0 20px rgba(96, 165, 250, 0.3);
  margin-top: 0.5rem;
}
.base-label {
  color: #60a5fa;
  font-size: 0.7rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}
.base-stats {
  color: #a3a3a3;
  font-size: 0.625rem;
  margin-top: 0.25rem;
}
.totals {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  font-size: 0.7rem;
  padding-top: 0.75rem;
  border-top: 1px dashed #374151;
}
.totals-bad {
  color: #e5e5e5;
}
.totals-good {
  color: #e5e5e5;
}
</style>

## The Mechanism: Linked Clones & Overlays

At the heart of Fluid's cloning engine is the Copy-On-Write (COW) strategy.

1.  The Golden Image: We start with a "base" or "golden" image (e.g., a standard Ubuntu cloud image). This file remains read-only and untouched.
2.  The Overlay (QCOW2): When you request a new environment, we don't copy that massive base image. Instead, we create a tiny "overlay" file using
    qemu-img:
    `qemu-img create -f qcow2 -F qcow2 -b /path/to/base.img /path/to/overlay.qcow2`
    This overlay records only the changes made by the new VM. It starts at a few kilobytes, making creation near-instantaneous and incredibly
    storage-efficient.

## The "Identity Crisis": Making Clones Unique

A raw clone of a disk is dangerous—it has the same SSH keys, the same static IP configs, and the same system identity as the parent. Fluid solves this
using a two-step "Identity Reset" during the clone process.

1. Libvirt XML Mutation
   Before defining the new VM in Libvirt, we parse the base VM's XML configuration and aggressively sanitize it:

- UUID Removal: We strip the old UUID so Libvirt assigns a brand new, unique identifier.
- MAC Address Regeneration: We generate a fresh, random MAC address (using the 52:54:00 prefix) to ensure the network stack sees a new device.
- Disk Swapping: We point the VM's primary drive to our new, empty overlay file instead of the base image.

2. The Cloud-Init "Amnesia" Trick
   This is the most critical safety feature. Linux distributions running cloud-init will typically run setup once and then mark themselves as "done." To
   force the clone to re-identify itself, we generate a custom cloud-init.iso for every single clone containing a new instance-id:

```
 # meta-data
 instance-id: <new-vm-name>
 local-hostname: <new-vm-name>
```

When the clone boots, it sees a new instance-id via the attached ISO. This signals cloud-init to run again, triggering:

- Fresh DHCP negotiation (getting a new IP for the new MAC).
- Regeneration of SSH host keys (if configured).
- User creation and SSH key injection.

This ensures that even though the disk is a clone, the OS thinks it's booting for the first time.

## Why It's Safe

- Isolation: The base image is locked. Corruption in one sandbox cannot affect others or the base.
- Network Safety: Unique MACs and forced DHCP renewal prevent IP conflicts on the bridge.
- Ephemeral Nature: Because the state lives in a disposable overlay, "wiping" a machine is as simple (and fast) as deleting a small file.

## Pre-flight Resource Checks

If a Libvirt host is running low on RAM or Disk space, the clone operation might succeed (because the overlay is small), but the VM will
fail to boot or crash later when it tries to write data.

Before creating a clone, Fluid querys the host's stats:

1.  RAM Check: Use `virsh nodeinfo` to calculate available memory vs. the requested VM size.
2.  Disk Space Projection: While overlays start small, they can grow to the virtual size of the base image. Fluid makes sure there is a 20% buffer before cloning.
    - Safety Policy: Ensure the host has enough headroom (e.g., at least 10-20% free buffer) to accommodate the potential growth of active overlays,
      or implement strict disk quotas (using `virtio-blk` quotas) to prevent one runaway log file from filling the host disk.

## The "Janitor": Sandbox Cleanup

To prevent idling sandboxes that are done being used but never got destroyed, there exists a Janitor process that checks the variables `DEFAULT_TTL` and if a sandbox exists older then that it gets removed. This is how the Fluid Remote server works.
The Fluid Terminal Agent on the otherhand keeps track of created sandboxes and if they are older than 24 hours or if the terminal agent is getting closed, they will then get removed, whichever happens first.

## What's Next

Now the thing is, I know this setup isn't perfect. It has its flaws, who wants to run additional VMs on your libvirt hosts? It's not ideal but it got me close enough to production that I didn't mind for now. The next step is QEMU microVMs - lightweight VMs that bring sandboxes onto the same network level as the host, with fast startup, copy-on-write overlays, and full isolation so sandboxes never touch your infrastructure. That work is already underway.
