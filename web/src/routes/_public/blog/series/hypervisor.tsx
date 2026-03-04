import { createFileRoute, Link } from '@tanstack/react-router'
import { useState } from 'react'
import { Menu, X } from 'lucide-react'
import { blogSeries } from '~/lib/blog-series'
import { useAuth } from '~/lib/auth'
import { useReturningVisitor } from '~/lib/use-returning-visitor'

interface BlogFrontmatter {
  title: string
  pubDate: string
  description: string
  author?: string
  authorImage?: string
}

interface BlogModule {
  frontmatter: BlogFrontmatter
  default: React.ComponentType
}

const modules = import.meta.glob<BlogModule>('/src/content/blog/*.{md,mdx}', { eager: true })

function getSlug(path: string) {
  const filename = path.split('/').pop() || ''
  return filename.replace(/\.(md|mdx)$/, '')
}

const series = blogSeries.find((s) => s.id === 'hypervisor')!

function getChapterMeta() {
  const slugToMod = new Map<string, BlogModule>()
  for (const [path, mod] of Object.entries(modules)) {
    slugToMod.set(getSlug(path), mod)
  }

  const now = new Date()
  return series.chapters.map((ch) => {
    const mod = slugToMod.get(ch.slug)
    const published = mod ? new Date(mod.frontmatter.pubDate) <= now : false
    return {
      ...ch,
      title: mod?.frontmatter.title ?? ch.shortTitle,
      description: mod?.frontmatter.description ?? '',
      pubDate: mod?.frontmatter.pubDate ?? '',
      published,
    }
  })
}

export const Route = createFileRoute('/_public/blog/series/hypervisor')({
  component: HypervisorSeries,
})

function HypervisorSeries() {
  const { isAuthenticated } = useAuth()
  const isReturning = useReturningVisitor()
  const chapters = getChapterMeta()
  const [mobileOpen, setMobileOpen] = useState(false)
  const publishedCount = chapters.filter((ch) => ch.published).length

  return (
    <>
      <header className="px-4 pt-24 pb-12 sm:px-6">
        <div className="mx-auto max-w-2xl">
          <div className="mb-6 flex items-center justify-between">
            <Link
              to="/"
              className="font-logo text-2xl tracking-tight text-white no-underline hover:no-underline md:text-3xl"
            >
              <span className="text-blue-400">$</span> fluid.sh
            </Link>
            <div className="hidden items-center gap-6 font-mono text-sm text-neutral-400 md:flex">
              <Link to="/docs/quickstart" className="transition-colors hover:text-neutral-200">
                Docs
              </Link>
              <Link to="/blog" className="transition-colors hover:text-neutral-200">
                Blog
              </Link>
              <Link to="/pricing" className="transition-colors hover:text-neutral-200">
                Pricing
              </Link>
              <a
                href="https://github.com/aspectrr/fluid.sh"
                target="_blank"
                rel="noopener noreferrer"
                className="transition-colors hover:text-neutral-200"
              >
                GitHub
              </a>
              <a
                href="https://discord.gg/4WGGXJWm8J"
                target="_blank"
                rel="noopener noreferrer"
                className="transition-colors hover:text-neutral-200"
              >
                Discord
              </a>
              {isAuthenticated ? (
                <Link
                  to="/dashboard"
                  className="rounded border border-neutral-700 px-3 py-1 text-neutral-300 transition-colors hover:border-neutral-500 hover:text-neutral-100"
                >
                  Dashboard
                </Link>
              ) : isReturning ? (
                <Link
                  to="/login"
                  className="rounded border border-neutral-700 px-3 py-1 text-neutral-300 transition-colors hover:border-neutral-500 hover:text-neutral-100"
                >
                  Login
                </Link>
              ) : (
                <Link
                  to="/register"
                  className="rounded border border-neutral-700 px-3 py-1 text-neutral-300 transition-colors hover:border-neutral-500 hover:text-neutral-100"
                >
                  Sign Up
                </Link>
              )}
            </div>
            <button
              className="text-neutral-400 hover:text-white md:hidden"
              onClick={() => setMobileOpen(!mobileOpen)}
            >
              {mobileOpen ? <X className="h-5 w-5" /> : <Menu className="h-5 w-5" />}
            </button>
          </div>

          {mobileOpen && (
            <div
              className="fixed inset-0 z-30 bg-black md:hidden"
              onClick={() => setMobileOpen(false)}
            >
              <nav
                className="flex flex-col gap-6 p-8 pt-20 font-mono text-lg text-neutral-300"
                onClick={(e) => e.stopPropagation()}
              >
                <Link
                  to="/docs/quickstart"
                  onClick={() => setMobileOpen(false)}
                  className="transition-colors hover:text-white"
                >
                  Docs
                </Link>
                <Link
                  to="/blog"
                  onClick={() => setMobileOpen(false)}
                  className="transition-colors hover:text-white"
                >
                  Blog
                </Link>
                <Link
                  to="/pricing"
                  onClick={() => setMobileOpen(false)}
                  className="transition-colors hover:text-white"
                >
                  Pricing
                </Link>
                <a
                  href="https://github.com/aspectrr/fluid.sh"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="transition-colors hover:text-white"
                  onClick={() => setMobileOpen(false)}
                >
                  GitHub
                </a>
                <a
                  href="https://discord.gg/4WGGXJWm8J"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="transition-colors hover:text-white"
                  onClick={() => setMobileOpen(false)}
                >
                  Discord
                </a>
                <Link
                  to={isAuthenticated ? '/dashboard' : isReturning ? '/login' : '/register'}
                  onClick={() => setMobileOpen(false)}
                  className="transition-colors hover:text-white"
                >
                  {isAuthenticated ? 'Dashboard' : isReturning ? 'Login' : 'Sign Up'}
                </Link>
              </nav>
            </div>
          )}
        </div>
      </header>

      <main className="px-4 pb-24 sm:px-6">
        <div className="mx-auto max-w-2xl">
          <div>
            <span className="font-mono text-[10px] tracking-wider text-blue-400 uppercase">
              Technical Deep Dive - {publishedCount}/{series.chapters.length} parts
            </span>
            <h1 className="font-logo mt-2 text-2xl tracking-tight md:text-3xl">
              <span className="text-blue-400">#</span> {series.title}
            </h1>
            <p className="mt-1 font-mono text-sm text-neutral-400">{series.subtitle}</p>
            <p className="mt-4 text-sm leading-relaxed text-neutral-500">{series.description}</p>

            {/* Author */}
            <div className="mt-6 flex items-center gap-3">
              <img
                src={series.authorImage}
                alt={series.author}
                className="h-10 w-10 rounded-full border border-neutral-700 object-cover"
              />
              <span className="font-mono text-sm text-neutral-400">{series.author}</span>
            </div>
          </div>

          {/* Chapter list */}
          <div className="mt-10 space-y-3">
            {chapters.map((ch) =>
              ch.published ? (
                <Link
                  key={ch.slug}
                  to="/blog/$slug"
                  params={{ slug: ch.slug }}
                  className="group flex items-start gap-4 rounded-lg border border-neutral-800 bg-neutral-900/50 p-4 no-underline transition-all duration-300 hover:border-blue-500/30 hover:no-underline"
                >
                  <span className="mt-0.5 shrink-0 font-mono text-sm text-blue-400">
                    {String(ch.part).padStart(2, '0')}
                  </span>
                  <div className="min-w-0 flex-1">
                    <div className="flex flex-col gap-1 sm:flex-row sm:items-start sm:justify-between">
                      <h2 className="font-mono text-sm text-neutral-200 transition-colors group-hover:text-blue-400">
                        {ch.title}
                      </h2>
                      <time
                        dateTime={new Date(ch.pubDate).toISOString()}
                        className="font-mono text-xs whitespace-nowrap text-neutral-600"
                      >
                        {new Date(ch.pubDate).toLocaleDateString('en-us', {
                          year: 'numeric',
                          month: 'short',
                          day: 'numeric',
                        })}
                      </time>
                    </div>
                    <p className="mt-1 line-clamp-2 text-sm text-neutral-500">{ch.description}</p>
                  </div>
                </Link>
              ) : (
                <div
                  key={ch.slug}
                  className="flex items-start gap-4 rounded-lg border border-neutral-800/50 bg-neutral-900/25 p-4 opacity-50"
                >
                  <span className="mt-0.5 shrink-0 font-mono text-sm text-neutral-600">
                    {String(ch.part).padStart(2, '0')}
                  </span>
                  <div className="min-w-0 flex-1">
                    <div className="flex items-start justify-between">
                      <h2 className="font-mono text-sm text-neutral-500">{ch.shortTitle}</h2>
                      <span className="font-mono text-[10px] text-neutral-600 uppercase">
                        Coming soon
                      </span>
                    </div>
                  </div>
                </div>
              )
            )}
          </div>
        </div>
      </main>
    </>
  )
}
