# Security Model

deer.sh uses defense-in-depth to isolate AI agent workloads in VM sandboxes. This document describes the security architecture across the CLI, daemon, and API control plane.

## Overview

Security is enforced across multiple layers:

1. **SSH Certificate Authority** - short-lived certificates replace persistent credentials
2. **Principal separation** - sandbox (`sandbox`) and read-only (`deer-readonly`) access use distinct SSH principals
3. **Read-only enforcement** - client-side allowlist + server-side restricted shell block destructive commands on source VMs
4. **VM isolation** - QEMU microVM hypervisor isolation with copy-on-write overlays
5. **Secrets redaction** - sensitive data stripped from LLM messages with deterministic tokens
6. **Human approval workflow** - blocking confirmation dialogs for network access, resource limits, and source VM preparation
7. **Hash-chained audit log** - tamper-evident append-only log of all agent actions
8. **Input validation** - shell argument sanitization, path traversal prevention, file size limits
9. **API authentication and authorization** - bcrypt passwords, session tokens, OAuth, RBAC
10. **Transport security** - CORS lockdown, rate limiting, optional mTLS for gRPC
11. **Encryption at rest** - AES-256-GCM for OAuth tokens and credentials
12. **Telemetry privacy** - enabled by default (opt-out), anonymous, no user content collected

## SSH Certificate Authority

The SSH CA signs short-lived certificates for all sandbox and source VM access. No persistent SSH keys are stored on VMs.

**Key generation**: Ed25519 CA key pair generated via `ssh-keygen`. Private key stored at configurable path (default `/etc/virsh-sandbox/ssh_ca`) with 0600 permissions. Public key at the same path with `.pub` suffix.

**Certificate identity format**:
```
user:{UserID}-vm:{VMID}-sbx:{SandboxID}-cert:{CertID}
```

**Certificate properties**:
- Default TTL: 30 minutes
- Maximum TTL: 60 minutes
- Minimum TTL: 1 minute
- Clock skew buffer: 1 minute (validity starts 1 minute before issuance)
- Serial numbers: random 64-bit, incremented per issuance
- Extensions: `permit-pty` only
- Restrictions: `no-port-forwarding`, `no-agent-forwarding`, `no-X11-forwarding`

**Permission validation**: the CA enforces that private key files have mode 0600 or 0400 (no group/world access) before signing.

Source: `deer-daemon/internal/sshca/ca.go`

## Sandbox Credentials

Each sandbox gets ephemeral Ed25519 key pairs, generated on demand and cached until expiry.

- **Principal**: `"sandbox"`
- **Key directory**: `{keyDir}/{sandboxID}/` with 0700 permissions
- **Private keys**: 0600 permissions
- **Certificates**: 0644 permissions
- **Auto-refresh**: credentials regenerate 30 seconds before certificate expiry
- **Thread safety**: per-sandbox mutexes prevent concurrent key generation
- **Cleanup**: key files and cache entries removed on sandbox destroy

Pre-flight permission checks run before every SSH connection: the runner verifies the private key file has no group/world permissions (`perm & 0077 == 0`) and rejects the connection otherwise.

Source: `deer-daemon/internal/sshkeys/manager.go`

## Source VM Read-Only Mode

Source (golden) VMs are accessible only for inspection, never modification. The CLI connects directly to source hosts via SSH for read-only operations (not through the daemon). Three enforcement layers ensure safety.

### Layer 1: Client-side allowlist

`ValidateCommand()` parses the command into pipeline segments and checks each segment's base command against an allowlist of ~70 safe commands.

**Allowed categories**:
- File inspection: `cat`, `ls`, `find`, `head`, `tail`, `stat`, `file`, `wc`, `du`, `tree`, `strings`, `md5sum`, `sha256sum`, `readlink`, `realpath`, `basename`, `dirname`, `base64`
- Process/system info: `ps`, `top`, `pgrep`, `systemctl`, `journalctl`, `dmesg`
- Network info: `ss`, `netstat`, `ip`, `ifconfig`, `dig`, `nslookup`, `ping`
- Disk info: `df`, `lsblk`, `blkid`
- Package queries: `dpkg`, `rpm`, `apt`, `pip` (restricted subcommands only)
- System info: `uname`, `hostname`, `uptime`, `free`, `lscpu`, `lsmod`, `lspci`, `lsusb`, `arch`, `nproc`
- User info: `whoami`, `id`, `groups`, `who`, `w`, `last`
- Misc: `env`, `printenv`, `date`, `which`, `type`, `echo`, `test`
- Pipe targets: `grep`, `awk`, `sed`, `sort`, `uniq`, `cut`, `tr`, `xargs`

**Subcommand restrictions** (first argument must match allowlist):
- `systemctl`: `status`, `show`, `list-units`, `is-active`, `is-enabled`
- `dpkg`: `-l`, `--list`
- `rpm`: `-qa`, `-q`
- `apt`: `list`
- `pip`: `list`

**Metacharacter blocking**:
- Command substitution: `$(...)` and backticks
- Process substitution: `<(...)` and `>(...)`
- Output redirection: `>` and `>>`
- Newlines: `\n` and `\r`

Source: `deer-daemon/internal/readonly/validate.go`

### Layer 2: Server-side restricted shell

A bash script installed at `/usr/local/bin/deer-readonly-shell` on source VMs acts as the login shell for the `deer-readonly` user. It:

1. Denies interactive login (requires `SSH_ORIGINAL_COMMAND`)
2. Blocks command substitution, subshells, output redirection, and newlines
3. Parses the command on pipe/semicolon/`&&`/`||` boundaries
4. Checks each segment against a blocklist of destructive command patterns

**Blocked command categories** (regex patterns on each pipeline segment):
- Privilege escalation: `sudo`, `su`
- File mutation: `rm`, `mv`, `cp`, `dd`, `chmod`, `chown`, `chgrp`
- Process control: `kill`, `killall`, `pkill`, `shutdown`, `reboot`, `halt`, `poweroff`
- User management: `useradd`, `userdel`, `usermod`, `groupadd`, `groupdel`, `passwd`
- Disk operations: `mkfs`, `mount`, `umount`, `fdisk`, `parted`
- Network tools: `wget`, `curl`, `scp`, `rsync`, `ftp`, `sftp`
- Interpreters/shells: `python`, `perl`, `ruby`, `node`, `bash`, `sh`, `zsh`, `dash`, `csh`
- Editors: `vi`, `vim`, `nano`, `emacs`
- Build tools: `make`, `gcc`, `g++`, `cc`
- Package installation: `apt install/remove/purge`, `apt-get`, `dpkg -i/--install/--remove/--purge`, `rpm -i/--install/-e/--erase`, `yum`, `dnf`, `pip install/uninstall`
- Service mutation: `systemctl start/stop/restart/reload/enable/disable/daemon/mask/unmask/edit/set`
- Firewall: `iptables`, `ip6tables`, `nft`
- Write tools: `sed -i`, `tee`, `install`

Source: `deer-daemon/internal/readonly/shell.go`

### Layer 3: SSH principal separation

Source VM credentials use the `"deer-readonly"` principal. The `sshd` on source VMs is configured with:
- `TrustedUserCAKeys /etc/ssh/deer_ca.pub`
- `AuthorizedPrincipalsFile /etc/ssh/authorized_principals/%u`

Only certificates with the `deer-readonly` principal are accepted for the `deer-readonly` user. Sandbox certificates (principal `"sandbox"`) cannot authenticate to source VMs.

Source VM preparation (`deer source prepare`) is idempotent and performs:
1. Install restricted shell at `/usr/local/bin/deer-readonly-shell`
2. Create `deer-readonly` system user with the restricted shell as login shell
3. Copy CA public key to `/etc/ssh/deer_ca.pub`
4. Configure `sshd` to trust the CA key and use per-user authorized principals
5. Create `/etc/ssh/authorized_principals/deer-readonly` containing `deer-readonly`
6. Restart `sshd`

Source: `deer-daemon/internal/readonly/prepare.go`, `deer-daemon/internal/sshkeys/manager.go`

## VM Isolation

- **Hypervisor**: QEMU microVMs provide hardware-level isolation between sandboxes
- **Copy-on-write overlays**: sandboxes are linked clones from golden images via qcow2 overlay files, so the source disk is never modified
- **Random MAC addresses**: each clone gets a random MAC in the `52:54:00` QEMU prefix via crypto/rand
- **Network isolation**: per-sandbox TAP devices attached to a bridge network; optional SSH `ProxyJump` for isolated networks not directly reachable from the host

Source: `deer-daemon/internal/microvm/manager.go`

## Secrets Redaction

Both the CLI and daemon include identical redaction packages that strip sensitive data from all outgoing LLM messages and restore tokens in responses before tool execution.

**Built-in detectors**:
- SSH private keys (`-----BEGIN ... PRIVATE KEY-----`)
- Connection strings: PostgreSQL, MySQL, MongoDB, Redis
- AWS access keys (`AKIA...`)
- API keys (`sk-`, `key-`, `Bearer`)
- IPv4 and IPv6 addresses

**Token format**: `[REDACTED_CATEGORY_N]` - deterministic per category, allowing the LLM to reference redacted values without seeing them.

**Configurability**: custom regex patterns and allowlists can be added.

Source: `deer-cli/internal/redact/`, `deer-daemon/internal/redact/`

## Human Approval Workflow

The TUI enforces human-in-the-loop confirmation for potentially dangerous operations via blocking dialogs. All dialogs default to "No"; Escape maps to "No".

**Network access**: blocking dialog before commands using `curl`, `wget`, `nc`, `ssh`, `scp`, `rsync`, and similar network tools. Default: deny.

**Resource limits**: warning dialog when sandbox creation exceeds available memory, CPU, or storage. Default: deny.

**Source VM preparation**: confirmation before running `deer source prepare`. Default: deny.

Source: `deer-cli/internal/tui/confirm.go`, `deer-cli/internal/tui/agent.go`

## Hash-Chained Audit Log

Append-only JSONL audit log at `~/.config/deer/audit.jsonl` with 0600 permissions.

**Hash chain**: each entry contains a SHA-256 hash computed from the previous entry's hash plus the current entry. The genesis entry uses an all-zeros hash.

**Logged events**:
- Session start/end
- User input (length only, never content)
- LLM requests and responses
- Tool calls with arguments, results, and duration

**Integrity verification**: `VerifyChain()` validates the entire chain and detects any tampering or insertion.

**Size protection**: configurable max file size; events are dropped when the limit is reached.

Source: `deer-cli/internal/audit/`, `deer-daemon/internal/audit/`

## MCP Input Validation

All MCP tool inputs are validated before execution.

**Shell argument validation**: rejects empty strings, arguments over 32 KB, null bytes, and control characters.

**Shell escaping**: POSIX single-quote wrapping for all shell arguments.

**File path validation**: paths must be absolute, must not contain `..` after cleaning, and must not contain null bytes.

**File size limit**: 10 MB maximum for file operations.

Source: `deer-cli/internal/mcp/validate.go`

## Config File Security

**Permission checking**: warns if config files are group- or world-readable (should be 0600).

**Secret detection**: flags insecure permissions when API keys or tokens are present in the config.

**File creation**: config files are saved with 0600 permissions.

Source: `deer-cli/internal/config/config.go`

## Telemetry Privacy

Telemetry is enabled by default (opt-out). Disable via `telemetry.enable_anonymous_usage: false` in config or `ENABLE_ANONYMOUS_USAGE=false` env var.

- Requires build-time API key injection; defaults to a no-op service otherwise
- Persistent anonymous UUID at `~/.config/deer/telemetry_id` for cross-session correlation
- `$ip` is set to `0.0.0.0` to prevent IP logging
- Tracks only: tool names, message counts, OS/arch
- Never collects: commands, file contents, IP addresses, hostnames, user input
- Daemon redaction scope: daemon audit uses built-in detectors only; CLI custom redaction patterns (`redact.custom_patterns`) do not apply on the daemon side

Source: `deer-cli/internal/telemetry/`, `deer-daemon/internal/telemetry/`

## API Authentication

**Password authentication**: bcrypt with cost factor 12, minimum 8-character passwords, generic error messages to prevent user enumeration.

**Session tokens**: 32 cryptographically random bytes; only the SHA-256 hash is stored server-side. Cookies are `HttpOnly`, `Secure`, `SameSite=Strict`.

**OAuth (GitHub, Google)**: CSRF state parameter uses 32 random bytes with constant-time comparison. OAuth tokens are encrypted at rest.

**Host tokens**: SHA-256 hashed in the database with expiry enforced at lookup time.

Source: `api/internal/auth/`

## API Authorization (RBAC)

Three roles with numeric levels: owner (3), admin (2), member (1).

- Per-resource membership verification on every request
- Escalated operations (create sandbox, manage hosts) require admin or higher
- Organization deletion: owner-only
- Role checks use numeric comparison for consistent enforcement

Source: `api/internal/rest/`, `api/internal/store/`

## API Transport Security

**CORS**: origin locked to configured frontend URL (not wildcard), credentials allowed.

**Rate limiting**: per-IP token bucket. Auth routes have custom limits:
- Registration: 0.1 requests/sec, burst 5
- Login: 0.2 requests/sec, burst 10

**Proxy IP resolution**: `X-Forwarded-For` only trusted from configured CIDR ranges.

Source: `api/internal/rest/server.go`, `api/internal/rest/ratelimit.go`

## Encryption at Rest

AES-256-GCM with random nonce for OAuth tokens and Proxmox credentials.

Sensitive fields are excluded from JSON serialization: `PasswordHash`, tokens, and secrets are all tagged `json:"-"`.

Source: `api/internal/crypto/crypto.go`

## gRPC Security (Control Plane)

**Daemon-to-API**: optional mTLS with client certificate and custom CA pool. Defaults to insecure for backwards compatibility.

**API-to-daemon**: host token authentication via stream interceptor. Optional TLS (warns if disabled).

**Concurrency limiting**: max 64 concurrent command handlers.

Source: `api/internal/grpc/`, `deer-daemon/internal/agent/`

## Network Isolation (Daemon)

- **Bridge name validation**: `^[a-zA-Z0-9_-]+$` regex
- **Per-sandbox TAP devices**: each sandbox gets a dedicated TAP device attached to the bridge
- **Random MAC addresses**: QEMU OUI prefix (`52:54:00`) with crypto/rand for remaining octets
- **IP discovery**: reads DHCP leases and ARP table; no direct guest communication required
- **Lease file path sanitization**: `filepath.Base()` prevents path traversal

Source: `deer-daemon/internal/network/`

## Sandbox Lifecycle (Janitor)

Background TTL enforcement for automatic sandbox cleanup.

- **Default TTL**: 24 hours, with per-sandbox override
- **Check interval**: every 1 minute
- **Cleanup**: destroys expired sandboxes (VM process + storage + state)

Source: `deer-daemon/internal/janitor/`

## Command Execution Security

- **Shell escaping**: environment variable values are single-quote escaped via `shellQuote()` (replaces `'` with `'\''`)
- **Environment variable name sanitization**: `safeShellIdent()` strips all characters except `[A-Za-z0-9_]`, replacing them with underscores
- **SSH retry with backoff**: transient connection failures retry up to 5 times with exponential backoff (2s initial, 30s max delay)
- **IP conflict detection**: before every command execution, the service re-discovers the VM IP and validates it is not assigned to another running or starting sandbox
- **StrictHostKeyChecking disabled**: ephemeral VMs have no stable host keys; trust is established via the CA certificate chain instead

Source: `deer-daemon/internal/microvm/manager.go`

## Path Traversal Prevention

VM names used in filesystem paths are sanitized via `sanitizeVMName()`:

```
regex: [^A-Za-z0-9_-]  ->  replaced with underscore
```

This prevents `../` sequences and absolute path injection in source VM names when constructing key directories.

Source: `deer-daemon/internal/sshkeys/manager.go`

## File Permissions Summary

| Asset | Permission | Notes |
|-------|-----------|-------|
| CA private key | 0600 | Enforced at initialization; 0400 also accepted |
| CA public key | 0644 | Readable by sshd on VMs |
| Key directories | 0700 | Per-sandbox and per-source-VM |
| Private keys | 0600 | Validated before every SSH connection |
| Certificates | 0644 | Standard SSH certificate permissions |
| CA work directory | 0700 | Temp directory for certificate operations |
| Restricted shell | 0755 | Executable on source VMs |
| Config file | 0600 | Warns if group/world readable |
| Audit log | 0600 | Append-only JSONL |
| State DB | default | SQLite, unencrypted |

## Timeouts

| Operation | Default | Notes |
|-----------|---------|-------|
| Command execution | 10 minutes | Configurable per-call |
| IP discovery | 2 minutes | Polls DHCP leases or ARP table |
| SSH readiness | 60 seconds | Exponential backoff probes after IP discovery |
| SSH connect | 15 seconds | Per-connection `ConnectTimeout` |
| Certificate TTL | 30 minutes | Max 60 minutes, min 1 minute |
| Credential refresh | 30 seconds before expiry | Auto-regenerates keys and certificates |
| Sandbox TTL | 24 hours | Per-sandbox override; janitor enforced |
| Janitor interval | 1 minute | Background cleanup cycle |
| OAuth state cookie | 600 seconds | CSRF state parameter lifetime |
| Rate limiter cleanup | 10 minutes | Expired per-IP bucket removal |
