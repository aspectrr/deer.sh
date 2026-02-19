import { createFileRoute } from '@tanstack/react-router'
import { StepTracker } from '~/components/docs/step-tracker'
import { quickstartSteps } from '~/components/docs/quickstart-steps'
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
        steps={quickstartSteps}
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
