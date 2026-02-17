import { createFileRoute } from '@tanstack/react-router'
import { H2, H3 } from '~/components/docs/heading-anchor'
import { DaemonConnectionAnimation } from '~/components/docs/daemon-connection-animation'
import { Callout } from '~/components/docs/callout'
import { TerminalBlock } from '~/components/docs/terminal-block'
import { CodeBlock } from '~/components/docs/code-block'
import { PrevNext } from '~/components/docs/prev-next'

export const Route = createFileRoute('/docs/upgrade')({
  component: UpgradePage,
})

function UpgradePage() {
  return (
    <div className="mx-auto max-w-2xl px-6 py-8">
      <h1 className="mb-1 text-lg font-medium text-white">Upgrade to Hosted</h1>
      <p className="text-muted-foreground mb-8 text-xs">
        Start with the CLI. When you need multi-host, add the daemon and control plane.
      </p>

      <DaemonConnectionAnimation />

      <H2>How the Connection Works</H2>
      <p className="mb-3 text-xs text-neutral-400">
        The daemon opens a persistent bidirectional gRPC stream to the control plane. The control
        plane sends sandbox creation/destruction commands down the stream. The daemon reports status
        updates back. The dashboard reads state via the control plane's REST API.
      </p>

      <H2>What Changes</H2>
      <div className="border-border mb-6 overflow-x-auto border">
        <table className="w-full text-xs">
          <thead>
            <tr className="border-border border-b bg-neutral-900/50">
              <th className="px-3 py-2 text-left font-medium text-neutral-300">Component</th>
              <th className="px-3 py-2 text-left font-medium text-neutral-300">CLI Only</th>
              <th className="px-3 py-2 text-left font-medium text-neutral-300">Hosted</th>
            </tr>
          </thead>
          <tbody className="text-neutral-400">
            <tr className="border-border border-b">
              <td className="px-3 py-1.5 text-neutral-300">State</td>
              <td className="px-3 py-1.5">Local SQLite</td>
              <td className="px-3 py-1.5">PostgreSQL (control plane)</td>
            </tr>
            <tr className="border-border border-b">
              <td className="px-3 py-1.5 text-neutral-300">Hosts</td>
              <td className="px-3 py-1.5">Single machine</td>
              <td className="px-3 py-1.5">Multi-host fleet</td>
            </tr>
            <tr className="border-border border-b">
              <td className="px-3 py-1.5 text-neutral-300">UI</td>
              <td className="px-3 py-1.5">CLI only</td>
              <td className="px-3 py-1.5">Web dashboard</td>
            </tr>
            <tr className="border-border border-b last:border-0">
              <td className="px-3 py-1.5 text-neutral-300">Placement</td>
              <td className="px-3 py-1.5">N/A</td>
              <td className="px-3 py-1.5">Auto-placement across hosts</td>
            </tr>
          </tbody>
        </table>
      </div>

      <H2>What Stays the Same</H2>
      <ul className="mb-6 ml-4 list-disc space-y-1 text-xs text-neutral-400">
        <li>TUI commands and MCP tools work the same way</li>
        <li>Sandbox behavior is unchanged</li>
        <li>MCP tools work the same way</li>
        <li>SSH access uses the same CA mechanism</li>
        <li>TTL janitor still runs locally on each host</li>
      </ul>

      <H2>Setup Steps</H2>

      <H3>1. Install the daemon</H3>
      <p className="mb-2 text-xs text-neutral-400">
        See{' '}
        <a href="/docs/daemon" className="text-blue-400 hover:text-blue-300">
          Daemon Setup
        </a>{' '}
        for full installation instructions.
      </p>

      <H3>2. Get your host token</H3>
      <p className="mb-2 text-xs text-neutral-400">
        From the web dashboard, navigate to Settings and generate a host registration token:
      </p>
      <TerminalBlock
        lines={[
          { output: 'Host token: host_abc123def456...' },
          { output: 'Copy this token to your daemon config.' },
        ]}
      />

      <H3>3. Configure the daemon</H3>
      <CodeBlock
        code={`# Add to daemon.yaml
control_plane:
  address: "cp.fluid.sh:9090"
  token: "host_abc123def456..."`}
        lang="yaml"
        filename="daemon.yaml"
      />

      <H3>4. Restart and verify</H3>
      <TerminalBlock lines={[{ command: 'sudo systemctl restart fluid-daemon' }]} />
      <p className="mb-3 text-xs text-neutral-400">
        Launch the TUI to verify the daemon and control plane connection:
      </p>
      <TerminalBlock
        lines={[
          { command: 'fluid' },
          { output: 'Connected to daemon at localhost:9091' },
          { output: 'Control plane: connected' },
        ]}
      />

      <Callout type="tip">
        Your sandboxes will now appear in the web dashboard. You can manage them from either the CLI
        or the UI.
      </Callout>

      <PrevNext />
    </div>
  )
}
