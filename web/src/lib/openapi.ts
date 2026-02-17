import YAML from 'yaml'

export interface OpenAPIParam {
  name: string
  in: 'path' | 'query' | 'header'
  description?: string
  required?: boolean
  schema?: { type?: string }
}

export interface OpenAPIEndpoint {
  method: string
  path: string
  summary?: string
  description?: string
  tags?: string[]
  parameters?: OpenAPIParam[]
  requestBody?: {
    content?: {
      'application/json'?: {
        schema?: Record<string, unknown>
      }
    }
  }
  responses?: Record<
    string,
    {
      description?: string
      content?: Record<string, { schema?: Record<string, unknown> }>
    }
  >
}

export interface ParsedSpec {
  endpoints: OpenAPIEndpoint[]
  tags: string[]
}

export function parseOpenAPISpec(yamlString: string): ParsedSpec {
  const spec = YAML.parse(yamlString)
  const endpoints: OpenAPIEndpoint[] = []
  const tagSet = new Set<string>()

  const paths = spec.paths || {}
  for (const [path, methods] of Object.entries(paths)) {
    for (const [method, details] of Object.entries(
      methods as Record<string, Record<string, unknown>>
    )) {
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
