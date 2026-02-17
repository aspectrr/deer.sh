import { useEffect, useState } from 'react'
import { cn } from '~/lib/utils'

export interface TocItem {
  id: string
  text: string
  level: number
}

export function DocsToc({ items }: { items: TocItem[] }) {
  const [activeId, setActiveId] = useState<string>('')

  useEffect(() => {
    if (items.length === 0) return

    const observer = new IntersectionObserver(
      (entries) => {
        for (const entry of entries) {
          if (entry.isIntersecting) {
            setActiveId(entry.target.id)
          }
        }
      },
      { rootMargin: '-80px 0px -70% 0px', threshold: 0 }
    )

    for (const item of items) {
      const el = document.getElementById(item.id)
      if (el) observer.observe(el)
    }

    return () => observer.disconnect()
  }, [items])

  if (items.length === 0) return null

  return (
    <nav className="hidden w-48 shrink-0 xl:block">
      <div className="sticky top-16 p-4">
        <h4 className="text-muted-foreground mb-3 text-[10px] font-medium tracking-wider uppercase">
          On this page
        </h4>
        <ul className="space-y-1">
          {items.map((item) => (
            <li key={item.id}>
              <a
                href={`#${item.id}`}
                className={cn(
                  'block border-l-2 py-0.5 text-[11px] transition-colors',
                  item.level === 3 ? 'pl-6' : 'pl-3',
                  activeId === item.id
                    ? 'border-blue-400 text-blue-400'
                    : 'text-muted-foreground hover:text-foreground border-transparent'
                )}
              >
                {item.text}
              </a>
            </li>
          ))}
        </ul>
      </div>
    </nav>
  )
}
