import { useState, type ReactNode } from 'react'
import { ChevronDown, ChevronRight } from 'lucide-react'
import type { Step } from '~/components/docs/step-tracker'
import { TerminalBlock } from '~/components/docs/terminal-block'
import { CodeBlock } from '~/components/docs/code-block'
import { Callout } from '~/components/docs/callout'
import { cn } from '~/lib/utils'

// -- Tabbed package manager components --

interface TabDef {
  id: string
  label: string
  lines: Array<{ command?: string; output?: string }>
}

function PackageManagerTabs({ tabs }: { tabs: TabDef[] }) {
  const [activeTab, setActiveTab] = useState<string>(tabs[0].id)
  const currentTab = tabs.find((t) => t.id === activeTab)!

  return (
    <div className="overflow-hidden rounded-lg bg-neutral-900">
      <div className="flex border-b border-neutral-800">
        {tabs.map((tab) => (
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
        <TerminalBlock lines={currentTab.lines} />
      </div>
    </div>
  )
}

// -- Step data --

const repoSetupTabs: TabDef[] = [
  {
    id: 'apt',
    label: 'apt (Debian/Ubuntu)',
    lines: [
      {
        command:
          'curl -fsSL https://packages.fluid.sh/gpg | sudo gpg --dearmor -o /usr/share/keyrings/fluid.gpg',
      },
      {
        command:
          'echo "deb [signed-by=/usr/share/keyrings/fluid.gpg] https://packages.fluid.sh/apt stable main" | sudo tee /etc/apt/sources.list.d/fluid.list',
      },
      { command: 'sudo apt update' },
    ],
  },
  {
    id: 'dnf',
    label: 'dnf (Fedora/RHEL)',
    lines: [
      { command: 'sudo rpm --import https://packages.fluid.sh/gpg' },
      {
        command: `sudo tee /etc/yum.repos.d/fluid.repo <<'EOF'
[fluid]
name=Fluid
baseurl=https://packages.fluid.sh/yum/$basearch
enabled=1
gpgcheck=1
gpgkey=https://packages.fluid.sh/gpg
EOF`,
      },
    ],
  },
  {
    id: 'yum',
    label: 'yum (CentOS)',
    lines: [
      { command: 'sudo rpm --import https://packages.fluid.sh/gpg' },
      {
        command: `sudo tee /etc/yum.repos.d/fluid.repo <<'EOF'
[fluid]
name=Fluid
baseurl=https://packages.fluid.sh/yum/$basearch
enabled=1
gpgcheck=1
gpgkey=https://packages.fluid.sh/gpg
EOF`,
      },
    ],
  },
]

const installTabs: TabDef[] = [
  {
    id: 'apt',
    label: 'apt',
    lines: [{ command: 'sudo apt install fluid-daemon' }],
  },
  {
    id: 'dnf',
    label: 'dnf',
    lines: [{ command: 'sudo dnf install fluid-daemon' }],
  },
  {
    id: 'yum',
    label: 'yum',
    lines: [{ command: 'sudo yum install fluid-daemon' }],
  },
]

export const daemonSetupSteps: Step[] = [
  {
    title: 'Add the package repository',
    content: (
      <>
        <p>Add the Fluid package repository for your distribution:</p>
        <PackageManagerTabs tabs={repoSetupTabs} />
      </>
    ),
  },
  {
    title: 'Install fluid-daemon',
    content: (
      <>
        <p>
          Install the daemon package. This creates a{' '}
          <code className="text-green-400">fluid-daemon</code> system user, installs the systemd
          unit, and places a default config at{' '}
          <code className="text-green-400">/etc/fluid-daemon/daemon.yaml</code>.
        </p>
        <PackageManagerTabs tabs={installTabs} />
        <Callout type="info">
          QEMU, libvirt, and networking tools are listed as recommended dependencies. On
          Debian/Ubuntu they install automatically; on RHEL/Fedora install them separately if
          needed: <code className="text-blue-400">sudo dnf install qemu-kvm libvirt</code>
        </Callout>
      </>
    ),
  },
  {
    title: 'Configure and start',
    content: (
      <>
        <p>
          Edit the config if needed (e.g. to change the network subnet or storage paths), then start
          the service:
        </p>
        <TerminalBlock
          lines={[
            { command: 'sudo vi /etc/fluid-daemon/daemon.yaml  # optional' },
            { command: 'sudo systemctl start fluid-daemon' },
            { command: 'sudo systemctl status fluid-daemon' },
            { output: 'Active: active (running)' },
          ]}
        />
        <Callout type="tip">
          The package already enables the service at boot. Your config file is preserved across
          upgrades.
        </Callout>
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
  address: "api.fluid.sh:9090"
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

// -- Manual installation steps (collapsed) --

const depsTabs: TabDef[] = [
  {
    id: 'apt',
    label: 'apt',
    lines: [
      {
        command:
          'sudo apt update && sudo apt install -y qemu-system-x86 qemu-utils libvirt-daemon-system libvirt-clients iproute2 bridge-utils openssh-client',
      },
    ],
  },
  {
    id: 'dnf',
    label: 'dnf',
    lines: [
      {
        command:
          'sudo dnf install -y qemu-kvm qemu-img libvirt libvirt-client iproute bridge-utils openssh-clients',
      },
    ],
  },
  {
    id: 'yum',
    label: 'yum',
    lines: [
      {
        command:
          'sudo yum install -y qemu-kvm qemu-img libvirt libvirt-client iproute bridge-utils openssh-clients',
      },
    ],
  },
  {
    id: 'pacman',
    label: 'pacman',
    lines: [
      {
        command: 'sudo pacman -S --noconfirm qemu-full libvirt iproute2 bridge-utils openssh',
      },
    ],
  },
]

const daemonConfig = `# /etc/fluid-daemon/daemon.yaml
listen:
  grpc: ":9091"

backend: qemu

storage:
  images: /var/lib/fluid-daemon/images
  overlays: /var/lib/fluid-daemon/overlays
  state: /var/lib/fluid-daemon/state.db

network:
  bridge: fluid0
  subnet: 10.0.0.0/24

ssh:
  ca_key_path: /etc/fluid-daemon/ssh_ca
  ca_pub_key_path: /etc/fluid-daemon/ssh_ca.pub
  key_dir: /etc/fluid-daemon/keys
  cert_ttl: 30m
  default_user: sandbox
  identity_file: /etc/fluid-daemon/identity

# Optional: connect to control plane
# control_plane:
#   address: "api.fluid.sh:9090"
#   token: "your-host-token"
`

const systemdUnit = `[Unit]
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

const manualSteps: Array<{ title: string; content: ReactNode }> = [
  {
    title: 'Install dependencies',
    content: (
      <>
        <p>Install QEMU, libvirt, and networking tools:</p>
        <PackageManagerTabs tabs={depsTabs} />
      </>
    ),
  },
  {
    title: 'Import GPG public key',
    content: (
      <>
        <p>Import the signing key to verify release artifacts:</p>
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
    title: 'Download and verify',
    content: (
      <>
        <p>Download the daemon binary, verify signature and checksum:</p>
        <TerminalBlock
          lines={[
            { command: 'VERSION=0.1.3' },
            { command: 'ARCH=amd64  # or arm64' },
            {
              command:
                'wget https://github.com/aspectrr/fluid.sh/releases/download/v${VERSION}/fluid-daemon_${VERSION}_linux_${ARCH}.tar.gz',
            },
            {
              command:
                'wget https://github.com/aspectrr/fluid.sh/releases/download/v${VERSION}/checksums.txt{,.sig}',
            },
            { command: 'gpg --verify checksums.txt.sig checksums.txt' },
            { command: 'sha256sum -c checksums.txt --ignore-missing' },
          ]}
        />
      </>
    ),
  },
  {
    title: 'Extract, install, and configure',
    content: (
      <>
        <TerminalBlock
          lines={[
            { command: 'tar -xzf fluid-daemon_${VERSION}_linux_${ARCH}.tar.gz' },
            { command: 'sudo install -m 755 fluid-daemon /usr/local/bin/' },
            {
              command:
                'sudo useradd --system --home /var/lib/fluid-daemon --shell /usr/sbin/nologin fluid-daemon',
            },
            {
              command:
                'sudo mkdir -p /etc/fluid-daemon /var/lib/fluid-daemon/{images,overlays} /var/log/fluid-daemon',
            },
            {
              command:
                'sudo chown -R fluid-daemon:fluid-daemon /var/lib/fluid-daemon /var/log/fluid-daemon',
            },
            {
              command:
                'sudo ssh-keygen -t ed25519 -f /etc/fluid-daemon/ssh_ca -N "" -C "fluid-daemon CA"',
            },
            {
              command:
                'sudo chown fluid-daemon:fluid-daemon /etc/fluid-daemon/ssh_ca /etc/fluid-daemon/ssh_ca.pub',
            },
            {
              command:
                'sudo ssh-keygen -t ed25519 -f /etc/fluid-daemon/identity -N "" -C "fluid-daemon"',
            },
            {
              command:
                'sudo chown fluid-daemon:fluid-daemon /etc/fluid-daemon/identity /etc/fluid-daemon/identity.pub',
            },
          ]}
        />
        <Callout type="info">
          Deploy <code className="text-green-400">/etc/fluid-daemon/identity.pub</code> to{' '}
          <code className="text-green-400">~/.ssh/authorized_keys</code> on each source VM host so
          the daemon can SSH to them for virsh and rsync operations. The daemon's source prepare
          step will automatically install this key on source VMs.
        </Callout>
        <p>Create the configuration file:</p>
        <CodeBlock code={daemonConfig} lang="yaml" filename="/etc/fluid-daemon/daemon.yaml" />
        <p>Create the systemd service file:</p>
        <CodeBlock
          code={systemdUnit}
          lang="ini"
          filename="/etc/systemd/system/fluid-daemon.service"
        />
        <p>Enable and start:</p>
        <TerminalBlock
          lines={[
            { command: 'sudo systemctl daemon-reload' },
            { command: 'sudo systemctl enable --now fluid-daemon' },
          ]}
        />
      </>
    ),
  },
]

export function ManualInstallSteps() {
  const [expanded, setExpanded] = useState(false)
  const [expandedStep, setExpandedStep] = useState<number>(-1)

  return (
    <div className="mt-4">
      <button
        onClick={() => setExpanded(!expanded)}
        className="text-xs text-blue-400 transition-colors hover:text-blue-300"
      >
        {expanded ? 'Hide' : 'Show'} manual installation
      </button>
      {expanded && (
        <div className="mt-3 border-l border-neutral-700 pl-4">
          <div className="space-y-0.5">
            {manualSteps.map((step, i) => {
              const isExpanded = expandedStep === i
              return (
                <div key={i}>
                  <button
                    onClick={() => setExpandedStep(isExpanded ? -1 : i)}
                    className="group flex w-full items-center gap-2 py-1 text-left"
                  >
                    <span
                      className={cn(
                        'flex h-5 w-5 shrink-0 items-center justify-center text-[10px] font-medium',
                        isExpanded
                          ? 'bg-blue-400/20 text-blue-400'
                          : 'bg-neutral-800 text-neutral-500'
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
                      {step.title}
                    </span>
                    {isExpanded ? (
                      <ChevronDown className="h-3 w-3 text-neutral-600" />
                    ) : (
                      <ChevronRight className="h-3 w-3 text-neutral-600" />
                    )}
                  </button>
                  {isExpanded && (
                    <div className="ml-7 pb-2">
                      <div className="docs-prose">{step.content}</div>
                    </div>
                  )}
                </div>
              )
            })}
          </div>
        </div>
      )}
    </div>
  )
}

// -- Substeps for quickstart page --

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
  // Reuse steps 0-2 from daemonSetupSteps (everything except "Connect to control plane")
  ...daemonSetupSteps.slice(0, 3),
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
