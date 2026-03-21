package microvm

import (
	"fmt"
	"os"
	"path/filepath"

	diskfs "github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/disk"
	"github.com/diskfs/go-diskfs/filesystem"
	"github.com/diskfs/go-diskfs/filesystem/iso9660"
)

const networkConfig = `network:
  version: 2
  ethernets:
    all-en:
      match:
        name: "e*"
      dhcp4: true
`

// generateUserData builds cloud-init user-data YAML with the CA public key
// embedded so the sandbox VM trusts cert-based SSH auth.
// If phoneHomeURL is non-empty, a phone_home module is appended so the
// sandbox signals readiness after runcmd completes.
func generateUserData(caPubKey, phoneHomeURL string) string {
	phoneHome := ""
	if phoneHomeURL != "" {
		phoneHome = fmt.Sprintf(`
phone_home:
  url: %s
  post: [instance_id]
  tries: 3
`, phoneHomeURL)
	}

	return fmt.Sprintf(`#cloud-config
users:
  - default
  - name: sandbox
    shell: /bin/bash
    sudo: ALL=(ALL) NOPASSWD:ALL
    lock_passwd: true

write_files:
  - path: /etc/ssh/authorized_principals/sandbox
    content: |
      sandbox
    owner: root:root
    permissions: '0644'
  - path: /etc/ssh/fluid_ca.pub
    content: |
      %s
    owner: root:root
    permissions: '0644'

runcmd:
  - grep -q 'TrustedUserCAKeys /etc/ssh/fluid_ca.pub' /etc/ssh/sshd_config || echo 'TrustedUserCAKeys /etc/ssh/fluid_ca.pub' >> /etc/ssh/sshd_config
  - grep -q 'AuthorizedPrincipalsFile /etc/ssh/authorized_principals/%%u' /etc/ssh/sshd_config || echo 'AuthorizedPrincipalsFile /etc/ssh/authorized_principals/%%u' >> /etc/ssh/sshd_config
  - systemctl restart sshd 2>/dev/null || systemctl restart ssh 2>/dev/null || service sshd restart 2>/dev/null || service ssh restart
%s`, caPubKey, phoneHome)
}

// GenerateCloudInitISO creates a NoCloud cloud-init ISO containing meta-data,
// network-config, and user-data with the CA public key for SSH cert auth.
// The ISO is written to <workDir>/<sandboxID>/cidata.iso and is cleaned up
// automatically by RemoveOverlay.
//
// A unique instance-id per sandbox forces cloud-init to re-run on cloned disks.
// The network-config uses a catch-all match so DHCP works regardless of the
// interface name assigned by the microvm machine type.
//
// If bridgeIP is non-empty, a cloud-init phone_home module is added that POSTs
// to the daemon readiness endpoint when runcmd completes.
func GenerateCloudInitISO(workDir, sandboxID, caPubKey, bridgeIP string) (string, error) {
	dir := filepath.Join(workDir, sandboxID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create sandbox dir: %w", err)
	}

	isoPath := filepath.Join(dir, "cidata.iso")

	// 2 MB is more than enough for cloud-init metadata
	const isoSize int64 = 2 * 1024 * 1024

	// ISO 9660 requires 2048-byte logical sectors.
	d, err := diskfs.Create(isoPath, isoSize, diskfs.SectorSize(2048))
	if err != nil {
		return "", fmt.Errorf("create disk image: %w", err)
	}

	fspec := disk.FilesystemSpec{
		Partition:   0,
		FSType:      filesystem.TypeISO9660,
		VolumeLabel: "cidata",
	}
	fs, err := d.CreateFilesystem(fspec)
	if err != nil {
		return "", fmt.Errorf("create filesystem: %w", err)
	}

	metaData := fmt.Sprintf("instance-id: %s\n", sandboxID)

	phoneHomeURL := ""
	if bridgeIP != "" {
		phoneHomeURL = fmt.Sprintf("http://%s:9092/ready/%s", bridgeIP, sandboxID)
	}

	files := map[string]string{
		"/meta-data":      metaData,
		"/network-config": networkConfig,
		"/user-data":      generateUserData(caPubKey, phoneHomeURL),
	}

	for name, content := range files {
		f, err := fs.OpenFile(name, os.O_CREATE|os.O_WRONLY)
		if err != nil {
			return "", fmt.Errorf("open %s: %w", name, err)
		}
		if _, err := f.Write([]byte(content)); err != nil {
			return "", fmt.Errorf("write %s: %w", name, err)
		}
	}

	iso, ok := fs.(*iso9660.FileSystem)
	if !ok {
		return "", fmt.Errorf("unexpected filesystem type")
	}
	if err := iso.Finalize(iso9660.FinalizeOptions{
		RockRidge:        true,
		VolumeIdentifier: "cidata",
	}); err != nil {
		return "", fmt.Errorf("finalize ISO: %w", err)
	}

	return isoPath, nil
}
