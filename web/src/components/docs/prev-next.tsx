import { Link, useLocation } from '@tanstack/react-router'
import { ChevronLeft, ChevronRight } from 'lucide-react'
import { flatNav } from '~/lib/docs-nav'

export function PrevNext() {
  const location = useLocation()
  const idx = flatNav.findIndex((item) => item.to === location.pathname)

  const prev = idx > 0 ? flatNav[idx - 1] : null
  const next = idx < flatNav.length - 1 ? flatNav[idx + 1] : null

  return (
    <div className="border-border mt-12 flex items-center justify-between border-t pt-4">
      {prev ? (
        <Link
          to={prev.to}
          className="text-muted-foreground hover:text-foreground flex items-center gap-1 text-xs transition-colors"
        >
          <ChevronLeft className="h-3 w-3" />
          {prev.label}
        </Link>
      ) : (
        <div />
      )}
      {next ? (
        <Link
          to={next.to}
          className="text-muted-foreground hover:text-foreground flex items-center gap-1 text-xs transition-colors"
        >
          {next.label}
          <ChevronRight className="h-3 w-3" />
        </Link>
      ) : (
        <div />
      )}
    </div>
  )
}
