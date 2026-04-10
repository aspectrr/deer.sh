#!/bin/bash
set -e

# Redpanda Cache Setup Script for Lima VM
# This script downloads Redpanda and provides commands to copy to Lima

echo "=== Redpanda Cache Setup for Deer.sh ==="
echo ""

# Detect architecture
ARCH=$(uname -m)
case "$ARCH" in
    arm64|aarch64)
        REDPANDA_FILE="redpanda-arm64.tar"
        REDPANDA_URL="https://vectorized-public.s3.us-west-2.amazonaws.com/releases/redpanda/25.2.7/redpanda-25.2.7-arm64.tar.gz"
        ;;
    x86_64|amd64)
        REDPANDA_FILE="redpanda-amd64.tar"
        REDPANDA_URL="https://vectorized-public.s3.us-west-2.amazonaws.com/releases/redpanda/25.2.7/redpanda-25.2.7-amd64.tar.gz"
        ;;
    *)
        echo "Unknown architecture: $ARCH"
        exit 1
        ;;
esac

# Create cache directory
CACHE_DIR="$HOME/Downloads/deer-cache"
mkdir -p "$CACHE_DIR"

echo "1. Downloading Redpanda for $ARCH..."
cd "$CACHE_DIR"

# Download
curl -L -o "${REDPANDA_FILE}.gz" "$REDPANDA_URL"

# Extract (remove .gz to get .tar)
echo "2. Extracting..."
gunzip "${REDPANDA_FILE}.gz"

echo "   Downloaded: $CACHE_DIR/$REDPANDA_FILE"
echo "   Size: $(du -h "$REDPANDA_FILE" | cut -f1)"
echo ""

# Provide Lima commands
echo "=== Copy to Lima VM ==="
echo ""
echo "Run these commands to copy to your Lima VM:"
echo ""
echo "limactl cp $CACHE_DIR/$REDPANDA_FILE default:/var/lib/deer-daemon/"
echo ""
echo "Or download directly inside Lima:"
echo "limactl shell default -- sudo mkdir -p /var/lib/deer-daemon"
echo "limactl shell default -- cd /var/lib/deer-daemon"
echo "limactl shell default -- sudo curl -L -o $REDPANDA_FILE $REDPANDA_URL.gz"
echo "limactl shell default -- sudo gunzip $REDPANDA_FILE.gz"
echo ""

# Provide config update commands
echo "=== Update Daemon Config ==="
echo ""
echo "Inside Lima VM, edit the config:"
echo "limactl shell default -- sudo nano /etc/deer-daemon/daemon.yaml"
echo ""
echo "Add or update this section:"
echo ""
cat <<EOF
microvm:
  redpanda_cache_path: "/var/lib/deer-daemon/$REDPANDA_FILE"
EOF
echo ""

# Provide restart command
echo "=== Restart Daemon ==="
echo ""
echo "After saving config, restart the daemon:"
echo "limactl shell default -- sudo systemctl restart deer-daemon"
echo ""

# Provide verification
echo "=== Verify Setup ==="
echo ""
echo "Check if daemon sees the cache:"
echo "limactl shell default -- sudo journalctl -u deer-daemon -n 50 | grep -i redpanda"
echo ""

echo "Or check config:"
echo "limactl shell default -- sudo cat /etc/deer-daemon/daemon.yaml | grep redpanda"
echo ""

echo "=== Done ==="
echo ""
echo "Redpanda cached at: $CACHE_DIR/$REDPANDA_FILE"
echo "After copying to Lima and updating config, new sandboxes with Kafka will use this cached file."
