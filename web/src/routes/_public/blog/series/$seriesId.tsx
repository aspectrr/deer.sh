import { createFileRoute, Link, notFound } from '@tanstack/react-router'
import { useState } from 'react'
import { Menu, X } from 'lucide-react'
import { blogSeries } from '~/lib/blog-series'
import { useAuth } from '~/lib/auth'
import { useReturningVisitor } from '~/lib/use-returning-visitor'

interface BlogModule {
  frontmatter: { title: string; pubDate: string; description: string }
  default: React.ComponentType
}

const modules = import.meta.glob<BlogModule>('/src/content/blog/**/*.{md,mdx}', {
  eager: true,
})

function getSlug(path: string) {
  const filename = path.split('/').pop() || ''
  return filename.replace(/\.(md|mdx)$/, '')
}

export const Route = createFileRoute('/_public/blog/series/$seriesId')({
  component: SeriesPage,
})

function SeriesPage() {
  const { seriesId } = Route.useParams()
  const series = blogSeries.find((s) => s.id === seriesId)
  if (!series) throw notFound()

  const { isAuthenticated } = useAuth()
  const isReturning = useReturningVisitor()
  const [mobileOpen, setMobileOpen] = useState(false)

  const now = new Date()
  const publishedChapters = series.chapters.filter((ch) => {
    for (const [path, mod] of Object.entries(modules)) {
      if (getSlug(path) === ch.slug && new Date(mod.frontmatter.pubDate) <= now) {
        return true
      }
    }
    return false
  })

  return (
    <>
      <header className="px-4 pt-24 pb-12 sm:px-6">
        <div className="mx-auto max-w-2xl">
          <div className="mb-6 flex items-center justify-between">
            <Link
              to="/"
              className="font-logo text-2xl tracking-tight text-white no-underline hover:no-underline md:text-3xl"
            >
              <span className="text-green-800">$</span> deer.sh
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
                href="https://github.com/aspectrr/deer.sh"
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
            <div className="fixed inset-0 z-30 bg-black md:hidden">
              <div className="flex flex-col gap-4 p-6">
                <button
                  className="self-end text-neutral-400 hover:text-white"
                  onClick={() => setMobileOpen(false)}
                >
                  <X className="h-5 w-5" />
                </button>
                <Link to="/docs/quickstart" className="font-mono text-neutral-200">
                  Docs
                </Link>
                <Link to="/blog" className="font-mono text-neutral-200">
                  Blog
                </Link>
                <Link to="/pricing" className="font-mono text-neutral-200">
                  Pricing
                </Link>
                <a
                  href="https://github.com/aspectrr/deer.sh"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="font-mono text-neutral-200"
                >
                  GitHub
                </a>
                <a
                  href="https://discord.gg/4WGGXJWm8J"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="font-mono text-neutral-200"
                >
                  Discord
                </a>
              </div>
            </div>
          )}

          <div className="mt-8">
            <Link
              to="/blog"
              className="font-mono text-xs text-neutral-500 transition-colors hover:text-green-800"
            >
              &larr; All Posts
            </Link>
          </div>

          <div className="mt-6">
            <p className="font-mono text-[10px] tracking-wider text-green-800 uppercase">
              Technical Deep Dive &middot; {publishedChapters.length}/{series.chapters.length} parts
            </p>
            <h1 className="mt-2 font-mono text-3xl font-bold text-white">{series.title}</h1>
            <p className="mt-2 font-mono text-sm text-neutral-400">{series.subtitle}</p>
            <p className="mt-4 text-sm leading-relaxed text-neutral-400">{series.description}</p>
          </div>
        </div>
      </header>

      <main className="px-4 pb-24 sm:px-6">
        <div className="mx-auto max-w-2xl space-y-3">
          {series.chapters.map((ch) => {
            const isPublished = publishedChapters.some((p) => p.slug === ch.slug)
            return (
              <Link
                key={ch.slug}
                to="/blog/$slug"
                params={{ slug: ch.slug }}
                className={
                  isPublished
                    ? 'group block rounded-lg border border-neutral-800 bg-neutral-900/50 p-4 no-underline transition-colors hover:border-green-900/30 hover:no-underline'
                    : 'pointer-events-none block rounded-lg border border-neutral-800/50 bg-neutral-900/20 p-4 no-underline opacity-50'
                }
              >
                <div className="flex items-center gap-3">
                  <span className="font-mono text-sm text-green-800">
                    {String(ch.part).padStart(2, '0')}
                  </span>
                  <div>
                    <span
                      className={
                        isPublished
                          ? 'font-mono text-sm text-neutral-300 transition-colors group-hover:text-green-800'
                          : 'font-mono text-sm text-neutral-500'
                      }
                    >
                      {ch.shortTitle}
                    </span>
                    {!isPublished && (
                      <span className="ml-2 font-mono text-[10px] text-neutral-600">
                        (coming soon)
                      </span>
                    )}
                  </div>
                </div>
              </Link>
            )
          })}
        </div>
      </main>
    </>
  )
}
