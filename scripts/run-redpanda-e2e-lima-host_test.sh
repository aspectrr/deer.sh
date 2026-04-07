#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
TARGET="${REPO_ROOT}/scripts/run-redpanda-e2e-lima-host.sh"

assert_contains() {
    local haystack="$1"
    local needle="$2"
    if [[ "$haystack" != *"$needle"* ]]; then
        printf 'expected output to contain: %s\n' "$needle" >&2
        exit 1
    fi
}

fixture_repo="$(mktemp -d)"
fake_bin_dir=""
fake_lima_home=""
trap 'rm -rf "$fixture_repo" "$fake_bin_dir" "$fake_lima_home"' EXIT

output="$("$TARGET" --dry-run --repo-root "$fixture_repo" --guest-repo-root /mnt/deer --arch arm64 --no-download)"
assert_contains "$output" "if [ -f "
assert_contains "$output" "deer-redpanda-e2e/lima.yaml"
assert_contains "$output" "limactl start deer-redpanda-e2e"
assert_contains "$output" "cat > /tmp/deer-redpanda-e2e-lima.yaml <<'EOF'"
assert_contains "$output" "base: template://ubuntu"
assert_contains "$output" "cpus: 4"
assert_contains "$output" "memory: 8GiB"
assert_contains "$output" "disk: 100GiB"
assert_contains "$output" "containerd:"
assert_contains "$output" "  user: false"
assert_contains "$output" "limactl start --name deer-redpanda-e2e /tmp/deer-redpanda-e2e-lima.yaml"
assert_contains "$output" "limactl shell deer-redpanda-e2e -- bash -lc"
assert_contains "$output" "test -d /mnt/deer"
assert_contains "$output" "sudo find /tmp /var/tmp -maxdepth 1"
assert_contains "$output" "sudo apt-get clean"
assert_contains "$output" "sudo env TMPDIR=/var/tmp apt-get update"
assert_contains "$output" "sudo env TMPDIR=/var/tmp apt-get install -y gpgv gnupg qemu-system qemu-utils libvirt-daemon-system libvirt-clients iproute2 openssh-client golang-go"
assert_contains "$output" "bash /mnt/deer/scripts/run-redpanda-e2e-lima.sh --repo-root /mnt/deer --bridge virbr0 --dhcp-mode libvirt --accel tcg --arch arm64 --no-download"

keep_output="$("$TARGET" --dry-run --repo-root "$fixture_repo" --guest-repo-root /mnt/deer --arch arm64 --no-download --workdir /var/tmp/deer-e2e-debug --keep-workdir)"
assert_contains "$keep_output" "bash /mnt/deer/scripts/run-redpanda-e2e-lima.sh --repo-root /mnt/deer --bridge virbr0 --dhcp-mode libvirt --accel tcg --arch arm64 --no-download --workdir /var/tmp/deer-e2e-debug --keep-workdir"

override_output="$("$TARGET" --dry-run --repo-root "$fixture_repo" --guest-repo-root /mnt/deer --arch arm64 --no-download --lima-name custom-redpanda-vm)"
assert_contains "$override_output" "limactl start custom-redpanda-vm"
assert_contains "$override_output" "limactl shell custom-redpanda-vm -- bash -lc"

fake_bin_dir="$(mktemp -d)"
fake_lima_home="$(mktemp -d)"
mkdir -p "$fake_lima_home/existing-low-mem"
cat >"$fake_lima_home/existing-low-mem/lima.yaml" <<'EOF'
cpus: 2
memory: null
EOF
cat >"$fake_bin_dir/limactl" <<'EOF'
#!/usr/bin/env bash
if [ "$1" = "start" ]; then
    exit 0
fi
if [ "$1" = "shell" ]; then
    printf 'unexpected limactl shell invocation\n' >&2
    exit 99
fi
printf 'unexpected limactl invocation: %s\n' "$*" >&2
exit 98
EOF
chmod +x "$fake_bin_dir/limactl"

set +e
under_output="$(PATH="$fake_bin_dir:$PATH" LIMA_HOME="$fake_lima_home" "$TARGET" --repo-root "$fixture_repo" --guest-repo-root /mnt/deer --arch arm64 --no-download --skip-guest-setup --lima-name existing-low-mem 2>&1)"
under_status=$?
set -e

if [ "$under_status" -eq 0 ]; then
    printf 'expected explicit underprovisioned lima instance to fail\n' >&2
    exit 1
fi
assert_contains "$under_output" "underprovisioned"
assert_contains "$under_output" "need at least 4 CPUs and 8GiB"
