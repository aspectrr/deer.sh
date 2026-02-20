#!/bin/bash
# reset-proxmox.sh
#
# Resets a Proxmox host by destroying all source VMs/CTs and sandboxes,
# then recreating the source LXC container and QEMU VM.
# WARN: This will destroy containers in VMID 100-199 (sources), 200-299 (QEMU sources),
# and 9000-9999 (sandboxes).
#
# Usage: sudo ./reset-proxmox.sh [VM_INDEX] [--ssh-users-file <path>]
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
            echo "Usage: sudo ./reset-proxmox.sh [VM_INDEX] [--ssh-users-file <path>]"
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
if ! command -v pvesh &>/dev/null || ! systemctl is-active --quiet pve-cluster 2>/dev/null; then
    log_error "Proxmox VE is not installed or pve-cluster is not running."
    log_error "Please run setup-proxmox.sh first."
    exit 1
fi

# VMID scheme
LXC_VMID=$((100 + VM_INDEX))
QEMU_VMID=$((200 + VM_INDEX))
LXC_NAME="test-vm-${VM_INDEX}"
QEMU_NAME="test-vm-qemu-${VM_INDEX}"
PVE_NODE=$(hostname)

log_warn "This script will DESTROY source CTs (100-199), source VMs (200-299), and sandboxes (9000-9999)."
log_warn "Then recreate ${LXC_NAME} and ${QEMU_NAME}."

# ============================================================================
# STEP 1: Destroy LXC containers in source range (100-199)
# ============================================================================
log_info "Destroying LXC source containers (VMID 100-199)..."

for vmid in $(pct list 2>/dev/null | awk 'NR>1{print $1}'); do
    if [[ "$vmid" -ge 100 ]] && [[ "$vmid" -le 199 ]]; then
        local_name=$(pct list 2>/dev/null | awk -v id="$vmid" '$1==id{print $3}')
        log_info "Destroying LXC container ${vmid} (${local_name:-unknown})..."
        pct stop "$vmid" --force 2>/dev/null || true
        sleep 1
        pct destroy "$vmid" --force --purge 2>/dev/null || true
        log_success "Destroyed container ${vmid}."
    fi
done

# ============================================================================
# STEP 2: Destroy sandbox LXC containers (9000-9999)
# ============================================================================
log_info "Destroying sandbox containers (VMID 9000-9999)..."

for vmid in $(pct list 2>/dev/null | awk 'NR>1{print $1}'); do
    if [[ "$vmid" -ge 9000 ]] && [[ "$vmid" -le 9999 ]]; then
        log_info "Destroying sandbox container ${vmid}..."
        pct stop "$vmid" --force 2>/dev/null || true
        sleep 1
        pct destroy "$vmid" --force --purge 2>/dev/null || true
        log_success "Destroyed sandbox ${vmid}."
    fi
done

# ============================================================================
# STEP 3: Destroy QEMU VMs in source range (200-299)
# ============================================================================
log_info "Destroying QEMU source VMs (VMID 200-299)..."

for vmid in $(qm list 2>/dev/null | awk 'NR>1{print $1}'); do
    if [[ "$vmid" -ge 200 ]] && [[ "$vmid" -le 299 ]]; then
        local_name=$(qm list 2>/dev/null | awk -v id="$vmid" '$1==id{print $2}')
        log_info "Destroying QEMU VM ${vmid} (${local_name:-unknown})..."
        qm stop "$vmid" --force 2>/dev/null || true
        sleep 1
        qm destroy "$vmid" --force --purge 2>/dev/null || true
        log_success "Destroyed VM ${vmid}."
    fi
done

log_success "Cleanup complete."

# ============================================================================
# STEP 4: Ensure LXC template is available
# ============================================================================
log_info "Verifying LXC template availability..."

LXC_TEMPLATE="ubuntu-22.04-standard_22.04-1_amd64.tar.zst"

pveam update 2>/dev/null || true

if pveam list local 2>/dev/null | grep -q "$LXC_TEMPLATE"; then
    log_success "LXC template available."
else
    log_info "Downloading $LXC_TEMPLATE..."
    if pveam download local "$LXC_TEMPLATE"; then
        log_success "LXC template downloaded."
    else
        LXC_TEMPLATE="ubuntu-22.04-standard_22.04-1_amd64.tar.gz"
        log_info "Trying alternate template: $LXC_TEMPLATE..."
        if pveam download local "$LXC_TEMPLATE"; then
            log_success "LXC template downloaded."
        else
            log_error "Failed to download LXC template."
            exit 1
        fi
    fi
fi

# Add SSH public keys to host
if [[ -n "$SSH_USERS_FILE" ]] && [[ -f "$SSH_USERS_FILE" ]]; then
    log_info "Adding SSH public keys to host authorized_keys..."
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

# Arrays to track created resources for summary
CREATED_NAMES=()
CREATED_IPS=()
CREATED_TYPES=()

# ============================================================================
# create_lxc: Create an LXC source container
#
# Arguments:
#   $1 - ct_name
#   $2 - vmid
# ============================================================================
create_lxc() {
    local ct_name="$1"
    local vmid="$2"

    log_info "Creating LXC container '${ct_name}' (VMID: ${vmid})..."

    # Destroy existing if present (safety)
    if pct status "$vmid" &>/dev/null; then
        pct stop "$vmid" --force 2>/dev/null || true
        sleep 2
        pct destroy "$vmid" --force --purge 2>/dev/null || true
    fi

    # Create container
    pct create "$vmid" "local:vztmpl/${LXC_TEMPLATE}" \
        --hostname "$ct_name" \
        --cores 2 \
        --memory 1024 \
        --swap 512 \
        --storage local-lvm \
        --rootfs local-lvm:8 \
        --net0 "name=eth0,bridge=vmbr0,ip=dhcp" \
        --unprivileged 1 \
        --features nesting=1 \
        --start 0

    log_success "Container '${ct_name}' created."

    # Start container
    log_info "Starting container '${ct_name}'..."
    pct start "$vmid"
    sleep 5

    # Wait for IP
    log_info "Waiting for '${ct_name}' to obtain IP address..."
    local max_wait=120
    local wait_interval=5
    local elapsed=0
    local ct_ip=""

    while [[ $elapsed -lt $max_wait ]]; do
        ct_ip=$(pct exec "$vmid" -- ip -4 addr show eth0 2>/dev/null | grep -oE '([0-9]{1,3}\.){3}[0-9]{1,3}' | head -1 || true)
        if [[ -n "$ct_ip" ]] && [[ "$ct_ip" != "127."* ]]; then
            log_success "Container '${ct_name}' IP: ${ct_ip}"
            break
        fi
        log_info "Waiting for '${ct_name}' IP... (${elapsed}s / ${max_wait}s)"
        sleep $wait_interval
        elapsed=$((elapsed + wait_interval))
    done

    if [[ -z "$ct_ip" ]] || [[ "$ct_ip" == "127."* ]]; then
        ct_ip=""
        log_warn "Container '${ct_name}' did not obtain IP within ${max_wait}s."
        log_warn "Check: pct exec ${vmid} -- ip addr"
    fi

    # Install openssh-server + basic tools
    log_info "Installing packages in container '${ct_name}'..."
    pct exec "$vmid" -- bash -c "apt-get update -qq && apt-get install -y -qq openssh-server curl wget sudo" 2>/dev/null || true

    # Ensure SSH is running
    pct exec "$vmid" -- bash -c "systemctl enable ssh && systemctl start ssh" 2>/dev/null || true

    # Add SSH users from file
    if [[ -n "$SSH_USERS_FILE" ]] && [[ -f "$SSH_USERS_FILE" ]]; then
        log_info "Adding SSH users to container '${ct_name}'..."
        while IFS= read -r line || [[ -n "$line" ]]; do
            [[ -z "$line" ]] && continue
            [[ "$line" =~ ^#.*$ ]] && continue
            local username="${line%% *}"
            local pubkey="${line#* }"
            pct exec "$vmid" -- bash -c "
                id '$username' &>/dev/null || useradd -m -s /bin/bash '$username'
                echo '$username ALL=(ALL) NOPASSWD:ALL' > /etc/sudoers.d/$username
                mkdir -p /home/$username/.ssh
                echo '$pubkey' >> /home/$username/.ssh/authorized_keys
                chmod 700 /home/$username/.ssh
                chmod 600 /home/$username/.ssh/authorized_keys
                chown -R $username:$username /home/$username/.ssh
            " 2>/dev/null || true
            log_success "Added user ${username} to container '${ct_name}'"
        done < "$SSH_USERS_FILE"
    fi

    # Set root password for fallback access
    pct exec "$vmid" -- bash -c "echo 'root:ubuntu' | chpasswd" 2>/dev/null || true

    # Track for summary
    CREATED_NAMES+=("$ct_name")
    CREATED_IPS+=("${ct_ip:-pending}")
    CREATED_TYPES+=("LXC (VMID: $vmid)")
}

# ============================================================================
# create_qemu_vm: Create a QEMU VM with cloud-init
#
# Arguments:
#   $1 - vm_name
#   $2 - vmid
# ============================================================================
create_qemu_vm() {
    local vm_name="$1"
    local vmid="$2"

    log_info "Creating QEMU VM '${vm_name}' (VMID: ${vmid})..."

    # Destroy existing if present (safety)
    if qm status "$vmid" &>/dev/null; then
        qm stop "$vmid" --force 2>/dev/null || true
        sleep 2
        qm destroy "$vmid" --force --purge 2>/dev/null || true
    fi

    # Download Ubuntu cloud image if missing
    local IMAGE_DIR="/var/lib/vz/template/qemu"
    local CLOUD_IMAGE="ubuntu-22.04-minimal-cloudimg-amd64.img"
    local CLOUD_IMAGE_URL="https://cloud-images.ubuntu.com/minimal/releases/jammy/release/${CLOUD_IMAGE}"
    local CLOUD_IMAGE_PATH="${IMAGE_DIR}/${CLOUD_IMAGE}"

    mkdir -p "$IMAGE_DIR"

    if [[ ! -f "$CLOUD_IMAGE_PATH" ]]; then
        log_info "Downloading Ubuntu cloud image..."
        if wget -q --show-progress -O "$CLOUD_IMAGE_PATH" "$CLOUD_IMAGE_URL"; then
            log_success "Cloud image downloaded."
        else
            log_error "Failed to download cloud image."
            exit 1
        fi
    else
        log_info "Cloud image already exists at $CLOUD_IMAGE_PATH"
    fi

    # Create VM
    qm create "$vmid" \
        --name "$vm_name" \
        --cores 2 \
        --memory 2048 \
        --net0 "virtio,bridge=vmbr0" \
        --agent enabled=1 \
        --ostype l26 \
        --scsihw virtio-scsi-single

    # Import disk to local storage
    log_info "Importing disk from cloud image..."
    qm importdisk "$vmid" "$CLOUD_IMAGE_PATH" local 2>/dev/null

    # Attach disk as scsi0
    qm set "$vmid" --scsi0 "local:${vmid}/vm-${vmid}-disk-0.raw"
    qm set "$vmid" --boot order=scsi0

    # Resize disk to 10G
    qm resize "$vmid" scsi0 10G

    # Add cloud-init drive
    qm set "$vmid" --ide2 local-lvm:cloudinit

    # Configure cloud-init
    qm set "$vmid" --ciuser ubuntu --cipassword ubuntu
    qm set "$vmid" --ipconfig0 ip=dhcp

    # Add SSH keys via cloud-init if users file exists
    if [[ -n "$SSH_USERS_FILE" ]] && [[ -f "$SSH_USERS_FILE" ]]; then
        local SSHKEYS_TMP
        SSHKEYS_TMP=$(mktemp)
        while IFS= read -r line || [[ -n "$line" ]]; do
            [[ -z "$line" ]] && continue
            [[ "$line" =~ ^#.*$ ]] && continue
            local pubkey="${line#* }"
            echo "$pubkey" >> "$SSHKEYS_TMP"
        done < "$SSH_USERS_FILE"
        qm set "$vmid" --sshkeys "$SSHKEYS_TMP"
        rm -f "$SSHKEYS_TMP"
    fi

    # Start VM
    log_info "Starting VM '${vm_name}'..."
    qm start "$vmid"

    # Wait for IP via guest agent
    log_info "Waiting for '${vm_name}' to obtain IP address (requires guest agent)..."
    local max_wait=180
    local wait_interval=10
    local elapsed=0
    local vm_ip=""

    while [[ $elapsed -lt $max_wait ]]; do
        vm_ip=$(qm guest cmd "$vmid" network-get-interfaces 2>/dev/null | \
            grep -oE '([0-9]{1,3}\.){3}[0-9]{1,3}' | grep -v '^127\.' | head -1 || true)

        if [[ -n "$vm_ip" ]]; then
            log_success "VM '${vm_name}' IP: ${vm_ip}"
            break
        fi

        log_info "Waiting for '${vm_name}' IP... (${elapsed}s / ${max_wait}s)"
        sleep $wait_interval
        elapsed=$((elapsed + wait_interval))
    done

    if [[ -z "$vm_ip" ]]; then
        log_warn "VM '${vm_name}' did not obtain IP within ${max_wait}s."
        log_warn "Check: qm guest cmd ${vmid} network-get-interfaces"
    fi

    # Track for summary
    CREATED_NAMES+=("$vm_name")
    CREATED_IPS+=("${vm_ip:-pending}")
    CREATED_TYPES+=("QEMU (VMID: $vmid)")
}

# ============================================================================
# STEP 5: Recreate source CT and QEMU VM
# ============================================================================
create_lxc "$LXC_NAME" "$LXC_VMID"
create_qemu_vm "$QEMU_NAME" "$QEMU_VMID"

# ============================================================================
# STEP 6: Final Summary
# ============================================================================
echo ""
echo "============================================================================"
log_success "Proxmox host reset complete!"
echo "============================================================================"
echo ""
echo "Reset Summary:"
echo "  - Source containers (100-199) destroyed and recreated"
echo "  - Source VMs (200-299) destroyed and recreated"
echo "  - Sandbox containers (9000-9999) destroyed"
echo "  - API token was NOT changed (existing token still valid)"
echo ""
for i in "${!CREATED_NAMES[@]}"; do
    echo "  ${CREATED_TYPES[$i]}: '${CREATED_NAMES[$i]}'"
    if [[ "${CREATED_IPS[$i]}" != "pending" ]]; then
        echo "    - IP Address: ${CREATED_IPS[$i]}"
    else
        echo "    - IP Address: (pending)"
    fi
done
echo ""
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
echo "  pct list                                   # List LXC containers"
echo "  qm list                                    # List QEMU VMs"
echo "  pct exec ${LXC_VMID} -- bash               # Shell into LXC container"
echo "  qm terminal ${QEMU_VMID}                   # Console to QEMU VM"
echo ""
