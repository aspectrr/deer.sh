import { createFileRoute } from '@tanstack/react-router'
import { useState } from 'react'
import { StepTracker } from '~/components/docs/step-tracker'
import { TerminalBlock } from '~/components/docs/terminal-block'
import { CodeBlock } from '~/components/docs/code-block'
import { Callout } from '~/components/docs/callout'
import { PrevNext } from '~/components/docs/prev-next'

export const Route = createFileRoute('/docs/daemon')({
  component: DaemonPage,
})

const daemonConfig = `# /etc/fluid-daemon/daemon.yaml
listen:
  grpc: ":9091"

# Sandbox backend
backend: firecracker  # or "qemu"

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

function DaemonPage() {
  const [showScript, setShowScript] = useState(false)
  const [showSource, setShowSource] = useState(false)

  return (
    <div className="mx-auto max-w-2xl px-6 py-8">
      <h1 className="mb-1 text-lg font-medium text-white">Daemon Setup</h1>
      <p className="text-muted-foreground mb-8 text-xs">
        Install and configure fluid-daemon on each sandbox host.
      </p>

      <Callout type="info" title="One daemon per sandbox host">
        The daemon runs on each machine where you want to run sandboxes. Each daemon can connect to
        multiple libvirt hosts over SSH for source VM access. Multiple daemons are typically needed
        for heavily NATed enterprise networks or separate data centers.
      </Callout>

      {/* Alternative install methods (collapsed) */}
      <div className="mb-6 space-y-2">
        <button
          onClick={() => setShowScript(!showScript)}
          className="text-xs text-blue-400 transition-colors hover:text-blue-300"
        >
          {showScript ? 'Hide' : 'Show'} install script method
        </button>
        {showScript && (
          <div className="border-border border bg-neutral-900/30 p-4">
            <p className="mb-2 text-xs text-neutral-400">
              Quick install (not recommended for production):
            </p>
            <TerminalBlock
              lines={[{ command: 'curl -fsSL https://fluid.sh/install.sh | bash -s -- --daemon' }]}
            />
          </div>
        )}
        <br />
        <button
          onClick={() => setShowSource(!showSource)}
          className="text-xs text-blue-400 transition-colors hover:text-blue-300"
        >
          {showSource ? 'Hide' : 'Show'} build from source
        </button>
        {showSource && (
          <div className="border-border border bg-neutral-900/30 p-4">
            <TerminalBlock
              lines={[
                { command: 'git clone https://github.com/aspectrr/fluid.sh.git' },
                {
                  command:
                    'cd fluid.sh/fluid-daemon && go build -o /usr/local/bin/fluid-daemon ./cmd/fluid-daemon',
                },
              ]}
            />
          </div>
        )}
      </div>

      <StepTracker
        storageKey="daemon"
        steps={[
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
                <p>
                  Verify the checksum file was signed by the official key, then validate the
                  archive:
                </p>
                <TerminalBlock
                  lines={[
                    { command: 'gpg --verify checksums.txt.sig checksums.txt' },
                    { command: 'sha256sum -c checksums.txt --ignore-missing' },
                  ]}
                />
                <Callout type="tip">
                  If GPG reports "Good signature from..." the checksum file is authentic. The
                  sha256sum check then confirms the archive has not been tampered with.
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
                <CodeBlock
                  code={daemonConfig}
                  lang="yaml"
                  filename="/etc/fluid-daemon/daemon.yaml"
                />
                <Callout type="tip">
                  The <code className="text-green-400">control_plane</code> section is optional.
                  Uncomment it when you are ready to connect to a hosted control plane.
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
                  The <code className="text-green-400">fluid-daemon</code> user needs access to
                  libvirt. Add it to the <code className="text-green-400">libvirt</code> group:{' '}
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
                  To enable multi-host orchestration and the web dashboard, connect the daemon to a
                  control plane instance:
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
        ]}
      />

      <PrevNext />
    </div>
  )
}
