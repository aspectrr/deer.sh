import { createFileRoute } from '@tanstack/react-router'
import { ApiTester } from '~/components/docs/api-tester'
import { PrevNext } from '~/components/docs/prev-next'

export const Route = createFileRoute('/docs/api')({
  component: ApiPage,
})

function ApiPage() {
  return (
    <div className="mx-auto max-w-3xl px-6 py-8">
      <h1 className="mb-1 text-lg font-medium text-white">API Reference</h1>
      <p className="text-muted-foreground mb-4 text-xs">
        Interactive reference for the Fluid REST API. Expand any endpoint to view details and send
        live requests.
      </p>

      <ApiTester />

      <PrevNext />
    </div>
  )
}
