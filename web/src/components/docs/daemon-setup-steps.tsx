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
                ? 'border-green-900 text-green-800'
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
          'curl -fsSL https://packages.deer.sh/gpg | sudo gpg --dearmor -o /usr/share/keyrings/deer.gpg',
      },
      {
        command:
          'echo "deb [signed-by=/usr/share/keyrings/deer.gpg] https://packages.deer.sh/apt stable main" | sudo tee /etc/apt/sources.list.d/deer.list',
      },
      { command: 'sudo apt update' },
    ],
  },
  {
    id: 'dnf',
    label: 'dnf (Fedora/RHEL)',
    lines: [
      { command: 'sudo rpm --import https://packages.deer.sh/gpg' },
      {
        command: `sudo tee /etc/yum.repos.d/deer.repo <<'EOF'
[deer]
name=Fluid
baseurl=https://packages.deer.sh/yum/$basearch
enabled=1
gpgcheck=1
gpgkey=https://packages.deer.sh/gpg
EOF`,
      },
    ],
  },
  {
    id: 'yum',
    label: 'yum (CentOS)',
    lines: [
      { command: 'sudo rpm --import https://packages.deer.sh/gpg' },
      {
        command: `sudo tee /etc/yum.repos.d/deer.repo <<'EOF'
[deer]
name=Fluid
baseurl=https://packages.deer.sh/yum/$basearch
enabled=1
gpgcheck=1
gpgkey=https://packages.deer.sh/gpg
EOF`,
      },
    ],
  },
]

const installTabs: TabDef[] = [
  {
    id: 'apt',
    label: 'apt',
    lines: [{ command: 'sudo apt install deer-daemon' }],
  },
  {
    id: 'dnf',
    label: 'dnf',
    lines: [{ command: 'sudo dnf install deer-daemon' }],
  },
  {
    id: 'yum',
    label: 'yum',
    lines: [{ command: 'sudo yum install deer-daemon' }],
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
    title: 'Install deer-daemon',
    content: (
      <>
        <p>
          Install the daemon package. This creates a{' '}
          <code className="text-green-800">deer-daemon</code> system user, installs the systemd
          unit, and places a default config at{' '}
          <code className="text-green-800">/etc/deer-daemon/daemon.yaml</code>.
        </p>
        <PackageManagerTabs tabs={installTabs} />
        <Callout type="info">
          QEMU, libvirt, libguestfs-tools, and networking tools are listed as recommended
          dependencies. On Debian/Ubuntu they install automatically; on RHEL/Fedora install them
          separately if needed:{' '}
          <code className="text-green-800">
            sudo dnf install qemu-kvm libvirt libguestfs-tools-c
          </code>
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
            { command: 'sudo vi /etc/deer-daemon/daemon.yaml  # optional' },
            { command: 'sudo systemctl start deer-daemon' },
            { command: 'sudo systemctl status deer-daemon' },
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
    title: 'Connect from the CLI',
    content: (
      <>
        <p>
          Once the daemon is running, connect your CLI to it. This tests the gRPC connection, runs
          health checks, and saves the daemon address to your config:
        </p>
        <TerminalBlock
          lines={[
            { command: 'deer connect your-sandbox-host:9091' },
            { output: '  [ok] Health check passed' },
            { output: '  [ok] Host info retrieved' },
            { output: '' },
            { output: '  Hostname:    your-sandbox-host' },
            { output: '  Version:     0.1.3' },
            { output: '  CPUs:        4' },
            { output: '  Memory:      8192 MB' },
            { output: '  Sandboxes:   0 active' },
            { output: '  Images:      2 available' },
            { output: '' },
            { output: '  [ok] Saved "your-sandbox-host" (your-sandbox-host:9091) to config' },
          ]}
        />
        <Callout type="tip">
          You can also use <code className="text-green-800">/connect</code> inside the TUI for a
          guided wizard experience. Use{' '}
          <code className="text-green-800">deer connect --no-save</code> to test without saving.
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
  address: "api.deer.sh:9090"
  token: "host_abc123..."`}
          lang="yaml"
          filename="daemon.yaml (append)"
        />
        <p>Restart the daemon to pick up the change:</p>
        <TerminalBlock lines={[{ command: 'sudo systemctl restart deer-daemon' }]} />
        <p>
          See{' '}
          <a href="/docs/upgrade" className="text-green-800 hover:text-green-900">
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
          'sudo apt update && sudo apt install -y qemu-system-x86 qemu-utils libvirt-daemon-system libvirt-clients libguestfs-tools iproute2 bridge-utils openssh-client',
      },
    ],
  },
  {
    id: 'dnf',
    label: 'dnf',
    lines: [
      {
        command:
          'sudo dnf install -y qemu-kvm qemu-img libvirt libvirt-client libguestfs-tools-c iproute bridge-utils openssh-clients',
      },
    ],
  },
  {
    id: 'yum',
    label: 'yum',
    lines: [
      {
        command:
          'sudo yum install -y qemu-kvm qemu-img libvirt libvirt-client libguestfs-tools-c iproute bridge-utils openssh-clients',
      },
    ],
  },
  {
    id: 'pacman',
    label: 'pacman',
    lines: [
      {
        command:
          'sudo pacman -S --noconfirm qemu-full libvirt libguestfs iproute2 bridge-utils openssh',
      },
    ],
  },
]

const daemonConfig = `# /etc/deer-daemon/daemon.yaml
listen:
  grpc: ":9091"

backend: qemu

storage:
  images: /var/lib/deer-daemon/images
  overlays: /var/lib/deer-daemon/overlays
  state: /var/lib/deer-daemon/state.db

network:
  bridge: deer0
  subnet: 10.0.0.0/24

ssh:
  ca_key_path: /etc/deer-daemon/ssh_ca
  ca_pub_key_path: /etc/deer-daemon/ssh_ca.pub
  key_dir: /var/lib/deer-daemon/keys
  cert_ttl: 30m
  default_user: sandbox
  identity_file: /etc/deer-daemon/identity
  # proxy_jump: "deer-daemon@vm-host"  # Set by CLI onboarding for NAT networks

# Optional: connect to control plane
# control_plane:
#   address: "api.deer.sh:9090"
#   token: "your-host-token"
`

const systemdUnit = `[Unit]
Description=deer-daemon sandbox host
After=network.target libvirtd.service

[Service]
User=deer-daemon
Group=deer-daemon
ExecStart=/usr/local/bin/deer-daemon --config /etc/deer-daemon/daemon.yaml
AmbientCapabilities=CAP_NET_ADMIN
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
                'curl -fsSL https://raw.githubusercontent.com/aspectrr/deer.sh/main/public-key.asc | gpg --import',
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
                'wget https://github.com/aspectrr/deer.sh/releases/download/v${VERSION}/deer-daemon_${VERSION}_linux_${ARCH}.tar.gz',
            },
            {
              command:
                'wget https://github.com/aspectrr/deer.sh/releases/download/v${VERSION}/checksums.txt{,.sig}',
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
            { command: 'tar -xzf deer-daemon_${VERSION}_linux_${ARCH}.tar.gz' },
            { command: 'sudo install -m 755 deer-daemon /usr/local/bin/' },
            {
              command:
                'sudo useradd --system --home /var/lib/deer-daemon --shell /usr/sbin/nologin deer-daemon',
            },
            {
              command:
                'sudo mkdir -p /etc/deer-daemon /var/lib/deer-daemon/{images,overlays} /var/log/deer-daemon',
            },
            {
              command:
                'sudo chown -R deer-daemon:deer-daemon /var/lib/deer-daemon /var/log/deer-daemon',
            },
            {
              command:
                'sudo ssh-keygen -t ed25519 -f /etc/deer-daemon/ssh_ca -N "" -C "deer-daemon CA"',
            },
            {
              command:
                'sudo chown deer-daemon:deer-daemon /etc/deer-daemon/ssh_ca /etc/deer-daemon/ssh_ca.pub',
            },
            {
              command:
                'sudo ssh-keygen -t ed25519 -f /etc/deer-daemon/identity -N "" -C "deer-daemon"',
            },
            {
              command:
                'sudo chown deer-daemon:deer-daemon /etc/deer-daemon/identity /etc/deer-daemon/identity.pub',
            },
          ]}
        />
        <Callout type="info">
          Deploy <code className="text-green-800">/etc/deer-daemon/identity.pub</code> to{' '}
          <code className="text-green-800">~/.ssh/authorized_keys</code> on each{' '}
          <strong>source VM host</strong> (the machine running libvirt) so the daemon can SSH for
          virsh and rsync operations.
        </Callout>
        <p>Create the configuration file:</p>
        <CodeBlock code={daemonConfig} lang="yaml" filename="/etc/deer-daemon/daemon.yaml" />
        <p>Create the systemd service file:</p>
        <CodeBlock
          code={systemdUnit}
          lang="ini"
          filename="/etc/systemd/system/deer-daemon.service"
        />
        <p>Enable and start:</p>
        <TerminalBlock
          lines={[
            { command: 'sudo systemctl daemon-reload' },
            { command: 'sudo systemctl enable --now deer-daemon' },
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
        className="text-xs text-green-800 transition-colors hover:text-green-900"
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
                          ? 'bg-green-800/20 text-green-800'
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
                    isExpanded
                      ? 'bg-green-800/20 text-green-800'
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
