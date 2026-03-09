import { useState, type ReactNode } from 'react'
import { ChevronDown, ChevronRight } from 'lucide-react'
import type { Step } from '~/components/docs/step-tracker'
import { TerminalBlock } from '~/components/docs/terminal-block'
import { CodeBlock } from '~/components/docs/code-block'
import { Callout } from '~/components/docs/callout'
import { DaemonSubsteps } from '~/components/docs/daemon-setup-steps'

export const quickstartSteps: Step[] = [
  {
    title: 'Install the Fluid CLI',
    content: (
      <>
        <p>Install the CLI using the install script:</p>
        <TerminalBlock lines={[{ command: 'curl -fsSL https://fluid.sh/install.sh | bash' }]} />
        <p>Or install from source with Go:</p>
        <TerminalBlock
          lines={[
            {
              command: 'go install github.com/aspectrr/fluid.sh/fluid-cli/cmd/fluid@latest',
            },
          ]}
        />
        <p>Verify the installation:</p>
        <TerminalBlock
          lines={[{ command: 'fluid --version' }, { output: 'fluid version 0.1.0' }]}
        />
      </>
    ),
  },
  {
    title: 'Launch the TUI',
    content: (
      <>
        <p>
          Run <code className="text-blue-400">fluid</code> to start the interactive agent TUI. On
          first run, onboarding walks you through source VM connection and LLM API key
          configuration.
        </p>
        <TerminalBlock
          lines={[
            { command: 'fluid' },
            { output: 'fluid.sh v0.1.0' },
            { output: '' },
            { output: 'Type your message... (type /settings to configure)' },
          ]}
        />
        <Callout type="tip">
          Use <code className="text-green-400">/settings</code> to configure hosts, LLM provider,
          and sandbox defaults at any time.
        </Callout>
      </>
    ),
  },
  {
    title: 'Prepare your source VMs',
    content: (
      <>
        <p>
          Set up read-only access on the VMs you want Fluid to inspect. This creates a restricted
          user that Fluid uses to safely read logs, check configs, and run diagnostic commands -
          without write access.
        </p>
        <TerminalBlock
          lines={[
            { command: 'fluid source prepare my-server' },
            { output: '  Connecting to my-server...' },
            { output: '  Creating read-only user...' },
            { output: '  Installing SSH key...' },
            { output: '  Verifying access...' },
            { output: '' },
            { output: 'Source VM my-server prepared successfully.' },
          ]}
        />
        <Callout type="info">
          During onboarding, you'll be asked to prepare the source VMs of your choosing. You can
          always prepare additional sources later in the TUI with{' '}
          <code className="text-green-400">/prepare</code>.
        </Callout>
        <Callout type="tip">
          You can prepare multiple source VMs. Run{' '}
          <code className="text-green-400">fluid source prepare</code> for each host you want Fluid
          to access. See{' '}
          <a href="/docs/source-prepare" className="text-blue-400 hover:text-blue-300">
            source prepare docs
          </a>{' '}
          for details on what happens during preparation.
        </Callout>
      </>
    ),
  },
  {
    title: 'Debug your source VMs',
    content: (
      <>
        <p>
          Ask Fluid to inspect your infrastructure. It uses read-only access to check logs, configs,
          and system state on your source VMs:
        </p>
        <TerminalBlock
          lines={[
            { command: 'check the nginx logs on my-server' },
            {
              output:
                '  ... run_source_command source_vm=my-server, command=journalctl -u nginx --no-pager -n 50',
            },
            { output: '  v run_source_command' },
            {
              output:
                '    -> {"exit_code":0,"stdout":"Mar 05 12:01:32 my-server nginx[1234]: 192.168.1.10 - - [05/Mar/2026:12:01:32 +0000] \\"GET /api/health HTTP/1.1\\" 200 ...\\nMar 05 12:01:35 my-server nginx[1234]: 192.168.1.15 - - [05/Mar/2026:12:01:35 +0000] \\"POST /api/login HTTP/1.1\\" 502 ..."}',
            },
            { output: '' },
            {
              output:
                'I can see 502 errors on POST /api/login starting around 12:01. The upstream backend may be down. Want me to check the backend service status?',
            },
          ]}
        />
        <p>
          This is fully read-only - Fluid can diagnose issues but won't make changes to your source
          VMs.
        </p>
      </>
    ),
  },
  {
    title: 'Connect MCP to your editor',
    content: (
      <>
        <p>
          Connect Claude Code, Codex, or Cursor to fluid via MCP for AI-driven sandbox management
          directly from your editor:
        </p>
        <CodeBlock
          code={`{
  "mcpServers": {
    "fluid": {
      "command": "fluid",
      "args": ["mcp"]
    }
  }
}`}
          lang="json"
          filename="Claude Code: ~/.claude/settings.json"
        />
        <CodeBlock
          code={`{
  "mcpServers": {
    "fluid": {
      "command": "fluid",
      "args": ["mcp"]
    }
  }
}`}
          lang="json"
          filename="Codex: ~/.codex/config.json"
        />
        <CodeBlock
          code={`{
  "mcpServers": {
    "fluid": {
      "command": "fluid",
      "args": ["mcp"]
    }
  }
}`}
          lang="json"
          filename="Cursor: .cursor/mcp.json"
        />
        <Callout type="tip">
          See{' '}
          <a href="/docs/mcp" className="text-blue-400 hover:text-blue-300">
            MCP Server
          </a>{' '}
          for the full tools reference.
        </Callout>
      </>
    ),
  },
]

// -- Daemon upgrade section for bottom of quickstart page --

interface CollapsibleSectionProps {
  title: string
  children: ReactNode
  defaultOpen?: boolean
}

function CollapsibleSection({ title, children, defaultOpen = false }: CollapsibleSectionProps) {
  const [expanded, setExpanded] = useState(defaultOpen)

  return (
    <div className="border-border overflow-hidden rounded-lg border">
      <button
        onClick={() => setExpanded(!expanded)}
        className="flex w-full items-center gap-2 px-4 py-3 text-left text-xs font-medium text-neutral-300 transition-colors hover:text-white"
      >
        {expanded ? (
          <ChevronDown className="h-3 w-3 text-neutral-500" />
        ) : (
          <ChevronRight className="h-3 w-3 text-neutral-500" />
        )}
        {title}
      </button>
      {expanded && <div className="docs-prose px-4 pb-4">{children}</div>}
    </div>
  )
}

export function DaemonUpgradeSection() {
  return (
    <div className="border-border mt-8 border-t pt-6">
      <h2 className="mb-2 text-sm font-medium text-white">Want Fluid to fix issues it finds?</h2>
      <p className="mb-4 text-xs text-neutral-400">
        The daemon enables sandboxes - isolated copy-on-write clones of your VMs where Fluid can
        make changes safely. Test fixes in a sandbox, then apply them to production when you're
        ready.
      </p>

      <div className="space-y-3">
        <CollapsibleSection title="Install the daemon on your sandbox host">
          <p>
            Choose a free host in your infrastructure that can connect to the VMs you want to work
            on over SSH. Install and configure the daemon - the CLI connects to it over gRPC.
          </p>
          <DaemonSubsteps />
        </CollapsibleSection>

        <CollapsibleSection title="Create and use sandboxes">
          <p>Once the daemon is running, ask Fluid to create sandboxes from your source VMs:</p>
          <TerminalBlock
            lines={[
              { command: 'create a sandbox from my-server' },
              { output: '  ... create_sandbox source_vm=my-server' },
              { output: '  v create_sandbox' },
              {
                output: '    -> {"sandbox_id":"SBX-a1b2c3d4","state":"RUNNING","ip":"10.0.0.2"}',
              },
              { output: '' },
              { output: 'Sandbox SBX-a1b2c3d4 created and running at 10.0.0.2' },
            ]}
          />
          <p>
            The sandbox is a full copy-on-write clone with its own network interface. Changes stay
            isolated from the source VM.
          </p>
          <p className="mt-3">Run commands in the sandbox:</p>
          <TerminalBlock
            lines={[
              { command: 'fix the nginx config in the sandbox' },
              {
                output: '  ... run_command sandbox_id=SBX-a1b2c3d4, command=nginx -t',
              },
              { output: '  v run_command' },
              {
                output:
                  '    -> {"exit_code":0,"stdout":"nginx: configuration file /etc/nginx/nginx.conf test is successful"}',
              },
            ]}
          />
          <p>
            Commands execute over SSH using ephemeral CA-signed certificates. No passwords needed.
          </p>
        </CollapsibleSection>
      </div>
    </div>
  )
}
