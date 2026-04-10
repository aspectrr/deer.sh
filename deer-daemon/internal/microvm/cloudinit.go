package microvm

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	diskfs "github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/disk"
	"github.com/diskfs/go-diskfs/filesystem"
	"github.com/diskfs/go-diskfs/filesystem/iso9660"
)

const networkConfig = `version: 2
ethernets:
  all-en:
    match:
      name: "e*"
    dhcp4: true
`

type KafkaBrokerOptions struct {
	Enabled          bool
	AdvertiseAddress string
	ArchiveURL       string
	Port             int
}

type CloudInitOptions struct {
	CAPubKey         string
	PhoneHomeURL     string
	KafkaBroker      KafkaBrokerOptions
	RedpandaCacheURL string // file:// URL for local Redpanda tarball (faster than S3 download)
	Disable          bool   // If true, skip cloud-init ISO creation entirely (for pre-baked images)
}

// generateUserData builds cloud-init user-data YAML with the CA public key
// embedded so the sandbox VM trusts cert-based SSH auth.
// If PhoneHomeURL is non-empty, an explicit notify script is appended to
// runcmd so readiness is signaled only after the guest-side checks complete.
func generateUserData(opts CloudInitOptions) string {
	notifyPort := opts.KafkaBroker.Port
	if notifyPort == 0 {
		notifyPort = 9092
	}

	writeFiles := `  - path: /etc/ssh/authorized_principals/sandbox
    content: |
      sandbox
    owner: root:root
    permissions: '0644'
  - path: /etc/ssh/deer_ca.pub
    content: |
      %s
    owner: root:root
    permissions: '0644'
`
	runcmd := []string{
		"grep -q 'TrustedUserCAKeys /etc/ssh/deer_ca.pub' /etc/ssh/sshd_config || echo 'TrustedUserCAKeys /etc/ssh/deer_ca.pub' >> /etc/ssh/sshd_config",
		"grep -q 'AuthorizedPrincipalsFile /etc/ssh/authorized_principals/%u' /etc/ssh/sshd_config || echo 'AuthorizedPrincipalsFile /etc/ssh/authorized_principals/%u' >> /etc/ssh/sshd_config",
		"systemctl restart sshd 2>/dev/null || systemctl restart ssh 2>/dev/null || service sshd restart 2>/dev/null || service ssh restart",
	}
	if opts.PhoneHomeURL != "" {
		writeFiles += fmt.Sprintf(`  - path: /usr/local/bin/fluid-notify-ready.sh
    content: |
      #!/bin/bash
      set -euo pipefail
      mkdir -p /var/log/fluid
      exec > >(tee -a /var/log/fluid/notify-ready.log /dev/console) 2>&1
      set -x
      export DEBIAN_FRONTEND=noninteractive
      broker_port=%d
      fail_stage() {
        local stage="$1"
        echo "fluid notify ready failure stage=${stage} $(date -Is)" >&2
        /usr/local/bin/fluid-redpanda-diagnostics.sh || true
        exit 1
      }
      listener_ready() {
        ss -H -ltn "( sport = :${broker_port} )" | awk 'END { exit(NR==0) }'
      }
      echo "fluid notify ready start $(date -Is)"
      if [ -f /etc/default/fluid-redpanda ]; then
        . /etc/default/fluid-redpanda
      fi
      if [ -n "${REDPANDA_BIN:-}" ]; then
        test -x "${REDPANDA_BIN}"
        systemctl is-enabled --quiet fluid-redpanda.service
        systemctl is-active --quiet fluid-redpanda.service
        listener_ready
      fi
      echo "fluid notify ready checks complete $(date -Is)"
      if ! command -v curl >/dev/null 2>&1; then
        apt-get update
        apt-get install -y ca-certificates curl
      fi
      echo "fluid phone_home start $(date -Is)"
      if ! timeout 1m curl --connect-timeout 10 --max-time 30 -fsS -X POST %q; then
        fail_stage "phone_home"
      fi
      echo "fluid notify ready complete $(date -Is)"
    owner: root:root
    permissions: '0755'
`, notifyPort, opts.PhoneHomeURL)
	}

	if opts.KafkaBroker.Enabled {
		port := opts.KafkaBroker.Port
		if port == 0 {
			port = 9092
		}
		advertiseAddr := opts.KafkaBroker.AdvertiseAddress
		configAdvertiseAddr := advertiseAddr
		if configAdvertiseAddr == "" {
			configAdvertiseAddr = "__FLUID_ADVERTISE_ADDRESS__"
		}
		archiveURL := opts.KafkaBroker.ArchiveURL
		if opts.RedpandaCacheURL != "" {
			archiveURL = opts.RedpandaCacheURL
		} else if archiveURL == "" {
			archiveURL = defaultRedpandaArchiveURL()
		}
		writeFiles += fmt.Sprintf(`  - path: /etc/redpanda/redpanda.yaml
    content: |
      redpanda:
        data_directory: /var/lib/redpanda/data
        empty_seed_starts_cluster: true
        rpc_server:
          address: 0.0.0.0
          port: 33145
        advertised_rpc_api:
          address: %s
          port: 33145
        kafka_api:
          - name: internal
            address: 0.0.0.0
            port: %d
        advertised_kafka_api:
          - name: internal
            address: %s
            port: %d
        admin:
          - address: 0.0.0.0
            port: 9644
    owner: root:root
    permissions: '0644'
  - path: /usr/local/bin/fluid-redpanda-diagnostics.sh
    content: |
      #!/bin/bash
      set +e
      broker_port=%d
      echo "fluid redpanda diagnostics start $(date -Is)"
      for path in \
        /var/log/fluid/redpanda-install.log \
        /var/log/fluid/redpanda-enable.log \
        /var/log/fluid/redpanda-start.log \
        /var/log/fluid/redpanda-wait.log \
        /var/log/fluid/notify-ready.log; do
        if [ -f "$path" ]; then
          echo "===== $path ====="
          cat "$path"
        fi
      done
      echo "===== systemctl status fluid-redpanda.service ====="
      systemctl status fluid-redpanda.service --no-pager || true
      echo "===== systemctl show fluid-redpanda.service (result/exit/restarts) ====="
      systemctl show fluid-redpanda.service --property=Result --property=ExecMainStatus --property=SubState --property=NRestarts --no-pager || true
      echo "===== systemctl show fluid-redpanda.service ====="
      systemctl show fluid-redpanda.service --no-pager || true
      echo "===== ss -H -ltn ( sport = :${broker_port} ) ====="
      ss -H -ltn "( sport = :${broker_port} )" || true
      echo "===== ss -H -ltnp ( sport = :${broker_port} ) ====="
      ss -H -ltnp "( sport = :${broker_port} )" || true
      echo "===== cloud-init status --long ====="
      cloud-init status --long || true
      echo "===== nproc ====="
      nproc || true
      echo "===== free -m ====="
      free -m || true
      echo "===== /proc/meminfo ====="
      cat /proc/meminfo || true
      echo "===== journalctl -u fluid-redpanda.service ====="
      journalctl -u fluid-redpanda.service --no-pager -n 200 || true
      echo "===== journalctl -u cloud-final ====="
      journalctl -u cloud-final --no-pager -n 200 || true
      echo "===== journalctl -k ====="
      journalctl -k --no-pager -n 200 || true
      echo "===== /etc/default/fluid-redpanda ====="
      cat /etc/default/fluid-redpanda || true
      if [ -f /etc/default/fluid-redpanda ]; then
        . /etc/default/fluid-redpanda
      fi
      if [ -n "${RPK_BIN:-}" ] && [ -x "${RPK_BIN:-}" ]; then
        echo "===== rpk cluster info ====="
        timeout 10s "${RPK_BIN}" cluster info --brokers 127.0.0.1:${broker_port} || true
        echo "===== rpk topic list ====="
        timeout 10s "${RPK_BIN}" topic list --brokers 127.0.0.1:${broker_port} || true
      fi
      echo "===== /usr/local/bin/fluid-redpanda-start.sh ====="
      sed -n '1,60p' /usr/local/bin/fluid-redpanda-start.sh || true
      echo "===== wrapper targets ====="
      readlink -f /usr/bin/redpanda || true
      readlink -f /usr/bin/rpk || true
      echo "===== /etc/redpanda/redpanda.yaml ====="
      cat /etc/redpanda/redpanda.yaml || true
      echo "===== /opt/redpanda/bin/redpanda ====="
      sed -n '1,20p' /opt/redpanda/bin/redpanda || true
      echo "===== /opt/redpanda/bin/rpk ====="
      sed -n '1,20p' /opt/redpanda/bin/rpk || true
      echo "===== ss -ltn ====="
      ss -ltn || true
      echo "===== redpanda log files ====="
      find /var/log /var/lib/redpanda -maxdepth 4 -type f \( -iname '*redpanda*.log' -o -iname '*redpanda*' -o -path '/var/log/fluid/*' \) 2>/dev/null | sort | while read -r log_path; do
        echo "===== $log_path ====="
        cat "$log_path" || true
      done
      echo "fluid redpanda diagnostics complete $(date -Is)"
    owner: root:root
    permissions: '0755'
  - path: /usr/local/bin/fluid-install-redpanda.sh
    content: |
      #!/bin/bash
      set -euo pipefail
      mkdir -p /var/log/fluid
      exec > >(tee -a /var/log/fluid/redpanda-install.log /dev/console) 2>&1
      set -x
      export DEBIAN_FRONTEND=noninteractive
      fail_stage() {
        local stage="$1"
        echo "fluid redpanda install failure stage=${stage} $(date -Is)" >&2
        /usr/local/bin/fluid-redpanda-diagnostics.sh || true
        exit 1
      }
      echo "fluid redpanda install start $(date -Is)"
      df -h / /opt /var/tmp || true
      if ! command -v curl >/dev/null 2>&1; then
        apt-get update
        apt-get install -y ca-certificates curl
      fi
      tmpdir=$(mktemp -d /var/tmp/fluid-redpanda.XXXXXX)
      archive_url=%q
      requested_advertise_addr=%q
      archive_path="$tmpdir/redpanda.tar.gz"
      if ! curl --connect-timeout 10 --max-time 600 -fsSL --retry 5 --retry-delay 2 -o "$archive_path" "$archive_url"; then
        fail_stage "archive_download"
      fi
      echo "fluid redpanda archive download complete $(date -Is)"
      extract_root="$tmpdir/extract"
      rm -rf "$extract_root" /opt/fluid-redpanda-root
      install -d -m 0755 "$extract_root" /opt/fluid-redpanda-root
      redpanda_bin=""
      rpk_bin=""
      redpanda_install_dir=""
      lib_dirs=""
      if tar -tzf "$archive_path" | grep -Eq '^(\./)?(usr/bin/redpanda|opt/redpanda/)'; then
        if ! timeout 15m tar -xzf "$archive_path" -C /; then
          fail_stage "archive_extract"
        fi
        redpanda_bin=/usr/bin/redpanda
        if [ -x /usr/bin/rpk ]; then
          rpk_bin=/usr/bin/rpk
        fi
      else
        if ! timeout 15m tar -xzf "$archive_path" -C "$extract_root"; then
          fail_stage "archive_extract"
        fi
        if ! cp -a "$extract_root"/. /opt/fluid-redpanda-root/; then
          fail_stage "archive_stage_copy"
        fi
        redpanda_bin=$(find /opt/fluid-redpanda-root -type f -path '*/bin/redpanda' -perm -u+x | head -n1)
        rpk_bin=$(find /opt/fluid-redpanda-root -type f -path '*/bin/rpk' -perm -u+x | head -n1)
        if [ -z "$redpanda_bin" ]; then
          redpanda_bin=$(find /opt/fluid-redpanda-root -type f -name redpanda -perm -u+x | head -n1)
        fi
        if [ -z "$rpk_bin" ]; then
          rpk_bin=$(find /opt/fluid-redpanda-root -type f -name rpk -perm -u+x | head -n1)
        fi
        if [ -n "$redpanda_bin" ]; then
          case "$redpanda_bin" in
            */bin/*)
              redpanda_install_dir=$(CDPATH= cd -- "$(dirname "$redpanda_bin")/.." && pwd)
              ;;
          esac
          lib_dirs=$(find /opt/fluid-redpanda-root -type d \( -name lib -o -name lib64 \) ! -path '*/var/lib*' | sort -u | paste -sd: -)
          if [ -z "$lib_dirs" ]; then
            lib_dirs=$(dirname "$redpanda_bin")
          fi
        fi
      fi
      echo "fluid redpanda extraction complete $(date -Is)"
      if [ -z "$redpanda_bin" ] || [ ! -x "$redpanda_bin" ]; then
        echo "redpanda binary not found after extracting $archive_url" >&2
        find "$extract_root" -maxdepth 6 -type f | sort >&2 || true
        fail_stage "binary_resolution"
      fi
      if [ -n "$redpanda_install_dir" ] && [ "$redpanda_install_dir" != "/opt/redpanda" ]; then
        ln -sfn "$redpanda_install_dir" /opt/redpanda
        redpanda_install_dir=/opt/redpanda
      elif [ -z "$redpanda_install_dir" ] && [ -d /opt/redpanda ]; then
        redpanda_install_dir=/opt/redpanda
      fi
      if [ -n "$redpanda_install_dir" ] && [ -x "$redpanda_install_dir/bin/redpanda" ]; then
        redpanda_bin="$redpanda_install_dir/bin/redpanda"
      fi
      if [ -n "$redpanda_install_dir" ] && [ -x "$redpanda_install_dir/bin/rpk" ]; then
        rpk_bin="$redpanda_install_dir/bin/rpk"
      fi
      redpanda_libexec=""
      rpk_libexec=""
      if [ -n "$redpanda_install_dir" ] && [ -x "$redpanda_install_dir/libexec/redpanda" ]; then
        redpanda_libexec="$redpanda_install_dir/libexec/redpanda"
      fi
      if [ -n "$redpanda_install_dir" ] && [ -x "$redpanda_install_dir/libexec/rpk" ]; then
        rpk_libexec="$redpanda_install_dir/libexec/rpk"
      fi
      if [ -n "$redpanda_install_dir" ] && [ -d "$redpanda_install_dir/lib" ]; then
        lib_dirs="$redpanda_install_dir/lib"
      fi
      echo "fluid redpanda binary resolution complete $(date -Is)"
      cat >/etc/default/fluid-redpanda <<EOF
      REDPANDA_BIN=$redpanda_bin
      RPK_BIN=$rpk_bin
      REDPANDA_INSTALL_DIR=$redpanda_install_dir
      REDPANDA_LD_LIBRARY_PATH=$lib_dirs
      REDPANDA_LIBEXEC_BIN=$redpanda_libexec
      RPK_LIBEXEC_BIN=$rpk_libexec
      REDPANDA_ADVERTISE_ADDRESS=$requested_advertise_addr
      EOF
      echo "fluid redpanda env file written $(date -Is)"
      if [ "$redpanda_bin" = "/usr/bin/redpanda" ] && [ -n "$redpanda_install_dir" ] && [ -x "$redpanda_install_dir/bin/redpanda" ]; then
        redpanda_bin="$redpanda_install_dir/bin/redpanda"
      fi
      if [ "$rpk_bin" = "/usr/bin/rpk" ] && [ -n "$redpanda_install_dir" ] && [ -x "$redpanda_install_dir/bin/rpk" ]; then
        rpk_bin="$redpanda_install_dir/bin/rpk"
      fi
      if [ -n "$redpanda_install_dir" ] && [ -x "$redpanda_install_dir/bin/redpanda" ] && [ "$redpanda_install_dir/bin/redpanda" != "/usr/bin/redpanda" ]; then
        ln -sf "$redpanda_install_dir/bin/redpanda" /usr/bin/redpanda
      elif [ "$redpanda_bin" != "/usr/bin/redpanda" ]; then
        ln -sf "$redpanda_bin" /usr/bin/redpanda
      fi
      if [ -n "$redpanda_install_dir" ] && [ -x "$redpanda_install_dir/bin/rpk" ] && [ "$redpanda_install_dir/bin/rpk" != "/usr/bin/rpk" ]; then
        ln -sf "$redpanda_install_dir/bin/rpk" /usr/bin/rpk
      elif [ -n "$rpk_bin" ] && [ "$rpk_bin" != "/usr/bin/rpk" ]; then
        ln -sf "$rpk_bin" /usr/bin/rpk
      fi
      . /etc/default/fluid-redpanda
      if ! command -v redpanda >/dev/null 2>&1; then
        fail_stage "binary_resolution"
      fi
      echo "fluid redpanda version probe start $(date -Is)"
      if test -x /usr/bin/redpanda; then
        timeout 10s /usr/bin/redpanda --version || true
      fi
      echo "fluid redpanda version probe complete $(date -Is)"
      echo "fluid rpk version probe skipped $(date -Is)"
      echo "resolved /usr/bin/redpanda: $(readlink -f /usr/bin/redpanda || true)"
      echo "resolved /usr/bin/rpk: $(readlink -f /usr/bin/rpk || true)"
      echo "resolved redpanda bin: $REDPANDA_BIN"
      echo "resolved rpk bin: $RPK_BIN"
      echo "resolved redpanda install dir: $REDPANDA_INSTALL_DIR"
      echo "resolved redpanda library path: $REDPANDA_LD_LIBRARY_PATH"
      df -h / /opt /var/tmp || true
      echo "fluid redpanda install complete $(date -Is)"
      echo "fluid redpanda temp cleanup skipped for ephemeral sandbox $(date -Is)"
    owner: root:root
    permissions: '0755'
  - path: /usr/local/bin/fluid-enable-redpanda.sh
    content: |
      #!/bin/bash
      set -euo pipefail
      mkdir -p /var/log/fluid
      exec > >(tee -a /var/log/fluid/redpanda-enable.log /dev/console) 2>&1
      set -x
      fail_stage() {
        local stage="$1"
        echo "fluid redpanda enable failure stage=${stage} $(date -Is)" >&2
        /usr/local/bin/fluid-redpanda-diagnostics.sh || true
        exit 1
      }
      echo "fluid redpanda enable start $(date -Is)"
      if ! timeout 2m systemctl daemon-reload; then
        fail_stage "daemon_reload"
      fi
      echo "fluid redpanda daemon reload complete $(date -Is)"
      echo "fluid redpanda service start invoked $(date -Is)"
      if ! timeout 10m systemctl enable --now fluid-redpanda.service; then
        fail_stage "systemd_enable_start"
      fi
      echo "fluid redpanda systemd enable complete $(date -Is)"
    owner: root:root
    permissions: '0755'
  - path: /usr/local/bin/fluid-redpanda-start.sh
    content: |
      #!/bin/bash
      set -euo pipefail
      mkdir -p /var/log/fluid
      exec > >(tee -a /var/log/fluid/redpanda-start.log /dev/console) 2>&1
      set -x
      echo "fluid redpanda start wrapper $(date -Is)"
      . /etc/default/fluid-redpanda
      resolve_advertise_addr() {
        if [ -n "${REDPANDA_ADVERTISE_ADDRESS:-}" ]; then
          echo "${REDPANDA_ADVERTISE_ADDRESS}"
          return 0
        fi
        local resolved
        resolved=$(ip -o -4 route get 1.1.1.1 2>/dev/null | awk '{for (i = 1; i <= NF; i++) if ($i == "src") {print $(i+1); exit}}')
        if [ -z "${resolved}" ]; then
          resolved=$(hostname -I 2>/dev/null | awk '{print $1}')
        fi
        echo "${resolved}"
      }
      echo "fluid redpanda version probe start $(date -Is)"
      if test -x /usr/bin/redpanda; then
        timeout 10s /usr/bin/redpanda --version || true
      fi
      echo "fluid redpanda version probe complete $(date -Is)"
      echo "fluid rpk version probe skipped $(date -Is)"
      echo "resolved /usr/bin/redpanda: $(readlink -f /usr/bin/redpanda || true)"
      echo "resolved /usr/bin/rpk: $(readlink -f /usr/bin/rpk || true)"
      advertise_addr=$(resolve_advertise_addr)
      if [ -n "${advertise_addr}" ] && [ -f /etc/redpanda/redpanda.yaml ]; then
        sed -i "s/__FLUID_ADVERTISE_ADDRESS__/${advertise_addr}/g" /etc/redpanda/redpanda.yaml
      fi
      echo "resolved redpanda advertise address: ${advertise_addr}"
      echo "fluid redpanda exec: /usr/bin/rpk redpanda start --install-dir /opt/redpanda --mode dev-container --smp 1 --default-log-level=info"
      exec /usr/bin/rpk redpanda start --install-dir /opt/redpanda --mode dev-container --smp 1 --default-log-level=info
    owner: root:root
    permissions: '0755'
  - path: /usr/local/bin/fluid-wait-redpanda.sh
    content: |
      #!/bin/bash
      set -euo pipefail
      mkdir -p /var/log/fluid
      exec > >(tee -a /var/log/fluid/redpanda-wait.log /dev/console) 2>&1
      set -x
      broker_port=%d
      fail_stage() {
        local stage="$1"
        echo "fluid redpanda readiness failure stage=${stage} $(date -Is)" >&2
        /usr/local/bin/fluid-redpanda-diagnostics.sh || true
        exit 1
      }
      listener_ready() {
        ss -H -ltn "( sport = :${broker_port} )" | awk 'END { exit(NR==0) }'
      }
      readiness_pending_stage=""
      note_pending_stage() {
        local stage="$1"
        if [ "${readiness_pending_stage}" != "${stage}" ]; then
          readiness_pending_stage="${stage}"
          echo "fluid redpanda readiness pending stage=${stage} $(date -Is)"
        fi
      }
      service_state() {
        local result exec_status exec_code active_state sub_state n_restarts
        result=$(systemctl show fluid-redpanda.service --property=Result --value 2>/dev/null || true)
        exec_status=$(systemctl show fluid-redpanda.service --property=ExecMainStatus --value 2>/dev/null || true)
        exec_code=$(systemctl show fluid-redpanda.service --property=ExecMainCode --value 2>/dev/null || true)
        active_state=$(systemctl show fluid-redpanda.service --property=ActiveState --value 2>/dev/null || true)
        sub_state=$(systemctl show fluid-redpanda.service --property=SubState --value 2>/dev/null || true)
        n_restarts=$(systemctl show fluid-redpanda.service --property=NRestarts --value 2>/dev/null || true)
        echo "fluid redpanda service_state active_state=${active_state} result=${result} sub_state=${sub_state} exec_main_code=${exec_code} exec_main_status=${exec_status} n_restarts=${n_restarts}"
        if [ "${sub_state}" = "failed" ] && [ -n "${exec_status}" ] && [ "${exec_status}" != "0" ]; then
          return 0
        fi
        if [ "${sub_state}" = "auto-restart" ] && [ -n "${exec_status}" ] && [ "${exec_status}" != "0" ]; then
          return 0
        fi
        if [ "${result}" = "exit-code" ] && [ -n "${exec_status}" ] && [ "${exec_status}" != "0" ]; then
          return 0
        fi
        if [ -n "${n_restarts}" ] && [ "${n_restarts}" -ge 5 ]; then
          return 0
        fi
        return 1
      }
      broker_ready() {
        if ! systemctl is-enabled --quiet fluid-redpanda.service; then
          note_pending_stage "service_enabled"
          return 1
        fi
        if ! systemctl is-active --quiet fluid-redpanda.service; then
          note_pending_stage "service_active"
          return 1
        fi
        if ! listener_ready; then
          note_pending_stage "listener_ready"
          return 1
        fi
        if [ -n "${RPK_BIN:-}" ] && [ -x "${RPK_BIN:-}" ]; then
          if ! timeout 10s "${RPK_BIN}" cluster info --brokers 127.0.0.1:${broker_port} >/dev/null 2>&1; then
            note_pending_stage "rpk_cluster_info"
            return 1
          fi
        fi
        readiness_pending_stage=""
        return 0
      }
      echo "fluid redpanda readiness wait started $(date -Is)"
      for attempt in $(seq 1 180); do
        if test -x /usr/bin/redpanda && broker_ready; then
          echo "fluid redpanda readiness wait success $(date -Is)"
          echo "fluid redpanda ready on attempt ${attempt} $(date -Is)"
          exit 0
        fi
        if service_state; then
          fail_stage "systemd_start"
        fi
        if systemctl is-failed --quiet fluid-redpanda.service; then
          fail_stage "systemd_start"
        fi
        if [ $((attempt %%%% 15)) -eq 0 ]; then
          if [ -n "${readiness_pending_stage}" ]; then
            echo "fluid redpanda readiness pending stage=${readiness_pending_stage} $(date -Is)"
          fi
          systemctl status fluid-redpanda.service --no-pager || true
          systemctl show fluid-redpanda.service --property=Result --property=ExecMainStatus --property=SubState --property=NRestarts --no-pager || true
          ss -ltn || true
          ss -H -ltn "( sport = :${broker_port} )" || true
        fi
        sleep 2
      done
      echo "fluid redpanda readiness wait timeout $(date -Is)"
      fail_stage "wait_timeout"
    owner: root:root
    permissions: '0755'
  - path: /etc/systemd/system/fluid-redpanda.service
    content: |
      [Unit]
      Description=Fluid Redpanda Broker
      Wants=network-online.target
      After=network-online.target
      StartLimitIntervalSec=60
      StartLimitBurst=5

      [Service]
      Type=simple
      EnvironmentFile=-/etc/default/fluid-redpanda
      ExecStart=/usr/local/bin/fluid-redpanda-start.sh
      StandardOutput=journal+console
      StandardError=journal+console
      Restart=on-failure
      RestartSec=5

      [Install]
      WantedBy=multi-user.target
    owner: root:root
    permissions: '0644'
`, configAdvertiseAddr, port, configAdvertiseAddr, port, port, archiveURL, advertiseAddr, port)
		runcmd = append(runcmd,
			"mkdir -p /etc/redpanda /var/lib/redpanda/data",
			"/usr/local/bin/fluid-install-redpanda.sh",
			"/usr/local/bin/fluid-enable-redpanda.sh",
			"/usr/local/bin/fluid-wait-redpanda.sh",
		)
	}
	if opts.PhoneHomeURL != "" {
		runcmd = append(runcmd, "/usr/local/bin/fluid-notify-ready.sh")
	}

	var runcmdBuilder strings.Builder
	for _, cmd := range runcmd {
		runcmdBuilder.WriteString("  - ")
		runcmdBuilder.WriteString(cmd)
		runcmdBuilder.WriteString("\n")
	}

	return fmt.Sprintf(`#cloud-config
users:
  - default
  - name: sandbox
    shell: /bin/bash
    sudo: ALL=(ALL) NOPASSWD:ALL
    lock_passwd: true

growpart:
  mode: auto
  devices: ['/']
resize_rootfs: true

write_files:
%s

runcmd:
%s
`, fmt.Sprintf(writeFiles, opts.CAPubKey), runcmdBuilder.String())
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
// If PhoneHomeURL is non-empty, an explicit notify script is added to runcmd
// that POSTs to the daemon readiness endpoint after guest-side checks complete.
func GenerateCloudInitISO(workDir, sandboxID string, opts CloudInitOptions) (string, error) {
	if opts.Disable {
		return "", nil
	}
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

	files := map[string]string{
		"/meta-data":      metaData,
		"/network-config": networkConfig,
		"/user-data":      generateUserData(opts),
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

func defaultRedpandaArchiveURL() string {
	if runtime.GOARCH == "arm64" {
		return "https://vectorized-public.s3.us-west-2.amazonaws.com/releases/redpanda/25.2.7/redpanda-25.2.7-arm64.tar.gz"
	}
	return "https://vectorized-public.s3.us-west-2.amazonaws.com/releases/redpanda/25.2.7/redpanda-25.2.7-amd64.tar.gz"
}
