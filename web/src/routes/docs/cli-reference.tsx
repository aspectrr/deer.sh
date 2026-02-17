import { createFileRoute } from '@tanstack/react-router'
import { H2 } from '~/components/docs/heading-anchor'
import { PrevNext } from '~/components/docs/prev-next'

export const Route = createFileRoute('/docs/cli-reference')({
  component: TuiMcpReferencePage,
})

const cliEntryPoints = [
  { name: 'fluid', description: 'Launch the interactive TUI agent (default)' },
  { name: 'fluid mcp', description: 'Start MCP server on stdio for editor integration' },
  { name: 'fluid update', description: 'Self-update to the latest release' },
  { name: 'fluid --version', description: 'Print version info' },
  { name: 'fluid --config <path>', description: 'Use a custom config file' },
]

const slashCommands = [
  { name: '/vms', description: 'List available VMs for cloning' },
  { name: '/sandboxes', description: 'List active sandboxes' },
  { name: '/hosts', description: 'List configured remote hosts' },
  { name: '/playbooks', description: 'List generated Ansible playbooks' },
  { name: '/compact', description: 'Summarize and compact conversation history' },
  { name: '/context', description: 'Show current context token usage' },
  { name: '/settings', description: 'Open configuration settings' },
  { name: '/clear', description: 'Clear conversation history' },
  { name: '/help', description: 'Show available commands' },
]

const shortcuts = [
  { key: 'Enter', action: 'Send message' },
  { key: 'Shift+Tab', action: 'Toggle edit / read-only mode' },
  { key: 'PgUp / PgDn', action: 'Scroll conversation' },
  { key: 'Ctrl+R', action: 'Reset conversation' },
  { key: 'Ctrl+C', action: 'Quit' },
]

const mcpTools = [
  {
    name: 'list_sandboxes',
    params: '(none)',
    description: 'List all sandboxes with state and IP addresses',
  },
  {
    name: 'create_sandbox',
    params: 'source_vm, cpu?, memory_mb?',
    description: 'Create a sandbox by cloning a source VM',
  },
  {
    name: 'destroy_sandbox',
    params: 'sandbox_id',
    description: 'Destroy a sandbox and remove its storage',
  },
  {
    name: 'run_command',
    params: 'sandbox_id, command, timeout_seconds?',
    description: 'Execute a shell command inside a sandbox via SSH',
  },
  { name: 'start_sandbox', params: 'sandbox_id', description: 'Start a stopped sandbox' },
  { name: 'stop_sandbox', params: 'sandbox_id', description: 'Stop a running sandbox' },
  { name: 'get_sandbox', params: 'sandbox_id', description: 'Get detailed info about a sandbox' },
  { name: 'list_vms', params: '(none)', description: 'List available source VMs for cloning' },
  {
    name: 'create_snapshot',
    params: 'sandbox_id, name?',
    description: 'Snapshot current sandbox state',
  },
  {
    name: 'create_playbook',
    params: 'name, hosts?, become?',
    description: 'Create an Ansible playbook',
  },
  {
    name: 'add_playbook_task',
    params: 'playbook_id, name, module, params?',
    description: 'Add a task to an Ansible playbook',
  },
  {
    name: 'edit_file',
    params: 'sandbox_id, path, new_str, old_str?, replace_all?',
    description: 'Edit or create a file in a sandbox',
  },
  { name: 'read_file', params: 'sandbox_id, path', description: 'Read a file from a sandbox' },
  { name: 'list_playbooks', params: '(none)', description: 'List all Ansible playbooks' },
  { name: 'get_playbook', params: 'playbook_id', description: 'Get playbook definition and YAML' },
  {
    name: 'run_source_command',
    params: 'source_vm, command, timeout_seconds?',
    description: 'Run a read-only command on a source VM',
  },
  {
    name: 'read_source_file',
    params: 'source_vm, path',
    description: 'Read a file from a source VM (read-only)',
  },
]

function TuiMcpReferencePage() {
  return (
    <div className="mx-auto max-w-2xl px-6 py-8">
      <h1 className="mb-1 text-lg font-medium text-white">TUI & MCP Reference</h1>
      <p className="text-muted-foreground mb-8 text-xs">
        Complete reference for the interactive TUI, slash commands, keyboard shortcuts, and MCP
        tools.
      </p>

      <H2>CLI Entry Points</H2>
      <div className="border-border mb-6 border">
        <table className="w-full text-xs">
          <thead>
            <tr className="border-border border-b bg-neutral-900/50">
              <th className="px-3 py-2 text-left font-medium text-neutral-300">Command</th>
              <th className="px-3 py-2 text-left font-medium text-neutral-300">Description</th>
            </tr>
          </thead>
          <tbody className="text-neutral-400">
            {cliEntryPoints.map((cmd) => (
              <tr key={cmd.name} className="border-border border-b last:border-0">
                <td className="px-3 py-1.5 font-mono whitespace-nowrap text-blue-400">
                  {cmd.name}
                </td>
                <td className="px-3 py-1.5">{cmd.description}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <H2>TUI Slash Commands</H2>
      <p className="mb-3 text-xs text-neutral-400">
        Type these in the TUI input to trigger built-in actions:
      </p>
      <div className="border-border mb-6 border">
        <table className="w-full text-xs">
          <thead>
            <tr className="border-border border-b bg-neutral-900/50">
              <th className="px-3 py-2 text-left font-medium text-neutral-300">Command</th>
              <th className="px-3 py-2 text-left font-medium text-neutral-300">Description</th>
            </tr>
          </thead>
          <tbody className="text-neutral-400">
            {slashCommands.map((cmd) => (
              <tr key={cmd.name} className="border-border border-b last:border-0">
                <td className="px-3 py-1.5 font-mono text-blue-400">{cmd.name}</td>
                <td className="px-3 py-1.5">{cmd.description}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <H2>Keyboard Shortcuts</H2>
      <div className="border-border mb-6 border">
        <table className="w-full text-xs">
          <thead>
            <tr className="border-border border-b bg-neutral-900/50">
              <th className="px-3 py-2 text-left font-medium text-neutral-300">Key</th>
              <th className="px-3 py-2 text-left font-medium text-neutral-300">Action</th>
            </tr>
          </thead>
          <tbody className="text-neutral-400">
            {shortcuts.map((s) => (
              <tr key={s.key} className="border-border border-b last:border-0">
                <td className="px-3 py-1.5 font-mono text-neutral-300">{s.key}</td>
                <td className="px-3 py-1.5">{s.action}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <H2>MCP Tools</H2>
      <p className="mb-3 text-xs text-neutral-400">
        These tools are exposed via <code className="text-blue-400">fluid mcp</code> for use with
        Claude Code, Cursor, Windsurf, and other MCP-compatible clients.
      </p>
      <div className="border-border mb-6 overflow-x-auto border">
        <table className="w-full text-xs">
          <thead>
            <tr className="border-border border-b bg-neutral-900/50">
              <th className="px-3 py-2 text-left font-medium text-neutral-300">Tool</th>
              <th className="px-3 py-2 text-left font-medium text-neutral-300">Parameters</th>
              <th className="px-3 py-2 text-left font-medium text-neutral-300">Description</th>
            </tr>
          </thead>
          <tbody className="text-neutral-400">
            {mcpTools.map((tool) => (
              <tr key={tool.name} className="border-border border-b last:border-0">
                <td className="px-3 py-1.5 font-mono whitespace-nowrap text-blue-400">
                  {tool.name}
                </td>
                <td className="px-3 py-1.5 font-mono whitespace-nowrap text-neutral-500">
                  {tool.params}
                </td>
                <td className="px-3 py-1.5">{tool.description}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <PrevNext />
    </div>
  )
}
