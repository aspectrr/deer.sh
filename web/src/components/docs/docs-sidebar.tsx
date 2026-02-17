import { Link, useLocation } from '@tanstack/react-router'
import { docsNav } from '~/lib/docs-nav'
import { cn } from '~/lib/utils'

export function DocsSidebar() {
  const location = useLocation()

  return (
    <aside className="border-border hidden w-56 shrink-0 border-r md:block">
      <div className="sticky top-0 h-screen overflow-y-auto p-4">
        {docsNav.map((section) => (
          <div key={section.title} className="mb-6">
            <h4 className="text-muted-foreground mb-2 text-[10px] font-medium tracking-wider uppercase">
              {section.title}
            </h4>
            <ul className="space-y-0.5">
              {section.items.map((item) => {
                const isActive = location.pathname === item.to
                return (
                  <li key={item.to}>
                    <Link
                      to={item.to}
                      className={cn(
                        'flex items-center gap-2 border-l-2 px-3 py-1.5 text-xs transition-colors',
                        isActive
                          ? 'border-blue-400 text-blue-400'
                          : 'text-muted-foreground hover:text-foreground border-transparent hover:border-neutral-700'
                      )}
                    >
                      <span className="w-6 shrink-0 font-mono text-[10px] opacity-50">
                        {item.icon}
                      </span>
                      {item.label}
                    </Link>
                  </li>
                )
              })}
            </ul>
          </div>
        ))}
      </div>
    </aside>
  )
}
