#!/bin/bash
# setup-ubuntu.sh
#
# Sets up libvirt and KVM on Ubuntu (x86 architecture).
# This script installs necessary packages, configures groups, and validates the installation.
#
# Usage: sudo ./setup-ubuntu.sh [VM_INDEX] [--ssh-users-file <path>]
#
# Options:
#   VM_INDEX                  VM index number (default: 1)
#   --ssh-users-file <path>   Path to file with SSH users (one per line: <username> <public-key>)

VM_INDEX=""
SSH_USERS_FILE=""

# Parse arguments: first positional arg is VM_INDEX, rest are named flags
while [[ $# -gt 0 ]]; do
    case "$1" in
        --ssh-users-file)
            SSH_USERS_FILE="$2"
            shift 2
            ;;
        --help|-h)
            echo "Usage: sudo ./setup-ubuntu.sh [VM_INDEX] [--ssh-users-file <path>]"
            echo ""
            echo "Options:"
            echo "  VM_INDEX                  VM index number (default: 1)"
            echo "  --ssh-users-file <path>   Path to file with SSH users (one per line: <username> <public-key>)"
            exit 0
            ;;
        *)
            if [[ -z "$VM_INDEX" ]]; then
                VM_INDEX="$1"
            else
                echo "Unknown argument: $1" >&2
                exit 1
            fi
            shift
            ;;
    esac
done

VM_INDEX="${VM_INDEX:-1}"

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

# Check if running as root
if [[ $EUID -ne 0 ]]; then
    log_error "This script must be run as root"
    exit 1
fi

log_info "Starting libvirt setup for Ubuntu..."

# ============================================================================
# STEP 1: Check for Virtualization Support
# ============================================================================
log_info "Checking for hardware virtualization support..."

if egrep -c '(vmx|svm)' /proc/cpuinfo > /dev/null; then
    log_success "Hardware virtualization support detected."
else
    log_warn "Hardware virtualization (VT-x/AMD-V) NOT detected in /proc/cpuinfo."
    log_warn "If you are running this inside a VM, ensure nested virtualization is enabled."
    log_warn "Proceeding, but KVM might not work..."
    # We don't exit here because sometimes it might be emulated or the check is flaky in some containers
fi

if command -v kvm-ok &>/dev/null; then
    if kvm-ok; then
        log_success "KVM acceleration can be used."
    else
        log_warn "KVM acceleration CANNOT be used."
    fi
else
    log_info "'kvm-ok' command not found. Installing cpu-checker to check KVM status..."
    apt-get update -qq
    apt-get install -y -qq cpu-checker
    if kvm-ok; then
        log_success "KVM acceleration can be used."
    else
        log_warn "KVM acceleration CANNOT be used."
    fi
fi

# ============================================================================
# STEP 2: Install Libvirt and QEMU Packages
# ============================================================================
log_info "Installing libvirt, qemu-kvm, and utilities..."

export DEBIAN_FRONTEND=noninteractive
apt-get update -qq
apt-get install -y -qq \
    wget \
    cloud-image-utils \
    qemu-kvm \
    libvirt-daemon-system \
    libvirt-clients \
    bridge-utils \
    virtinst \
    virt-manager \
    genisoimage

log_success "Packages installed successfully."

# ============================================================================
# STEP 3: Enable and Start libvirtd Service
# ============================================================================
log_info "Enabling and starting libvirtd service..."

systemctl enable --now libvirtd
systemctl start libvirtd

if systemctl is-active --quiet libvirtd; then
    log_success "libvirtd is running."
else
    log_error "libvirtd failed to start. Check 'systemctl status libvirtd'."
    exit 1
fi

# ============================================================================
# STEP 4: Ensure default network is active
# ============================================================================
log_info "Ensuring default network is active..."

if ! virsh net-info default &>/dev/null; then
    log_info "Default network not found, creating it..."
    virsh net-define /usr/share/libvirt/networks/default.xml || true
fi

if ! virsh net-list | grep -q "default.*active"; then
    log_info "Starting default network..."
    virsh net-start default || true
    virsh net-autostart default || true
fi

log_success "Default network is active."

# Verify DHCP is configured
if ! virsh net-dumpxml default | grep -q "<dhcp>"; then
    log_warn "Default network does not have DHCP configured!"
    log_warn "VMs may not get IP addresses automatically."
fi

# ============================================================================
# STEP 5: Configure User Groups
# ============================================================================
log_info "Configuring user groups..."

# Get the user who invoked sudo (if applicable), otherwise assume root or current user
REAL_USER=${SUDO_USER:-$USER}

if [[ -n "$REAL_USER" ]]; then
    log_info "Adding user '$REAL_USER' to 'libvirt' and 'kvm' groups..."
    usermod -aG libvirt "$REAL_USER"
    usermod -aG kvm "$REAL_USER"
    log_success "User '$REAL_USER' added to groups."
else
    log_warn "Could not determine real user. Skipping group addition."
fi

# ============================================================================
# STEP 6: Verify Installation
# ============================================================================
log_info "Verifying installation..."

if virsh list --all > /dev/null; then
    log_success "virsh command working correctly."
else
    log_error "virsh command failed. Check permissions or libvirtd status."
    exit 1
fi

# ============================================================================
# STEP 7: Create Test VMs (Ubuntu 22.04 Cloud Image)
# ============================================================================
log_info "Creating test VMs..."

IMAGE_DIR="/var/lib/libvirt/images"
CLOUD_INIT_DIR="${IMAGE_DIR}/cloud-init"
BASE_IMAGE="ubuntu-22.04-minimal-cloudimg-amd64.img"
BASE_IMAGE_URL="https://cloud-images.ubuntu.com/minimal/releases/jammy/release/${BASE_IMAGE}"
BASE_IMAGE_PATH="${IMAGE_DIR}/${BASE_IMAGE}"

# Ensure directories exist
mkdir -p "$IMAGE_DIR"
mkdir -p "$CLOUD_INIT_DIR"

# 1. Download Base Image if missing
if [[ ! -f "$BASE_IMAGE_PATH" ]]; then
    log_info "Downloading Ubuntu Minimal Cloud Image (approx 300MB)..."
    if wget -q --show-progress -O "$BASE_IMAGE_PATH" "$BASE_IMAGE_URL"; then
        log_success "Image downloaded."
    else
        log_error "Failed to download image from $BASE_IMAGE_URL"
        exit 1
    fi
else
    log_info "Base image already exists at $BASE_IMAGE_PATH"
fi

# Add SSH public keys to KVM host for proxy jump access (once, not per VM)
if [[ -n "$SSH_USERS_FILE" ]] && [[ -f "$SSH_USERS_FILE" ]]; then
    log_info "Adding SSH public keys to KVM host authorized_keys..."
    HOST_SSH_DIR="/root/.ssh"
    mkdir -p "$HOST_SSH_DIR"
    chmod 700 "$HOST_SSH_DIR"
    touch "$HOST_SSH_DIR/authorized_keys"
    chmod 600 "$HOST_SSH_DIR/authorized_keys"
    while IFS= read -r line || [[ -n "$line" ]]; do
        [[ -z "$line" ]] && continue
        [[ "$line" =~ ^#.*$ ]] && continue
        pubkey="${line#* }"
        if ! grep -qF "$pubkey" "$HOST_SSH_DIR/authorized_keys"; then
            echo "$pubkey" >> "$HOST_SSH_DIR/authorized_keys"
            username="${line%% *}"
            log_success "Added key for ${username} to host authorized_keys"
        fi
    done < "$SSH_USERS_FILE"
fi

# Arrays to track created VMs for summary
CREATED_VM_NAMES=()
CREATED_VM_IPS=()
CREATED_VM_MACS=()
CREATED_VM_DISKS=()

# ============================================================================
# create_vm: Create a single VM with cloud-init, wait for IP, verify network
#
# Arguments:
#   $1 - vm_name (e.g. "test-vm-1")
#   $2 - vm_index (numeric, for MAC generation)
#   $3 - mac_prefix (e.g. "52:54:00" or "52:54:01")
# ============================================================================
create_vm() {
    local vm_name="$1"
    local vm_index="$2"
    local mac_prefix="$3"

    log_info "Creating VM '${vm_name}'..."

    # Create Disk (Copy-on-Write)
    local vm_disk="${IMAGE_DIR}/${vm_name}.qcow2"
    log_info "Creating VM disk: $vm_disk"
    if [[ -f "$vm_disk" ]]; then
        log_warn "Disk $vm_disk already exists, overwriting..."
        rm -f "$vm_disk"
    fi
    qemu-img create -f qcow2 -F qcow2 -b "$BASE_IMAGE_PATH" "$vm_disk" 10G

    # Create Cloud-Init Config
    local seed_dir="${CLOUD_INIT_DIR}/${vm_name}"
    mkdir -p "$seed_dir"

    local user_data="${seed_dir}/user-data"
    local meta_data="${seed_dir}/meta-data"
    local network_config="${seed_dir}/network-config"
    local instance_id="${vm_name}-$(date +%s)"

    log_info "Creating cloud-init configuration for '${vm_name}'..."

    # User-data: password, SSH, guest agent
    cat > "$user_data" <<EOF
#cloud-config
password: ubuntu
chpasswd: { expire: False }
ssh_pwauth: True
EOF

    # Add SSH users from file if provided
    if [[ -n "$SSH_USERS_FILE" ]] && [[ -f "$SSH_USERS_FILE" ]]; then
        echo "" >> "$user_data"
        echo "users:" >> "$user_data"
        echo "  - default" >> "$user_data"
        while IFS= read -r line || [[ -n "$line" ]]; do
            [[ -z "$line" ]] && continue
            [[ "$line" =~ ^#.*$ ]] && continue
            local username="${line%% *}"
            local pubkey="${line#* }"
            cat >> "$user_data" <<EOF
  - name: ${username}
    sudo: ALL=(ALL) NOPASSWD:ALL
    shell: /bin/bash
    ssh_authorized_keys:
      - ${pubkey}
EOF
        done < "$SSH_USERS_FILE"
    fi

    cat >> "$user_data" <<EOF

# Install and enable guest agent for better VM management
packages:
  - qemu-guest-agent

# Enable guest agent on boot
runcmd:
  - systemctl enable qemu-guest-agent
  - systemctl start qemu-guest-agent
EOF

    # Meta-data: unique instance-id is CRITICAL for cloud-init to run on clones
    cat > "$meta_data" <<EOF
instance-id: ${instance_id}
local-hostname: ${vm_name}
EOF

    # Network-config (NoCloud v2 format)
    cat > "$network_config" <<EOF
version: 2
ethernets:
  id0:
    match:
      name: en*
    dhcp4: true
EOF

    log_success "Cloud-init config files created for '${vm_name}'."

    # Generate a deterministic MAC address based on VM index
    local mac_suffix
    mac_suffix=$(printf '%02x:%02x:%02x' $((vm_index / 256 / 256 % 256)) $((vm_index / 256 % 256)) $((vm_index % 256)))
    local mac_address="${mac_prefix}:${mac_suffix}"

    log_info "Using MAC address: ${mac_address}"

    # Boot VM with virt-install
    log_info "Booting VM '${vm_name}' with virt-install --cloud-init..."

    virt-install \
        --name "${vm_name}" \
        --memory 2048 \
        --vcpus 2 \
        --disk "${vm_disk},device=disk,bus=virtio" \
        --cloud-init user-data="${user_data}",meta-data="${meta_data}",network-config="${network_config}" \
        --os-variant ubuntu22.04 \
        --import \
        --noautoconsole \
        --graphics none \
        --console pty,target_type=serial \
        --network network=default,model=virtio,mac="${mac_address}"

    log_success "VM '${vm_name}' started!"

    # Wait for VM to get IP address
    log_info "Waiting for '${vm_name}' to obtain IP address (this may take 30-60 seconds)..."

    local max_wait=120
    local wait_interval=5
    local elapsed=0
    local vm_ip=""

    while [[ $elapsed -lt $max_wait ]]; do
        vm_ip=$(virsh domifaddr "${vm_name}" --source lease 2>/dev/null | grep -oE '([0-9]{1,3}\.){3}[0-9]{1,3}' | head -1 || true)

        if [[ -n "$vm_ip" ]]; then
            log_success "VM '${vm_name}' obtained IP address: ${vm_ip}"
            break
        fi

        log_info "Waiting for '${vm_name}' IP... (${elapsed}s / ${max_wait}s)"
        sleep $wait_interval
        elapsed=$((elapsed + wait_interval))
    done

    if [[ -z "$vm_ip" ]]; then
        log_warn "VM '${vm_name}' did not obtain IP address within ${max_wait} seconds."
        log_warn "Check: virsh domifaddr ${vm_name} --source lease"
        log_warn "Check: virsh console ${vm_name} (login: ubuntu/ubuntu)"
    fi

    # Verify VM network interface
    log_info "Verifying '${vm_name}' network configuration..."

    local vm_mac
    vm_mac=$(virsh domiflist "${vm_name}" 2>/dev/null | grep -oE '([0-9a-f]{2}:){5}[0-9a-f]{2}' | head -1 || true)
    if [[ -n "$vm_mac" ]]; then
        log_success "VM '${vm_name}' MAC address: ${vm_mac}"
    else
        log_warn "Could not determine MAC address for '${vm_name}'"
    fi

    local iface
    iface=$(virsh domiflist "${vm_name}" 2>/dev/null | awk 'NR>2 && $1 != "" {print $1}' | head -1 || true)
    if [[ -n "$iface" ]]; then
        log_info "Network interface: ${iface}"
        virsh domifstat "${vm_name}" "${iface}" 2>/dev/null || true
    fi

    # Track results for summary
    CREATED_VM_NAMES+=("$vm_name")
    CREATED_VM_IPS+=("$vm_ip")
    CREATED_VM_MACS+=("$vm_mac")
    CREATED_VM_DISKS+=("$vm_disk")
}

# Create both VMs
create_vm "test-vm-${VM_INDEX}" "$VM_INDEX" "52:54:00"
create_vm "sandbox-host-${VM_INDEX}" "$VM_INDEX" "52:54:01"

# ============================================================================
# STEP 8: Final Summary
# ============================================================================
echo ""
echo "============================================================================"
log_success "Libvirt setup complete!"
echo "============================================================================"
echo ""
echo "Setup Summary:"
echo "  - Installed: qemu-kvm, libvirt-daemon-system, libvirt-clients, bridge-utils, virtinst"
echo "  - Service: libvirtd enabled and started"
echo "  - Network: default network active with DHCP"
echo ""
for i in "${!CREATED_VM_NAMES[@]}"; do
    echo "  VM: '${CREATED_VM_NAMES[$i]}'"
    echo "    - Disk: ${CREATED_VM_DISKS[$i]}"
    if [[ -n "${CREATED_VM_MACS[$i]}" ]]; then
        echo "    - MAC Address: ${CREATED_VM_MACS[$i]}"
    fi
    if [[ -n "${CREATED_VM_IPS[$i]}" ]]; then
        echo "    - IP Address: ${CREATED_VM_IPS[$i]}"
    else
        echo "    - IP Address: (pending - check with 'virsh domifaddr ${CREATED_VM_NAMES[$i]} --source lease')"
    fi
done
echo ""
echo "  - Cloud-Init: virt-install --cloud-init (native injection)"
echo "  - Login: ubuntu / ubuntu (password)"
if [[ -n "$SSH_USERS_FILE" ]] && [[ -f "$SSH_USERS_FILE" ]]; then
    echo "  - SSH Users:"
    while IFS= read -r line || [[ -n "$line" ]]; do
        [[ -z "$line" ]] && continue
        [[ "$line" =~ ^#.*$ ]] && continue
        username="${line%% *}"
        echo "      ${username} (key-based auth)"
    done < "$SSH_USERS_FILE"
fi
if [[ -n "$REAL_USER" ]]; then
    echo "  - User: '$REAL_USER' added to 'libvirt' and 'kvm' groups"
    echo "    NOTE: You may need to log out and log back in for group changes to take effect."
fi
echo ""
echo "Useful commands:"
echo "  virsh list --all                          # List all VMs"
for i in "${!CREATED_VM_NAMES[@]}"; do
    echo "  virsh domifaddr ${CREATED_VM_NAMES[$i]} --source lease # Get ${CREATED_VM_NAMES[$i]} IP"
done
echo ""
