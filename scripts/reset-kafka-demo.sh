#!/bin/bash
# reset-kafka-demo.sh
#
# Destroys kafka-demo-{INDEX} and logstash-demo-{INDEX} VMs and recreates
# them from scratch by running setup-kafka-demo.sh.
#
# Unlike reset-ubuntu.sh, this script only removes the kafka/logstash demo VMs
# and leaves any other VMs on the host untouched.
#
# Usage: sudo ./reset-kafka-demo.sh [VM_INDEX] [--ssh-users-file <path>]
#
# Options:
#   VM_INDEX                  VM index number (default: 1)
#   --ssh-users-file <path>   Path to file with SSH users (one per line: <username> <public-key>)

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

VM_INDEX=""
SSH_USERS_FILE=""

while [[ $# -gt 0 ]]; do
    case "$1" in
        --ssh-users-file)
            SSH_USERS_FILE="$2"
            shift 2
            ;;
        --help|-h)
            echo "Usage: sudo ./reset-kafka-demo.sh [VM_INDEX] [--ssh-users-file <path>]"
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

RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info()    { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
log_warn()    { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error()   { echo -e "${RED}[ERROR]${NC} $1" >&2; }

if [[ $EUID -ne 0 ]]; then
    log_error "This script must be run as root"
    exit 1
fi

if ! command -v virsh &>/dev/null; then
    log_error "virsh not found. Please run setup-ubuntu.sh first."
    exit 1
fi

KAFKA_VM="kafka-demo-${VM_INDEX}"
LOGSTASH_VM="logstash-demo-${VM_INDEX}"
IMAGE_DIR="/var/lib/libvirt/images"
CLOUD_INIT_DIR="${IMAGE_DIR}/cloud-init"

log_warn "Destroying ${KAFKA_VM} and ${LOGSTASH_VM} (other VMs will not be touched)."

# ============================================================================
# Destroy and undefine the demo VMs
# ============================================================================
for VM in "$KAFKA_VM" "$LOGSTASH_VM"; do
    if virsh dominfo "$VM" &>/dev/null; then
        log_info "Destroying VM: ${VM}..."
        virsh destroy "$VM" > /dev/null 2>&1 || true
        virsh undefine "$VM" --nvram > /dev/null 2>&1 || virsh undefine "$VM" > /dev/null 2>&1 || true
        log_success "Removed: ${VM}"
    else
        log_info "VM '${VM}' does not exist, skipping."
    fi
done

# ============================================================================
# Clean up disks and cloud-init data
# ============================================================================
log_info "Cleaning up disks and cloud-init data..."
rm -f "${IMAGE_DIR}/${KAFKA_VM}.qcow2" 2>/dev/null || true
rm -f "${IMAGE_DIR}/${LOGSTASH_VM}.qcow2" 2>/dev/null || true
rm -rf "${CLOUD_INIT_DIR}/${KAFKA_VM}" 2>/dev/null || true
rm -rf "${CLOUD_INIT_DIR}/${LOGSTASH_VM}" 2>/dev/null || true
log_success "Cleanup complete."

# ============================================================================
# Flush DHCP leases to prevent IP conflicts on re-creation
# ============================================================================
log_info "Flushing DHCP leases..."
virsh net-destroy default > /dev/null 2>&1 || true
rm -f /var/lib/libvirt/dnsmasq/virbr0.status \
      /var/lib/libvirt/dnsmasq/virbr0.leases \
      /var/lib/libvirt/dnsmasq/default.leases 2>/dev/null || true
virsh net-start default > /dev/null 2>&1 || true
log_success "DHCP leases flushed."

# ============================================================================
# Re-run setup
# ============================================================================
log_info "Re-running setup-kafka-demo.sh..."
echo ""

SETUP_SCRIPT="${SCRIPT_DIR}/setup-kafka-demo.sh"
if [[ ! -f "$SETUP_SCRIPT" ]]; then
    log_error "setup-kafka-demo.sh not found at: ${SETUP_SCRIPT}"
    exit 1
fi

SETUP_ARGS=("$VM_INDEX")
if [[ -n "$SSH_USERS_FILE" ]]; then
    SETUP_ARGS+=("--ssh-users-file" "$SSH_USERS_FILE")
fi

exec "$SETUP_SCRIPT" "${SETUP_ARGS[@]}"
