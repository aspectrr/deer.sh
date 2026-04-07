#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

repo_root="${REPO_ROOT}"
arch=""
asset_dir=""
bridge="virbr0"
dhcp_mode="libvirt"
accel="tcg"
download_assets=1
workdir=""
keep_workdir=0
dry_run=0

usage() {
    cat <<'EOF'
Usage: ./scripts/run-redpanda-e2e-lima.sh [options]

Run the live Redpanda microVM integration test inside a Linux Lima guest.

Options:
  --repo-root <path>      Repository root inside the Lima guest.
                          Default: directory above this script
  --arch <amd64|arm64>    Guest architecture. Default: derived from uname -m
  --asset-dir <path>      Override downloaded asset directory.
  --bridge <name>         Bridge to use for the test. Default: virbr0
  --dhcp-mode <mode>      IP discovery mode. Default: libvirt
  --accel <mode>          QEMU acceleration mode. Default: tcg
  --workdir <path>        Preserve E2E artifacts in this directory.
  --keep-workdir          Preserve the generated temp E2E workdir on failure.
  --no-download           Do not download assets automatically.
  --dry-run               Print commands without executing them.
  -h, --help              Show this help text.
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

parse_args() {
    while [ "$#" -gt 0 ]; do
        case "$1" in
            --repo-root)
                [ "$#" -ge 2 ] || fail "missing value for --repo-root"
                repo_root="$2"
                shift 2
                ;;
            --arch)
                [ "$#" -ge 2 ] || fail "missing value for --arch"
                arch="$2"
                shift 2
                ;;
            --asset-dir)
                [ "$#" -ge 2 ] || fail "missing value for --asset-dir"
                asset_dir="$2"
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
                workdir="$2"
                shift 2
                ;;
            --keep-workdir)
                keep_workdir=1
                shift
                ;;
            --no-download)
                download_assets=0
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

resolve_arch() {
    if [ -n "$arch" ]; then
        return 0
    fi

    case "$(uname -m)" in
        x86_64|amd64)
            arch="amd64"
            ;;
        aarch64|arm64)
            arch="arm64"
            ;;
        *)
            fail "unsupported host architecture: $(uname -m)"
            ;;
    esac
}

parse_args "$@"

if [ "$dry_run" -eq 0 ] && [ "$(uname -s)" != "Linux" ]; then
    fail "this helper is intended to run inside a Linux Lima guest"
fi

resolve_arch

case "$arch" in
    amd64)
        qemu_bin="qemu-system-x86_64"
        image_name="ubuntu-24.04-server-cloudimg-amd64"
        default_asset_dir="${repo_root}/.cache/deer/e2e/noble-amd64"
        ;;
    arm64)
        qemu_bin="qemu-system-aarch64"
        image_name="ubuntu-24.04-server-cloudimg-arm64"
        default_asset_dir="${repo_root}/.cache/deer/e2e/noble-arm64"
        ;;
    *)
        fail "unsupported arch: ${arch}"
        ;;
esac

if [ -z "$asset_dir" ]; then
    asset_dir="${default_asset_dir}"
fi

log "repo_root=${repo_root}"
log "arch=${arch}"
log "asset_dir=${asset_dir}"
if [ -n "$workdir" ]; then
    log "workdir=${workdir}"
fi
if [ "$keep_workdir" -eq 1 ]; then
    log "keeping E2E workdir on failure"
fi

if [ "$download_assets" -eq 1 ]; then
    run_cmd "${repo_root}/scripts/download-microvm-assets.sh" --arch "$arch" --output-dir "$asset_dir"
fi

base_image="${asset_dir}/${image_name}.qcow2"
kernel_path="${asset_dir}/${image_name}-vmlinuz-generic"
initrd_path="${asset_dir}/${image_name}-initrd-generic"

if [ "$dry_run" -eq 0 ]; then
    [ -d "${repo_root}/deer-daemon" ] || fail "deer-daemon directory not found under ${repo_root}"
    [ -f "$base_image" ] || fail "base image not found: $base_image"
    [ -f "$kernel_path" ] || fail "kernel not found: $kernel_path"
    [ -f "$initrd_path" ] || fail "initrd not found: $initrd_path"
    command -v go >/dev/null 2>&1 || fail "go not found in PATH inside the Lima guest; rerun the host wrapper without --skip-guest-setup"
    log "guest memory snapshot"
    free -h || true
fi

cmd=(
    sudo env
    TMPDIR=/var/tmp
    DEER_E2E_MICROVM=1
    DEER_E2E_STARTUP_TIMEOUT=25m
    "DEER_E2E_BASE_IMAGE=${base_image}"
    "DEER_E2E_KERNEL=${kernel_path}"
    "DEER_E2E_INITRD=${initrd_path}"
    "DEER_E2E_QEMU_BINARY=${qemu_bin}"
    "DEER_E2E_BRIDGE=${bridge}"
    "DEER_E2E_DHCP_MODE=${dhcp_mode}"
    "DEER_E2E_ACCEL=${accel}"
    DEER_E2E_ROOT_DEVICE=/dev/vda1
    "FLUID_QEMU_KERNEL_APPEND=earlycon=pl011,0x09000000 ignore_loglevel loglevel=8"
)

if [ -n "$workdir" ]; then
    cmd+=( "DEER_E2E_WORKDIR=${workdir}" )
fi
if [ "$keep_workdir" -eq 1 ]; then
    cmd+=( DEER_E2E_KEEP_WORKDIR=1 )
fi
cmd+=(
    GOTOOLCHAIN=auto
    GOCACHE=/var/tmp/deer-daemon-go-build
    go test -timeout 30m -v -run TestProviderIntegration_RedpandaStartsInGuest ./internal/provider/microvm
)

if [ "$dry_run" -eq 1 ]; then
    printf 'cd %q/deer-daemon && ' "$repo_root"
    quote_cmd "${cmd[@]}"
else
    (
        cd "${repo_root}/deer-daemon"
        "${cmd[@]}"
    )
fi
