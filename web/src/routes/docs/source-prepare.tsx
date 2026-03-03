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
# fluid-readonly-shell - restricted shell for read-only VM access.
# Installed by: fluid source prepare
# This shell is set as the login shell for the fluid-readonly user.
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
        Prepare source VMs for read-only access by fluid agents.
      </p>

      <Callout type="info" title="Automatic during onboarding">
        These commands run automatically when you use{' '}
        <code className="text-blue-400">fluid source prepare</code> during CLI onboarding. This page
        is for manual setup or understanding what the preparation does.
      </Callout>

      <H2>On Each Source VM</H2>
      <p className="mb-4 text-xs text-neutral-400">
        Run these commands as root on every VM you want fluid agents to inspect.
      </p>

      <H3>1. Install the restricted shell</H3>
      <p className="mb-2 text-xs text-neutral-400">
        Write this script to the source VM. It becomes the login shell for the fluid-readonly user.
      </p>
      <CodeBlock
        code={restrictedShellScript}
        lang="bash"
        filename="/usr/local/bin/fluid-readonly-shell"
      />
      <TerminalBlock lines={[{ command: 'chmod 755 /usr/local/bin/fluid-readonly-shell' }]} />

      <H3>2. Create the fluid-readonly user</H3>
      <TerminalBlock
        lines={[
          { command: 'mkdir -p /var/empty' },
          {
            command:
              'useradd -r -s /usr/local/bin/fluid-readonly-shell -d /var/empty -M fluid-readonly',
          },
        ]}
      />

      <H3>3. Install the SSH CA public key</H3>
      <TerminalBlock
        lines={[
          { command: "cat > /etc/ssh/fluid_ca.pub << 'EOF'" },
          { output: '<your CA public key>' },
          { output: 'EOF' },
          { command: 'chmod 644 /etc/ssh/fluid_ca.pub' },
        ]}
      />
      <Callout type="info" title="Where to find the CA key">
        The CA public key is at <code className="text-blue-400">~/.fluid/ca.pub</code> on the
        machine running the fluid CLI, or at{' '}
        <code className="text-blue-400">/etc/fluid-daemon/ca.pub</code> on the daemon host.
      </Callout>

      <H3>4. Configure sshd</H3>
      <p className="mb-2 text-xs text-neutral-400">
        Add CA trust and principal-based authorization to sshd_config.
      </p>
      <TerminalBlock
        lines={[
          {
            command:
              "grep -q 'TrustedUserCAKeys /etc/ssh/fluid_ca.pub' /etc/ssh/sshd_config || \\\n  echo 'TrustedUserCAKeys /etc/ssh/fluid_ca.pub' >> /etc/ssh/sshd_config",
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
            command: "echo 'fluid-readonly' > /etc/ssh/authorized_principals/fluid-readonly",
          },
          { command: 'chmod 644 /etc/ssh/authorized_principals/fluid-readonly' },
        ]}
      />

      <H3>6. Restart sshd</H3>
      <TerminalBlock lines={[{ command: 'systemctl restart sshd' }]} />

      <H2>On Each VM Host</H2>
      <p className="mb-4 text-xs text-neutral-400">
        Configure the host machine where fluid-daemon runs so it can SSH into source VMs.
      </p>

      <H3>1. Create fluid-daemon system user</H3>
      <TerminalBlock
        lines={[
          {
            command:
              'id fluid-daemon >/dev/null 2>&1 || useradd --system --shell /bin/bash -m fluid-daemon',
          },
          { command: 'usermod -aG libvirt fluid-daemon' },
        ]}
      />

      <H3>2. Set up SSH directory</H3>
      <TerminalBlock
        lines={[
          { command: 'mkdir -p /home/fluid-daemon/.ssh' },
          { command: 'chmod 700 /home/fluid-daemon/.ssh' },
          { command: 'chown fluid-daemon:fluid-daemon /home/fluid-daemon/.ssh' },
        ]}
      />

      <H3>3. Deploy the daemon SSH public key</H3>
      <TerminalBlock
        lines={[
          {
            command: "echo '<daemon-pub-key>' >> /home/fluid-daemon/.ssh/authorized_keys",
          },
          {
            command: 'chown fluid-daemon:fluid-daemon /home/fluid-daemon/.ssh/authorized_keys',
          },
          { command: 'chmod 600 /home/fluid-daemon/.ssh/authorized_keys' },
        ]}
      />
      <Callout type="info" title="Key location">
        The daemon's SSH public key is at{' '}
        <code className="text-blue-400">/etc/fluid-daemon/identity.pub</code> on the daemon host.
      </Callout>

      <Callout type="tip" title="About the restricted shell">
        The restricted shell blocks destructive commands (rm, mv, sudo, curl, etc.), command
        substitution, output redirection, and multi-line commands. Only read-only inspection is
        allowed through the fluid-readonly user.
      </Callout>

      <PrevNext />
    </div>
  )
}
