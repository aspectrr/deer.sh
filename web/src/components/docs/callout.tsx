import { Info, AlertTriangle, Lightbulb, AlertOctagon } from 'lucide-react'
import { cn } from '~/lib/utils'
import type { ReactNode } from 'react'

const variants = {
  info: {
    icon: Info,
    border: 'border-blue-400/30',
    bg: 'bg-blue-400/5',
    text: 'text-blue-400',
  },
  warning: {
    icon: AlertTriangle,
    border: 'border-amber-400/30',
    bg: 'bg-amber-400/5',
    text: 'text-amber-400',
  },
  tip: {
    icon: Lightbulb,
    border: 'border-green-400/30',
    bg: 'bg-green-400/5',
    text: 'text-green-400',
  },
  danger: {
    icon: AlertOctagon,
    border: 'border-red-400/30',
    bg: 'bg-red-400/5',
    text: 'text-red-400',
  },
}

interface CalloutProps {
  type: keyof typeof variants
  title?: string
  children: ReactNode
}

export function Callout({ type, title, children }: CalloutProps) {
  const v = variants[type]
  const Icon = v.icon

  return (
    <div className={cn('mb-3 border-l-2 p-3', v.border, v.bg)}>
      <div className="flex items-start gap-2">
        <Icon className={cn('mt-0.5 h-3 w-3 shrink-0', v.text)} />
        <div className="min-w-0">
          {title && <p className={cn('mb-1 text-xs font-medium', v.text)}>{title}</p>}
          <div className="text-xs text-neutral-400">{children}</div>
        </div>
      </div>
    </div>
  )
}
