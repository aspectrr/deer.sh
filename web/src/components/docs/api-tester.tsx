import { useEffect, useState } from 'react'
import YAML from 'yaml'
import { ApiEndpointCard } from '~/components/docs/api-endpoint-card'
import { H2 } from '~/components/docs/heading-anchor'
import type { OpenAPIEndpoint, OpenAPIParam } from '~/lib/openapi'

interface ParsedSpec {
  endpoints: OpenAPIEndpoint[]
  tags: string[]
}

function parseSwaggerSpec(spec: Record<string, unknown>): ParsedSpec {
  const endpoints: OpenAPIEndpoint[] = []
  const tagSet = new Set<string>()

  const paths = (spec.paths || {}) as Record<string, Record<string, Record<string, unknown>>>
  for (const [path, methods] of Object.entries(paths)) {
    for (const [method, details] of Object.entries(methods)) {
      if (['get', 'post', 'put', 'patch', 'delete'].includes(method)) {
        const endpoint: OpenAPIEndpoint = {
          method: method.toUpperCase(),
          path,
          summary: details.summary as string | undefined,
          description: details.description as string | undefined,
          tags: details.tags as string[] | undefined,
          parameters: details.parameters as OpenAPIParam[] | undefined,
          requestBody: details.requestBody as OpenAPIEndpoint['requestBody'],
          responses: details.responses as OpenAPIEndpoint['responses'],
        }
        endpoints.push(endpoint)
        ;(details.tags as string[] | undefined)?.forEach((t) => tagSet.add(t))
      }
    }
  }

  return { endpoints, tags: Array.from(tagSet) }
}

export function ApiTester() {
  const [spec, setSpec] = useState<ParsedSpec | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    fetch('/v1/docs/openapi.yaml')
      .then((res) => {
        if (!res.ok) throw new Error(`Failed to fetch spec: ${res.status}`)
        return res.text()
      })
      .then((text) => {
        const parsed = YAML.parse(text) as Record<string, unknown>
        setSpec(parseSwaggerSpec(parsed))
        setLoading(false)
      })
      .catch((err) => {
        setError(err instanceof Error ? err.message : 'Unknown error')
        setLoading(false)
      })
  }, [])

  if (loading) {
    return <div className="text-muted-foreground p-4 text-xs">Loading API spec...</div>
  }

  if (error) {
    return (
      <div className="border border-amber-400/20 bg-amber-400/5 p-4">
        <p className="mb-1 text-xs text-amber-400">Could not load API spec</p>
        <p className="text-[10px] text-neutral-500">{error}</p>
        <p className="mt-1 text-[10px] text-neutral-500">
          Make sure the API server is running on port 8080.
        </p>
      </div>
    )
  }

  if (!spec || spec.endpoints.length === 0) {
    return (
      <div className="border border-amber-400/20 bg-amber-400/5 p-4">
        <p className="text-xs text-amber-400">No endpoints found in spec.</p>
      </div>
    )
  }

  // Group by tag
  const grouped = new Map<string, typeof spec.endpoints>()
  for (const endpoint of spec.endpoints) {
    const tag = endpoint.tags?.[0] || 'Other'
    if (!grouped.has(tag)) grouped.set(tag, [])
    grouped.get(tag)!.push(endpoint)
  }

  return (
    <div className="space-y-6">
      {Array.from(grouped.entries()).map(([tag, endpoints]) => (
        <div key={tag}>
          <H2>{tag}</H2>
          <div className="border-border border">
            {endpoints.map((endpoint, i) => (
              <ApiEndpointCard
                key={`${endpoint.method}-${endpoint.path}-${i}`}
                endpoint={endpoint}
              />
            ))}
          </div>
        </div>
      ))}
    </div>
  )
}
