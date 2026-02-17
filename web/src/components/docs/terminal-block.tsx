import { Copy, Check } from 'lucide-react'
import { useState } from 'react'
import { cn } from '~/lib/utils'

interface TerminalLine {
  command?: string
  output?: string
}

interface TerminalBlockProps {
  lines: TerminalLine[]
  className?: string
}

export function TerminalBlock({ lines, className }: TerminalBlockProps) {
  const [copied, setCopied] = useState(false)

  const commands = lines.filter((l) => l.command).map((l) => l.command!)

  const handleCopy = async () => {
    await navigator.clipboard.writeText(commands.join('\n'))
    setCopied(true)
    setTimeout(() => setCopied(false), 1500)
  }

  return (
    <div className={cn('group border-border relative mb-3 border bg-neutral-900', className)}>
      <button
        onClick={handleCopy}
        className="text-muted-foreground hover:text-foreground absolute top-2 right-2 p-1 opacity-0 transition-opacity group-hover:opacity-100"
      >
        {copied ? <Check className="h-3 w-3 text-green-400" /> : <Copy className="h-3 w-3" />}
      </button>
      <div className="p-3 font-mono text-xs">
        {lines.map((line, i) => (
          <div key={i}>
            {line.command && (
              <div className="text-neutral-200">
                <span className="text-blue-400">$ </span>
                {line.command}
              </div>
            )}
            {line.output && <div className="text-neutral-500">{line.output}</div>}
          </div>
        ))}
      </div>
    </div>
  )
}
