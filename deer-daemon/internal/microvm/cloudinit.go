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

type ElasticsearchBrokerOptions struct {
	Enabled    bool
	Port       int
	ArchiveURL string
}

type CloudInitOptions struct {
	CAPubKey            string
	PhoneHomeURL        string
	KafkaBroker         KafkaBrokerOptions
	ElasticsearchBroker ElasticsearchBrokerOptions
	RedpandaCacheURL    string // file:// URL for local Redpanda tarball (faster than S3 download)
	Disable             bool   // If true, skip cloud-init ISO creation entirely (for pre-baked images)
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
	esPort := opts.ElasticsearchBroker.Port
	if esPort == 0 {
		esPort = 9200
	}

	runcmd := []string{
		"grep -q 'TrustedUserCAKeys /etc/ssh/deer_ca.pub' /etc/ssh/sshd_config || echo 'TrustedUserCAKeys /etc/ssh/deer_ca.pub' >> /etc/ssh/sshd_config",
		"grep -q 'AuthorizedPrincipalsFile /etc/ssh/authorized_principals/%u' /etc/ssh/sshd_config || echo 'AuthorizedPrincipalsFile /etc/ssh/authorized_principals/%u' >> /etc/ssh/sshd_config",
		"systemctl restart sshd 2>/dev/null || systemctl restart ssh 2>/dev/null || service sshd restart 2>/dev/null || service ssh restart",
	}
	if opts.PhoneHomeURL != "" {
		writeFiles += fmt.Sprintf(`  - path: /usr/local/bin/deer-notify-ready.sh
    content: |
      #!/bin/bash
      set -euo pipefail
      mkdir -p /var/log/deer
      exec > >(tee -a /var/log/deer/notify-ready.log /dev/console) 2>&1
      set -x
      export DEBIAN_FRONTEND=noninteractive
      broker_port=%d
      es_port=%d
      fail_stage() {
        local stage="$1"
        echo "deer notify ready failure stage=${stage} $(date -Is)" >&2
        /usr/local/bin/deer-redpanda-diagnostics.sh || true
        exit 1
      }
      listener_ready() {
        ss -H -ltn "( sport = :${broker_port} )" | awk 'END { exit(NR==0) }'
      }
      echo "deer notify ready start $(date -Is)"
      if [ -f /etc/default/deer-redpanda ]; then
        . /etc/default/deer-redpanda
      fi
      if [ -n "${REDPANDA_BIN:-}" ]; then
        test -x "${REDPANDA_BIN}"
        systemctl is-enabled --quiet deer-redpanda.service
        systemctl is-active --quiet deer-redpanda.service
        listener_ready
      fi
      if [ -f /etc/default/deer-elasticsearch ] && systemctl is-enabled --quiet deer-elasticsearch.service 2>/dev/null; then
        echo "deer elasticsearch readiness check start $(date -Is)"
        if ! timeout 5m curl -sf "http://localhost:${es_port}/_cluster/health" >/dev/null 2>&1; then
          echo "deer elasticsearch readiness check failed $(date -Is)" >&2
          journalctl -u deer-elasticsearch.service --no-pager -n 50 || true
          fail_stage "elasticsearch_readiness"
        fi
        echo "deer elasticsearch readiness check complete $(date -Is)"
      fi
      echo "deer notify ready checks complete $(date -Is)"
      if ! command -v curl >/dev/null 2>&1; then
        apt-get update
        apt-get install -y ca-certificates curl
      fi
      echo "deer phone_home start $(date -Is)"
      if ! timeout 1m curl --connect-timeout 10 --max-time 30 -fsS -X POST %q; then
        fail_stage "phone_home"
      fi
      echo "deer notify ready complete $(date -Is)"
    owner: root:root
    permissions: '0755'
`, notifyPort, esPort, opts.PhoneHomeURL)
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
  - path: /usr/local/bin/deer-redpanda-diagnostics.sh
    content: |
      #!/bin/bash
      set +e
      broker_port=%d
      echo "deer redpanda diagnostics start $(date -Is)"
      for path in \
        /var/log/deer/redpanda-install.log \
        /var/log/deer/redpanda-enable.log \
        /var/log/deer/redpanda-start.log \
        /var/log/deer/redpanda-wait.log \
        /var/log/deer/notify-ready.log; do
        if [ -f "$path" ]; then
          echo "===== $path ====="
          cat "$path"
        fi
      done
      echo "===== systemctl status deer-redpanda.service ====="
      systemctl status deer-redpanda.service --no-pager || true
      echo "===== systemctl show deer-redpanda.service (result/exit/restarts) ====="
      systemctl show deer-redpanda.service --property=Result --property=ExecMainStatus --property=SubState --property=NRestarts --no-pager || true
      echo "===== systemctl show deer-redpanda.service ====="
      systemctl show deer-redpanda.service --no-pager || true
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
      echo "===== journalctl -u deer-redpanda.service ====="
      journalctl -u deer-redpanda.service --no-pager -n 200 || true
      echo "===== journalctl -u cloud-final ====="
      journalctl -u cloud-final --no-pager -n 200 || true
      echo "===== journalctl -k ====="
      journalctl -k --no-pager -n 200 || true
      echo "===== /etc/default/deer-redpanda ====="
      cat /etc/default/deer-redpanda || true
      if [ -f /etc/default/deer-redpanda ]; then
        . /etc/default/deer-redpanda
      fi
      if [ -n "${RPK_BIN:-}" ] && [ -x "${RPK_BIN:-}" ]; then
        echo "===== rpk cluster info ====="
        timeout 10s "${RPK_BIN}" cluster info --brokers 127.0.0.1:${broker_port} || true
        echo "===== rpk topic list ====="
        timeout 10s "${RPK_BIN}" topic list --brokers 127.0.0.1:${broker_port} || true
      fi
      echo "===== /usr/local/bin/deer-redpanda-start.sh ====="
      sed -n '1,60p' /usr/local/bin/deer-redpanda-start.sh || true
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
      find /var/log /var/lib/redpanda -maxdepth 4 -type f \( -iname '*redpanda*.log' -o -iname '*redpanda*' -o -path '/var/log/deer/*' \) 2>/dev/null | sort | while read -r log_path; do
        echo "===== $log_path ====="
        cat "$log_path" || true
      done
      echo "deer redpanda diagnostics complete $(date -Is)"
    owner: root:root
    permissions: '0755'
  - path: /usr/local/bin/deer-install-redpanda.sh
    content: |
      #!/bin/bash
      set -euo pipefail
      mkdir -p /var/log/deer
      exec > >(tee -a /var/log/deer/redpanda-install.log /dev/console) 2>&1
      set -x
      export DEBIAN_FRONTEND=noninteractive
      fail_stage() {
        local stage="$1"
        echo "deer redpanda install failure stage=${stage} $(date -Is)" >&2
        /usr/local/bin/deer-redpanda-diagnostics.sh || true
        exit 1
      }
      echo "deer redpanda install start $(date -Is)"
      df -h / /opt /var/tmp || true
      if ! command -v curl >/dev/null 2>&1; then
        apt-get update
        apt-get install -y ca-certificates curl
      fi
      tmpdir=$(mktemp -d /var/tmp/deer-redpanda.XXXXXX)
      archive_url=%q
      requested_advertise_addr=%q
      archive_path="$tmpdir/redpanda.tar.gz"
      if ! curl --connect-timeout 10 --max-time 600 -fsSL --retry 5 --retry-delay 2 -o "$archive_path" "$archive_url"; then
        fail_stage "archive_download"
      fi
      echo "deer redpanda archive download complete $(date -Is)"
      extract_root="$tmpdir/extract"
      rm -rf "$extract_root" /opt/deer-redpanda-root
      install -d -m 0755 "$extract_root" /opt/deer-redpanda-root
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
        if ! cp -a "$extract_root"/. /opt/deer-redpanda-root/; then
          fail_stage "archive_stage_copy"
        fi
        redpanda_bin=$(find /opt/deer-redpanda-root -type f -path '*/bin/redpanda' -perm -u+x | head -n1)
        rpk_bin=$(find /opt/deer-redpanda-root -type f -path '*/bin/rpk' -perm -u+x | head -n1)
        if [ -z "$redpanda_bin" ]; then
          redpanda_bin=$(find /opt/deer-redpanda-root -type f -name redpanda -perm -u+x | head -n1)
        fi
        if [ -z "$rpk_bin" ]; then
          rpk_bin=$(find /opt/deer-redpanda-root -type f -name rpk -perm -u+x | head -n1)
        fi
        if [ -n "$redpanda_bin" ]; then
          case "$redpanda_bin" in
            */bin/*)
              redpanda_install_dir=$(CDPATH= cd -- "$(dirname "$redpanda_bin")/.." && pwd)
              ;;
          esac
          lib_dirs=$(find /opt/deer-redpanda-root -type d \( -name lib -o -name lib64 \) ! -path '*/var/lib*' | sort -u | paste -sd: -)
          if [ -z "$lib_dirs" ]; then
            lib_dirs=$(dirname "$redpanda_bin")
          fi
        fi
      fi
      echo "deer redpanda extraction complete $(date -Is)"
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
      echo "deer redpanda binary resolution complete $(date -Is)"
      cat >/etc/default/deer-redpanda <<EOF
      REDPANDA_BIN=$redpanda_bin
      RPK_BIN=$rpk_bin
      REDPANDA_INSTALL_DIR=$redpanda_install_dir
      REDPANDA_LD_LIBRARY_PATH=$lib_dirs
      REDPANDA_LIBEXEC_BIN=$redpanda_libexec
      RPK_LIBEXEC_BIN=$rpk_libexec
      REDPANDA_ADVERTISE_ADDRESS=$requested_advertise_addr
      EOF
      echo "deer redpanda env file written $(date -Is)"
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
      . /etc/default/deer-redpanda
      if ! command -v redpanda >/dev/null 2>&1; then
        fail_stage "binary_resolution"
      fi
      echo "deer redpanda version probe start $(date -Is)"
      if test -x /usr/bin/redpanda; then
        timeout 10s /usr/bin/redpanda --version || true
      fi
      echo "deer redpanda version probe complete $(date -Is)"
      echo "deer rpk version probe skipped $(date -Is)"
      echo "resolved /usr/bin/redpanda: $(readlink -f /usr/bin/redpanda || true)"
      echo "resolved /usr/bin/rpk: $(readlink -f /usr/bin/rpk || true)"
      echo "resolved redpanda bin: $REDPANDA_BIN"
      echo "resolved rpk bin: $RPK_BIN"
      echo "resolved redpanda install dir: $REDPANDA_INSTALL_DIR"
      echo "resolved redpanda library path: $REDPANDA_LD_LIBRARY_PATH"
      df -h / /opt /var/tmp || true
      echo "deer redpanda install complete $(date -Is)"
      rm -rf "$tmpdir"
      echo "deer redpanda temp cleanup complete $(date -Is)"
    owner: root:root
    permissions: '0755'
  - path: /usr/local/bin/deer-enable-redpanda.sh
    content: |
      #!/bin/bash
      set -euo pipefail
      mkdir -p /var/log/deer
      exec > >(tee -a /var/log/deer/redpanda-enable.log /dev/console) 2>&1
      set -x
      fail_stage() {
        local stage="$1"
        echo "deer redpanda enable failure stage=${stage} $(date -Is)" >&2
        /usr/local/bin/deer-redpanda-diagnostics.sh || true
        exit 1
      }
      echo "deer redpanda enable start $(date -Is)"
      if ! timeout 2m systemctl daemon-reload; then
        fail_stage "daemon_reload"
      fi
      echo "deer redpanda daemon reload complete $(date -Is)"
      echo "deer redpanda service start invoked $(date -Is)"
      if ! timeout 10m systemctl enable --now deer-redpanda.service; then
        fail_stage "systemd_enable_start"
      fi
      echo "deer redpanda systemd enable complete $(date -Is)"
    owner: root:root
    permissions: '0755'
  - path: /usr/local/bin/deer-redpanda-start.sh
    content: |
      #!/bin/bash
      set -euo pipefail
      mkdir -p /var/log/deer
      exec > >(tee -a /var/log/deer/redpanda-start.log /dev/console) 2>&1
      set -x
      echo "deer redpanda start wrapper $(date -Is)"
      . /etc/default/deer-redpanda
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
      echo "deer redpanda version probe start $(date -Is)"
      if test -x /usr/bin/redpanda; then
        timeout 10s /usr/bin/redpanda --version || true
      fi
      echo "deer redpanda version probe complete $(date -Is)"
      echo "deer rpk version probe skipped $(date -Is)"
      echo "resolved /usr/bin/redpanda: $(readlink -f /usr/bin/redpanda || true)"
      echo "resolved /usr/bin/rpk: $(readlink -f /usr/bin/rpk || true)"
      advertise_addr=$(resolve_advertise_addr)
      if [ -n "${advertise_addr}" ] && [ -f /etc/redpanda/redpanda.yaml ]; then
        sed -i "s/__FLUID_ADVERTISE_ADDRESS__/${advertise_addr}/g" /etc/redpanda/redpanda.yaml
      fi
      echo "resolved redpanda advertise address: ${advertise_addr}"
      echo "deer redpanda exec: /usr/bin/rpk redpanda start --install-dir /opt/redpanda --mode dev-container --smp 1 --default-log-level=info"
      exec /usr/bin/rpk redpanda start --install-dir /opt/redpanda --mode dev-container --smp 1 --default-log-level=info
    owner: root:root
    permissions: '0755'
  - path: /usr/local/bin/deer-wait-redpanda.sh
    content: |
      #!/bin/bash
      set -euo pipefail
      mkdir -p /var/log/deer
      exec > >(tee -a /var/log/deer/redpanda-wait.log /dev/console) 2>&1
      set -x
      broker_port=%d
      fail_stage() {
        local stage="$1"
        echo "deer redpanda readiness failure stage=${stage} $(date -Is)" >&2
        /usr/local/bin/deer-redpanda-diagnostics.sh || true
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
          echo "deer redpanda readiness pending stage=${stage} $(date -Is)"
        fi
      }
      service_state() {
        local result exec_status exec_code active_state sub_state n_restarts
        result=$(systemctl show deer-redpanda.service --property=Result --value 2>/dev/null || true)
        exec_status=$(systemctl show deer-redpanda.service --property=ExecMainStatus --value 2>/dev/null || true)
        exec_code=$(systemctl show deer-redpanda.service --property=ExecMainCode --value 2>/dev/null || true)
        active_state=$(systemctl show deer-redpanda.service --property=ActiveState --value 2>/dev/null || true)
        sub_state=$(systemctl show deer-redpanda.service --property=SubState --value 2>/dev/null || true)
        n_restarts=$(systemctl show deer-redpanda.service --property=NRestarts --value 2>/dev/null || true)
        echo "deer redpanda service_state active_state=${active_state} result=${result} sub_state=${sub_state} exec_main_code=${exec_code} exec_main_status=${exec_status} n_restarts=${n_restarts}"
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
        if ! systemctl is-enabled --quiet deer-redpanda.service; then
          note_pending_stage "service_enabled"
          return 1
        fi
        if ! systemctl is-active --quiet deer-redpanda.service; then
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
      echo "deer redpanda readiness wait started $(date -Is)"
      for attempt in $(seq 1 180); do
        if test -x /usr/bin/redpanda && broker_ready; then
          echo "deer redpanda readiness wait success $(date -Is)"
          echo "deer redpanda ready on attempt ${attempt} $(date -Is)"
          exit 0
        fi
        if service_state; then
          fail_stage "systemd_start"
        fi
        if systemctl is-failed --quiet deer-redpanda.service; then
          fail_stage "systemd_start"
        fi
        if [ $((attempt %%%% 15)) -eq 0 ]; then
          if [ -n "${readiness_pending_stage}" ]; then
            echo "deer redpanda readiness pending stage=${readiness_pending_stage} $(date -Is)"
          fi
          systemctl status deer-redpanda.service --no-pager || true
          systemctl show deer-redpanda.service --property=Result --property=ExecMainStatus --property=SubState --property=NRestarts --no-pager || true
          ss -ltn || true
          ss -H -ltn "( sport = :${broker_port} )" || true
        fi
        sleep 2
      done
      echo "deer redpanda readiness wait timeout $(date -Is)"
      fail_stage "wait_timeout"
    owner: root:root
    permissions: '0755'
  - path: /etc/systemd/system/deer-redpanda.service
    content: |
      [Unit]
      Description=Fluid Redpanda Broker
      Wants=network-online.target
      After=network-online.target
      StartLimitIntervalSec=60
      StartLimitBurst=5

      [Service]
      Type=simple
      EnvironmentFile=-/etc/default/deer-redpanda
      ExecStart=/usr/local/bin/deer-redpanda-start.sh
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
			"/usr/local/bin/deer-install-redpanda.sh",
			"/usr/local/bin/deer-enable-redpanda.sh",
			"/usr/local/bin/deer-wait-redpanda.sh",
		)
	}

	if opts.ElasticsearchBroker.Enabled {
		esPort := opts.ElasticsearchBroker.Port
		if esPort == 0 {
			esPort = 9200
		}
		esArchiveURL := opts.ElasticsearchBroker.ArchiveURL
		if esArchiveURL == "" {
			esArchiveURL = defaultElasticsearchArchiveURL()
		}
		writeFiles += fmt.Sprintf(`  - path: /usr/local/bin/deer-install-elasticsearch.sh
    content: |
      #!/bin/bash
      set -euo pipefail
      mkdir -p /var/log/deer
      exec > >(tee -a /var/log/deer/elasticsearch-install.log /dev/console) 2>&1
      set -x
      export DEBIAN_FRONTEND=noninteractive
      echo "deer elasticsearch install start $(date -Is)"
      if ! command -v curl >/dev/null 2>&1; then
        apt-get update
        apt-get install -y ca-certificates curl
      fi
      tmpdir=$(mktemp -d /var/tmp/deer-elasticsearch.XXXXXX)
      archive_url=%q
      archive_path="$tmpdir/elasticsearch.tar.gz"
      if ! curl --connect-timeout 10 --max-time 900 -fsSL --retry 5 --retry-delay 2 -o "$archive_path" "$archive_url"; then
        echo "deer elasticsearch archive download failed $(date -Is)" >&2
        exit 1
      fi
      echo "deer elasticsearch archive download complete $(date -Is)"
      rm -rf /opt/deer-elasticsearch
      mkdir -p /opt/deer-elasticsearch
      if ! tar -xzf "$archive_path" -C /opt/deer-elasticsearch --strip-components=1; then
        echo "deer elasticsearch extraction failed $(date -Is)" >&2
        exit 1
      fi
      echo "deer elasticsearch extraction complete $(date -Is)"
      es_bin=$(find /opt/deer-elasticsearch -type f -name elasticsearch -path '*/bin/*' | head -n1)
      if [ -z "$es_bin" ]; then
        echo "elasticsearch binary not found" >&2
        exit 1
      fi
      ln -sf "$(dirname "$es_bin")/../" /opt/elasticsearch
      cp /etc/elasticsearch/elasticsearch.yml /opt/deer-elasticsearch/config/elasticsearch.yml
      echo "ES_HOME=/opt/elasticsearch" > /etc/default/deer-elasticsearch
      echo "ES_JAVA_OPTS=-Xms512m -Xmx512m" >> /etc/default/deer-elasticsearch
      echo "ES_PORT=%d" >> /etc/default/deer-elasticsearch
      id elasticsearch 2>/dev/null || useradd -r -s /bin/false elasticsearch
      mkdir -p /var/lib/elasticsearch /var/log/elasticsearch
      chown -R elasticsearch:elasticsearch /opt/deer-elasticsearch /var/lib/elasticsearch /var/log/elasticsearch /var/run/elasticsearch
      rm -rf "$tmpdir"
      echo "deer elasticsearch install complete $(date -Is)"
    owner: root:root
    permissions: '0755'
  - path: /etc/elasticsearch/elasticsearch.yml
    content: |
      cluster.name: deer-sandbox
      node.name: sandbox-node-1
      path.data: /var/lib/elasticsearch
      path.logs: /var/log/elasticsearch
      network.host: 0.0.0.0
      http.port: %d
      discovery.type: single-node
      xpack.security.enabled: false
      xpack.security.enrollment.enabled: false
      xpack.security.http.ssl.enabled: false
      xpack.security.transport.ssl.enabled: false
    owner: root:root
    permissions: '0644'
  - path: /etc/systemd/system/deer-elasticsearch.service
    content: |
      [Unit]
      Description=Deer Sandbox Elasticsearch
      Wants=network-online.target
      After=network-online.target

      [Service]
      Type=simple
      EnvironmentFile=-/etc/default/deer-elasticsearch
      User=elasticsearch
      Group=elasticsearch
      ExecStart=/opt/elasticsearch/bin/elasticsearch -p /var/run/elasticsearch/es.pid
      StandardOutput=journal+console
      StandardError=journal+console
      Restart=on-failure
      RestartSec=10

      [Install]
      WantedBy=multi-user.target
    owner: root:root
    permissions: '0644'
  - path: /usr/local/bin/deer-wait-elasticsearch.sh
    content: |
      #!/bin/bash
      set -euo pipefail
      mkdir -p /var/log/deer
      exec > >(tee -a /var/log/deer/elasticsearch-wait.log /dev/console) 2>&1
      set -x
      es_port=%d
      echo "deer elasticsearch readiness wait started $(date -Is)"
      for attempt in $(seq 1 120); do
        if curl -sf "http://localhost:${es_port}/_cluster/health" >/dev/null 2>&1; then
          echo "deer elasticsearch ready on attempt ${attempt} $(date -Is)"
          exit 0
        fi
        if systemctl is-failed --quiet deer-elasticsearch.service; then
          echo "deer elasticsearch service failed $(date -Is)" >&2
          systemctl status deer-elasticsearch.service --no-pager || true
          journalctl -u deer-elasticsearch.service --no-pager -n 50 || true
          exit 1
        fi
        if [ $((attempt %%%% 15)) -eq 0 ]; then
          echo "deer elasticsearch readiness pending attempt ${attempt} $(date -Is)"
          systemctl status deer-elasticsearch.service --no-pager || true
        fi
        sleep 5
      done
      echo "deer elasticsearch readiness wait timeout $(date -Is)" >&2
      systemctl status deer-elasticsearch.service --no-pager || true
      journalctl -u deer-elasticsearch.service --no-pager -n 100 || true
      exit 1
    owner: root:root
    permissions: '0755'
`, esArchiveURL, esPort, esPort, esPort)
		runcmd = append(runcmd,
			"mkdir -p /etc/elasticsearch /var/lib/elasticsearch /var/log/elasticsearch /var/run/elasticsearch",
			"/usr/local/bin/deer-install-elasticsearch.sh",
			"systemctl daemon-reload",
			"systemctl enable --now deer-elasticsearch.service",
			"/usr/local/bin/deer-wait-elasticsearch.sh",
		)
	}

	if opts.PhoneHomeURL != "" {
		runcmd = append(runcmd, "/usr/local/bin/deer-notify-ready.sh")
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

	// 4 MB to accommodate ES broker cloud-init scripts alongside Redpanda.
	const isoSize int64 = 4 * 1024 * 1024

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

func defaultElasticsearchArchiveURL() string {
	if runtime.GOARCH == "arm64" {
		return "https://artifacts.elastic.co/downloads/elasticsearch/elasticsearch-8.13.4-linux-aarch64.tar.gz"
	}
	return "https://artifacts.elastic.co/downloads/elasticsearch/elasticsearch-8.13.4-linux-x86_64.tar.gz"
}
