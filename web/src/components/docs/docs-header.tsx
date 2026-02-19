import { Link } from '@tanstack/react-router'
import { ArrowLeft, Github, Menu, X } from 'lucide-react'
import { useAuth } from '~/lib/auth'
import { useState } from 'react'
import { docsNav } from '~/lib/docs-nav'
import { useLocation } from '@tanstack/react-router'
import { cn } from '~/lib/utils'

export function DocsHeader() {
  const { isAuthenticated } = useAuth()
  const [mobileOpen, setMobileOpen] = useState(false)
  const location = useLocation()

  return (
    <>
      <header className="border-border sticky top-0 z-40 flex h-12 items-center justify-between border-b bg-black px-4">
        <div className="flex items-center gap-2 p-4">
          <button
            className="text-muted-foreground hover:text-foreground md:hidden"
            onClick={() => setMobileOpen(!mobileOpen)}
          >
            {mobileOpen ? <X className="h-4 w-4" /> : <Menu className="h-4 w-4" />}
          </button>
          <Link to="/dashboard" className="font-logo text-lg font-extrabold text-white">
            fluid.sh
          </Link>
          <span className="text-muted-foreground text-sm">/</span>
          <Link
            to="/docs/quickstart"
            className="text-muted-foreground text-sm transition-colors hover:text-white"
          >
            docs
          </Link>
        </div>
        <div className="flex items-center gap-2">
          <a
            href="https://github.com/aspectrr/fluid.sh"
            target="_blank"
            rel="noopener noreferrer"
            className="text-muted-foreground hover:text-foreground p-1.5"
          >
            <Github className="h-3.5 w-3.5" />
          </a>
          <Link
            to={isAuthenticated ? '/dashboard' : '/login'}
            className="border-border hover:bg-muted flex items-center gap-1.5 border px-2.5 py-1 text-xs text-white"
          >
            <ArrowLeft className="h-3 w-3" />
            {isAuthenticated ? 'Dashboard' : 'Sign in'}
          </Link>
        </div>
      </header>

      {/* Mobile sidebar overlay */}
      {mobileOpen && (
        <div
          className="fixed inset-0 top-12 z-30 bg-black/80 md:hidden"
          onClick={() => setMobileOpen(false)}
        >
          <div
            className="border-border w-56 border-r bg-black p-4"
            onClick={(e) => e.stopPropagation()}
          >
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
                          onClick={() => setMobileOpen(false)}
                          className={cn(
                            'flex items-center gap-2 border-l-2 px-3 py-1.5 text-xs transition-colors',
                            isActive
                              ? 'border-blue-400 text-blue-400'
                              : 'text-muted-foreground hover:text-foreground border-transparent'
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
        </div>
      )}
    </>
  )
}
