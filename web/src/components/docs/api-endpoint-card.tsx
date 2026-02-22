import { useState } from 'react'
import { ChevronDown, ChevronRight, Send, Loader2 } from 'lucide-react'
import { cn } from '~/lib/utils'
import { Button } from '~/components/ui/button'
import type { OpenAPIEndpoint } from '~/lib/openapi'
import { axios } from '~/lib/axios'

const methodColors: Record<string, string> = {
  GET: 'bg-green-400/10 text-green-400 border-green-400/30',
  POST: 'bg-blue-400/10 text-blue-400 border-blue-400/30',
  PUT: 'bg-amber-400/10 text-amber-400 border-amber-400/30',
  PATCH: 'bg-amber-400/10 text-amber-400 border-amber-400/30',
  DELETE: 'bg-red-400/10 text-red-400 border-red-400/30',
}

interface ApiEndpointCardProps {
  endpoint: OpenAPIEndpoint
}

export function ApiEndpointCard({ endpoint }: ApiEndpointCardProps) {
  const [expanded, setExpanded] = useState(false)
  const [pathParams, setPathParams] = useState<Record<string, string>>({})
  const [body, setBody] = useState('')
  const [response, setResponse] = useState<{ status: number; data: string } | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  const pathParamNames =
    endpoint.parameters?.filter((p) => p.in === 'path').map((p) => p.name) || []

  const hasBody = ['POST', 'PUT', 'PATCH'].includes(endpoint.method)

  const buildPath = () => {
    let path = endpoint.path
    for (const [key, val] of Object.entries(pathParams)) {
      path = path.replace(`{${key}}`, encodeURIComponent(val))
    }
    return path
  }

  const sendRequest = async () => {
    setLoading(true)
    setError(null)
    setResponse(null)

    try {
      const url = buildPath()
      let data: unknown = undefined
      if (hasBody && body.trim()) {
        try {
          data = JSON.parse(body)
        } catch {
          setError('Invalid JSON body')
          setLoading(false)
          return
        }
      }

      const res = await axios({
        method: endpoint.method.toLowerCase() as 'get' | 'post' | 'put' | 'patch' | 'delete',
        url,
        data,
        validateStatus: () => true,
      })
      setResponse({
        status: res.status,
        data: JSON.stringify(res.data, null, 2),
      })
    } catch (err) {
      setError(
        err instanceof Error && err.message.includes('Network Error')
          ? 'Backend offline - start the API server to test endpoints'
          : `Request failed: ${err instanceof Error ? err.message : 'Unknown error'}`
      )
    } finally {
      setLoading(false)
    }
  }

  // Highlight path params in the path
  const renderPath = () => {
    const parts = endpoint.path.split(/(\{[^}]+\})/)
    return parts.map((part, i) => {
      if (part.startsWith('{') && part.endsWith('}')) {
        return (
          <span key={i} className="text-amber-400">
            {part}
          </span>
        )
      }
      return <span key={i}>{part}</span>
    })
  }

  return (
    <div className="border-border border-b last:border-0">
      <button
        onClick={() => setExpanded(!expanded)}
        className="flex w-full items-center gap-3 px-3 py-2 text-left transition-colors hover:bg-neutral-900/50"
      >
        {expanded ? (
          <ChevronDown className="h-3 w-3 shrink-0 text-neutral-600" />
        ) : (
          <ChevronRight className="h-3 w-3 shrink-0 text-neutral-600" />
        )}
        <span
          className={cn(
            'w-14 shrink-0 border px-1.5 py-0.5 text-center text-[10px] font-medium',
            methodColors[endpoint.method] || 'border-neutral-600 text-neutral-400'
          )}
        >
          {endpoint.method}
        </span>
        <span className="font-mono text-xs text-neutral-300">{renderPath()}</span>
        {endpoint.summary && (
          <span className="ml-auto truncate text-[10px] text-neutral-500">{endpoint.summary}</span>
        )}
      </button>

      {expanded && (
        <div className="space-y-3 px-3 pb-3 pl-9">
          {endpoint.description && (
            <p className="text-xs text-neutral-400">{endpoint.description}</p>
          )}

          {/* Path params */}
          {pathParamNames.length > 0 && (
            <div>
              <span className="text-[10px] tracking-wider text-neutral-500 uppercase">
                Path Parameters
              </span>
              <div className="mt-1 space-y-1">
                {pathParamNames.map((name) => (
                  <div key={name} className="flex items-center gap-2">
                    <label className="w-32 shrink-0 font-mono text-xs text-neutral-300">
                      {name}
                    </label>
                    <input
                      type="text"
                      value={pathParams[name] || ''}
                      onChange={(e) => setPathParams({ ...pathParams, [name]: e.target.value })}
                      placeholder={name}
                      className="border-border w-full border bg-neutral-900 px-2 py-1 text-xs text-neutral-200 focus:border-blue-400 focus:outline-none"
                    />
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Request body */}
          {hasBody && (
            <div>
              <span className="text-[10px] tracking-wider text-neutral-500 uppercase">
                Request Body (JSON)
              </span>
              <textarea
                value={body}
                onChange={(e) => setBody(e.target.value)}
                placeholder='{"key": "value"}'
                rows={4}
                className="border-border mt-1 w-full resize-y border bg-neutral-900 px-2 py-1.5 font-mono text-xs text-neutral-200 focus:border-blue-400 focus:outline-none"
              />
            </div>
          )}

          {/* Send button */}
          <Button size="xs" onClick={sendRequest} disabled={loading}>
            {loading ? <Loader2 className="h-3 w-3 animate-spin" /> : <Send className="h-3 w-3" />}
            Send
          </Button>

          {/* Error */}
          {error && (
            <div className="border border-red-400/20 bg-red-400/5 p-2 text-xs text-red-400">
              {error}
            </div>
          )}

          {/* Response */}
          {response && (
            <div>
              <div className="mb-1 flex items-center gap-2">
                <span className="text-[10px] tracking-wider text-neutral-500 uppercase">
                  Response
                </span>
                <span
                  className={cn(
                    'font-mono text-[10px]',
                    response.status < 300
                      ? 'text-green-400'
                      : response.status < 500
                        ? 'text-amber-400'
                        : 'text-red-400'
                  )}
                >
                  {response.status}
                </span>
              </div>
              <pre className="border-border max-h-64 overflow-auto border bg-neutral-900 p-2 font-mono text-xs text-neutral-300">
                {response.data}
              </pre>
            </div>
          )}
        </div>
      )}
    </div>
  )
}
