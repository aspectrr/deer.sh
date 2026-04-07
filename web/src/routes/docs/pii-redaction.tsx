import { createFileRoute } from '@tanstack/react-router'
import { H2, H3 } from '~/components/docs/heading-anchor'
import { Callout } from '~/components/docs/callout'
import { CodeBlock } from '~/components/docs/code-block'
import { PrevNext } from '~/components/docs/prev-next'

export const Route = createFileRoute('/docs/pii-redaction')({
  component: PIIRedactionPage,
})

function PIIRedactionPage() {
  return (
    <div className="mx-auto max-w-2xl px-6 py-8">
      <h1 className="mb-1 text-lg font-medium text-white">Sensitive Data Redaction</h1>
      <p className="text-muted-foreground mb-8 text-xs">
        How deer prevents credentials from reaching the AI agent.
      </p>

      <H2>The Problem</H2>
      <p className="mb-3 text-xs text-neutral-400">
        AI agents run commands and read files on VMs. Output goes to the LLM. Private keys, K8s
        secrets, and credentials must never reach the model.
      </p>

      <H2>Two-Layer Architecture</H2>

      <H3>Layer 1 - Inline Tool Redaction</H3>
      <p className="mb-3 text-xs text-neutral-400">
        Applied at tool execution time. When <code>run_command</code>, <code>read_file</code>,{' '}
        <code>run_source_command</code>, or <code>read_source_file</code> return results, sensitive
        content is scanned and replaced before it enters the conversation.
      </p>

      <H3>Layer 2 - Pattern-Based Message Redaction</H3>
      <p className="mb-3 text-xs text-neutral-400">
        Applied to all messages before LLM API calls. Uses deterministic tokens so the same value
        always maps to the same placeholder within a session.
      </p>

      <Callout type="info">
        The live output stream in the TUI shows what the LLM sees - content is redacted before it
        reaches the model.
      </Callout>

      <H2>What Gets Detected</H2>
      <ul className="mb-3 ml-4 list-disc space-y-1 text-xs text-neutral-400">
        <li>PEM private keys (RSA, EC, ED25519, OPENSSH)</li>
        <li>
          Base64-encoded PEM keys (e.g., <code>cat key.pem | base64</code>, K8s secret values)
        </li>
        <li>
          Kubernetes secret fields (<code>tls.key</code>, <code>ssh-privatekey</code>,{' '}
          <code>private_key</code>, <code>secret_key</code>, etc.)
        </li>
        <li>
          API keys and tokens (<code>sk-...</code>, <code>key-...</code>, Bearer tokens)
        </li>
        <li>
          AWS access keys (<code>AKIA...</code>)
        </li>
        <li>
          Connection strings (<code>postgres://</code>, <code>mysql://</code>,{' '}
          <code>mongodb://</code>, <code>redis://</code>)
        </li>
        <li>IPv4 and IPv6 addresses</li>
        <li>Configured host names, addresses, key paths</li>
      </ul>

      <H2>Where Redaction Applies</H2>
      <div className="border-border mb-6 overflow-x-auto border bg-neutral-900 p-4 font-mono text-xs text-neutral-400">
        <table className="w-full">
          <thead>
            <tr className="border-b border-neutral-700 text-left text-neutral-300">
              <th className="pr-4 pb-2">Tool</th>
              <th className="pb-2">Coverage</th>
            </tr>
          </thead>
          <tbody>
            <tr className="border-b border-neutral-800">
              <td className="py-2 pr-4">
                <code>run_command</code> / <code>run_source_command</code>
              </td>
              <td className="py-2">Both layers</td>
            </tr>
            <tr className="border-b border-neutral-800">
              <td className="py-2 pr-4">
                <code>read_file</code> / <code>read_source_file</code>
              </td>
              <td className="py-2">Both layers</td>
            </tr>
            <tr>
              <td className="py-2 pr-4">Other tool results</td>
              <td className="py-2">Layer 2 only</td>
            </tr>
          </tbody>
        </table>
      </div>

      <H2>How It Works</H2>
      <p className="mb-3 text-xs text-neutral-400">
        The agent sees <code>[REDACTED: ...]</code> placeholders. Original values are never stored
        in conversation history. The redactor uses deterministic tokens so the same value always
        maps to the same placeholder within a session.
      </p>
      <CodeBlock
        code={`# What the agent sees:
Server [REDACTED_HOST_1] ([REDACTED_IP_1]) uses key [REDACTED_PATH_1]
DB: [REDACTED_SECRET_1]`}
        lang="bash"
      />

      <H2>Custom Patterns</H2>
      <CodeBlock
        code={`redact:
  enabled: true
  custom_patterns:
    - '\\bTOKEN_[A-Z0-9]{10,}\\b'
  allowlist:
    - '192.168.1.1'`}
        lang="yaml"
      />

      <H2>Limitations</H2>
      <ul className="mb-3 ml-4 list-disc space-y-1 text-xs text-neutral-400">
        <li>Hex-encoded keys and non-standard key formats are not detected</li>
        <li>
          Keys embedded inside nested encodings (e.g., base64 of JSON of base64) are not detected
        </li>
        <li>
          Base64-PEM detection requires the PEM block to be at the start of the encoded content
        </li>
      </ul>

      <PrevNext />
    </div>
  )
}
