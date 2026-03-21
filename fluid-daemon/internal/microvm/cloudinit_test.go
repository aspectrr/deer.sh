package microvm

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	diskfs "github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/filesystem/iso9660"
)

const testCAPubKey = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAITestCAKeyForUnitTests fluid-ca@test"

func TestGenerateCloudInitISO(t *testing.T) {
	workDir := t.TempDir()
	sandboxID := "SBX-test-1234"

	isoPath, err := GenerateCloudInitISO(workDir, sandboxID, testCAPubKey, "")
	if err != nil {
		t.Fatalf("GenerateCloudInitISO: %v", err)
	}

	// Verify file exists with nonzero size
	info, err := os.Stat(isoPath)
	if err != nil {
		t.Fatalf("stat ISO: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("ISO file is empty")
	}

	// Verify path is under the sandbox directory
	expected := filepath.Join(workDir, sandboxID, "cidata.iso")
	if isoPath != expected {
		t.Errorf("path = %q, want %q", isoPath, expected)
	}

	// Open the ISO and verify contents
	d, err := diskfs.Open(isoPath)
	if err != nil {
		t.Fatalf("open ISO: %v", err)
	}

	fs, err := d.GetFilesystem(0)
	if err != nil {
		t.Fatalf("get filesystem: %v", err)
	}

	isoFS, ok := fs.(*iso9660.FileSystem)
	if !ok {
		t.Fatal("filesystem is not ISO 9660")
	}

	// Read meta-data
	metaFile, err := isoFS.OpenFile("/meta-data", os.O_RDONLY)
	if err != nil {
		t.Fatalf("open meta-data: %v", err)
	}
	metaBytes, err := io.ReadAll(metaFile)
	if err != nil {
		t.Fatalf("read meta-data: %v", err)
	}
	metaContent := string(metaBytes)

	if !strings.Contains(metaContent, "instance-id: "+sandboxID) {
		t.Errorf("meta-data missing instance-id, got: %q", metaContent)
	}

	// Read network-config
	netFile, err := isoFS.OpenFile("/network-config", os.O_RDONLY)
	if err != nil {
		t.Fatalf("open network-config: %v", err)
	}
	netBytes, err := io.ReadAll(netFile)
	if err != nil {
		t.Fatalf("read network-config: %v", err)
	}
	netContent := string(netBytes)

	if !strings.Contains(netContent, "dhcp4: true") {
		t.Errorf("network-config missing dhcp4, got: %q", netContent)
	}
	if !strings.Contains(netContent, `name: "e*"`) {
		t.Errorf("network-config missing match pattern, got: %q", netContent)
	}

	// Read user-data
	userFile, err := isoFS.OpenFile("/user-data", os.O_RDONLY)
	if err != nil {
		t.Fatalf("open user-data: %v", err)
	}
	userBytes, err := io.ReadAll(userFile)
	if err != nil {
		t.Fatalf("read user-data: %v", err)
	}
	userContent := string(userBytes)

	if !strings.Contains(userContent, "name: sandbox") {
		t.Errorf("user-data missing sandbox user, got: %q", userContent)
	}
	if !strings.Contains(userContent, "authorized_principals/sandbox") {
		t.Errorf("user-data missing authorized_principals, got: %q", userContent)
	}
	if !strings.Contains(userContent, "fluid_ca.pub") {
		t.Errorf("user-data missing fluid_ca.pub, got: %q", userContent)
	}
	if !strings.Contains(userContent, testCAPubKey) {
		t.Errorf("user-data missing CA public key, got: %q", userContent)
	}
	if !strings.Contains(userContent, "TrustedUserCAKeys /etc/ssh/fluid_ca.pub") {
		t.Errorf("user-data missing TrustedUserCAKeys, got: %q", userContent)
	}
	if !strings.Contains(userContent, "AuthorizedPrincipalsFile /etc/ssh/authorized_principals/%u") {
		t.Errorf("user-data missing AuthorizedPrincipalsFile, got: %q", userContent)
	}
}

func TestGenerateCloudInitISO_DifferentSandboxIDs(t *testing.T) {
	workDir := t.TempDir()

	path1, err := GenerateCloudInitISO(workDir, "SBX-aaa", testCAPubKey, "")
	if err != nil {
		t.Fatalf("first ISO: %v", err)
	}
	path2, err := GenerateCloudInitISO(workDir, "SBX-bbb", testCAPubKey, "")
	if err != nil {
		t.Fatalf("second ISO: %v", err)
	}

	if path1 == path2 {
		t.Error("different sandbox IDs produced same ISO path")
	}

	data1, err := os.ReadFile(path1)
	if err != nil {
		t.Fatalf("read first ISO: %v", err)
	}
	data2, err := os.ReadFile(path2)
	if err != nil {
		t.Fatalf("read second ISO: %v", err)
	}

	if string(data1) == string(data2) {
		t.Error("different sandbox IDs produced identical ISO content")
	}
}
