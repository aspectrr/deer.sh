import { useState } from 'react'
import { Copy, Check } from 'lucide-react'
import { cn } from '~/lib/utils'

interface TerminalCodeBlockProps {
  code: string
  title?: string
  className?: string
}

export function TerminalCodeBlock({ code, title, className }: TerminalCodeBlockProps) {
  const [copied, setCopied] = useState(false)
  const trimmed = code.trim()
  const lines = trimmed.split('\n')

  const handleCopy = async () => {
    await navigator.clipboard.writeText(trimmed)
    setCopied(true)
    setTimeout(() => setCopied(false), 1500)
  }

  return (
    <div className={cn('group relative my-6 rounded-2xl', className)}>
      {/* Title bar */}
      <div className="flex items-center gap-2 rounded-t-2xl bg-neutral-800 px-4 py-2.5">
        <div className="flex items-center gap-1.5">
          <div className="h-3 w-3 rounded-full bg-neutral-600" />
          <div className="h-3 w-3 rounded-full bg-neutral-600" />
          <div className="h-3 w-3 rounded-full bg-neutral-600" />
        </div>
        <span className="ml-1 font-mono text-xs font-medium text-neutral-400">
          {title || 'terminal'}
        </span>
        <button
          onClick={handleCopy}
          className="ml-auto p-1 text-neutral-500 opacity-0 transition-opacity group-hover:opacity-100 hover:text-neutral-300"
        >
          {copied ? <Check className="h-3 w-3 text-green-400" /> : <Copy className="h-3 w-3" />}
        </button>
      </div>

      {/* Terminal output */}
      <div className="rounded-b-2xl border border-t-0 border-neutral-800 bg-black">
        <pre className="!m-0 overflow-x-auto !border-0 !bg-transparent !p-4 font-mono text-xs leading-relaxed">
          <code>
            {lines.map((line, i) => (
              <span key={i}>
                {styleLine(line)}
                {i < lines.length - 1 ? '\n' : ''}
              </span>
            ))}
          </code>
        </pre>
      </div>
    </div>
  )
}

function styleLine(line: string) {
  // Shell prompt: / # or $ prefix
  const shellMatch = line.match(/^(\/\s*#\s*|.*?\$\s)(.*)$/)
  if (shellMatch) {
    const [, prompt, cmd] = shellMatch
    if (!cmd) {
      // Bare prompt with no command
      return <span className="text-green-400">{prompt}</span>
    }
    return (
      <>
        <span className="text-green-400">{prompt}</span>
        <span className="text-green-300">{cmd}</span>
      </>
    )
  }

  // Kernel log lines: [    0.000000] ...
  if (/^\[[\s\d.]+\]/.test(line)) {
    return <span className="text-neutral-400">{line}</span>
  }

  // Ellipsis / truncation markers
  if (line === '...' || line === '---') {
    return <span className="text-neutral-600">{line}</span>
  }

  // Default output
  return <span className="text-neutral-300">{line}</span>
}
