#!/bin/bash
#
# reset-libvirt-macos.sh
#
# Deletes all domains in the local macOS libvirt and recreates the test-vm.
# Designed for macOS with native libvirt (homebrew).
#
# Usage: ./reset-libvirt-macos.sh [vm-name] [ca-pub-path] [ca-key-path]
#
# For SSH-based connection (Docker compatible):
#   LIBVIRT_URI=qemu+ssh://username@localhost/session ./reset-libvirt-macos.sh
#
# For CA-based authentication:
#   SSH_CA_PUB_PATH=/path/to/ssh_ca.pub SSH_CA_KEY_PATH=/path/to/ssh_ca ./reset-libvirt-macos.sh
#

set -euo pipefail

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[0;33m'
RED='\033[0;31m'
NC='\033[0m'

# Configuration
# Default to local session; set LIBVIRT_URI for SSH-based connection
LIBVIRT_URI="${LIBVIRT_URI:-qemu:///session}"
VM_NAME="${1:-test-vm}"
SSH_CA_PUB_PATH="${2:-${SSH_CA_PUB_PATH:-/etc/fluid/ssh_ca.pub}}"
SSH_CA_KEY_PATH="${3:-${SSH_CA_KEY_PATH:-/etc/fluid/ssh_ca}}"
VM_MEMORY_KB=2097152  # 2GB
VM_VCPUS=2
VM_DISK_SIZE="10G"
BASE_IMAGE_DIR="/var/lib/libvirt/images/base"
JOBS_DIR="/var/lib/libvirt/images/jobs"
NVRAM_DIR="${HOME}/.config/libvirt/qemu/nvram"

# QEMU paths for macOS homebrew (resolve symlinks for libvirtd)
QEMU_EMULATOR="$(readlink -f /opt/homebrew/bin/qemu-system-aarch64 2>/dev/null || echo /opt/homebrew/bin/qemu-system-aarch64)"
UEFI_CODE="/opt/homebrew/share/qemu/edk2-aarch64-code.fd"

# Cloud image URL for ARM64
CLOUD_IMAGE_URL="https://cloud-images.ubuntu.com/releases/jammy/release/ubuntu-22.04-server-cloudimg-arm64.img"
CLOUD_IMAGE="${BASE_IMAGE_DIR}/ubuntu-22.04-server-cloudimg-arm64.img"

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
    echo -e "${RED}[ERROR]${NC} $1"
}

virsh_cmd() {
    virsh -c "${LIBVIRT_URI}" "$@"
}

# =============================================================================
# Step 1: Delete all existing domains
# =============================================================================
delete_all_domains() {
    log_info "Deleting all existing domains..."

    # Get list of all domains
    local domains
    domains=$(virsh_cmd list --all --name 2>/dev/null | grep -v '^$' || true)

    if [ -z "$domains" ]; then
        log_info "No domains found"
        return 0
    fi

    for domain in $domains; do
        log_info "Processing domain: $domain"

        # Check if running
        local state
        state=$(virsh_cmd domstate "$domain" 2>/dev/null || echo "unknown")

        if [ "$state" = "running" ]; then
            log_info "  Destroying running domain: $domain"
            virsh_cmd destroy "$domain" 2>/dev/null || true
        fi

        # Undefine the domain (with nvram if applicable)
        log_info "  Undefining domain: $domain"
        virsh_cmd undefine "$domain" --nvram 2>/dev/null || \
        virsh_cmd undefine "$domain" 2>/dev/null || true
    done

    log_success "All domains deleted"
}

# =============================================================================
# Step 2: Clean up job overlay disks
# =============================================================================
cleanup_jobs() {
    log_info "Cleaning up job overlay disks in ${JOBS_DIR}..."

    if [ -d "$JOBS_DIR" ]; then
        # Remove all files in jobs directory
        if [ -w "$JOBS_DIR" ]; then
            rm -rf "${JOBS_DIR:?}"/* 2>/dev/null || true
        else
            log_warn "No write permission to $JOBS_DIR, trying with sudo..."
            sudo rm -rf "${JOBS_DIR:?}"/*
        fi
        log_success "Job overlays cleaned up"
    else
        log_warn "Jobs directory does not exist: $JOBS_DIR"
    fi
}

# =============================================================================
# Step 3: Download cloud image if needed
# =============================================================================
download_cloud_image() {
    if [ -f "$CLOUD_IMAGE" ]; then
        log_info "Cloud image already exists: $CLOUD_IMAGE"
        return 0
    fi

    log_info "Downloading Ubuntu cloud image..."
    mkdir -p "$BASE_IMAGE_DIR" 2>/dev/null || sudo mkdir -p "$BASE_IMAGE_DIR"
    curl -L --progress-bar -o "/tmp/ubuntu-cloud.img" "$CLOUD_IMAGE_URL"
    mv "/tmp/ubuntu-cloud.img" "$CLOUD_IMAGE" 2>/dev/null || sudo mv "/tmp/ubuntu-cloud.img" "$CLOUD_IMAGE"
    chmod 644 "$CLOUD_IMAGE" 2>/dev/null || sudo chmod 644 "$CLOUD_IMAGE"
    log_success "Cloud image downloaded"
}

# =============================================================================
# Step 4: Create VM disk
# =============================================================================
create_vm_disk() {
    local vm_disk="${BASE_IMAGE_DIR}/${VM_NAME}.qcow2"

    if [ -f "$vm_disk" ]; then
        log_info "Removing existing VM disk: $vm_disk"
        rm -f "$vm_disk" 2>/dev/null || sudo rm -f "$vm_disk"
    fi

    log_info "Creating VM disk from cloud image..."
    qemu-img create -f qcow2 -b "$CLOUD_IMAGE" -F qcow2 "$vm_disk" "$VM_DISK_SIZE" 2>/dev/null || \
        sudo qemu-img create -f qcow2 -b "$CLOUD_IMAGE" -F qcow2 "$vm_disk" "$VM_DISK_SIZE"
    chmod 666 "$vm_disk" 2>/dev/null || sudo chmod 666 "$vm_disk"
    log_success "VM disk created: $vm_disk"
}

# =============================================================================
# Step 5: Create cloud-init ISO
# =============================================================================
create_cloud_init_iso() {
    local cloud_init_dir="/tmp/cloud-init-${VM_NAME}"
    local cloud_init_iso="${BASE_IMAGE_DIR}/${VM_NAME}-cloud-init.iso"

    log_info "Creating cloud-init configuration..."
    mkdir -p "$cloud_init_dir"

    # Get CA public key
    local ca_pub_key
    if [[ -f "$SSH_CA_PUB_PATH" ]]; then
        ca_pub_key=$(cat "$SSH_CA_PUB_PATH")
        log_info "Using CA public key from: $SSH_CA_PUB_PATH"
    else
        ca_pub_key="ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIO0e/MeLFYx1jCQv0qFJvSBEco+2z9TYrwN6wQAlR31E fluid-ssh-ca"
        log_warn "CA public key file not found at $SSH_CA_PUB_PATH, using default fallback"
    fi

    # User data
    cat > "${cloud_init_dir}/user-data" << 'USERDATA'
#cloud-config
hostname: VM_NAME_PLACEHOLDER
manage_etc_hosts: true

# Configure networking to use DHCP without a MAC address match
network:
  version: 2
  ethernets:
    id0:
      dhcp4: true
      match:
        name: en*

users:
  - name: testuser
    sudo: ALL=(ALL) NOPASSWD:ALL
    shell: /bin/bash
    lock_passwd: false
  - name: sandbox
    sudo: ALL=(ALL) NOPASSWD:ALL
    shell: /bin/bash
    lock_passwd: false

chpasswd:
  list: |
    testuser:testpassword
    root:rootpassword
    sandbox:sandboxpassword
  expire: false

ssh_pwauth: true

# Configure SSH daemon for Certificate Authority (CA) based authentication
write_files:
  - path: /etc/ssh/trusted_ca.pub
    permissions: '0644'
    content: |
      CA_PUB_KEY_PLACEHOLDER
  - path: /etc/ssh/sshd_config.d/60-trusted-ca.conf
    permissions: '0644'
    content: |
      # Enable CA-based authentication
      TrustedUserCAKeys /etc/ssh/trusted_ca.pub
      # Disable password-based logins (security best practice)
      PasswordAuthentication no

packages:
  - curl
  - wget
  - vim
  - htop
  - tmux
  - ufw # Ensure ufw is installed to manage it

runcmd:
  - echo "Test VM is ready for fluid.sh testing" > /etc/motd
  # Disable UFW (Uncomplicated Firewall) to ensure SSH connections are not blocked
  - ufw disable || true
  # Restart ssh.service to apply all configuration changes
  - systemctl restart ssh.service

final_message: "Test VM boot completed in $UPTIME seconds"
USERDATA

    # Replace placeholder with actual CA public key
    # Use | as delimiter to avoid issues with / in the key
    sed -i '' "s|CA_PUB_KEY_PLACEHOLDER|${ca_pub_key}|" "${cloud_init_dir}/user-data"
    sed -i '' "s|VM_NAME_PLACEHOLDER|${VM_NAME}|" "${cloud_init_dir}/user-data"

    # Meta data
    cat > "${cloud_init_dir}/meta-data" << METADATA
instance-id: ${VM_NAME}
local-hostname: ${VM_NAME}
METADATA

    # Create ISO
    log_info "Creating cloud-init ISO..."
    if [ -f "$cloud_init_iso" ]; then
        rm -f "$cloud_init_iso" 2>/dev/null || sudo rm -f "$cloud_init_iso"
    fi

    # Use mkisofs (available on macOS via homebrew cdrtools)
    mkisofs -output "/tmp/${VM_NAME}-cloud-init.iso" \
        -volid cidata -joliet -rock \
        "${cloud_init_dir}/user-data" \
        "${cloud_init_dir}/meta-data" 2>/dev/null

    mv "/tmp/${VM_NAME}-cloud-init.iso" "$cloud_init_iso" 2>/dev/null || \
        sudo mv "/tmp/${VM_NAME}-cloud-init.iso" "$cloud_init_iso"
    chmod 644 "$cloud_init_iso" 2>/dev/null || sudo chmod 644 "$cloud_init_iso"

    # Cleanup
    rm -rf "$cloud_init_dir"
    log_success "Cloud-init ISO created: $cloud_init_iso"
}

# =============================================================================
# Step 6: Create NVRAM file for UEFI
# =============================================================================
create_nvram() {
    mkdir -p "$NVRAM_DIR"
    local nvram_file="${NVRAM_DIR}/${VM_NAME}_VARS.fd"

    if [ -f "$nvram_file" ]; then
        log_info "Removing existing NVRAM: $nvram_file"
        rm -f "$nvram_file"
    fi

    log_info "Creating NVRAM file for UEFI..."
    # Create empty NVRAM file (64MB for aarch64)
    dd if=/dev/zero of="$nvram_file" bs=1m count=64 2>/dev/null
    chmod 644 "$nvram_file"
    log_success "NVRAM created: $nvram_file"
}

# =============================================================================
# Step 7: Define the VM
# =============================================================================
define_vm() {
    local vm_disk="${BASE_IMAGE_DIR}/${VM_NAME}.qcow2"
    local cloud_init_iso="${BASE_IMAGE_DIR}/${VM_NAME}-cloud-init.iso"
    local nvram_file="${NVRAM_DIR}/${VM_NAME}_VARS.fd"
    local xml_file="/tmp/${VM_NAME}.xml"
    local wrapper_script

    # Determine absolute path to the wrapper script relative to this script
    wrapper_script="$(cd "$(dirname "$0")" && pwd)/qemu-socket-vmnet-wrapper.sh"

    # Generate a random MAC address
    local mac_address
    mac_address=$(printf '52:54:00:%02x:%02x:%02x' $((RANDOM%256)) $((RANDOM%256)) $((RANDOM%256)))

    log_info "Creating VM definition for socket_vmnet..."
    log_info "Generated MAC address: ${mac_address}"
    log_info "Emulator wrapper: ${wrapper_script}"

    cat > "$xml_file" << VMXML
<domain type='qemu' xmlns:qemu='http://libvirt.org/schemas/domain/qemu/1.0'>
  <name>${VM_NAME}</name>
  <memory unit='KiB'>${VM_MEMORY_KB}</memory>
  <currentMemory unit='KiB'>${VM_MEMORY_KB}</currentMemory>
  <vcpu placement='static'>${VM_VCPUS}</vcpu>
  <os>
    <type arch='aarch64' machine='virt'>hvm</type>
    <loader readonly='yes' type='pflash'>${UEFI_CODE}</loader>
    <nvram>${nvram_file}</nvram>
    <boot dev='hd'/>
  </os>
  <features>
    <acpi/>
    <gic version='3'/>
  </features>
  <cpu mode='custom' match='exact' check='none'>
    <model fallback='allow'>cortex-a57</model>
  </cpu>
  <clock offset='utc'/>
  <on_poweroff>destroy</on_poweroff>
  <on_reboot>restart</on_reboot>
  <on_crash>destroy</on_crash>
  <devices>
    <emulator>${wrapper_script}</emulator>
    <disk type='file' device='disk'>
      <driver name='qemu' type='qcow2' cache='writeback'/>
      <source file='${vm_disk}'/>
      <target dev='vda' bus='virtio'/>
    </disk>
    <disk type='file' device='cdrom'>
      <driver name='qemu' type='raw'/>
      <source file='${cloud_init_iso}'/>
      <target dev='sda' bus='scsi'/>
      <readonly/>
    </disk>
    <controller type='scsi' index='0' model='virtio-scsi'/>
    <controller type='pci' index='0' model='pcie-root'/>
    <controller type='pci' index='1' model='pcie-root-port'/>
    <controller type='pci' index='2' model='pcie-root-port'/>
    <controller type='pci' index='3' model='pcie-root-port'/>
    <controller type='pci' index='4' model='pcie-root-port'/>
    <serial type='pty'>
      <target type='system-serial' port='0'>
        <model name='pl011'/>
      </target>
    </serial>
    <console type='pty'>
      <target type='serial' port='0'/>
    </console>
    <graphics type='vnc' port='-1' autoport='yes' listen='0.0.0.0'/>
    <video>
      <model type='virtio' heads='1' primary='yes'/>
    </video>
  </devices>
  <qemu:commandline>
    <qemu:arg value='-netdev'/>
    <qemu:arg value='socket,id=vnet,fd=3'/>
    <qemu:arg value='-device'/>
    <qemu:arg value='virtio-net-pci,netdev=vnet,mac=${mac_address},addr=0x03'/>
  </qemu:commandline>
</domain>
VMXML

    log_info "Defining VM in libvirt..."
    virsh_cmd define "$xml_file"
    rm -f "$xml_file"
    log_success "VM defined: ${VM_NAME}"
}

# =============================================================================
# Step 8: Start the VM (optional)
# =============================================================================
start_vm() {
    log_info "Starting VM: ${VM_NAME}..."

    if virsh_cmd start "$VM_NAME" 2>&1; then
        log_success "VM started successfully"
    else
        log_warn "VM may have failed to start - this can happen on macOS due to HVF limitations"
        log_info "Check VM state with: virsh -c ${LIBVIRT_URI} domstate ${VM_NAME}"
    fi
}

# =============================================================================
# Main
# =============================================================================
main() {
    echo ""
    echo "=================================================="
    echo "  macOS Libvirt Reset Script"
    echo "=================================================="
    echo ""
    echo "LIBVIRT_URI: ${LIBVIRT_URI}"
    echo "VM Name:     ${VM_NAME}"
    echo "CA Pub Path: ${SSH_CA_PUB_PATH}"
    echo "CA Key Path: ${SSH_CA_KEY_PATH}"
    echo ""

    # Verify libvirt connection
    log_info "Verifying libvirt connection..."
    if ! virsh_cmd version &>/dev/null; then
        log_error "Cannot connect to libvirt at ${LIBVIRT_URI}"
        log_error "Make sure libvirtd is running with --listen flag"
        exit 1
    fi
    log_success "Connected to libvirt"

    # Run steps
    delete_all_domains
    cleanup_jobs
    download_cloud_image
    create_vm_disk
    create_cloud_init_iso
    create_nvram
    define_vm
    start_vm

    echo ""
    echo "=================================================="
    echo "  Setup Complete"
    echo "=================================================="
    echo ""
    echo "VM Details:"
    virsh_cmd dominfo "$VM_NAME" 2>/dev/null || true
    echo ""
    echo "Commands:"
    echo "  List VMs:     virsh -c ${LIBVIRT_URI} list --all"
    echo "  Start VM:     virsh -c ${LIBVIRT_URI} start ${VM_NAME}"
    echo "  Console:      virsh -c ${LIBVIRT_URI} console ${VM_NAME}"
    echo "  Get IP:       virsh -c ${LIBVIRT_URI} domifaddr ${VM_NAME}"
    echo ""
    echo "Default credentials:"
    echo "  Username: testuser"
    echo "  Password: testpassword"
    echo ""
}

main "$@"
