#!/bin/bash
set -e

# Create system user if it doesn't exist
if ! id fluid-daemon >/dev/null 2>&1; then
    useradd --system --home /var/lib/fluid-daemon --shell /usr/sbin/nologin fluid-daemon
fi

# Create required directories
mkdir -p /etc/fluid-daemon /var/lib/fluid-daemon/images /var/lib/fluid-daemon/overlays /var/lib/fluid-daemon/keys /var/log/fluid-daemon

# Set ownership
chown -R fluid-daemon:fluid-daemon /var/lib/fluid-daemon /var/log/fluid-daemon

# Add to libvirt group if available
if getent group libvirt >/dev/null 2>&1; then
    usermod -aG libvirt fluid-daemon
fi

# Generate SSH CA keypair if missing (used to sign ephemeral sandbox certs)
if [ ! -f /etc/fluid-daemon/ssh_ca ]; then
    ssh-keygen -t ed25519 -f /etc/fluid-daemon/ssh_ca -N "" -C "fluid-daemon CA"
    chown fluid-daemon:fluid-daemon /etc/fluid-daemon/ssh_ca /etc/fluid-daemon/ssh_ca.pub
fi

# Generate identity keypair if missing (used for SSH to source VM hosts)
if [ ! -f /etc/fluid-daemon/identity ]; then
    ssh-keygen -t ed25519 -f /etc/fluid-daemon/identity -N "" -C "fluid-daemon"
    chown fluid-daemon:fluid-daemon /etc/fluid-daemon/identity /etc/fluid-daemon/identity.pub
fi

# Reload systemd
systemctl daemon-reload

# Enable on fresh install (deb: "configure", rpm: "1")
if [ "$1" = "configure" ] || [ "$1" = "1" ]; then
    systemctl enable fluid-daemon
fi
