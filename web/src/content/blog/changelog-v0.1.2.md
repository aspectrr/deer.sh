---
title: 'Changelog #002 v0.1.2'
pubDate: 2026-02-10
description: 'Changelog for v0.1.2'
author: 'Collin @ Fluid.sh'
authorImage: '/images/skeleton_smoking_cigarette.jpg'
authorEmail: 'cpfeifer@madcactus.org'
authorPhone: '+3179955114'
authorDiscord: 'https://discordapp.com/users/301068417685913600'
---

## Changelog #002

Hi everyone!

This release brings a major addition to Fluid's read-only capabilities -- the ability to read and run diagnostic commands directly on source/golden VMs without creating a sandbox.

## Update to v0.1.2

Update your CLI agent with `go install`

```bash
go install github.com/aspectrr/fluid.sh/fluid/cmd/fluid@latest
```

## Reading Source VMs

The biggest feature in this release is source VM read-only access. Previously, the agent could only interact with sandboxes -- cloned VMs that were safe to modify. Now, the agent can read files and run diagnostic commands directly on your source/golden VMs without ever modifying them.

This is incredibly useful for investigation and debugging on production VMs. Instead of having to clone a VM just to look at a log file or check a service status, the agent can inspect the source VM directly.

### How It Works

Source VM access uses a defense-in-depth security model with two layers of protection:

**Client-side validation** -- Before any command is sent to the VM, it's validated against an allowlist of safe, read-only commands. Things like `cat`, `ls`, `journalctl`, `systemctl status`, `ps`, `df`, and other diagnostic tools are allowed. Destructive commands like `rm`, `kill`, `apt install`, etc. are blocked before they ever reach the VM.

**Server-side restricted shell** -- On the VM itself, a dedicated `fluid-readonly` user is created with a restricted shell. This shell blocks destructive commands as a second line of defense, prevents command substitution and output redirection, and denies interactive login entirely.

### Preparing a Source VM

To enable read-only access on a source VM, run:

```bash
fluid source prepare <vm-name>
```

This will:

1. Install a restricted shell script on the VM
2. Create a `fluid-readonly` user with that restricted shell
3. Set up SSH CA trust so the agent can authenticate with ephemeral certificates
4. Configure authorized principals for the read-only user

This is also run automatically during onboarding or when you try to investigate a VM in read-only mode.

### New Tools

Two new tools are available to the agent in read-only mode:

| Tool                 | Description                                                                                             |
| -------------------- | ------------------------------------------------------------------------------------------------------- |
| `run_source_command` | Execute a read-only diagnostic command on a source VM (ps, ls, cat, systemctl status, journalctl, etc.) |
| `read_source_file`   | Read the contents of a file on a source VM                                                              |

### Allowed Commands

The read-only command allowlist includes:

- **File inspection**: `cat`, `ls`, `find`, `head`, `tail`, `stat`, `file`, `wc`, `du`, `tree`, `strings`, `md5sum`, `sha256sum`
- **Process/system**: `ps`, `top`, `pgrep`, `systemctl status`, `journalctl`, `dmesg`
- **Network**: `ss`, `netstat`, `ip`, `ifconfig`, `dig`, `nslookup`, `ping`
- **Disk**: `df`, `lsblk`, `blkid`
- **Package query**: `dpkg -l`, `rpm -q`, `apt list`, `pip list`
- **System info**: `uname`, `hostname`, `uptime`, `free`, `lscpu`, `lsmod`
- **Pipe targets**: `grep`, `awk`, `sed`, `sort`, `uniq`, `cut`, `tr`

Subcommands are also restricted -- for example, `systemctl` only allows `status`, `show`, `list-units`, `is-active`, and `is-enabled`.

## Security Hardening

This release also includes several security fixes that came out of hardening the source VM access:

- **Shell injection prevention** -- Path parameters in `read_source_file` are properly escaped to prevent shell injection
- **Metacharacter blocking** -- Command substitution (`$(...)`, backticks), process substitution, output redirection, and newline injection are all blocked at both the client and server level
- **VM name sanitization** -- VM names used in filesystem paths are sanitized to prevent path traversal attacks
- **Pipeline validation** -- Commands chained with `|`, `;`, `&&`, `||` have each segment individually validated

## The 4-Phase Strategy

This feature completes the read phase of Fluid's 4-phase strategy:

**Read:** Read through the source VM for context and initial debugging.

**Edit:** Edit and test failure modes with potentially destructive commands in a VM sandbox.

**Ansible:** After fixing the issue or after debugging, an Ansible playbook will be created to fix this issue on production.

**Cleanup:** Finally, any sandboxes created will be cleaned up when the CLI is closed, or can be closed manually by the agent.

## Come Hang Out

Questions? Join us on [Discord](https://discord.gg/4WGGXJWm8J)

Found a bug? Open an issue on [GitHub](https://github.com/aspectrr/fluid.sh/issues)
