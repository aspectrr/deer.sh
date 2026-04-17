#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

release="noble"
version="24.04"
arch="amd64"
output_dir=""
force=0
dry_run=0

usage() {
    cat <<'EOF'
Usage: ./scripts/download-microvm-assets.sh [options]

Download the Ubuntu cloud image, kernel, and initrd needed for the live
microVM guest integration tests.

Options:
  --arch <amd64|arm64>    Guest architecture to download. Default: amd64
  --output-dir <path>     Target directory for the assets.
                          Default: <repo>/.cache/deer/e2e/<release>-<arch>
  --force                 Re-download files even if they already exist.
  --dry-run               Print the download/verify commands without executing.
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

require_tool() {
    if ! command -v "$1" >/dev/null 2>&1; then
        fail "missing required tool: $1"
    fi
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

checksum_verify_cmd() {
    if command -v sha256sum >/dev/null 2>&1; then
        printf '%s\n' "sha256sum"
        return 0
    fi
    if command -v shasum >/dev/null 2>&1; then
        printf '%s\n' "shasum -a 256"
        return 0
    fi
    return 1
}

parse_args() {
    while [ "$#" -gt 0 ]; do
        case "$1" in
            --arch)
                [ "$#" -ge 2 ] || fail "missing value for --arch"
                arch="$2"
                shift 2
                ;;
            --output-dir)
                [ "$#" -ge 2 ] || fail "missing value for --output-dir"
                output_dir="$2"
                shift 2
                ;;
            --force)
                force=1
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

download_if_needed() {
    local dest="$1"
    local url="$2"

    if [ -f "$dest" ] && [ "$force" -eq 0 ]; then
        log "using existing $(basename "$dest")"
        return 0
    fi

    run_cmd curl -fL --retry 3 -o "$dest" "$url"
}

ensure_qcow2_link() {
    if [ -L "$qcow2_path" ]; then
        local current_target
        current_target="$(readlink "$qcow2_path")"
        if [ "$current_target" = "$image_name" ]; then
            log "using existing $(basename "$qcow2_path") symlink"
            return 0
        fi
    fi

    if [ -e "$qcow2_path" ] && [ ! -L "$qcow2_path" ]; then
        fail "existing qcow2 path is not a symlink: $qcow2_path"
    fi

    if [ "$dry_run" -eq 1 ]; then
        quote_cmd ln -sf "$image_name" "$qcow2_path"
        return 0
    fi

    (
        cd "$output_dir"
        rm -f "$qcow2_name"
        ln -s "$image_name" "$qcow2_name"
    )
}

parse_args "$@"

case "$arch" in
    amd64|arm64)
        ;;
    *)
        fail "unsupported arch: $arch"
        ;;
esac

if [ -z "$output_dir" ]; then
    output_dir="${REPO_ROOT}/.cache/deer/e2e/${release}-${arch}"
fi

image_name="ubuntu-${version}-server-cloudimg-${arch}.img"
kernel_name="ubuntu-${version}-server-cloudimg-${arch}-vmlinuz-generic"
initrd_name="ubuntu-${version}-server-cloudimg-${arch}-initrd-generic"
qcow2_name="ubuntu-${version}-server-cloudimg-${arch}.qcow2"
sha_name="SHA256SUMS"

release_base="https://cloud-images.ubuntu.com/releases/${release}/release"
unpacked_base="${release_base}/unpacked"

image_url="${release_base}/${image_name}"
kernel_url="${unpacked_base}/${kernel_name}"
initrd_url="${unpacked_base}/${initrd_name}"
sha_url="${release_base}/${sha_name}"

image_path="${output_dir}/${image_name}"
kernel_path="${output_dir}/${kernel_name}"
initrd_path="${output_dir}/${initrd_name}"
qcow2_path="${output_dir}/${qcow2_name}"
sha_path="${output_dir}/${sha_name}"

if [ "$dry_run" -eq 0 ]; then
    require_tool curl
    checksum_tool="$(checksum_verify_cmd)" || fail "missing checksum tool: need sha256sum or shasum"
    mkdir -p "$output_dir"
else
    checksum_tool="$(checksum_verify_cmd || true)"
    if [ -z "$checksum_tool" ]; then
        checksum_tool="sha256sum"
    fi
fi

log "release=${release} arch=${arch}"
log "output=${output_dir}"

download_if_needed "$image_path" "$image_url"
download_if_needed "$kernel_path" "$kernel_url"
download_if_needed "$initrd_path" "$initrd_url"
download_if_needed "$sha_path" "$sha_url"

pattern="${image_name}\\|${kernel_name}\\|${initrd_name}"
verify_cmd="grep '${pattern}' '${sha_path}' | ${checksum_tool} -c -"

if [ "$dry_run" -eq 1 ]; then
    printf '%s\n' "$verify_cmd"
else
    (
        cd "$output_dir"
        eval "$verify_cmd"
    )
fi

ensure_qcow2_link

cat <<EOF
DEER_E2E_BASE_IMAGE=${qcow2_path}
DEER_E2E_KERNEL=${kernel_path}
DEER_E2E_INITRD=${initrd_path}
EOF
