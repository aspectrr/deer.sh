import { useState, useCallback, useRef } from 'react'
import { Link } from '@tanstack/react-router'

const INSTALL_COMMAND = 'curl -fsSL https://deer.sh/install.sh | bash'

function CopyButton() {
  const [copied, setCopied] = useState(false)
  const timeoutRef = useRef<ReturnType<typeof setTimeout>>(null)

  const handleCopy = useCallback(async () => {
    await navigator.clipboard.writeText(INSTALL_COMMAND)
    if (timeoutRef.current) clearTimeout(timeoutRef.current)
    setCopied(true)
    timeoutRef.current = setTimeout(() => setCopied(false), 2000)
  }, [])

  return (
    <button
      onClick={handleCopy}
      className="rounded p-2 text-neutral-600 transition-colors hover:bg-neutral-800 hover:text-neutral-300"
      aria-label="Copy to clipboard"
    >
      {copied ? (
        <svg
          className="h-4 w-4 text-green-800"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
        </svg>
      ) : (
        <svg className="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"
          />
        </svg>
      )}
    </button>
  )
}

export function BlogCTA() {
  return (
    <div className="mt-16 overflow-hidden border border-neutral-800 bg-neutral-900">
      <div className="flex items-center gap-2 border-b border-neutral-800 px-4 py-2">
        <div className="flex gap-1.5">
          <span className="h-2.5 w-2.5 rounded-full bg-neutral-700" />
          <span className="h-2.5 w-2.5 rounded-full bg-neutral-700" />
          <span className="h-2.5 w-2.5 rounded-full bg-neutral-700" />
        </div>
        <span className="font-mono text-xs text-neutral-500">install</span>
      </div>
      <div className="p-5">
        <div className="flex items-center justify-between gap-4 overflow-hidden font-mono text-sm">
          <div className="min-w-0 overflow-x-auto text-neutral-400">
            <span className="text-green-800 select-none">$ </span>
            <span className="whitespace-nowrap">{INSTALL_COMMAND}</span>
          </div>
          <CopyButton />
        </div>
        <div className="mt-4 border-t border-neutral-800 pt-4">
          <p className="mb-3 text-sm text-neutral-400">
            Spend most of your time managing or debugging Linux servers? Fluid can help.
          </p>
          <Link
            to="/docs/quickstart"
            className="inline-block bg-green-900 px-4 py-2 font-mono text-sm text-white transition-colors hover:bg-green-950"
          >
            Try Fluid
          </Link>
        </div>
      </div>
    </div>
  )
}
