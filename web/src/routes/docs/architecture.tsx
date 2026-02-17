import { createFileRoute } from '@tanstack/react-router'
import { H2, H3 } from '~/components/docs/heading-anchor'
import { ArchitectureDiagram } from '~/components/docs/architecture-diagram'
import { Callout } from '~/components/docs/callout'
import { PrevNext } from '~/components/docs/prev-next'

export const Route = createFileRoute('/docs/architecture')({
  component: ArchitecturePage,
})

function ArchitecturePage() {
  return (
    <div className="mx-auto max-w-2xl px-6 py-8">
      <h1 className="mb-1 text-lg font-medium text-white">Architecture</h1>
      <p className="text-muted-foreground mb-8 text-xs">
        How the three tiers of fluid.sh fit together.
      </p>

      <ArchitectureDiagram />

      <H2>Three Ways to Use Fluid</H2>
      <p className="mb-4 text-xs text-neutral-400">
        fluid.sh is designed to scale with your needs. Start with the CLI on a single machine, add
        the daemon for persistence, then connect to the control plane for multi-host.
      </p>

      <H3>Tier 1: Single Host (CLI + Daemon)</H3>
      <p className="mb-3 text-xs text-neutral-400">
        The simplest setup. One daemon runs per sandbox host, managing VMs locally via libvirt. The
        daemon can also reach remote libvirt hosts over SSH for source VM access.
      </p>
      <ul className="mb-4 ml-4 list-disc space-y-1 text-xs text-neutral-400">
        <li>CLI connects to daemon over gRPC (port 9091)</li>
        <li>Daemon manages libvirt/QEMU on the local host</li>
        <li>Can connect to remote libvirt hosts over SSH</li>
        <li>SQLite for local state tracking</li>
        <li>No control plane required</li>
      </ul>

      <H3>Tier 2: CLI + Daemon (Multi-Host SSH)</H3>
      <p className="mb-3 text-xs text-neutral-400">
        One daemon per sandbox host, running as a background service. The CLI connects over gRPC on
        port 9091. Each daemon can SSH into multiple libvirt hosts for source VM discovery and
        read-only access. Multiple daemons are typically needed for heavily NATed networks or
        separate data centers.
      </p>
      <ul className="mb-4 ml-4 list-disc space-y-1 text-xs text-neutral-400">
        <li>gRPC server for CLI communication</li>
        <li>Background sandbox lifecycle management</li>
        <li>SSH connectivity to multiple libvirt hosts</li>
        <li>Image extraction and caching</li>
        <li>TTL-based cleanup (janitor)</li>
        <li>Network bridge and TAP management</li>
      </ul>

      <H3>Tier 3: Full Hosted (Control Plane)</H3>
      <p className="mb-3 text-xs text-neutral-400">
        The control plane is a central server that orchestrates multiple daemon instances across
        hosts. It provides a REST API, web dashboard, multi-host placement, and uses PostgreSQL for
        persistent state.
      </p>
      <ul className="mb-4 ml-4 list-disc space-y-1 text-xs text-neutral-400">
        <li>Multi-host sandbox placement</li>
        <li>REST API for external integrations</li>
        <li>Web dashboard for monitoring and approval</li>
        <li>PostgreSQL-backed state</li>
        <li>Source VM registry across hosts</li>
        <li>Bidirectional gRPC stream with daemons</li>
      </ul>

      <H2>Decision Matrix</H2>
      <div className="border-border mb-6 overflow-x-auto border">
        <table className="w-full text-xs">
          <thead>
            <tr className="border-border border-b bg-neutral-900/50">
              <th className="px-3 py-2 text-left font-medium text-neutral-300">Use Case</th>
              <th className="px-3 py-2 text-left font-medium text-neutral-300">Tier</th>
            </tr>
          </thead>
          <tbody className="text-neutral-400">
            <tr className="border-border border-b">
              <td className="px-3 py-1.5">Local dev, quick experiments</td>
              <td className="px-3 py-1.5 text-blue-400">Single Host</td>
            </tr>
            <tr className="border-border border-b">
              <td className="px-3 py-1.5">Background sandbox management</td>
              <td className="px-3 py-1.5 text-blue-400">CLI + Daemon</td>
            </tr>
            <tr className="border-border border-b">
              <td className="px-3 py-1.5">AI agent integration (MCP)</td>
              <td className="px-3 py-1.5 text-blue-400">CLI + Daemon</td>
            </tr>
            <tr className="border-border border-b">
              <td className="px-3 py-1.5">Multi-host fleet</td>
              <td className="px-3 py-1.5 text-blue-400">Full Hosted</td>
            </tr>
            <tr className="border-border border-b">
              <td className="px-3 py-1.5">Web dashboard + approval flow</td>
              <td className="px-3 py-1.5 text-blue-400">Full Hosted</td>
            </tr>
            <tr className="border-border border-b last:border-0">
              <td className="px-3 py-1.5">Team collaboration</td>
              <td className="px-3 py-1.5 text-blue-400">Full Hosted</td>
            </tr>
          </tbody>
        </table>
      </div>

      <Callout type="tip">
        Start with Tier 1 (Single Host). You can connect to the control plane later without changing
        your workflow. The TUI and MCP tools work the same way regardless of tier.
      </Callout>

      <PrevNext />
    </div>
  )
}
