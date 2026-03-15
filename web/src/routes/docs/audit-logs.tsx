import { createFileRoute } from '@tanstack/react-router'
import { H2, H3 } from '~/components/docs/heading-anchor'
import { Callout } from '~/components/docs/callout'
import { CodeBlock } from '~/components/docs/code-block'
import { PrevNext } from '~/components/docs/prev-next'

export const Route = createFileRoute('/docs/audit-logs')({
  component: AuditLogsPage,
})

function AuditLogsPage() {
  return (
    <div className="mx-auto max-w-2xl px-6 py-8">
      <h1 className="mb-1 text-lg font-medium text-white">Audit Logs</h1>
      <p className="text-muted-foreground mb-8 text-xs">
        Hash-chained, tamper-evident audit trail for all CLI operations.
      </p>

      <H2>Overview</H2>
      <p className="mb-3 text-xs text-neutral-400">
        Every significant CLI operation is recorded in a hash-chained audit log. Each entry is
        SHA-256 linked to the previous one, making tampering detectable.
      </p>

      <H2>What Gets Logged</H2>
      <ul className="mb-3 ml-4 list-disc space-y-1 text-xs text-neutral-400">
        <li>
          <code>session_start</code> - Session initialization
        </li>
        <li>
          <code>session_end</code> - Session completion with stats
        </li>
        <li>
          <code>user_input</code> - User input (length only, not content - privacy preserving)
        </li>
        <li>
          <code>llm_request</code> - Outgoing LLM API calls (message count, token estimate, model)
        </li>
        <li>
          <code>llm_response</code> - Incoming LLM responses (token count, tool call count)
        </li>
        <li>
          <code>tool_call</code> - Tool execution (name, args, result, error, duration_ms)
        </li>
      </ul>

      <H2>Entry Format</H2>
      <CodeBlock
        code={`{
  "seq": 42,
  "ts": "2026-03-15T10:30:00Z",
  "type": "tool_call",
  "tool": "run_command",
  "args": {"sandbox_id": "sbx-abc", "command": "ls -la"},
  "result": "success",
  "duration_ms": 1200,
  "prev_hash": "a1b2c3...",
  "hash": "d4e5f6..."
}`}
        lang="json"
      />

      <H2>Hash Chain Integrity</H2>
      <p className="mb-3 text-xs text-neutral-400">
        Each entry's SHA-256 hash includes the previous entry's hash. If any entry is modified,
        deleted, or reordered, verification fails. This creates a tamper-evident chain from the
        first entry forward.
      </p>

      <H2>Configuration</H2>
      <CodeBlock
        code={`audit:
  enabled: true
  log_path: ~/.config/fluid/audit.jsonl
  max_size_mb: 50`}
        lang="yaml"
      />
      <p className="mb-3 text-xs text-neutral-400">
        Also configurable via <code>/settings</code> in the TUI.
      </p>

      <H2>Accessing Audit Logs</H2>

      <H3>CLI Commands</H3>
      <CodeBlock
        code={`# View recent entries (last 50)
fluid audit show

# Verify hash chain integrity
fluid audit verify`}
        lang="bash"
      />

      <H3>Direct Access</H3>
      <CodeBlock code={`cat ~/.config/fluid/audit.jsonl | jq .`} lang="bash" />

      <H2>Data Redaction</H2>
      <p className="mb-3 text-xs text-neutral-400">
        Tool arguments are automatically redacted in audit entries when redaction is enabled.
        Sensitive values are masked before being written to the log.
      </p>

      <H2>Enterprise</H2>
      <Callout type="info">
        Need exportable audit logs across your whole team? Reach out to{' '}
        <a className="text-blue-400 hover:text-blue-300" href="mailto:cpfeifer@madcactus.org">
          Collin
        </a>{' '}
        about enterprise plans.
      </Callout>

      <PrevNext />
    </div>
  )
}
