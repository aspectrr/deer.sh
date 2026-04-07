import { useEffect, useState } from 'react'
import { Copy, Check } from 'lucide-react'
import { getHighlighter } from '~/lib/shiki'
import { cn } from '~/lib/utils'

interface IdeCodeBlockProps {
  code: string
  lang?: string
  filename?: string
  className?: string
}

export function IdeCodeBlock({ code, lang, filename, className }: IdeCodeBlockProps) {
  const [html, setHtml] = useState<string>('')
  const [copied, setCopied] = useState(false)
  const trimmed = code.trim()
  const lineCount = trimmed.split('\n').length
  const displayLang = lang || 'text'
  const showLineNumbers = displayLang !== 'bash' && displayLang !== 'text'

  useEffect(() => {
    let cancelled = false
    getHighlighter().then((h) => {
      if (cancelled) return
      try {
        const result = h.codeToHtml(trimmed, {
          lang: displayLang,
          theme: 'github-dark',
        })
        setHtml(sanitizeHtml(result))
      } catch {
        setHtml(`<pre class="shiki"><code>${escapeHtml(trimmed)}</code></pre>`)
      }
    })
    return () => {
      cancelled = true
    }
  }, [trimmed, displayLang])

  const handleCopy = async () => {
    await navigator.clipboard.writeText(trimmed)
    setCopied(true)
    setTimeout(() => setCopied(false), 1500)
  }

  return (
    <div
      className={cn(
        'group relative my-6 border border-[rgba(96,165,250,0.4)] bg-[rgba(96,165,250,0.05)]',
        className
      )}
    >
      {/* Title bar */}
      <div className="border-b border-neutral-800 px-3 py-2">
        <div className="flex items-center gap-2">
          <div className="flex gap-1">
            <div className="h-1 w-1 rounded-sm bg-neutral-600" />
            <div className="h-1 w-1 rounded-sm bg-neutral-600" />
            <div className="h-1 w-1 rounded-sm bg-neutral-600" />
          </div>
          <span className="font-mono text-[10px] text-neutral-500">{displayLang}</span>
        </div>
        {filename && (
          <div className="mt-1 font-mono text-xs font-medium text-white">{filename}</div>
        )}
      </div>

      {/* Code area */}
      <div className="relative">
        <button
          onClick={handleCopy}
          className="absolute top-2 right-2 z-10 p-1 text-neutral-500 opacity-0 transition-opacity group-hover:opacity-100 hover:text-neutral-300"
        >
          {copied ? <Check className="h-3 w-3 text-green-800" /> : <Copy className="h-3 w-3" />}
        </button>
        <div className="flex overflow-x-auto text-xs">
          {showLineNumbers && <LineNumbers count={lineCount} />}
          {html ? (
            <div
              className="min-w-0 flex-1 py-3 pr-3 pl-4 [&_code]:!bg-transparent [&_code]:!leading-[20px] [&_pre]:!m-0 [&_pre]:!border-0 [&_pre]:!bg-transparent [&_pre]:!p-0 [&_pre]:!leading-[20px]"
              dangerouslySetInnerHTML={{ __html: html }}
            />
          ) : (
            <pre className="min-w-0 flex-1 py-3 pr-3 pl-4 text-neutral-300">
              <code>{trimmed}</code>
            </pre>
          )}
        </div>
      </div>
    </div>
  )
}

function LineNumbers({ count }: { count: number }) {
  return (
    <div
      aria-hidden
      className="sticky left-0 border-r border-neutral-800 py-3 pr-3 pl-3 text-right font-mono text-xs leading-[20px] text-neutral-600 select-none"
    >
      {Array.from({ length: count }, (_, i) => (
        <div key={i}>{i + 1}</div>
      ))}
    </div>
  )
}

// Strip script tags and event handlers from HTML as defense-in-depth.
// Shiki output is trusted but this guards against library vulnerabilities.
function sanitizeHtml(html: string): string {
  return html
    .replace(/<script\b[^>]*>[\s\S]*?<\/script>/gi, '')
    .replace(/\bon\w+\s*=\s*(?:"[^"]*"|'[^']*'|[^\s>]*)/gi, '')
}

function escapeHtml(str: string): string {
  return str.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
}
