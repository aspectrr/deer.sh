#!/bin/bash
# reset-ubuntu.sh
#
# Resets the Ubuntu host to contain ONLY test-vm-{INDEX} and sandbox-host-{INDEX}.
# WARN: This will delete ALL other VMs on the system to ensure a clean state.
#
# Usage: sudo ./reset-ubuntu.sh [VM_INDEX] [--ssh-users-file <path>]
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
            echo "Usage: sudo ./reset-ubuntu.sh [VM_INDEX] [--ssh-users-file <path>]"
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

# Check for required commands
if ! command -v virsh &> /dev/null || ! command -v virt-install &> /dev/null; then
    log_error "Required commands (virsh, virt-install) not found."
    log_error "Please run setup-ubuntu.sh first to install dependencies."
    exit 1
fi

log_warn "This script will DESTROY ALL VMs on this host and recreate test-vm-${VM_INDEX} + sandbox-host-${VM_INDEX}."

# ============================================================================
# STEP 1: Ensure default network is active
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
# STEP 2: Destroy and Undefine ALL VMs
# ============================================================================
log_info "Cleaning up ALL existing VMs..."

# Get list of all VMs (running and shut off)
VMS=$(virsh list --all --name 2>/dev/null || true)

for VM in $VMS; do
    if [[ -n "$VM" ]]; then
        log_info "Removing VM: $VM"
        # Destroy (stop) if running
        virsh destroy "$VM" > /dev/null 2>&1 || true
        # Undefine and remove NVRAM if applicable
        virsh undefine "$VM" --nvram > /dev/null 2>&1 || virsh undefine "$VM" > /dev/null 2>&1 || true
    fi
done

# Clean up old cloud-init directories
log_info "Cleaning up old cloud-init data..."
rm -rf /var/lib/libvirt/images/cloud-init/* 2>/dev/null || true

# Clean up old VM disks (except base images)
log_info "Cleaning up old VM disks..."
rm -f /var/lib/libvirt/images/test-vm-*.qcow2 2>/dev/null || true
rm -f /var/lib/libvirt/images/sandbox-host-*.qcow2 2>/dev/null || true
rm -f /var/lib/libvirt/images/sbx-*.qcow2 2>/dev/null || true

# Clean up sandbox work directories
rm -rf /var/lib/libvirt/images/sandboxes/* 2>/dev/null || true

log_success "Cleanup complete."

# Flush stale DHCP leases by bouncing the default network
# Prevents multiple IPs when reusing deterministic MAC addresses
log_info "Flushing DHCP leases by restarting default network..."
virsh net-destroy default > /dev/null 2>&1 || true
# Delete on-disk lease files so dnsmasq doesn't reload stale leases
rm -f /var/lib/libvirt/dnsmasq/virbr0.status /var/lib/libvirt/dnsmasq/virbr0.leases 2>/dev/null || true
rm -f /var/lib/libvirt/dnsmasq/default.leases 2>/dev/null || true
virsh net-start default > /dev/null 2>&1 || true
log_success "DHCP leases flushed."

# ============================================================================
# STEP 3: Create Test VMs (Ubuntu 22.04 Cloud Image)
# ============================================================================
log_info "Creating fresh test VMs..."

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

    local max_wait=180
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
        log_warn "Troubleshooting steps:"
        log_warn "  1. Check VM is running: virsh list --all"
        log_warn "  2. Check network interface: virsh domiflist ${vm_name}"
        log_warn "  3. Check DHCP leases: virsh net-dhcp-leases default"
        log_warn "  4. Access VM console: virsh console ${vm_name} (login: ubuntu/ubuntu)"
        log_warn "  5. Inside VM, check: ip addr show; cloud-init status"
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
# STEP 4: Final Summary
# ============================================================================
echo ""
echo "============================================================================"
log_success "Host reset complete!"
echo "============================================================================"
echo ""
echo "Reset Summary:"
echo "  - All previous VMs destroyed and undefined"
echo "  - Cloud-init data cleaned up"
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
echo ""
echo "Useful commands:"
echo "  virsh list --all                          # List all VMs"
for i in "${!CREATED_VM_NAMES[@]}"; do
    echo "  virsh domifaddr ${CREATED_VM_NAMES[$i]} --source lease # Get ${CREATED_VM_NAMES[$i]} IP"
done
echo ""

# Verify the VMs are in a good state for cloning
log_info "Validating VMs are ready for use as sandbox sources..."

ALL_READY=true
for i in "${!CREATED_VM_NAMES[@]}"; do
    if [[ -n "${CREATED_VM_IPS[$i]}" ]] && [[ -n "${CREATED_VM_MACS[$i]}" ]]; then
        log_success "VM '${CREATED_VM_NAMES[$i]}' is ready for use as a sandbox source!"
    else
        log_warn "VM '${CREATED_VM_NAMES[$i]}' may not be fully ready. Please verify:"
        log_warn "  - VM has IP: virsh domifaddr ${CREATED_VM_NAMES[$i]} --source lease"
        log_warn "  - VM has MAC: virsh domiflist ${CREATED_VM_NAMES[$i]}"
        ALL_READY=false
    fi
done

if [[ "$ALL_READY" == true ]]; then
    log_success "All VMs are ready!"
fi
