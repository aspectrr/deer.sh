import { createFileRoute, Link } from '@tanstack/react-router'
import { useState } from 'react'
import { Menu, X } from 'lucide-react'
import { getSeriesSlugs, blogSeries } from '~/lib/blog-series'
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

interface PostEntry {
  kind: 'post'
  slug: string
  title: string
  pubDate: string
  description: string
  author?: string
  authorImage?: string
}

interface SeriesEntry {
  kind: 'series'
  id: string
  title: string
  subtitle: string
  author: string
  authorImage: string
  pubDate: string
  latestPubDate: string
  publishedCount: number
  totalCount: number
}

type FeedEntry = PostEntry | SeriesEntry

function getFeed(): FeedEntry[] {
  const now = new Date()
  const seriesSlugs = getSeriesSlugs()

  const posts: FeedEntry[] = Object.entries(modules)
    .map(([path, mod]) => ({
      kind: 'post' as const,
      slug: getSlug(path),
      ...mod.frontmatter,
    }))
    .filter((post) => new Date(post.pubDate) <= now && !seriesSlugs.has(post.slug))

  // Build series cards
  const seriesCards: FeedEntry[] = blogSeries
    .map((s) => {
      const publishedDates: Date[] = []
      for (const ch of s.chapters) {
        for (const [path, mod] of Object.entries(modules)) {
          if (getSlug(path) === ch.slug && new Date(mod.frontmatter.pubDate) <= now) {
            publishedDates.push(new Date(mod.frontmatter.pubDate))
          }
        }
      }
      if (publishedDates.length === 0) return null
      const earliest = publishedDates.reduce((a, b) => (a < b ? a : b))
      const latest = publishedDates.reduce((a, b) => (a > b ? a : b))
      return {
        kind: 'series' as const,
        id: s.id,
        title: s.title,
        subtitle: s.subtitle,
        author: s.author,
        authorImage: s.authorImage,
        pubDate: earliest.toISOString(),
        latestPubDate: latest.toISOString(),
        publishedCount: publishedDates.length,
        totalCount: s.chapters.length,
      }
    })
    .filter((x): x is SeriesEntry => x !== null)

  return [...posts, ...seriesCards].sort(
    (a, b) => new Date(b.pubDate).valueOf() - new Date(a.pubDate).valueOf()
  )
}

export const Route = createFileRoute('/_public/blog/')({
  component: BlogIndex,
})

function BlogIndex() {
  const { isAuthenticated } = useAuth()
  const isReturning = useReturningVisitor()
  const feed = getFeed()
  const [mobileOpen, setMobileOpen] = useState(false)

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
              <Link to="/blog" className="text-neutral-200">
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

          {/* Mobile nav overlay */}
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
        <div className="mx-auto max-w-2xl space-y-3">
          {feed.map((entry) =>
            entry.kind === 'series' ? (
              <Link
                key={`series-${entry.id}`}
                to="/blog/series/hypervisor"
                className="group block rounded-lg border border-blue-500/20 bg-neutral-900/50 p-4 no-underline transition-all duration-300 hover:border-blue-500/40 hover:no-underline"
              >
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <span className="font-mono text-[10px] tracking-wider text-blue-400 uppercase">
                      Technical Deep Dive
                    </span>
                    <span className="rounded bg-blue-500/10 px-1.5 py-0.5 font-mono text-[10px] text-blue-400">
                      {entry.publishedCount}/{entry.totalCount} parts
                    </span>
                  </div>
                  <time
                    dateTime={new Date(entry.latestPubDate).toISOString()}
                    className="font-mono text-xs whitespace-nowrap text-neutral-600"
                  >
                    {new Date(entry.latestPubDate).toLocaleDateString('en-us', {
                      year: 'numeric',
                      month: 'short',
                      day: 'numeric',
                    })}
                  </time>
                </div>
                <h2 className="mt-2 font-mono text-sm text-neutral-200 transition-colors group-hover:text-blue-400">
                  {entry.title}
                </h2>
                <p className="mt-1 text-sm text-neutral-500">{entry.subtitle}</p>
                <div className="mt-3 flex items-center gap-2">
                  <img
                    src={entry.authorImage}
                    alt={entry.author}
                    className="h-5 w-5 rounded-full border border-neutral-700"
                  />
                  <span className="font-mono text-xs text-neutral-600">{entry.author}</span>
                </div>
              </Link>
            ) : (
              <Link
                key={entry.slug}
                to="/blog/$slug"
                params={{ slug: entry.slug }}
                className="group block rounded-lg border border-neutral-800 bg-neutral-900/50 p-4 no-underline transition-all duration-300 hover:border-blue-500/30 hover:no-underline"
              >
                <div className="flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
                  <h2 className="font-mono text-sm text-neutral-200 transition-colors group-hover:text-blue-400">
                    {entry.title}
                  </h2>
                  <time
                    dateTime={new Date(entry.pubDate).toISOString()}
                    className="font-mono text-xs whitespace-nowrap text-neutral-600"
                  >
                    {new Date(entry.pubDate).toLocaleDateString('en-us', {
                      year: 'numeric',
                      month: 'short',
                      day: 'numeric',
                    })}
                  </time>
                </div>
                <p className="mt-2 line-clamp-2 text-sm text-neutral-500">{entry.description}</p>
                {entry.author && (
                  <div className="mt-3 flex items-center gap-2">
                    {entry.authorImage ? (
                      <img
                        src={entry.authorImage}
                        alt={entry.author}
                        className="h-5 w-5 rounded-full border border-neutral-700"
                      />
                    ) : (
                      <div className="flex h-5 w-5 items-center justify-center rounded-full border border-neutral-700 bg-neutral-800 font-mono text-[10px] text-blue-400">
                        {entry.author.charAt(0)}
                      </div>
                    )}
                    <span className="font-mono text-xs text-neutral-600">{entry.author}</span>
                  </div>
                )}
              </Link>
            )
          )}
        </div>
      </main>
    </>
  )
}
