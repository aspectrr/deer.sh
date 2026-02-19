import { createFileRoute, Link } from '@tanstack/react-router'
import { StepTracker } from '~/components/docs/step-tracker'
import { TerminalBlock } from '~/components/docs/terminal-block'
import { CodeBlock } from '~/components/docs/code-block'
import { Callout } from '~/components/docs/callout'
import { PrevNext } from '~/components/docs/prev-next'
import { H2 } from '~/components/docs/heading-anchor'

export const Route = createFileRoute('/docs/mcp')({
  component: McpPage,
})

const claudeConfig = `{
  "mcpServers": {
    "fluid": {
      "command": "fluid",
      "args": ["mcp"],
      "env": {}
    }
  }
}`

const codexConfig = `{
  "mcpServers": {
    "fluid": {
      "command": "fluid",
      "args": ["mcp"],
      "env": {}
    }
  }
}`

const cursorConfig = `{
  "mcpServers": {
    "fluid": {
      "command": "fluid",
      "args": ["mcp"],
      "env": {}
    }
  }
}`

const mcpTools = [
  { name: 'list_sandboxes', description: 'List all sandboxes with state and IP addresses' },
  { name: 'create_sandbox', description: 'Create a sandbox by cloning a source VM' },
  { name: 'destroy_sandbox', description: 'Destroy a sandbox and remove its storage' },
  { name: 'run_command', description: 'Execute a shell command inside a sandbox via SSH' },
  { name: 'start_sandbox', description: 'Start a stopped sandbox' },
  { name: 'stop_sandbox', description: 'Stop a running sandbox' },
  { name: 'get_sandbox', description: 'Get detailed info about a sandbox' },
  { name: 'list_vms', description: 'List available source VMs for cloning' },
  { name: 'create_snapshot', description: 'Snapshot current sandbox state' },
  { name: 'create_playbook', description: 'Create an Ansible playbook' },
  { name: 'add_playbook_task', description: 'Add a task to an Ansible playbook' },
  { name: 'edit_file', description: 'Edit or create a file in a sandbox' },
  { name: 'read_file', description: 'Read a file from a sandbox' },
  { name: 'list_playbooks', description: 'List all Ansible playbooks' },
  { name: 'get_playbook', description: 'Get playbook definition and YAML' },
  { name: 'run_source_command', description: 'Run a read-only command on a source VM' },
  { name: 'read_source_file', description: 'Read a file from a source VM (read-only)' },
]

function McpPage() {
  return (
    <div className="mx-auto max-w-2xl px-6 py-8">
      <h1 className="mb-1 text-lg font-medium text-white">MCP Server</h1>
      <p className="text-muted-foreground mb-8 text-xs">
        Connect AI coding tools to fluid.sh sandboxes via the Model Context Protocol.
      </p>

      <Callout type="info">
        The MCP server lets Claude Code, Cursor, and other MCP-compatible tools create and manage
        sandboxes directly. Your AI agent can spin up isolated VMs, run commands, and clean up - all
        without leaving your editor.
      </Callout>

      <StepTracker
        storageKey="mcp"
        steps={[
          {
            title: 'Install the CLI',
            content: (
              <>
                <p>
                  Install the fluid CLI before configuring MCP. Follow the{' '}
                  <Link to="/docs/quickstart" className="text-blue-400 underline">
                    quickstart guide
                  </Link>{' '}
                  to get set up.
                </p>
              </>
            ),
          },
          {
            title: 'Start the MCP server',
            content: (
              <>
                <p>
                  The MCP server is built into the CLI. It communicates over stdin/stdout using the
                  MCP protocol:
                </p>
                <TerminalBlock lines={[{ command: 'fluid mcp' }]} />
                <Callout type="tip">
                  You do not need to run this manually. MCP clients (Claude Code, Cursor) will start
                  the server automatically when configured.
                </Callout>
              </>
            ),
          },
          {
            title: 'Configure',
            content: (
              <>
                <p>Add the fluid MCP server to your editor of choice:</p>
                <CodeBlock
                  code={claudeConfig}
                  lang="json"
                  filename="Claude Code: ~/.claude/settings.json"
                />
                <CodeBlock code={codexConfig} lang="json" filename="Codex: ~/.codex/config.json" />
                <CodeBlock code={cursorConfig} lang="json" filename="Cursor: .cursor/mcp.json" />
                <p>Restart your editor to pick up the change.</p>
              </>
            ),
          },
          {
            title: 'Verify tools are available',
            content: (
              <>
                <p>
                  Ask your AI tool to list available fluid tools. You should see the sandbox
                  management tools listed below.
                </p>
                <Callout type="tip">
                  Try asking: "Create a sandbox from ubuntu-base and run whoami in it"
                </Callout>
              </>
            ),
          },
        ]}
      />

      <H2>Available Tools</H2>
      <div className="border-border border">
        <table className="w-full text-xs">
          <thead>
            <tr className="border-border border-b bg-neutral-900/50">
              <th className="px-3 py-2 text-left font-medium text-neutral-300">Tool</th>
              <th className="px-3 py-2 text-left font-medium text-neutral-300">Description</th>
            </tr>
          </thead>
          <tbody>
            {mcpTools.map((tool) => (
              <tr key={tool.name} className="border-border border-b last:border-0">
                <td className="px-3 py-1.5 font-mono text-blue-400">{tool.name}</td>
                <td className="px-3 py-1.5 text-neutral-400">{tool.description}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <PrevNext />
    </div>
  )
}
