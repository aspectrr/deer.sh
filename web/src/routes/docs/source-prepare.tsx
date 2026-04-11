import { createFileRoute } from '@tanstack/react-router'
import { H2, H3 } from '~/components/docs/heading-anchor'
import { CodeBlock } from '~/components/docs/code-block'
import { TerminalBlock } from '~/components/docs/terminal-block'
import { Callout } from '~/components/docs/callout'
import { PrevNext } from '~/components/docs/prev-next'

export const Route = createFileRoute('/docs/source-prepare')({
  component: SourcePreparePage,
})

// Escaped for JS template literal: \${ and \` where needed
const restrictedShellScript = `#!/bin/bash
# deer-readonly-shell - restricted shell for read-only VM access.
# Installed by: deer source prepare
# This shell is set as the login shell for the deer-readonly user.
# Commands are accepted via SSH_ORIGINAL_COMMAND (ForceCommand) or -c arg (login shell).

set -euo pipefail

# Extract command from SSH_ORIGINAL_COMMAND or login shell -c invocation
if [ -n "\${SSH_ORIGINAL_COMMAND:-}" ]; then
    CMD="$SSH_ORIGINAL_COMMAND"
elif [ "\${1:-}" = "-c" ] && [ -n "\${2:-}" ]; then
    CMD="$2"
else
    echo "ERROR: Interactive login is not permitted. This account is for read-only SSH commands only." >&2
    exit 1
fi

# Blocked command patterns (destructive operations)
BLOCKED_PATTERNS=(
    "^sudo "
    "^su "
    "^rm "
    "^mv "
    "^cp "
    "^dd "
    "^kill "
    "^killall "
    "^pkill "
    "^shutdown "
    "^reboot "
    "^halt "
    "^poweroff "
    "^init "
    "^telinit "
    "^chmod "
    "^chown "
    "^chgrp "
    "^useradd "
    "^userdel "
    "^usermod "
    "^groupadd "
    "^groupdel "
    "^groupmod "
    "^passwd "
    "^chpasswd "
    "^mkfs"
    "^mount "
    "^umount "
    "^fdisk "
    "^parted "
    "^lvm "
    "^mdadm "
    "^wget "
    "^curl "
    "^scp "
    "^rsync "
    "^ftp "
    "^sftp "
    "^python"
    "^perl "
    "^ruby "
    "^node "
    "^bash "
    "^sh "
    "^zsh "
    "^dash "
    "^csh "
    "^vi "
    "^vim "
    "^nano "
    "^emacs "
    "^sed -i"
    "^tee "
    "^install "
    "^make "
    "^gcc "
    "^g++ "
    "^cc "
    "^iptables "
    "^ip6tables "
    "^nft "
    "^systemctl start"
    "^systemctl stop"
    "^systemctl restart"
    "^systemctl reload"
    "^systemctl enable"
    "^systemctl disable"
    "^systemctl daemon"
    "^systemctl mask"
    "^systemctl unmask"
    "^systemctl edit"
    "^systemctl set"
    "^apt install"
    "^apt remove"
    "^apt purge"
    "^apt autoremove"
    "^apt-get "
    "^dpkg -i"
    "^dpkg --install"
    "^dpkg --remove"
    "^dpkg --purge"
    "^rpm -i"
    "^rpm --install"
    "^rpm -e"
    "^rpm --erase"
    "^yum "
    "^dnf "
    "^pip install"
    "^pip uninstall"
    "^pip3 install"
    "^pip3 uninstall"
)

# Block command substitution and subshells
# Check for $(...), backticks, <(...), >(...)
if echo "$CMD" | grep -qE '\\$\\(|\\\`|<\\(|>\\('; then
    echo "ERROR: Command substitution and subshells are not permitted." >&2
    exit 126
fi

# Block output redirection
if echo "$CMD" | grep -qE '[^"'"'"']>[^&]|[^"'"'"']>>'; then
    echo "ERROR: Output redirection is not permitted." >&2
    exit 126
fi

# Block newlines (commands must be single-line)
if [[ "$CMD" == *$'\\n'* ]]; then
    echo "ERROR: Multi-line commands are not permitted." >&2
    exit 126
fi

# Split command on all shell separators: | || ; && (and newlines, already blocked above)
# We need to parse the command to split on these operators outside of quotes.
# For defense-in-depth, we'll use a bash function to split properly.

# Parse and validate each segment
parse_and_validate_segments() {
    local cmd="$1"
    local segment=""
    local in_single_quote=false
    local in_double_quote=false
    local prev_char=""
    local i

    for (( i=0; i<\${#cmd}; i++ )); do
        local char="\${cmd:$i:1}"
        local next_char="\${cmd:$((i+1)):1}"

        # Track quote state
        if [[ "$char" == "'" && "$in_double_quote" == false && "$prev_char" != "\\\\" ]]; then
            if [[ "$in_single_quote" == true ]]; then
                in_single_quote=false
            else
                in_single_quote=true
            fi
            segment+="$char"
        elif [[ "$char" == '"' && "$in_single_quote" == false && "$prev_char" != "\\\\" ]]; then
            if [[ "$in_double_quote" == true ]]; then
                in_double_quote=false
            else
                in_double_quote=true
            fi
            segment+="$char"
        # Check for separators outside quotes
        elif [[ "$in_single_quote" == false && "$in_double_quote" == false ]]; then
            if [[ "$char" == "|" ]]; then
                # Check for ||
                if [[ "$next_char" == "|" ]]; then
                    validate_segment "$segment"
                    segment=""
                    ((i++))  # Skip next |
                else
                    validate_segment "$segment"
                    segment=""
                fi
            elif [[ "$char" == ";" ]]; then
                validate_segment "$segment"
                segment=""
            elif [[ "$char" == "&" && "$next_char" == "&" ]]; then
                validate_segment "$segment"
                segment=""
                ((i++))  # Skip next &
            else
                segment+="$char"
            fi
        else
            segment+="$char"
        fi

        prev_char="$char"
    done

    # Validate the last segment
    if [[ -n "$segment" ]]; then
        validate_segment "$segment"
    fi
}

validate_segment() {
    local segment="$1"
    # Trim leading whitespace
    segment="\${segment#"\${segment%%[![:space:]]*}"}"

    # Skip empty segments
    [[ -z "$segment" ]] && return

    for pattern in "\${BLOCKED_PATTERNS[@]}"; do
        if echo "$segment" | grep -qE "$pattern"; then
            echo "ERROR: Command blocked by restricted shell: $segment" >&2
            exit 126
        fi
    done
}

# Validate all segments
parse_and_validate_segments "$CMD"

# Execute the command
exec /bin/bash -c "$CMD"
`

function SourcePreparePage() {
  return (
    <div className="mx-auto max-w-2xl px-6 py-8">
      <h1 className="mb-1 text-lg font-medium text-white">Source VM Preparation</h1>
      <p className="text-muted-foreground mb-8 text-xs">
        Prepare source VMs for read-only access by deer agents.
      </p>

      <Callout type="info" title="Automatic during onboarding">
        These commands run automatically when you use{' '}
        <code className="text-green-800">deer source prepare</code> during CLI onboarding. This page
        is for manual setup or understanding what the preparation does.
      </Callout>

      <H2>On Each Source VM</H2>
      <p className="mb-4 text-xs text-neutral-400">
        Run these commands as root on every VM you want deer agents to inspect.
      </p>

      <H3>1. Install the restricted shell</H3>
      <p className="mb-2 text-xs text-neutral-400">
        Write this script to the source VM. It becomes the login shell for the deer-readonly user.
      </p>
      <CodeBlock
        code={restrictedShellScript}
        lang="bash"
        filename="/usr/local/bin/deer-readonly-shell"
      />
      <TerminalBlock lines={[{ command: 'chmod 755 /usr/local/bin/deer-readonly-shell' }]} />

      <H3>2. Create the deer-readonly user</H3>
      <TerminalBlock
        lines={[
          { command: 'mkdir -p /var/empty' },
          {
            command:
              'useradd -r -s /usr/local/bin/deer-readonly-shell -d /var/empty -M deer-readonly',
          },
        ]}
      />

      <H3>3. Install the SSH CA public key</H3>
      <TerminalBlock
        lines={[
          { command: "cat > /etc/ssh/deer_ca.pub << 'EOF'" },
          { output: '<your CA public key>' },
          { output: 'EOF' },
          { command: 'chmod 644 /etc/ssh/deer_ca.pub' },
        ]}
      />
      <Callout type="info" title="Where to find the CA key">
        The CA public key is at <code className="text-green-800">~/.deer/ca.pub</code> on the
        machine running the deer CLI, or at{' '}
        <code className="text-green-800">/etc/deer-daemon/ca.pub</code> on the daemon host.
      </Callout>

      <H3>4. Configure sshd</H3>
      <p className="mb-2 text-xs text-neutral-400">
        Add CA trust and principal-based authorization to sshd_config.
      </p>
      <TerminalBlock
        lines={[
          {
            command:
              "grep -q 'TrustedUserCAKeys /etc/ssh/deer_ca.pub' /etc/ssh/sshd_config || \\\n  echo 'TrustedUserCAKeys /etc/ssh/deer_ca.pub' >> /etc/ssh/sshd_config",
          },
          {
            command:
              "grep -q 'AuthorizedPrincipalsFile /etc/ssh/authorized_principals/%u' /etc/ssh/sshd_config || \\\n  echo 'AuthorizedPrincipalsFile /etc/ssh/authorized_principals/%u' >> /etc/ssh/sshd_config",
          },
        ]}
      />

      <H3>5. Create authorized principals</H3>
      <TerminalBlock
        lines={[
          { command: 'mkdir -p /etc/ssh/authorized_principals' },
          {
            command: "echo 'deer-readonly' > /etc/ssh/authorized_principals/deer-readonly",
          },
          { command: 'chmod 644 /etc/ssh/authorized_principals/deer-readonly' },
        ]}
      />

      <H3>6. Restart sshd</H3>
      <TerminalBlock lines={[{ command: 'systemctl restart sshd' }]} />

      <H2>On Each Source Host</H2>
      <p className="mb-4 text-xs text-neutral-400">
        The daemon needs SSH access to each source host (hypervisor) to manage VMs via{' '}
        <code className="text-green-800">qemu+ssh://deer-daemon@host/system</code>. Source hosts are
        configured in the daemon's <code className="text-green-800">daemon.yaml</code> under{' '}
        <code className="text-green-800">source_hosts</code>.
      </p>

      <H3>1. Create deer-daemon system user</H3>
      <p className="mb-2 text-xs text-neutral-400">
        On each source host, create the user that the daemon will SSH in as.
      </p>
      <TerminalBlock
        lines={[
          {
            command:
              'id deer-daemon >/dev/null 2>&1 || useradd --system --shell /bin/bash -m deer-daemon',
          },
          { command: 'usermod -aG libvirt deer-daemon' },
        ]}
      />

      <H3>2. Deploy the daemon SSH key</H3>
      <p className="mb-2 text-xs text-neutral-400">
        The daemon generates an SSH identity key pair on first start. Deploy the public key to each
        source host so the daemon can SSH in as <code className="text-green-800">deer-daemon</code>.
      </p>

      <H3>Option A: Automatic (recommended)</H3>
      <p className="mb-2 text-xs text-neutral-400">
        When you run <code className="text-green-800">deer connect</code> or use the{' '}
        <code className="text-green-800">/connect</code> TUI command, the CLI fetches the daemon's
        source host list and identity key, then offers to deploy the key to each source host
        automatically. Your local SSH user must have sudo access on the source hosts for this to
        work.
      </p>
      <TerminalBlock
        lines={[
          { command: 'deer connect 192.168.1.100:9091' },
          { output: 'Connected!' },
          { output: 'Running doctor checks...' },
          { output: 'Enter: deploy daemon key to source hosts' },
        ]}
      />

      <H3>Option B: Manual</H3>
      <p className="mb-2 text-xs text-neutral-400">
        Copy the daemon's public key from the daemon host, then deploy it on each source host.
      </p>
      <TerminalBlock
        lines={[
          { command: '# On the daemon host - get the public key' },
          { command: 'cat /etc/deer-daemon/identity.pub' },
        ]}
      />
      <TerminalBlock
        lines={[
          { command: '# On each source host - deploy to deer-daemon user' },
          { command: 'mkdir -p ~deer-daemon/.ssh && chmod 700 ~deer-daemon/.ssh' },
          {
            command: "echo '<daemon-pub-key>' >> ~deer-daemon/.ssh/authorized_keys",
          },
          { command: 'chmod 600 ~deer-daemon/.ssh/authorized_keys' },
          { command: 'chown -R deer-daemon:deer-daemon ~deer-daemon/.ssh' },
        ]}
      />

      <H3>3. Add source host keys to daemon's known_hosts</H3>
      <p className="mb-2 text-xs text-neutral-400">
        The daemon needs the source host's SSH host key in its known_hosts file. The{' '}
        <code className="text-green-800">deer connect</code> wizard does this automatically via the
        daemon's <code className="text-green-800">ScanSourceHostKeys</code> RPC after deploying
        keys. For manual setup:
      </p>
      <TerminalBlock
        lines={[
          { command: '# On the daemon host' },
          {
            command:
              'ssh-keyscan -H <source-host-ip> | sudo -u deer-daemon tee -a ~deer-daemon/.ssh/known_hosts',
          },
        ]}
      />

      <H2>Verification</H2>
      <p className="mb-4 text-xs text-neutral-400">
        Run these commands on the daemon host to confirm the daemon can reach the source host.
      </p>

      <H3>Test SSH connectivity</H3>
      <TerminalBlock
        lines={[
          {
            command:
              'sudo -u deer-daemon ssh -i /etc/deer-daemon/identity deer-daemon@<source-host-ip> "echo ok"',
          },
        ]}
      />

      <H3>Test libvirt connectivity</H3>
      <TerminalBlock
        lines={[
          {
            command:
              'sudo -u deer-daemon virsh -c "qemu+ssh://deer-daemon@<source-host-ip>/system?keyfile=/etc/deer-daemon/identity" list --all',
          },
        ]}
      />

      <Callout type="info" title="Doctor checks verify connectivity">
        After connecting, run <code className="text-green-800">deer doctor</code> or press{' '}
        <code className="text-green-800">r</code> in the connect wizard to re-run doctor checks. The
        daemon will verify SSH+libvirt connectivity to each configured source host and report any
        failures with fix commands.
      </Callout>

      <Callout type="tip" title="About the restricted shell">
        The restricted shell blocks destructive commands (rm, mv, sudo, curl, etc.), command
        substitution, output redirection, and multi-line commands. Only read-only inspection is
        allowed through the deer-readonly user.
      </Callout>

      <PrevNext />
    </div>
  )
}
