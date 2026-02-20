import { useState, type ReactNode } from 'react'
import { ChevronDown, ChevronRight } from 'lucide-react'
import type { Step } from '~/components/docs/step-tracker'
import { TerminalBlock } from '~/components/docs/terminal-block'
import { CodeBlock } from '~/components/docs/code-block'
import { Callout } from '~/components/docs/callout'
import { cn } from '~/lib/utils'

const packageTabs = [
  {
    id: 'apt',
    label: 'apt',
    command:
      'sudo apt update && sudo apt install -y qemu-system-x86 qemu-utils libvirt-daemon-system libvirt-clients iproute2 bridge-utils openssh-client',
  },
  {
    id: 'dnf',
    label: 'dnf',
    command:
      'sudo dnf install -y qemu-kvm qemu-img libvirt libvirt-client iproute bridge-utils openssh-clients',
  },
  {
    id: 'yum',
    label: 'yum',
    command:
      'sudo yum install -y qemu-kvm qemu-img libvirt libvirt-client iproute bridge-utils openssh-clients',
  },
  {
    id: 'pacman',
    label: 'pacman',
    command: 'sudo pacman -S --noconfirm qemu-full libvirt iproute2 bridge-utils openssh',
  },
] as const

function PackageInstallTabs() {
  const [activeTab, setActiveTab] = useState<string>('apt')
  const currentTab = packageTabs.find((t) => t.id === activeTab)!

  return (
    <div className="overflow-hidden rounded-lg bg-neutral-900">
      <div className="flex border-b border-neutral-800">
        {packageTabs.map((tab) => (
          <button
            key={tab.id}
            onClick={() => setActiveTab(tab.id)}
            className={`border-b-2 px-4 py-2 font-mono text-sm transition-colors ${
              activeTab === tab.id
                ? 'border-blue-400 text-blue-400'
                : 'border-transparent text-neutral-500 hover:text-neutral-300'
            }`}
          >
            {tab.label}
          </button>
        ))}
      </div>
      <div className="p-4">
        <TerminalBlock lines={[{ command: currentTab.command }]} />
      </div>
    </div>
  )
}

export const daemonConfig = `# /etc/fluid-daemon/daemon.yaml
listen:
  grpc: ":9091"

# Sandbox backend
backend: qemu

# Storage paths
storage:
  images: /var/lib/fluid-daemon/images
  overlays: /var/lib/fluid-daemon/overlays
  state: /var/lib/fluid-daemon/state.db

# Network
network:
  bridge: fluid0
  subnet: 10.0.0.0/24

# Optional: connect to control plane
# control_plane:
#   address: "cp.fluid.sh:9090"
#   token: "your-host-token"
`

export const systemdUnit = `[Unit]
Description=fluid-daemon sandbox host
After=network.target libvirtd.service

[Service]
User=fluid-daemon
Group=fluid-daemon
ExecStart=/usr/local/bin/fluid-daemon --config /etc/fluid-daemon/daemon.yaml
Restart=on-failure
RestartSec=5
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
`

export const daemonSetupSteps: Step[] = [
  {
    title: 'Import GPG public key',
    content: (
      <>
        <p>Import the signing key to verify release artifacts:</p>
        <TerminalBlock
          lines={[
            {
              command:
                'gpg --keyserver keys.openpgp.org --recv-keys B27DED65CFB30427EE85F8209DD0911D6CB0B643',
            },
          ]}
        />
        <p>Or import from file:</p>
        <TerminalBlock
          lines={[
            {
              command:
                'curl -fsSL https://raw.githubusercontent.com/aspectrr/fluid.sh/main/public-key.asc | gpg --import',
            },
          ]}
        />
      </>
    ),
  },
  {
    title: 'Download release assets',
    content: (
      <>
        <p>Download the daemon binary, checksums, and signature:</p>
        <TerminalBlock
          lines={[
            { command: 'VERSION=0.1.0' },
            { command: 'ARCH=amd64  # or arm64' },
            {
              command:
                'wget https://github.com/aspectrr/fluid.sh/releases/download/v${VERSION}/fluid-daemon_${VERSION}_linux_${ARCH}.tar.gz',
            },
            {
              command:
                'wget https://github.com/aspectrr/fluid.sh/releases/download/v${VERSION}/checksums.txt',
            },
            {
              command:
                'wget https://github.com/aspectrr/fluid.sh/releases/download/v${VERSION}/checksums.txt.sig',
            },
          ]}
        />
      </>
    ),
  },
  {
    title: 'Verify signature and checksum',
    content: (
      <>
        <p>Verify the checksum file was signed by the official key, then validate the archive:</p>
        <TerminalBlock
          lines={[
            { command: 'gpg --verify checksums.txt.sig checksums.txt' },
            { command: 'sha256sum -c checksums.txt --ignore-missing' },
          ]}
        />
        <Callout type="tip">
          If GPG reports "Good signature from..." the checksum file is authentic. The sha256sum
          check then confirms the archive has not been tampered with.
        </Callout>
      </>
    ),
  },
  {
    title: 'Extract and install',
    content: (
      <>
        <p>Extract the archive and install the binary:</p>
        <TerminalBlock
          lines={[
            { command: 'tar -xzf fluid-daemon_${VERSION}_linux_${ARCH}.tar.gz' },
            { command: 'sudo install -m 755 fluid-daemon /usr/local/bin/' },
          ]}
        />
      </>
    ),
  },
  {
    title: 'Install dependencies',
    content: (
      <>
        <p>Install QEMU, libvirt, and networking tools for your distribution:</p>
        <PackageInstallTabs />
        <Callout type="info">
          KVM/hardware virtualization must be enabled in your BIOS/UEFI. Verify with{' '}
          <code className="text-blue-400">egrep -c '(vmx|svm)' /proc/cpuinfo</code> - a non-zero
          result means it is supported.
        </Callout>
      </>
    ),
  },
  {
    title: 'Create system user and directories',
    content: (
      <>
        <p>Create a dedicated system user and the required filesystem layout:</p>
        <TerminalBlock
          lines={[
            {
              command:
                'sudo useradd --system --home /var/lib/fluid-daemon --shell /usr/sbin/nologin fluid-daemon',
            },
            {
              command:
                'sudo mkdir -p /etc/fluid-daemon /var/lib/fluid-daemon /var/log/fluid-daemon',
            },
            {
              command:
                'sudo chown -R fluid-daemon:fluid-daemon /var/lib/fluid-daemon /var/log/fluid-daemon',
            },
          ]}
        />
        <p>Filesystem layout:</p>
        <CodeBlock
          code={`/usr/local/bin/fluid-daemon        # binary
/etc/fluid-daemon/daemon.yaml       # configuration
/var/lib/fluid-daemon/              # state, images, overlays
/var/log/fluid-daemon/              # logs`}
          lang="text"
          filename="filesystem layout"
        />
      </>
    ),
  },
  {
    title: 'Configure daemon.yaml',
    content: (
      <>
        <p>Create the configuration file:</p>
        <CodeBlock code={daemonConfig} lang="yaml" filename="/etc/fluid-daemon/daemon.yaml" />
        <Callout type="tip">
          The <code className="text-green-400">control_plane</code> section is optional. Uncomment
          it when you are ready to connect to the hosted Fluid control plane (or self-host locally).
        </Callout>
      </>
    ),
  },
  {
    title: 'Create systemd unit',
    content: (
      <>
        <p>Create the systemd service file:</p>
        <CodeBlock
          code={systemdUnit}
          lang="ini"
          filename="/etc/systemd/system/fluid-daemon.service"
        />
        <Callout type="info">
          The <code className="text-green-400">fluid-daemon</code> user needs access to libvirt. Add
          it to the <code className="text-green-400">libvirt</code> group:{' '}
          <code className="text-blue-400">sudo usermod -aG libvirt fluid-daemon</code>
        </Callout>
      </>
    ),
  },
  {
    title: 'Enable and start',
    content: (
      <>
        <p>Reload systemd, enable, and start the daemon:</p>
        <TerminalBlock
          lines={[
            { command: 'sudo systemctl daemon-reload' },
            { command: 'sudo systemctl enable fluid-daemon' },
            { command: 'sudo systemctl start fluid-daemon' },
            { command: 'sudo systemctl status fluid-daemon' },
            { output: 'Active: active (running)' },
          ]}
        />
      </>
    ),
  },
  {
    title: 'Verify via CLI',
    content: (
      <>
        <p>Launch the TUI to verify the daemon is reachable:</p>
        <TerminalBlock
          lines={[
            { command: 'fluid' },
            { output: 'fluid.sh v0.1.0' },
            { output: 'Connected to daemon at localhost:9091' },
          ]}
        />
        <p>
          If the connection fails, check the daemon logs with{' '}
          <code className="text-blue-400">journalctl -u fluid-daemon</code>.
        </p>
      </>
    ),
  },
  {
    title: 'Connect to control plane (optional)',
    content: (
      <>
        <p>
          To enable multi-host orchestration and the web dashboard, connect the daemon to a control
          plane instance:
        </p>
        <CodeBlock
          code={`control_plane:
  address: "cp.fluid.sh:9090"
  token: "host_abc123..."`}
          lang="yaml"
          filename="daemon.yaml (append)"
        />
        <p>Restart the daemon to pick up the change:</p>
        <TerminalBlock lines={[{ command: 'sudo systemctl restart fluid-daemon' }]} />
        <p>
          See{' '}
          <a href="/docs/upgrade" className="text-blue-400 hover:text-blue-300">
            Upgrade to Hosted
          </a>{' '}
          for the full walkthrough.
        </p>
      </>
    ),
  },
]

// Substeps for the quickstart page - excludes the optional "Connect to control plane" step
// and adds a "Navigate to your sandbox host" intro step
interface Substep {
  title: string
  content: ReactNode
}

const quickstartSubsteps: Substep[] = [
  {
    title: 'Navigate to your sandbox host',
    content: (
      <>
        <p>
          SSH into the free host that can connect to the VMs you want to work on. This will be the
          host where the sandboxes will run. This should be a Linux ARM/AMD host with nothing else
          running.
        </p>
        <TerminalBlock lines={[{ command: 'ssh user@your-sandbox-host' }]} />
      </>
    ),
  },
  // Reuse steps 0-9 from daemonSetupSteps (everything except "Connect to control plane")
  ...daemonSetupSteps.slice(0, 10),
]

export function DaemonSubsteps() {
  const [expandedSubstep, setExpandedSubstep] = useState<number>(-1)

  return (
    <div className="mt-3 border-l border-neutral-700 pl-4">
      <div className="space-y-0.5">
        {quickstartSubsteps.map((substep, i) => {
          const isExpanded = expandedSubstep === i

          return (
            <div key={i}>
              <button
                onClick={() => setExpandedSubstep(isExpanded ? -1 : i)}
                className="group flex w-full items-center gap-2 py-1 text-left"
              >
                <span
                  className={cn(
                    'flex h-5 w-5 shrink-0 items-center justify-center text-[10px] font-medium',
                    isExpanded ? 'bg-blue-400/20 text-blue-400' : 'bg-neutral-800 text-neutral-500'
                  )}
                >
                  {i + 1}
                </span>
                <span
                  className={cn(
                    'flex-1 text-xs',
                    isExpanded ? 'text-white' : 'text-neutral-400 group-hover:text-neutral-300'
                  )}
                >
                  {substep.title}
                </span>
                {isExpanded ? (
                  <ChevronDown className="h-3 w-3 text-neutral-600" />
                ) : (
                  <ChevronRight className="h-3 w-3 text-neutral-600" />
                )}
              </button>

              {isExpanded && (
                <div className="ml-7 pb-2">
                  <div className="docs-prose">{substep.content}</div>
                </div>
              )}
            </div>
          )
        })}
      </div>
    </div>
  )
}
