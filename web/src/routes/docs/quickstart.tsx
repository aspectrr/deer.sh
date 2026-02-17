import { createFileRoute } from '@tanstack/react-router'
import { StepTracker } from '~/components/docs/step-tracker'
import { TerminalBlock } from '~/components/docs/terminal-block'
import { CodeBlock } from '~/components/docs/code-block'
import { Callout } from '~/components/docs/callout'
import { PrevNext } from '~/components/docs/prev-next'

export const Route = createFileRoute('/docs/quickstart')({
  validateSearch: (search: Record<string, unknown>): { code?: string } => ({
    code: (search.code as string) || undefined,
  }),
  component: QuickstartPage,
})

function QuickstartPage() {
  const { code } = Route.useSearch()

  return (
    <div className="mx-auto max-w-2xl px-6 py-8">
      <h1 className="mb-1 text-lg font-medium text-white">Quickstart</h1>
      <p className="text-muted-foreground mb-8 text-xs">
        Get up and running with fluid.sh in under 5 minutes.
      </p>

      <StepTracker
        storageKey="quickstart"
        progressEndpoint="/v1/docs-progress"
        sessionCode={code}
        steps={[
          {
            title: 'Install the Fluid CLI',
            content: (
              <>
                <p>Install the CLI using the install script:</p>
                <TerminalBlock
                  lines={[{ command: 'curl -fsSL https://fluid.sh/install.sh | bash' }]}
                />
                <p>Or install from source with Go:</p>
                <TerminalBlock
                  lines={[
                    {
                      command:
                        'go install github.com/aspectrr/fluid.sh/fluid-cli/cmd/fluid-cli@latest',
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
            title: 'Set up the daemon',
            content: (
              <>
                <p>
                  The daemon manages sandboxes on the host machine. The CLI connects to it over
                  gRPC.
                </p>
                <TerminalBlock
                  lines={[
                    { command: 'curl -fsSL https://fluid.sh/install.sh | bash -s -- --daemon' },
                  ]}
                />
                <p>Configure and start:</p>
                <TerminalBlock
                  lines={[
                    { command: 'sudo systemctl enable --now fluid-daemon' },
                    { output: 'Created symlink ...' },
                  ]}
                />
                <Callout type="info">
                  See{' '}
                  <a href="/docs/daemon" className="text-blue-400 hover:text-blue-300">
                    Daemon Setup
                  </a>{' '}
                  for full configuration options.
                </Callout>
              </>
            ),
          },
          {
            title: 'Launch the TUI',
            content: (
              <>
                <p>
                  Run <code className="text-blue-400">fluid</code> to start the interactive agent
                  TUI. On first run, onboarding walks you through host setup, SSH CA generation, and
                  LLM API key configuration.
                </p>
                <TerminalBlock
                  lines={[
                    { command: 'fluid' },
                    { output: 'fluid.sh v0.1.0' },
                    { output: 'Connected to daemon at localhost:9091' },
                    { output: '' },
                    { output: 'Type your message... (type /settings to configure)' },
                  ]}
                />
                <Callout type="tip">
                  Use <code className="text-green-400">/settings</code> to configure hosts, LLM
                  provider, and sandbox defaults at any time.
                </Callout>
              </>
            ),
          },
          {
            title: 'Create a sandbox',
            content: (
              <>
                <p>
                  In the TUI, type a natural language request. The agent will call the appropriate
                  tools:
                </p>
                <TerminalBlock
                  lines={[
                    { command: 'create a sandbox from ubuntu-base' },
                    { output: '  ... create_sandbox source_vm=ubuntu-base' },
                    { output: '  v create_sandbox' },
                    {
                      output:
                        '    -> {"sandbox_id":"SBX-a1b2c3d4","state":"RUNNING","ip":"10.0.0.2"}',
                    },
                    { output: '' },
                    { output: 'Sandbox SBX-a1b2c3d4 created and running at 10.0.0.2' },
                  ]}
                />
                <p>
                  The sandbox is a full copy-on-write clone with its own network interface. Changes
                  stay isolated from the source VM.
                </p>
              </>
            ),
          },
          {
            title: 'Run commands in the sandbox',
            content: (
              <>
                <p>Ask the agent to run commands:</p>
                <TerminalBlock
                  lines={[
                    { command: 'run whoami in the sandbox' },
                    { output: '  ... run_command sandbox_id=SBX-a1b2c3d4, command=whoami' },
                    { output: '  v run_command' },
                    { output: '    -> {"exit_code":0,"stdout":"root\\n"}' },
                    { output: '' },
                    { output: 'root' },
                  ]}
                />
                <p>
                  Commands execute over SSH using ephemeral CA-signed certificates. No passwords
                  needed.
                </p>
              </>
            ),
          },
          {
            title: 'Connect MCP to your editor',
            content: (
              <>
                <p>
                  Connect Claude Code, Cursor, or Windsurf to fluid via MCP for AI-driven sandbox
                  management directly from your editor:
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
                  filename="Cursor: .cursor/mcp.json"
                />
                <Callout type="tip">
                  See{' '}
                  <a href="/docs/mcp" className="text-blue-400 hover:text-blue-300">
                    MCP Server
                  </a>{' '}
                  for the full tools reference and Windsurf configuration.
                </Callout>
              </>
            ),
          },
        ]}
      />

      <div className="border-border mt-8 border-t pt-6">
        <h2 className="mb-2 text-sm font-medium text-white">What's next?</h2>
        <ul className="ml-4 list-disc space-y-1 text-xs text-neutral-400">
          <li>
            Set up the{' '}
            <a href="/docs/daemon" className="text-blue-400 hover:text-blue-300">
              daemon
            </a>{' '}
            for persistent sandbox management
          </li>
          <li>
            Learn about{' '}
            <a href="/docs/sandboxes" className="text-blue-400 hover:text-blue-300">
              sandbox internals
            </a>
          </li>
          <li>
            Connect your{' '}
            <a href="/docs/mcp" className="text-blue-400 hover:text-blue-300">
              AI tools via MCP
            </a>
          </li>
          <li>
            See the full{' '}
            <a href="/docs/cli-reference" className="text-blue-400 hover:text-blue-300">
              TUI & MCP reference
            </a>
          </li>
        </ul>
      </div>

      <PrevNext />
    </div>
  )
}
