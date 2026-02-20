#!/bin/bash
# setup-proxmox.sh
#
# Sets up Proxmox VE on a Debian host and creates source LXC containers + QEMU VMs.
# Requires a reboot after initial Proxmox install (PVE kernel). Re-run after reboot
# to complete VM/CT creation.
#
# Usage: sudo ./setup-proxmox.sh [VM_INDEX] [--ssh-users-file <path>]
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
            echo "Usage: sudo ./setup-proxmox.sh [VM_INDEX] [--ssh-users-file <path>]"
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

# VMID scheme
LXC_VMID=$((100 + VM_INDEX))
QEMU_VMID=$((200 + VM_INDEX))
LXC_NAME="test-vm-${VM_INDEX}"
QEMU_NAME="test-vm-qemu-${VM_INDEX}"

log_info "Starting Proxmox setup (VM_INDEX=${VM_INDEX})..."

# ============================================================================
# STEP 1: Check if Proxmox is already installed
# ============================================================================
log_info "Checking if Proxmox VE is already installed..."

PROXMOX_INSTALLED=false
if command -v pvesh &>/dev/null && systemctl is-active --quiet pve-cluster 2>/dev/null; then
    PROXMOX_INSTALLED=true
    log_success "Proxmox VE is already installed and running."
fi

# ============================================================================
# STEP 2: Install Proxmox VE (if not installed)
# ============================================================================
if [[ "$PROXMOX_INSTALLED" == false ]]; then
    log_info "Proxmox VE not detected. Installing..."

    # Verify this is a Debian host
    if [[ ! -f /etc/os-release ]]; then
        log_error "Cannot detect OS. /etc/os-release missing."
        exit 1
    fi
    source /etc/os-release
    if [[ "$ID" != "debian" ]]; then
        log_error "This script requires Debian. Detected: $ID"
        exit 1
    fi
    log_info "Detected Debian $VERSION_CODENAME"

    # Ensure hostname resolves in /etc/hosts (Proxmox requirement)
    HOSTNAME_FQDN=$(hostname -f 2>/dev/null || hostname)
    HOSTNAME_SHORT=$(hostname -s 2>/dev/null || hostname)
    HOST_IP=$(hostname -I | awk '{print $1}')
    if ! grep -q "$HOSTNAME_FQDN" /etc/hosts 2>/dev/null; then
        log_info "Adding hostname to /etc/hosts..."
        echo "${HOST_IP} ${HOSTNAME_FQDN} ${HOSTNAME_SHORT}" >> /etc/hosts
        log_success "Added ${HOST_IP} ${HOSTNAME_FQDN} ${HOSTNAME_SHORT} to /etc/hosts"
    fi

    # Add Proxmox GPG key + no-subscription repo
    export DEBIAN_FRONTEND=noninteractive
    log_info "Adding Proxmox VE repository..."
    apt-get update -qq
    apt-get install -y -qq wget gnupg2

    wget -qO /etc/apt/trusted.gpg.d/proxmox-release-$VERSION_CODENAME.gpg \
        "http://download.proxmox.com/debian/proxmox-release-$VERSION_CODENAME.gpg"

    echo "deb http://download.proxmox.com/debian/pve $VERSION_CODENAME pve-no-subscription" \
        > /etc/apt/sources.list.d/pve-install-repo.list

    apt-get update -qq
    log_success "Proxmox repository added."

    # Preseed postfix debconf (local only)
    log_info "Preseeding postfix configuration..."
    debconf-set-selections <<< "postfix postfix/mailname string $(hostname -f)"
    debconf-set-selections <<< "postfix postfix/main_mailer_type string 'Local only'"

    # Install Proxmox VE
    log_info "Installing Proxmox VE packages (this may take several minutes)..."
    apt-get install -y proxmox-ve postfix open-iscsi chrony

    log_success "Proxmox VE packages installed."

    # Remove Debian default kernel and os-prober
    log_info "Removing Debian default kernel and os-prober..."
    apt-get remove -y os-prober 2>/dev/null || true
    # Remove non-PVE kernels
    DEBIAN_KERNELS=$(dpkg -l | awk '/linux-image-[0-9]/{print $2}' | grep -v pve || true)
    if [[ -n "$DEBIAN_KERNELS" ]]; then
        log_info "Removing Debian kernels: $DEBIAN_KERNELS"
        apt-get remove -y $DEBIAN_KERNELS 2>/dev/null || true
    fi
    apt-get autoremove -y 2>/dev/null || true

    log_success "Cleanup complete."
fi

# ============================================================================
# REBOOT GATE: Check if running PVE kernel
# ============================================================================
log_info "Checking running kernel..."

RUNNING_KERNEL=$(uname -r)
if [[ "$RUNNING_KERNEL" != *-pve ]]; then
    echo ""
    echo "============================================================================"
    log_warn "Proxmox VE is installed but NOT running the PVE kernel."
    log_warn "Current kernel: $RUNNING_KERNEL"
    echo ""
    log_warn "A REBOOT is required before Proxmox services can start."
    log_warn "After reboot, re-run this script to complete setup (VM/CT creation)."
    echo ""
    log_info "To reboot: sudo reboot"
    echo "============================================================================"
    exit 0
fi

log_success "Running PVE kernel: $RUNNING_KERNEL"

# ============================================================================
# STEP 3: Verify Proxmox services
# ============================================================================
log_info "Verifying Proxmox services..."

SERVICES=("pve-cluster" "pvedaemon" "pveproxy" "pvestatd")
ALL_OK=true
for svc in "${SERVICES[@]}"; do
    if systemctl is-active --quiet "$svc"; then
        log_success "$svc is running."
    else
        log_warn "$svc is not running. Attempting to start..."
        systemctl start "$svc" 2>/dev/null || true
        sleep 2
        if systemctl is-active --quiet "$svc"; then
            log_success "$svc started."
        else
            log_error "$svc failed to start."
            ALL_OK=false
        fi
    fi
done

if [[ "$ALL_OK" == false ]]; then
    log_error "Some Proxmox services failed. Check 'systemctl status pve-cluster'."
    exit 1
fi

# ============================================================================
# STEP 4: Disable enterprise repo nag
# ============================================================================
log_info "Disabling enterprise repository sources..."

rm -f /etc/apt/sources.list.d/pve-enterprise.list 2>/dev/null || true
rm -f /etc/apt/sources.list.d/ceph.list 2>/dev/null || true

log_success "Enterprise repo sources removed."

# ============================================================================
# STEP 5: Create API token
# ============================================================================
log_info "Creating API token for daemon access..."

API_TOKEN_ID="root@pam!fluid"
API_SECRET=""

# Check if token already exists
if pveum user token list root@pam 2>/dev/null | grep -q "fluid"; then
    log_warn "API token 'fluid' already exists. Removing and recreating..."
    pveum user token remove root@pam fluid 2>/dev/null || true
fi

TOKEN_OUTPUT=$(pveum user token add root@pam fluid --privsep 0 2>&1)
API_SECRET=$(echo "$TOKEN_OUTPUT" | grep -oE '[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}')

if [[ -z "$API_SECRET" ]]; then
    log_error "Failed to extract API token secret from output:"
    echo "$TOKEN_OUTPUT"
    exit 1
fi

log_success "API token created: $API_TOKEN_ID"

# ============================================================================
# STEP 6: Detect node name + verify storage
# ============================================================================
log_info "Detecting node name and verifying storage..."

PVE_NODE=$(hostname)
log_info "Node name: $PVE_NODE"

# Verify local-lvm exists (for LXC rootfs)
if pvesm status | grep -q "local-lvm"; then
    log_success "Storage 'local-lvm' available."
else
    log_warn "Storage 'local-lvm' not found. LXC containers may need manual storage config."
fi

# Verify local exists (for QEMU disk images, ISOs, vzdump)
if pvesm status | grep -q "local "; then
    log_success "Storage 'local' available."
else
    log_error "Storage 'local' not found. Cannot continue."
    exit 1
fi

# ============================================================================
# STEP 7: Add SSH keys to host
# ============================================================================
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

# ============================================================================
# STEP 8: Download Ubuntu LXC template
# ============================================================================
log_info "Downloading Ubuntu LXC template..."

LXC_TEMPLATE="ubuntu-22.04-standard_22.04-1_amd64.tar.zst"

pveam update 2>/dev/null || true

# Check if template already exists
if pveam list local 2>/dev/null | grep -q "$LXC_TEMPLATE"; then
    log_success "LXC template already available."
else
    log_info "Downloading $LXC_TEMPLATE..."
    if pveam download local "$LXC_TEMPLATE"; then
        log_success "LXC template downloaded."
    else
        # Try alternate template name
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

    # Destroy existing if present
    if pct status "$vmid" &>/dev/null; then
        log_warn "Container VMID ${vmid} already exists. Destroying..."
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
        # Skip localhost
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

    # Also set root password for fallback access
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

    # Destroy existing if present
    if qm status "$vmid" &>/dev/null; then
        log_warn "VM VMID ${vmid} already exists. Destroying..."
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
        # Try guest agent first
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
        log_warn "The guest agent may not be ready yet. Check manually:"
        log_warn "  qm guest cmd ${vmid} network-get-interfaces"
        log_warn "  qm terminal ${vmid}"
    fi

    # Track for summary
    CREATED_NAMES+=("$vm_name")
    CREATED_IPS+=("${vm_ip:-pending}")
    CREATED_TYPES+=("QEMU (VMID: $vmid)")
}

# ============================================================================
# STEP 9: Create source LXC container
# ============================================================================
create_lxc "$LXC_NAME" "$LXC_VMID"

# ============================================================================
# STEP 10: Create source QEMU VM
# ============================================================================
create_qemu_vm "$QEMU_NAME" "$QEMU_VMID"

# ============================================================================
# STEP 11: Final Summary
# ============================================================================
echo ""
echo "============================================================================"
log_success "Proxmox setup complete!"
echo "============================================================================"
echo ""
echo "Setup Summary:"
echo "  - Proxmox VE running on kernel: $(uname -r)"
echo "  - Node: ${PVE_NODE}"
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
echo "  API Token:"
echo "    - Token ID: ${API_TOKEN_ID}"
echo "    - Secret:   ${API_SECRET}"
echo ""
if [[ -n "$SSH_USERS_FILE" ]] && [[ -f "$SSH_USERS_FILE" ]]; then
    echo "  SSH Users:"
    while IFS= read -r line || [[ -n "$line" ]]; do
        [[ -z "$line" ]] && continue
        [[ "$line" =~ ^#.*$ ]] && continue
        username="${line%% *}"
        echo "      ${username} (key-based auth)"
    done < "$SSH_USERS_FILE"
    echo ""
fi
echo "  Daemon config snippet (daemon.yaml):"
echo "    provider: lxc"
echo "    lxc:"
echo "      host: \"https://$(hostname -I | awk '{print $1}'):8006\""
echo "      token_id: \"${API_TOKEN_ID}\""
echo "      secret: \"${API_SECRET}\""
echo "      node: \"${PVE_NODE}\""
echo "      storage: \"local-lvm\""
echo "      bridge: \"vmbr0\""
echo "      vmid_start: 9000"
echo "      vmid_end: 9999"
echo "      verify_ssl: false"
echo ""
echo "Useful commands:"
echo "  pct list                                   # List LXC containers"
echo "  qm list                                    # List QEMU VMs"
echo "  pvesh get /access/users/root@pam/token     # List API tokens"
echo "  pct exec ${LXC_VMID} -- bash               # Shell into LXC container"
echo "  qm terminal ${QEMU_VMID}                   # Console to QEMU VM"
echo ""
