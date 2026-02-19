import { createFileRoute } from '@tanstack/react-router'
import { H2, H3 } from '~/components/docs/heading-anchor'
import { Callout } from '~/components/docs/callout'
import { CodeBlock } from '~/components/docs/code-block'
import { PrevNext } from '~/components/docs/prev-next'

export const Route = createFileRoute('/docs/sandboxes')({
  component: SandboxesPage,
})

function SandboxesPage() {
  return (
    <div className="mx-auto max-w-2xl px-6 py-8">
      <h1 className="mb-1 text-lg font-medium text-white">Sandboxes</h1>
      <p className="text-muted-foreground mb-8 text-xs">
        How isolated VM sandboxes work under the hood.
      </p>

      <H2>What is a Sandbox?</H2>
      <p className="mb-3 text-xs text-neutral-400">
        A sandbox is an isolated, ephemeral clone of a source VM. It gets its own filesystem
        (copy-on-write), network interface (TAP device), and SSH access (CA-signed certificate).
        Changes inside the sandbox never affect the source VM.
      </p>

      <H2>MicroVM Path (QEMU)</H2>

      <H3>QCOW2 Overlay</H3>
      <p className="mb-3 text-xs text-neutral-400">
        Instead of copying the full disk image, fluid creates a QCOW2 overlay backed by the source
        VM's disk. This means sandbox creation is near-instant regardless of disk size - only
        changed blocks are written to the overlay.
      </p>
      <CodeBlock
        code={`# The overlay chain:
source.qcow2        # Read-only base image (shared)
  └── sandbox.qcow2 # Copy-on-write overlay (per-sandbox)`}
        lang="bash"
      />

      <H3>TAP Devices and Bridges</H3>
      <p className="mb-3 text-xs text-neutral-400">
        Each sandbox gets a dedicated TAP network device attached to a bridge interface. The bridge
        handles routing between sandboxes and to the host network.
      </p>
      <CodeBlock
        code={`# Network topology:
host-eth0
  └── fluid0 (bridge, 10.0.0.1/24)
        ├── tap-sbx-001 (10.0.0.2)
        ├── tap-sbx-002 (10.0.0.3)
        └── tap-sbx-003 (10.0.0.4)`}
        lang="bash"
      />

      <H3>IP Discovery</H3>
      <p className="mb-3 text-xs text-neutral-400">
        After boot, fluid discovers the sandbox IP by monitoring ARP tables and DHCP leases on the
        bridge interface. The CLI polls until the IP is available, typically within 2-5 seconds.
      </p>

      <H2>LXC Path</H2>
      <p className="mb-3 text-xs text-neutral-400">
        For lighter-weight sandboxes, fluid supports LXC containers as an alternative backend. LXC
        sandboxes boot faster and use less memory, but provide weaker isolation than full VMs.
      </p>

      <Callout type="warning">
        LXC sandboxes share the host kernel. For untrusted workloads, use the microVM path (QEMU).
      </Callout>

      <H2>SSH Access</H2>

      <H3>Ephemeral CA-Signed Certificates</H3>
      <p className="mb-3 text-xs text-neutral-400">
        fluid runs an internal SSH Certificate Authority. When a sandbox is created, fluid signs a
        short-lived SSH certificate and injects the CA public key into the sandbox. This means:
      </p>
      <ul className="mb-3 ml-4 list-disc space-y-1 text-xs text-neutral-400">
        <li>No passwords needed</li>
        <li>No key distribution required</li>
        <li>Certificates auto-expire with the sandbox TTL</li>
        <li>Each sandbox trusts only its parent CA</li>
      </ul>

      <H2>Lifecycle</H2>
      <div className="border-border mb-6 border bg-neutral-900 p-4 font-mono text-xs text-neutral-400">
        <div className="flex flex-wrap items-center gap-2">
          <span className="border border-blue-400/30 bg-blue-400/10 px-2 py-0.5 text-blue-400">
            create
          </span>
          <span className="text-neutral-600">{'->'}</span>
          <span className="border border-green-400/30 bg-green-400/10 px-2 py-0.5 text-green-400">
            running
          </span>
          <span className="text-neutral-600">{'->'}</span>
          <span className="border border-amber-400/30 bg-amber-400/10 px-2 py-0.5 text-amber-400">
            stop/start
          </span>
          <span className="text-neutral-600">{'->'}</span>
          <span className="border border-red-400/30 bg-red-400/10 px-2 py-0.5 text-red-400">
            destroy
          </span>
        </div>
      </div>

      <H3>TTL Janitor</H3>
      <p className="mb-3 text-xs text-neutral-400">
        The janitor runs periodically and destroys sandboxes that have exceeded their configured
        TTL. Default TTL is 1 hour. This prevents resource leaks from forgotten sandboxes.
      </p>
      <p className="mb-3 text-xs text-neutral-400">
        In the TUI, ask: "create a sandbox from ubuntu-base with a 30 minute TTL". Or via MCP:
      </p>
      <CodeBlock
        code={`// MCP tool call
create_sandbox({ source_vm: "ubuntu-base" })`}
        lang="javascript"
      />

      <Callout type="info">
        Sandboxes are cheap. Create them freely, destroy them when done. The overlay filesystem
        means minimal disk overhead per sandbox.
      </Callout>

      <PrevNext />
    </div>
  )
}
