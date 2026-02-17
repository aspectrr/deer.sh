import { createFileRoute } from '@tanstack/react-router'
import { H2, H3 } from '~/components/docs/heading-anchor'
import { TerminalBlock } from '~/components/docs/terminal-block'
import { CodeBlock } from '~/components/docs/code-block'
import { Callout } from '~/components/docs/callout'
import { PrevNext } from '~/components/docs/prev-next'

export const Route = createFileRoute('/docs/local-setup')({
  component: LocalSetupPage,
})

function LocalSetupPage() {
  return (
    <div className="mx-auto max-w-2xl px-6 py-8">
      <h1 className="mb-1 text-lg font-medium text-white">Local Setup</h1>
      <p className="text-muted-foreground mb-8 text-xs">
        Set up libvirt and QEMU on your local machine for development and testing.
      </p>

      <Callout type="info">
        This guide is for local development. For production deployments, the daemon runs on a
        dedicated Linux host with KVM support.
      </Callout>

      <H2>macOS</H2>

      <H3>Prerequisites</H3>
      <TerminalBlock
        lines={[
          { command: 'brew install qemu libvirt socket_vmnet cdrtools' },
          { command: 'brew services start libvirt' },
        ]}
      />

      <H3>Enable Remote Login (SSH)</H3>
      <p className="mb-3 text-xs text-neutral-400">
        The daemon connects to libvirt via SSH. Enable Remote Login in System Settings &gt; General
        &gt; Sharing.
      </p>

      <H3>Create image directories</H3>
      <TerminalBlock
        lines={[
          { command: 'sudo mkdir -p /var/lib/libvirt/images/{base,jobs}' },
          { command: 'sudo chown -R $(whoami) /var/lib/libvirt/images/{base,jobs}' },
        ]}
      />

      <H3>Verify SSH and libvirt</H3>
      <TerminalBlock
        lines={[
          { command: 'ssh $(whoami)@localhost' },
          { command: 'virsh -c qemu+ssh://$(whoami)@localhost/session list --all' },
        ]}
      />

      <H3>Create a test VM</H3>
      <p className="mb-3 text-xs text-neutral-400">
        Use the provided script to set up a base VM with cloud-init:
      </p>
      <TerminalBlock lines={[{ command: './scripts/reset-libvirt-macos.sh' }]} />

      <H2>Linux (x86_64 / ARM64)</H2>

      <H3>Prerequisites</H3>
      <p className="mb-3 text-xs text-neutral-400">Ubuntu / Debian:</p>
      <TerminalBlock
        lines={[
          {
            command:
              'sudo apt update && sudo apt install -y qemu-kvm qemu-utils libvirt-daemon-system libvirt-clients virtinst bridge-utils ovmf cpu-checker cloud-image-utils genisoimage',
          },
        ]}
      />
      <p className="mt-3 mb-3 text-xs text-neutral-400">Fedora / RHEL:</p>
      <TerminalBlock
        lines={[
          {
            command:
              'sudo dnf install -y qemu-kvm qemu-img libvirt libvirt-client virt-install bridge-utils edk2-ovmf cloud-utils genisoimage',
          },
        ]}
      />

      <H3>Enable libvirt</H3>
      <TerminalBlock
        lines={[
          { command: 'sudo systemctl enable --now libvirtd' },
          { command: 'sudo usermod -aG libvirt,kvm $(whoami)' },
          { command: 'newgrp libvirt' },
        ]}
      />

      <H3>Verify KVM</H3>
      <TerminalBlock
        lines={[{ command: 'kvm-ok' }, { command: 'virsh -c qemu:///system list --all' }]}
      />

      <H3>Set up the default network</H3>
      <TerminalBlock
        lines={[
          { command: 'sudo virsh net-autostart default' },
          { command: 'sudo virsh net-start default' },
        ]}
      />

      <H2>Setting Up a Base VM Image</H2>
      <p className="mb-3 text-xs text-neutral-400">
        Download an Ubuntu cloud image and create a VM with cloud-init:
      </p>
      <TerminalBlock
        lines={[
          { command: 'sudo mkdir -p /var/lib/libvirt/images/base' },
          {
            command:
              'cd /var/lib/libvirt/images/base && sudo wget https://cloud-images.ubuntu.com/jammy/current/jammy-server-cloudimg-amd64.img',
          },
        ]}
      />
      <Callout type="tip">
        For ARM64 hosts (Apple Silicon dev, Graviton, etc.), download the arm64 variant:{' '}
        <code className="text-green-400">jammy-server-cloudimg-arm64.img</code>
      </Callout>

      <H2>Configuration</H2>
      <p className="mb-3 text-xs text-neutral-400">
        Set the libvirt URI in your environment or daemon config:
      </p>
      <CodeBlock
        code={`# macOS (SSH to local libvirt)
LIBVIRT_URI=qemu+ssh://$(whoami)@localhost/session

# Linux (direct socket)
LIBVIRT_URI=qemu:///system

# Remote host
LIBVIRT_URI=qemu+ssh://user@remote-host/system`}
        lang="bash"
        filename="Environment variables"
      />

      <H2>Quick Reference</H2>
      <TerminalBlock
        lines={[
          { command: 'virsh -c $LIBVIRT_URI list --all', output: '# List all VMs' },
          { command: 'virsh -c $LIBVIRT_URI start test-vm', output: '# Start a VM' },
          { command: 'virsh -c $LIBVIRT_URI destroy test-vm', output: '# Stop a VM' },
          { command: 'virsh -c $LIBVIRT_URI console test-vm', output: '# VM console' },
          { command: 'virsh -c $LIBVIRT_URI undefine test-vm --nvram', output: '# Delete a VM' },
        ]}
      />

      <H2>Troubleshooting</H2>

      <H3>SSH connection refused</H3>
      <p className="mb-3 text-xs text-neutral-400">
        Verify Remote Login is enabled (macOS) or sshd is running (Linux). Test manually:{' '}
        <code className="text-blue-400">ssh $(whoami)@localhost</code>
      </p>

      <H3>VM won't start</H3>
      <p className="mb-3 text-xs text-neutral-400">Check libvirt is running:</p>
      <TerminalBlock
        lines={[
          { command: 'brew services list | grep libvirt', output: '# macOS' },
          { command: 'systemctl status libvirtd', output: '# Linux' },
        ]}
      />

      <PrevNext />
    </div>
  )
}
