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
              command: 'go install github.com/aspectrr/fluid.sh/fluid-cli/cmd/fluid-cli@latest',
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
          first run, onboarding walks you through host setup, SSH CA generation, and LLM API key
          configuration.
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
          Use <code className="text-green-400">/settings</code> to configure hosts, LLM provider,
          and sandbox defaults at any time.
        </Callout>
      </>
    ),
  },
  {
    title: 'Set up the daemon',
    content: (
      <>
        <p>
          The daemon manages sandboxes on the sandbox host. Choose a free host in your
          infrastructure that can connect to the VMs that you want to work on over SSH. Install and
          configure the daemon on each sandbox host - the CLI connects to it over gRPC.
        </p>
        <DaemonSubsteps />
      </>
    ),
  },
  {
    title: 'Create a sandbox',
    content: (
      <>
        <p>
          In the TUI, type a natural language request. The agent will call the appropriate tools:
        </p>
        <TerminalBlock
          lines={[
            { command: 'create a sandbox from ubuntu-base' },
            { output: '  ... create_sandbox source_vm=ubuntu-base' },
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
          Commands execute over SSH using ephemeral CA-signed certificates. No passwords needed.
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
