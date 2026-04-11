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

const testCAPubKey = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAITestCAKeyForUnitTests deer-ca@test"

func TestGenerateCloudInitISO(t *testing.T) {
	workDir := t.TempDir()
	sandboxID := "SBX-test-1234"

	isoPath, err := GenerateCloudInitISO(workDir, sandboxID, CloudInitOptions{
		CAPubKey: testCAPubKey,
	})
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
	if !strings.Contains(userContent, "deer_ca.pub") {
		t.Errorf("user-data missing deer_ca.pub, got: %q", userContent)
	}
	if !strings.Contains(userContent, testCAPubKey) {
		t.Errorf("user-data missing CA public key, got: %q", userContent)
	}
	if !strings.Contains(userContent, "TrustedUserCAKeys /etc/ssh/deer_ca.pub") {
		t.Errorf("user-data missing TrustedUserCAKeys, got: %q", userContent)
	}
	if !strings.Contains(userContent, "AuthorizedPrincipalsFile /etc/ssh/authorized_principals/%u") {
		t.Errorf("user-data missing AuthorizedPrincipalsFile, got: %q", userContent)
	}
	if !strings.Contains(userContent, "growpart:") {
		t.Errorf("user-data missing growpart config, got: %q", userContent)
	}
	if !strings.Contains(userContent, "resize_rootfs: true") {
		t.Errorf("user-data missing resize_rootfs, got: %q", userContent)
	}
	if strings.Contains(userContent, "deer-install-redpanda.sh") || strings.Contains(userContent, "deer-redpanda.service") {
		t.Errorf("non-kafka user-data should not include redpanda assets, got: %q", userContent)
	}
}

func TestGenerateCloudInitISO_DifferentSandboxIDs(t *testing.T) {
	workDir := t.TempDir()

	path1, err := GenerateCloudInitISO(workDir, "SBX-aaa", CloudInitOptions{CAPubKey: testCAPubKey})
	if err != nil {
		t.Fatalf("first ISO: %v", err)
	}
	path2, err := GenerateCloudInitISO(workDir, "SBX-bbb", CloudInitOptions{CAPubKey: testCAPubKey})
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

func TestGenerateCloudInitISO_WithKafkaBroker(t *testing.T) {
	workDir := t.TempDir()

	isoPath, err := GenerateCloudInitISO(workDir, "SBX-kafka", CloudInitOptions{
		CAPubKey:     testCAPubKey,
		PhoneHomeURL: "http://10.0.0.1:9092/ready/SBX-kafka",
		KafkaBroker: KafkaBrokerOptions{
			Enabled:          true,
			AdvertiseAddress: "10.0.0.15",
			ArchiveURL:       "http://10.0.0.1:9088/redpanda.tar.gz",
			Port:             9092,
		},
	})
	if err != nil {
		t.Fatalf("GenerateCloudInitISO: %v", err)
	}

	d, err := diskfs.Open(isoPath)
	if err != nil {
		t.Fatalf("open ISO: %v", err)
	}
	fs, err := d.GetFilesystem(0)
	if err != nil {
		t.Fatalf("get filesystem: %v", err)
	}
	isoFS := fs.(*iso9660.FileSystem)
	userFile, err := isoFS.OpenFile("/user-data", os.O_RDONLY)
	if err != nil {
		t.Fatalf("open user-data: %v", err)
	}
	userBytes, err := io.ReadAll(userFile)
	if err != nil {
		t.Fatalf("read user-data: %v", err)
	}
	userContent := string(userBytes)

	if !strings.Contains(userContent, "deer-install-redpanda.sh") {
		t.Fatalf("expected redpanda install script in user-data, got %q", userContent)
	}
	if !strings.Contains(userContent, "deer-redpanda-start.sh") {
		t.Fatalf("expected redpanda start wrapper in user-data, got %q", userContent)
	}
	if !strings.Contains(userContent, "deer-enable-redpanda.sh") {
		t.Fatalf("expected redpanda enable wrapper in user-data, got %q", userContent)
	}
	if !strings.Contains(userContent, "deer-wait-redpanda.sh") {
		t.Fatalf("expected redpanda readiness wait script in user-data, got %q", userContent)
	}
	if !strings.Contains(userContent, "deer-redpanda-diagnostics.sh") {
		t.Fatalf("expected redpanda diagnostics script in user-data, got %q", userContent)
	}
	if !strings.Contains(userContent, "empty_seed_starts_cluster: true") {
		t.Fatalf("expected single-node bootstrap in redpanda.yaml, got %q", userContent)
	}
	if !strings.Contains(userContent, "advertised_rpc_api:") {
		t.Fatalf("expected advertised RPC listener in user-data, got %q", userContent)
	}
	if !strings.Contains(userContent, "- name: internal") {
		t.Fatalf("expected named kafka listener in user-data, got %q", userContent)
	}
	if strings.Contains(userContent, "pandaproxy: {}") || strings.Contains(userContent, "schema_registry: {}") {
		t.Fatalf("did not expect pandaproxy or schema registry in default kafka stub config, got %q", userContent)
	}
	if !strings.Contains(userContent, "cloud-init status --long") {
		t.Fatalf("expected expanded cloud-init diagnostics in user-data, got %q", userContent)
	}
	if !strings.Contains(userContent, "journalctl -k --no-pager -n 200") {
		t.Fatalf("expected kernel journal diagnostics in user-data, got %q", userContent)
	}
	if !strings.Contains(userContent, "nproc || true") || !strings.Contains(userContent, "free -m || true") || !strings.Contains(userContent, "cat /proc/meminfo || true") {
		t.Fatalf("expected cpu and memory diagnostics in user-data, got %q", userContent)
	}
	if !strings.Contains(userContent, "find /var/log /var/lib/redpanda -maxdepth 4 -type f") {
		t.Fatalf("expected redpanda log discovery in user-data, got %q", userContent)
	}
	if !strings.Contains(userContent, "deer-redpanda.service") {
		t.Fatalf("expected redpanda systemd unit in user-data, got %q", userContent)
	}
	if !strings.Contains(userContent, "StandardOutput=journal+console") {
		t.Fatalf("expected redpanda service console logging in user-data, got %q", userContent)
	}
	if !strings.Contains(userContent, "10.0.0.15") {
		t.Fatalf("expected advertised broker address in user-data, got %q", userContent)
	}
	if !strings.Contains(userContent, "http://10.0.0.1:9088/redpanda.tar.gz") {
		t.Fatalf("expected custom archive URL in user-data, got %q", userContent)
	}
	if !strings.Contains(userContent, "archive_path=\"$tmpdir/redpanda.tar.gz\"") {
		t.Fatalf("expected archive-based install path in user-data, got %q", userContent)
	}
	if !strings.Contains(userContent, "tmpdir=$(mktemp -d /var/tmp/deer-redpanda.XXXXXX)") {
		t.Fatalf("expected /var/tmp install workspace in user-data, got %q", userContent)
	}
	if strings.Contains(userContent, "trap 'rm -rf \"$tmpdir\"' EXIT") {
		t.Fatalf("did not expect install workspace cleanup trap in user-data, got %q", userContent)
	}
	if !strings.Contains(userContent, "deer redpanda archive download complete") {
		t.Fatalf("expected archive download progress marker in user-data, got %q", userContent)
	}
	if !strings.Contains(userContent, "deer redpanda extraction complete") {
		t.Fatalf("expected extraction progress marker in user-data, got %q", userContent)
	}
	if !strings.Contains(userContent, "deer redpanda binary resolution complete") {
		t.Fatalf("expected binary resolution progress marker in user-data, got %q", userContent)
	}
	if !strings.Contains(userContent, "deer redpanda env file written") {
		t.Fatalf("expected env-file progress marker in user-data, got %q", userContent)
	}
	if !strings.Contains(userContent, "deer redpanda temp cleanup complete") {
		t.Fatalf("expected explicit temp cleanup marker in user-data, got %q", userContent)
	}
	if !strings.Contains(userContent, "if tar -tzf \"$archive_path\" | grep -Eq '^(\\./)?(usr/bin/redpanda|opt/redpanda/)'; then") {
		t.Fatalf("expected rootfs archive detection in user-data, got %q", userContent)
	}
	if !strings.Contains(userContent, "timeout 15m tar -xzf \"$archive_path\" -C /") {
		t.Fatalf("expected bounded archive extraction in user-data, got %q", userContent)
	}
	if !strings.Contains(userContent, "tar -xzf \"$archive_path\" -C \"$extract_root\"") {
		t.Fatalf("expected fallback tar extraction into a staging dir in user-data, got %q", userContent)
	}
	if !strings.Contains(userContent, "find /opt/deer-redpanda-root -type f -path '*/bin/redpanda' -perm -u+x") {
		t.Fatalf("expected redpanda binary discovery in user-data, got %q", userContent)
	}
	if !strings.Contains(userContent, "ln -sfn \"$redpanda_install_dir\" /opt/redpanda") {
		t.Fatalf("expected normalized /opt/redpanda install path in user-data, got %q", userContent)
	}
	if !strings.Contains(userContent, "redpanda_install_dir=/opt/redpanda") {
		t.Fatalf("expected normalized install dir assignment in user-data, got %q", userContent)
	}
	if strings.Contains(userContent, "redpanda_bin=\"$redpanda_install_dir/libexec/redpanda\"") {
		t.Fatalf("did not expect libexec redpanda runtime preference in user-data, got %q", userContent)
	}
	if strings.Contains(userContent, "rpk_bin=\"$redpanda_install_dir/libexec/rpk\"") {
		t.Fatalf("did not expect libexec rpk runtime preference in user-data, got %q", userContent)
	}
	if !strings.Contains(userContent, "find /opt/deer-redpanda-root -type f -path '*/bin/rpk' -perm -u+x") {
		t.Fatalf("expected rpk binary discovery in user-data, got %q", userContent)
	}
	if !strings.Contains(userContent, "cat >/etc/default/deer-redpanda <<EOF") {
		t.Fatalf("expected resolved runtime env file in user-data, got %q", userContent)
	}
	if !strings.Contains(userContent, "REDPANDA_LIBEXEC_BIN=$redpanda_libexec") || !strings.Contains(userContent, "RPK_LIBEXEC_BIN=$rpk_libexec") {
		t.Fatalf("expected libexec metadata in user-data, got %q", userContent)
	}
	if !strings.Contains(userContent, "REDPANDA_INSTALL_DIR=$redpanda_install_dir") {
		t.Fatalf("expected resolved redpanda install dir in user-data, got %q", userContent)
	}
	if !strings.Contains(userContent, "REDPANDA_LD_LIBRARY_PATH=$lib_dirs") {
		t.Fatalf("expected scoped redpanda library path in user-data, got %q", userContent)
	}
	if !strings.Contains(userContent, "requested_advertise_addr=\"10.0.0.15\"") {
		t.Fatalf("expected explicit advertise address handoff in user-data, got %q", userContent)
	}
	if !strings.Contains(userContent, "REDPANDA_ADVERTISE_ADDRESS=$requested_advertise_addr") {
		t.Fatalf("expected advertise address in env file, got %q", userContent)
	}
	if !strings.Contains(userContent, "timeout 10s /usr/bin/redpanda --version || true") {
		t.Fatalf("expected bounded redpanda version probe in user-data, got %q", userContent)
	}
	if strings.Contains(userContent, "timeout 10s /usr/bin/rpk --version || true") || strings.Contains(userContent, "\"${RPK_BIN}\" version") {
		t.Fatalf("did not expect rpk version execution in user-data, got %q", userContent)
	}
	if !strings.Contains(userContent, "resolved redpanda library path: $REDPANDA_LD_LIBRARY_PATH") {
		t.Fatalf("expected resolved redpanda library path output in user-data, got %q", userContent)
	}
	if !strings.Contains(userContent, "deer redpanda version probe start") || !strings.Contains(userContent, "deer redpanda version probe complete") {
		t.Fatalf("expected redpanda version probe markers in user-data, got %q", userContent)
	}
	if !strings.Contains(userContent, "deer rpk version probe skipped") {
		t.Fatalf("expected rpk version probe skip marker in user-data, got %q", userContent)
	}
	if !strings.Contains(userContent, "resolved /usr/bin/redpanda: $(readlink -f /usr/bin/redpanda || true)") {
		t.Fatalf("expected redpanda wrapper target logging in user-data, got %q", userContent)
	}
	if !strings.Contains(userContent, "resolved /usr/bin/rpk: $(readlink -f /usr/bin/rpk || true)") {
		t.Fatalf("expected rpk wrapper target logging in user-data, got %q", userContent)
	}
	if !strings.Contains(userContent, "resolve_advertise_addr() {") {
		t.Fatalf("expected runtime advertise-address resolver in user-data, got %q", userContent)
	}
	if !strings.Contains(userContent, "echo \"${REDPANDA_ADVERTISE_ADDRESS}\"") {
		t.Fatalf("expected explicit advertise-address echo in user-data, got %q", userContent)
	}
	if !strings.Contains(userContent, "echo \"${resolved}\"") {
		t.Fatalf("expected resolved advertise-address echo in user-data, got %q", userContent)
	}
	if !strings.Contains(userContent, "ip -o -4 route get 1.1.1.1") || !strings.Contains(userContent, "hostname -I 2>/dev/null | awk '{print $1}'") {
		t.Fatalf("expected guest IP fallback discovery in user-data, got %q", userContent)
	}
	if !strings.Contains(userContent, "resolved redpanda advertise address: ${advertise_addr}") {
		t.Fatalf("expected advertise-address diagnostics in user-data, got %q", userContent)
	}
	if !strings.Contains(userContent, "===== /usr/local/bin/deer-redpanda-start.sh =====") || !strings.Contains(userContent, "===== wrapper targets =====") || !strings.Contains(userContent, "===== /opt/redpanda/bin/redpanda =====") || !strings.Contains(userContent, "===== /opt/redpanda/bin/rpk =====") {
		t.Fatalf("expected wrapper diagnostics in user-data, got %q", userContent)
	}
	if !strings.Contains(userContent, "deer redpanda service start invoked") {
		t.Fatalf("expected service start progress marker in user-data, got %q", userContent)
	}
	if !strings.Contains(userContent, "deer redpanda systemd enable complete") {
		t.Fatalf("expected systemd enable progress marker in user-data, got %q", userContent)
	}
	if !strings.Contains(userContent, "deer redpanda readiness wait started") {
		t.Fatalf("expected readiness wait start marker in user-data, got %q", userContent)
	}
	if !strings.Contains(userContent, "deer redpanda readiness wait success") {
		t.Fatalf("expected readiness wait success marker in user-data, got %q", userContent)
	}
	if !strings.Contains(userContent, "deer redpanda readiness wait timeout") {
		t.Fatalf("expected readiness wait timeout marker in user-data, got %q", userContent)
	}
	if !strings.Contains(userContent, "service_state() {") {
		t.Fatalf("expected service_state helper in user-data, got %q", userContent)
	}
	if !strings.Contains(userContent, "Result --value") || !strings.Contains(userContent, "ExecMainStatus --value") || !strings.Contains(userContent, "ExecMainCode --value") || !strings.Contains(userContent, "ActiveState --value") || !strings.Contains(userContent, "NRestarts --value") {
		t.Fatalf("expected systemd failure-state inspection in user-data, got %q", userContent)
	}
	if !strings.Contains(userContent, "if [ \"${sub_state}\" = \"auto-restart\" ]") {
		t.Fatalf("expected auto-restart failure detection in user-data, got %q", userContent)
	}
	if !strings.Contains(userContent, "$((attempt % 15))") {
		t.Fatalf("expected periodic readiness diagnostics loop in user-data, got %q", userContent)
	}
	if !strings.Contains(userContent, "broker_ready() {") {
		t.Fatalf("expected broker readiness helper in user-data, got %q", userContent)
	}
	if !strings.Contains(userContent, "listener_ready() {") {
		t.Fatalf("expected shared listener readiness helper in user-data, got %q", userContent)
	}
	if !strings.Contains(userContent, "broker_port=9092") {
		t.Fatalf("expected broker port variable in user-data, got %q", userContent)
	}
	if !strings.Contains(userContent, `ss -H -ltn "( sport = :${broker_port} )" | awk 'END { exit(NR==0) }'`) {
		t.Fatalf("expected stable listener probe in user-data, got %q", userContent)
	}
	if !strings.Contains(userContent, "\"${RPK_BIN}\" cluster info --brokers 127.0.0.1:${broker_port}") {
		t.Fatalf("expected rpk cluster readiness probe in user-data, got %q", userContent)
	}
	if !strings.Contains(userContent, "note_pending_stage \"rpk_cluster_info\"") {
		t.Fatalf("expected readiness stage marker for cluster info in user-data, got %q", userContent)
	}
	if !strings.Contains(userContent, "deer redpanda readiness pending stage=${stage}") {
		t.Fatalf("expected dynamic readiness pending marker in user-data, got %q", userContent)
	}
	if strings.Contains(userContent, "if timeout 30s \"${RPK_BIN}\" topic list --brokers 127.0.0.1:9092 >/dev/null 2>&1; then") {
		t.Fatalf("did not expect topic-list readiness fallback in user-data, got %q", userContent)
	}
	if !strings.Contains(userContent, "\"${RPK_BIN}\" topic list --brokers 127.0.0.1:${broker_port} || true") {
		t.Fatalf("expected rpk topic-list diagnostics in user-data, got %q", userContent)
	}
	if !strings.Contains(userContent, "if test -x /usr/bin/redpanda && broker_ready; then") {
		t.Fatalf("expected strengthened redpanda readiness check in user-data, got %q", userContent)
	}
	if strings.Contains(userContent, "--memory 256M") {
		t.Fatalf("did not expect hardcoded 256M cap in user-data, got %q", userContent)
	}
	if strings.Contains(userContent, "--config /etc/redpanda/redpanda.yaml") {
		t.Fatalf("did not expect unsupported --config flag in user-data, got %q", userContent)
	}
	if !strings.Contains(userContent, "exec /usr/bin/rpk redpanda start --install-dir /opt/redpanda --mode dev-container --smp 1 --default-log-level=info") {
		t.Fatalf("expected wrapper-based rpk startup without config flag, got %q", userContent)
	}
	if !strings.Contains(userContent, "deer redpanda exec: /usr/bin/rpk redpanda start --install-dir /opt/redpanda --mode dev-container --smp 1 --default-log-level=info") {
		t.Fatalf("expected explicit rpk startup diagnostics in user-data, got %q", userContent)
	}
	if strings.Contains(userContent, "printf '%%s\\n'") {
		t.Fatalf("did not expect printf-based advertise-address output in user-data, got %q", userContent)
	}
	if strings.Contains(userContent, "%!s(MISSING)") || strings.Contains(userContent, "%!") {
		t.Fatalf("did not expect fmt formatting artifacts in user-data, got %q", userContent)
	}
	if strings.Contains(userContent, "env LD_LIBRARY_PATH") {
		t.Fatalf("did not expect direct LD_LIBRARY_PATH version probes in user-data, got %q", userContent)
	}
	if !strings.Contains(userContent, "Restart=on-failure") {
		t.Fatalf("expected bounded restart policy in user-data, got %q", userContent)
	}
	if !strings.Contains(userContent, "After=network-online.target\n      StartLimitIntervalSec=60\n      StartLimitBurst=5\n\n      [Service]") {
		t.Fatalf("expected start-limit settings under [Unit], got %q", userContent)
	}
	if strings.Contains(userContent, "RestartSec=5\n      StartLimitIntervalSec=60") {
		t.Fatalf("did not expect start-limit settings under [Service], got %q", userContent)
	}
	if !strings.Contains(userContent, "StartLimitIntervalSec=60") || !strings.Contains(userContent, "StartLimitBurst=5") {
		t.Fatalf("expected systemd start limit configuration in user-data, got %q", userContent)
	}
	if !strings.Contains(userContent, "if [ -n \"${REDPANDA_BIN:-}\" ]; then") {
		t.Fatalf("expected readiness phone-home guard in user-data, got %q", userContent)
	}
	if !strings.Contains(userContent, "deer phone_home start") {
		t.Fatalf("expected explicit phone-home start marker in user-data, got %q", userContent)
	}
	if !strings.Contains(userContent, "deer notify ready complete") {
		t.Fatalf("expected explicit notify completion marker in user-data, got %q", userContent)
	}
	if strings.Contains(userContent, "\nphone_home:") {
		t.Fatalf("did not expect built-in cloud-init phone_home stanza in user-data, got %q", userContent)
	}
	if !strings.Contains(userContent, "listener_ready") {
		t.Fatalf("expected phone-home to reuse the shared listener helper, got %q", userContent)
	}
	if strings.Contains(userContent, "ss -ltn | grep -q ':9092 '") {
		t.Fatalf("did not expect brittle ss|grep listener checks in user-data, got %q", userContent)
	}
}

func TestGenerateCloudInitISO_WithKafkaBrokerRuntimeAdvertiseAddress(t *testing.T) {
	workDir := t.TempDir()

	isoPath, err := GenerateCloudInitISO(workDir, "SBX-kafka-runtime-addr", CloudInitOptions{
		CAPubKey:     testCAPubKey,
		PhoneHomeURL: "http://10.0.0.1:9092/ready/SBX-kafka-runtime-addr",
		KafkaBroker: KafkaBrokerOptions{
			Enabled:    true,
			ArchiveURL: "http://10.0.0.1:9088/redpanda.tar.gz",
			Port:       9092,
		},
	})
	if err != nil {
		t.Fatalf("GenerateCloudInitISO: %v", err)
	}

	d, err := diskfs.Open(isoPath)
	if err != nil {
		t.Fatalf("open ISO: %v", err)
	}
	fs, err := d.GetFilesystem(0)
	if err != nil {
		t.Fatalf("get filesystem: %v", err)
	}
	isoFS := fs.(*iso9660.FileSystem)
	userFile, err := isoFS.OpenFile("/user-data", os.O_RDONLY)
	if err != nil {
		t.Fatalf("open user-data: %v", err)
	}
	userBytes, err := io.ReadAll(userFile)
	if err != nil {
		t.Fatalf("read user-data: %v", err)
	}
	userContent := string(userBytes)

	if !strings.Contains(userContent, "requested_advertise_addr=\"\"") {
		t.Fatalf("expected empty advertise-address handoff in user-data, got %q", userContent)
	}
	if !strings.Contains(userContent, "address: __FLUID_ADVERTISE_ADDRESS__") {
		t.Fatalf("expected runtime advertise-address placeholder in user-data, got %q", userContent)
	}
	if strings.Contains(userContent, "address: 127.0.0.1") {
		t.Fatalf("did not expect loopback advertised address in generated config, got %q", userContent)
	}
	if !strings.Contains(userContent, "echo \"${REDPANDA_ADVERTISE_ADDRESS}\"") {
		t.Fatalf("expected explicit advertise-address echo helper in user-data, got %q", userContent)
	}
	if !strings.Contains(userContent, "echo \"${resolved}\"") {
		t.Fatalf("expected resolved advertise-address echo helper in user-data, got %q", userContent)
	}
	if !strings.Contains(userContent, "sed -i \"s/__FLUID_ADVERTISE_ADDRESS__/${advertise_addr}/g\" /etc/redpanda/redpanda.yaml") {
		t.Fatalf("expected runtime advertise-address replacement in user-data, got %q", userContent)
	}
	if strings.Contains(userContent, "\nphone_home:") {
		t.Fatalf("did not expect built-in cloud-init phone_home stanza in runtime advertise-address config, got %q", userContent)
	}
	if strings.Contains(userContent, "printf '%%s\\n'") {
		t.Fatalf("did not expect printf-based advertise-address output in user-data, got %q", userContent)
	}
	if strings.Contains(userContent, "%!s(MISSING)") || strings.Contains(userContent, "%!") {
		t.Fatalf("did not expect fmt formatting artifacts in user-data, got %q", userContent)
	}
}
