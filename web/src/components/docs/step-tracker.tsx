import { useState, useEffect, useRef, useCallback, type ReactNode } from 'react'
import { Check, ChevronDown, ChevronRight } from 'lucide-react'
import { cn } from '~/lib/utils'

export interface Step {
  title: string
  content: ReactNode
}

interface StepTrackerProps {
  storageKey: string
  steps: Step[]
  progressEndpoint?: string
  sessionCode?: string
}

function generateSessionCode(): string {
  const chars = 'ABCDEFGHJKLMNPQRSTUVWXYZ23456789'
  let code = ''
  for (let i = 0; i < 6; i++) {
    code += chars[Math.floor(Math.random() * chars.length)]
  }
  return code
}

export function StepTracker({
  storageKey,
  steps,
  progressEndpoint,
  sessionCode: externalCode,
}: StepTrackerProps) {
  const fullKey = `fluid-docs-progress-${storageKey}`

  const [completedSteps, setCompletedSteps] = useState<Set<number>>(() => {
    try {
      const saved = localStorage.getItem(fullKey)
      return saved ? new Set(JSON.parse(saved) as number[]) : new Set()
    } catch {
      return new Set()
    }
  })

  const [expandedStep, setExpandedStep] = useState<number>(() => {
    // Find first incomplete step
    for (let i = 0; i < steps.length; i++) {
      if (!completedSteps.has(i)) return i
    }
    return steps.length - 1
  })

  const [sessionCode] = useState<string | null>(() => {
    if (!progressEndpoint) return null
    return externalCode || generateSessionCode()
  })
  const registeredRef = useRef(false)
  const pollingRef = useRef<ReturnType<typeof setInterval> | null>(null)

  useEffect(() => {
    localStorage.setItem(fullKey, JSON.stringify(Array.from(completedSteps)))
  }, [completedSteps, fullKey])

  const markComplete = useCallback(
    (index: number) => {
      setCompletedSteps((prev) => {
        const next = new Set(prev)
        next.add(index)
        return next
      })
      setExpandedStep((current) => {
        // Expand next incomplete step
        for (let i = index + 1; i < steps.length; i++) {
          if (!completedSteps.has(i) && i !== index) {
            return i
          }
        }
        return current
      })
    },
    [completedSteps, steps.length]
  )

  // Register session when progressEndpoint is set (client-generated code only)
  useEffect(() => {
    if (!progressEndpoint || !sessionCode || externalCode || registeredRef.current) return
    registeredRef.current = true

    fetch(`${progressEndpoint}/register`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ storage_key: storageKey }),
    }).catch(() => {
      // Registration failed - polling won't find anything
    })
  }, [progressEndpoint, storageKey, externalCode, sessionCode])

  // Poll for progress when we have a session code
  useEffect(() => {
    if (!progressEndpoint || !sessionCode) return

    pollingRef.current = setInterval(async () => {
      try {
        const res = await fetch(`${progressEndpoint}/progress?code=${sessionCode}`)
        if (!res.ok) return
        const data = (await res.json()) as { completed_steps: number[] }
        if (data.completed_steps && data.completed_steps.length > 0) {
          for (const stepIndex of data.completed_steps) {
            markComplete(stepIndex)
          }
        }
      } catch {
        // Polling error - silently ignore
      }
    }, 3000)

    return () => {
      if (pollingRef.current) {
        clearInterval(pollingRef.current)
      }
    }
  }, [progressEndpoint, sessionCode, markComplete])

  const allComplete = completedSteps.size === steps.length

  return (
    <div className="mb-6">
      <div className="mb-4 flex items-center justify-between">
        <span className="text-muted-foreground text-[10px]">
          {completedSteps.size}/{steps.length} completed
        </span>
      </div>

      {sessionCode && (
        <div className="mb-4 border border-blue-400/20 bg-blue-400/5 p-3">
          {externalCode ? (
            <p className="text-xs text-blue-400">Progress syncs with your CLI session.</p>
          ) : (
            <>
              <p className="text-xs text-blue-400">
                Session code: <code className="font-mono font-bold text-white">{sessionCode}</code>
              </p>
              <p className="mt-1 text-[10px] text-neutral-500">
                Progress will auto-update as you complete each step in the CLI.
              </p>
            </>
          )}
        </div>
      )}

      <div className="relative">
        {/* Vertical line */}
        <div className="absolute top-2 bottom-2 left-[7px] w-px bg-neutral-800" />

        <div className="space-y-1">
          {steps.map((step, i) => {
            const isComplete = completedSteps.has(i)
            const isExpanded = expandedStep === i
            const isCurrent = !isComplete && i === expandedStep

            return (
              <div key={i}>
                <button
                  onClick={() => {
                    if (isExpanded) {
                      setExpandedStep(-1)
                    } else {
                      setExpandedStep(i)
                      // Bidirectional: mark 0..i-1 complete, unmark i and beyond
                      setCompletedSteps(() => {
                        const next = new Set<number>()
                        for (let j = 0; j < i; j++) {
                          next.add(j)
                        }
                        return next
                      })
                    }
                  }}
                  className="group flex w-full items-center gap-3 py-1.5 text-left"
                >
                  {/* Dot */}
                  <div
                    className={cn(
                      'relative z-10 flex h-[15px] w-[15px] shrink-0 items-center justify-center',
                      isComplete && 'bg-blue-400/20',
                      isCurrent && 'shadow-[0_0_8px_2px_rgba(96,165,250,0.6)]'
                    )}
                  >
                    {isComplete ? (
                      <Check className="h-2.5 w-2.5 text-blue-400" />
                    ) : (
                      <div
                        className={cn('h-2 w-2', isCurrent ? 'bg-blue-400' : 'bg-neutral-700')}
                      />
                    )}
                  </div>

                  <span
                    className={cn(
                      'flex-1 text-xs',
                      isComplete
                        ? 'text-neutral-500 line-through'
                        : isCurrent
                          ? 'text-white'
                          : 'text-neutral-400'
                    )}
                  >
                    {step.title}
                  </span>

                  {isExpanded ? (
                    <ChevronDown className="h-3 w-3 text-neutral-600" />
                  ) : (
                    <ChevronRight className="h-3 w-3 text-neutral-600" />
                  )}
                </button>

                {isExpanded && (
                  <div className="ml-[27px] pb-3">
                    <div className="docs-prose">{step.content}</div>
                  </div>
                )}
              </div>
            )
          })}
        </div>
      </div>

      {allComplete && (
        <div className="mt-4 border border-blue-400/20 bg-blue-400/5 p-3">
          <p className="text-xs text-blue-400">All steps complete.</p>
        </div>
      )}
    </div>
  )
}
