#!/bin/bash
set -e

# Create system user if it doesn't exist
if ! id deer-daemon >/dev/null 2>&1; then
    useradd --system --home /var/lib/deer-daemon --shell /usr/sbin/nologin deer-daemon
fi

# Create required directories
mkdir -p /etc/deer-daemon /var/lib/deer-daemon/images /var/lib/deer-daemon/overlays /var/lib/deer-daemon/keys /var/log/deer-daemon

# Set ownership
chown -R deer-daemon:deer-daemon /var/lib/deer-daemon /var/log/deer-daemon

# Add to libvirt group if available
if getent group libvirt >/dev/null 2>&1; then
    usermod -aG libvirt deer-daemon
fi

# Generate SSH CA keypair if missing (used to sign ephemeral sandbox certs)
if [ ! -f /etc/deer-daemon/ssh_ca ]; then
    ssh-keygen -t ed25519 -f /etc/deer-daemon/ssh_ca -N "" -C "deer-daemon CA"
    chown deer-daemon:deer-daemon /etc/deer-daemon/ssh_ca /etc/deer-daemon/ssh_ca.pub
fi

# Generate identity keypair if missing (used for SSH to source VM hosts)
if [ ! -f /etc/deer-daemon/identity ]; then
    ssh-keygen -t ed25519 -f /etc/deer-daemon/identity -N "" -C "deer-daemon"
    chown deer-daemon:deer-daemon /etc/deer-daemon/identity /etc/deer-daemon/identity.pub
fi

# Reload systemd
systemctl daemon-reload

# Enable on fresh install (deb: "configure", rpm: "1")
if [ "$1" = "configure" ] || [ "$1" = "1" ]; then
    systemctl enable deer-daemon
fi
