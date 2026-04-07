#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

lima_name="deer-redpanda-e2e"
lima_template="template://ubuntu"
host_repo_root="${REPO_ROOT}"
guest_repo_root=""
arch=""
bridge="virbr0"
dhcp_mode="libvirt"
accel="tcg"
skip_guest_setup=0
no_download=0
guest_workdir=""
keep_workdir=0
dry_run=0
lima_name_explicit=0
required_lima_cpus=4
required_lima_memory="8GiB"
required_lima_memory_mib=8192
required_lima_disk="100GiB"

usage() {
    cat <<'EOF'
Usage: ./scripts/run-redpanda-e2e-lima-host.sh [options]

Create or start a Lima VM from the host, install guest dependencies, and run
the Redpanda live microVM integration test inside the Linux guest.

Options:
  --lima-name <name>        Lima VM name. Default: deer-redpanda-e2e
  --template <template>     Lima template for first boot. Default: template://ubuntu
  --repo-root <path>        Repository root on the host. Default: directory above this script
  --guest-repo-root <path>  Repository root inside the Lima guest.
                            Default: same path as --repo-root
  --arch <amd64|arm64>      Force guest asset architecture.
  --bridge <name>           Guest bridge name. Default: virbr0
  --dhcp-mode <mode>        Guest IP discovery mode. Default: libvirt
  --accel <mode>            QEMU acceleration mode. Default: tcg
  --workdir <path>          Preserve E2E artifacts in this guest directory.
  --keep-workdir            Preserve the generated temp E2E workdir on failure.
  --skip-guest-setup        Skip apt/libvirt setup inside the guest
  --no-download             Reuse existing guest assets without downloading
  --dry-run                 Print the commands without executing them
  -h, --help                Show this help text
EOF
}

log() {
    printf '[INFO] %s\n' "$*"
}

fail() {
    printf '[ERROR] %s\n' "$*" >&2
    exit 1
}

quote_cmd() {
    printf '%q' "$1"
    shift
    while [ "$#" -gt 0 ]; do
        printf ' %q' "$1"
        shift
    done
    printf '\n'
}

run_cmd() {
    if [ "$dry_run" -eq 1 ]; then
        quote_cmd "$@"
        return 0
    fi
    "$@"
}

run_guest_shell() {
    local command="$1"
    if [ "$dry_run" -eq 1 ]; then
        quote_cmd limactl shell "$lima_name" -- bash -lc "$command"
        return 0
    fi
    limactl shell "$lima_name" -- bash -lc "$command"
}

lima_config_path() {
    printf '%s/%s/lima.yaml\n' "${LIMA_HOME:-$HOME/.lima}" "$lima_name"
}

render_lima_template() {
    cat <<EOF
base: ${lima_template}
cpus: ${required_lima_cpus}
memory: ${required_lima_memory}
disk: ${required_lima_disk}
containerd:
  user: false
EOF
}

write_lima_template() {
    local template_path="$1"
    render_lima_template >"$template_path"
}

dry_run_create_lima() {
    local template_path="/tmp/${lima_name}-lima.yaml"
    printf "cat > %q <<'EOF'\n" "$template_path"
    render_lima_template
    printf "EOF\n"
    printf 'limactl start --name %q %q\n' "$lima_name" "$template_path"
}

read_lima_config_value() {
    local key="$1"
    local path="$2"
    awk -F': *' -v key="$key" '$1 == key { print $2; exit }' "$path" | tr -d '"'
}

memory_to_mib() {
    local raw="${1//\"/}"
    case "$raw" in
        ''|null)
            printf '0\n'
            ;;
        *GiB|*G)
            raw="${raw%GiB}"
            raw="${raw%G}"
            printf '%d\n' $((raw * 1024))
            ;;
        *MiB|*M)
            raw="${raw%MiB}"
            raw="${raw%M}"
            printf '%d\n' "$raw"
            ;;
        *)
            printf '0\n'
            ;;
    esac
}

validate_lima_resources() {
    local config_path="$1"
    local cpus_raw
    local memory_raw
    local cpus=0
    local memory_mib=0

    [ -f "$config_path" ] || fail "expected Lima config at $config_path"

    cpus_raw="$(read_lima_config_value cpus "$config_path")"
    memory_raw="$(read_lima_config_value memory "$config_path")"

    if [[ "$cpus_raw" =~ ^[0-9]+$ ]]; then
        cpus="$cpus_raw"
    fi
    memory_mib="$(memory_to_mib "$memory_raw")"

    if [ "$cpus" -lt "$required_lima_cpus" ] || [ "$memory_mib" -lt "$required_lima_memory_mib" ]; then
        fail "Lima instance \"$lima_name\" is underprovisioned for the Redpanda E2E run (cpus=${cpus_raw:-unset}, memory=${memory_raw:-unset}; need at least ${required_lima_cpus} CPUs and ${required_lima_memory}). Recreate it or choose a different --lima-name."
    fi
}

ensure_lima_instance() {
    local config_path
    local template_path

    config_path="$(lima_config_path)"
    if [ -f "$config_path" ]; then
        validate_lima_resources "$config_path"
        run_cmd limactl start "$lima_name"
        return 0
    fi

    template_path="$(mktemp "/tmp/${lima_name}.XXXXXX.yaml")"
    trap 'rm -f "$template_path"' RETURN
    write_lima_template "$template_path"
    run_cmd limactl start --name "$lima_name" "$template_path"
}

parse_args() {
    while [ "$#" -gt 0 ]; do
        case "$1" in
            --lima-name)
                [ "$#" -ge 2 ] || fail "missing value for --lima-name"
                lima_name="$2"
                lima_name_explicit=1
                shift 2
                ;;
            --template)
                [ "$#" -ge 2 ] || fail "missing value for --template"
                lima_template="$2"
                shift 2
                ;;
            --repo-root)
                [ "$#" -ge 2 ] || fail "missing value for --repo-root"
                host_repo_root="$2"
                shift 2
                ;;
            --guest-repo-root)
                [ "$#" -ge 2 ] || fail "missing value for --guest-repo-root"
                guest_repo_root="$2"
                shift 2
                ;;
            --arch)
                [ "$#" -ge 2 ] || fail "missing value for --arch"
                arch="$2"
                shift 2
                ;;
            --bridge)
                [ "$#" -ge 2 ] || fail "missing value for --bridge"
                bridge="$2"
                shift 2
                ;;
            --dhcp-mode)
                [ "$#" -ge 2 ] || fail "missing value for --dhcp-mode"
                dhcp_mode="$2"
                shift 2
                ;;
            --accel)
                [ "$#" -ge 2 ] || fail "missing value for --accel"
                accel="$2"
                shift 2
                ;;
            --workdir)
                [ "$#" -ge 2 ] || fail "missing value for --workdir"
                guest_workdir="$2"
                shift 2
                ;;
            --keep-workdir)
                keep_workdir=1
                shift
                ;;
            --skip-guest-setup)
                skip_guest_setup=1
                shift
                ;;
            --no-download)
                no_download=1
                shift
                ;;
            --dry-run)
                dry_run=1
                shift
                ;;
            -h|--help)
                usage
                exit 0
                ;;
            *)
                fail "unknown argument: $1"
                ;;
        esac
    done
}

parse_args "$@"

if [ -z "$guest_repo_root" ]; then
    guest_repo_root="$host_repo_root"
fi

guest_repo_q="$(printf '%q' "$guest_repo_root")"
guest_script_q="$(printf '%q' "${guest_repo_root}/scripts/run-redpanda-e2e-lima.sh")"

if [ "$dry_run" -eq 0 ]; then
    command -v limactl >/dev/null 2>&1 || fail "limactl not found in PATH"
    [ -d "$host_repo_root" ] || fail "repo root not found: $host_repo_root"
fi

log "lima_name=${lima_name}"
log "host_repo_root=${host_repo_root}"
log "guest_repo_root=${guest_repo_root}"
if [ -n "$guest_workdir" ]; then
    log "guest_workdir=${guest_workdir}"
fi
if [ "$keep_workdir" -eq 1 ]; then
    log "keeping guest E2E workdir on failure"
fi
if [ "$lima_name_explicit" -eq 1 ]; then
    log "using explicit lima_name=${lima_name}"
fi

if [ "$dry_run" -eq 1 ]; then
    printf 'if [ -f %q ]; then\n' "$(lima_config_path)"
    printf '  # existing instance must meet the Redpanda E2E minimums before reuse\n'
    printf '  limactl start %q\n' "$lima_name"
    printf 'else\n'
    dry_run_create_lima
    printf 'fi\n'
else
    ensure_lima_instance
fi

if [ "$skip_guest_setup" -eq 0 ]; then
    read -r -d '' guest_setup_cmd <<EOF || true
set -euo pipefail
test -d ${guest_repo_q}
sudo find /tmp /var/tmp -maxdepth 1 \\( -name 'deer-*' -o -name 'go-build*' \\) -exec rm -rf {} + 2>/dev/null || true
sudo apt-get clean
sudo env TMPDIR=/var/tmp apt-get update
sudo env TMPDIR=/var/tmp apt-get install -y gpgv gnupg qemu-system qemu-utils libvirt-daemon-system libvirt-clients iproute2 openssh-client golang-go
sudo systemctl enable --now libvirtd
sudo virsh net-autostart default
sudo virsh net-start default || true
ip -4 addr show ${bridge}
EOF
    run_guest_shell "$guest_setup_cmd"
fi

guest_run_cmd="set -euo pipefail
bash ${guest_script_q} --repo-root ${guest_repo_q} --bridge $(printf '%q' "$bridge") --dhcp-mode $(printf '%q' "$dhcp_mode") --accel $(printf '%q' "$accel")"

if [ -n "$arch" ]; then
    guest_run_cmd+=" --arch $(printf '%q' "$arch")"
fi
if [ "$no_download" -eq 1 ]; then
    guest_run_cmd+=" --no-download"
fi
if [ -n "$guest_workdir" ]; then
    guest_run_cmd+=" --workdir $(printf '%q' "$guest_workdir")"
fi
if [ "$keep_workdir" -eq 1 ]; then
    guest_run_cmd+=" --keep-workdir"
fi

run_guest_shell "$guest_run_cmd"
