import { createFileRoute, Link, notFound } from '@tanstack/react-router'
import { useState, useCallback, useRef } from 'react'
import { Menu, X } from 'lucide-react'
import { YouTube } from '~/components/youtube'
import { BlogPreBlock } from '~/components/blog/blog-pre-block'
import { BlogCTA } from '~/components/blog/blog-cta'
import { BlogInlineCTA } from '~/components/blog/blog-inline-cta'
import { findSeriesForSlug } from '~/lib/blog-series'
import { useAuth } from '~/lib/auth'
import { useReturningVisitor } from '~/lib/use-returning-visitor'

interface BlogFrontmatter {
  title: string
  pubDate: string
  description: string
  author?: string
  authorImage?: string
  authorEmail?: string
  authorPhone?: string
  authorDiscord?: string
}

interface BlogModule {
  frontmatter: BlogFrontmatter
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  default: React.ComponentType<{ components?: Record<string, React.ComponentType<any>> }>
}

const modules = import.meta.glob<BlogModule>('/src/content/blog/**/*.{md,mdx}', { eager: true })

function getSlug(path: string) {
  const filename = path.split('/').pop() || ''
  return filename.replace(/\.(md|mdx)$/, '')
}

function isConsultingPost(path: string): boolean {
  return path.includes('/consulting/')
}

function findPost(slug: string): { mod: BlogModule; path: string } | null {
  const now = new Date()
  for (const [path, mod] of Object.entries(modules)) {
    if (getSlug(path) === slug) {
      if (new Date(mod.frontmatter.pubDate) > now) return null
      return { mod, path }
    }
  }
  return null
}

function isSafeUrl(url: string): boolean {
  try {
    const parsed = new URL(url, window.location.origin)
    return parsed.protocol === 'https:' || parsed.protocol === 'http:'
  } catch {
    return false
  }
}

function ShareButton({ title, consulting }: { title: string; consulting: boolean }) {
  const [label, setLabel] = useState('share')
  const timeoutRef = useRef<ReturnType<typeof setTimeout>>(null)

  const handleShare = useCallback(async () => {
    const url = window.location.href
    try {
      await navigator.share({ title, url })
    } catch {
      await navigator.clipboard.writeText(url)
      if (timeoutRef.current) clearTimeout(timeoutRef.current)
      setLabel('copied')
      timeoutRef.current = setTimeout(() => setLabel('share'), 2000)
    }
  }, [title])

  return (
    <button
      onClick={handleShare}
      className={`text-xs transition-colors ${
        consulting
          ? 'text-stone-400 hover:text-green-700'
          : 'font-mono text-neutral-500 hover:text-green-800'
      }`}
    >
      {label}
    </button>
  )
}

export const Route = createFileRoute('/_public/blog/$slug')({
  component: BlogPost,
})

function BlogPost() {
  const { isAuthenticated } = useAuth()
  const isReturning = useReturningVisitor()
  const { slug } = Route.useParams()
  const result = findPost(slug)
  const [mobileOpen, setMobileOpen] = useState(false)

  if (!result) {
    throw notFound()
  }

  const { mod: post, path: postPath } = result
  const consulting = isConsultingPost(postPath)
  const { frontmatter, default: Content } = post
  const seriesInfo = findSeriesForSlug(slug)

  const now = new Date()
  let prevChapter: { slug: string; part: number; shortTitle: string } | null = null
  let nextChapter: { slug: string; part: number; shortTitle: string; published: boolean } | null =
    null
  if (seriesInfo) {
    const { series, chapterIndex } = seriesInfo
    if (chapterIndex > 0) {
      prevChapter = series.chapters[chapterIndex - 1]
    }
    if (chapterIndex < series.chapters.length - 1) {
      const next = series.chapters[chapterIndex + 1]
      let published = false
      for (const [path, mod] of Object.entries(modules)) {
        if (getSlug(path) === next.slug && new Date(mod.frontmatter.pubDate) <= now) {
          published = true
        }
      }
      nextChapter = { ...next, published }
    }
  }

  if (consulting) {
    return (
      <div className="font-inter min-h-screen bg-stone-50">
        {/* Nav */}
        <header className="px-4 pt-8 pb-0 sm:px-6">
          <div className="mx-auto max-w-3xl">
            <div className="mb-0 flex items-center justify-between">
              <Link
                to="/"
                className="font-logo text-2xl tracking-tight text-stone-900 no-underline hover:no-underline md:text-3xl"
              >
                🦌 <span className="text-green-700">deer.sh</span>
              </Link>
              <div className="hidden items-center gap-6 text-sm text-stone-500 md:flex">
                <Link to="/" className="transition-colors hover:text-stone-800">
                  Home
                </Link>
                <a href="/#services" className="transition-colors hover:text-stone-800">
                  Services
                </a>
                <Link to="/product" className="transition-colors hover:text-stone-800">
                  Product
                </Link>
                <Link to="/blog" className="font-medium text-stone-800">
                  Blog
                </Link>
                <a
                  href="/#contact"
                  className="inline-flex items-center gap-1 rounded-full border border-green-900/40 bg-green-900/10 px-4 py-1.5 text-green-700 transition-colors hover:border-green-900/60 hover:bg-green-900/20"
                >
                  Get in Touch
                </a>
              </div>
              <button
                className="text-stone-500 hover:text-stone-800 md:hidden"
                onClick={() => setMobileOpen(!mobileOpen)}
              >
                {mobileOpen ? <X className="h-5 w-5" /> : <Menu className="h-5 w-5" />}
              </button>
            </div>

            {mobileOpen && (
              <div
                className="fixed inset-0 z-30 bg-stone-900 md:hidden"
                onClick={() => setMobileOpen(false)}
              >
                <nav
                  className="flex flex-col gap-6 p-8 pt-20 text-lg text-stone-300"
                  onClick={(e) => e.stopPropagation()}
                >
                  <Link
                    to="/"
                    onClick={() => setMobileOpen(false)}
                    className="transition-colors hover:text-white"
                  >
                    Home
                  </Link>
                  <Link
                    to="/blog"
                    onClick={() => setMobileOpen(false)}
                    className="transition-colors hover:text-white"
                  >
                    Blog
                  </Link>
                  <a
                    href="/#contact"
                    onClick={() => setMobileOpen(false)}
                    className="transition-colors hover:text-white"
                  >
                    Get in Touch
                  </a>
                </nav>
              </div>
            )}
          </div>
        </header>

        {/* Post */}
        <main className="px-4 pt-12 pb-24 sm:px-6">
          <article className="mx-auto max-w-3xl">
            <header className="mb-12">
              <h1 className="font-logo text-3xl font-bold tracking-tight text-stone-900 md:text-4xl">
                {frontmatter.title}
              </h1>

              <div className="mt-3 text-sm text-stone-400">
                {new Date(frontmatter.pubDate).toLocaleDateString('en-us', {
                  year: 'numeric',
                  month: 'long',
                  day: 'numeric',
                })}
              </div>

              {frontmatter.author && (
                <div className="mt-6 flex items-center gap-4">
                  {frontmatter.authorImage ? (
                    <img
                      src={frontmatter.authorImage}
                      alt={frontmatter.author}
                      className="h-12 w-12 rounded-full border border-stone-200 object-cover"
                    />
                  ) : (
                    <div className="flex h-12 w-12 items-center justify-center rounded-full border border-stone-200 bg-stone-100 text-lg text-green-700">
                      {frontmatter.author.charAt(0)}
                    </div>
                  )}
                  <div>
                    <div className="font-medium text-stone-800">{frontmatter.author}</div>
                    <div className="mt-1 flex gap-3">
                      {frontmatter.authorEmail && (
                        <a
                          href={`mailto:${frontmatter.authorEmail}`}
                          className="text-xs text-stone-400 transition-colors hover:text-green-700"
                        >
                          email
                        </a>
                      )}
                      {frontmatter.authorPhone && (
                        <a
                          href={`tel:${frontmatter.authorPhone}`}
                          className="text-xs text-stone-400 transition-colors hover:text-green-700"
                        >
                          text
                        </a>
                      )}
                      {frontmatter.authorDiscord && isSafeUrl(frontmatter.authorDiscord) && (
                        <a
                          href={frontmatter.authorDiscord}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="text-xs text-stone-400 transition-colors hover:text-green-700"
                        >
                          discord
                        </a>
                      )}
                      <ShareButton title={frontmatter.title} consulting />
                    </div>
                  </div>
                </div>
              )}
            </header>

            <div className="consulting-blog-prose max-w-none">
              <Content components={{ YouTube, BlogInlineCTA, pre: BlogPreBlock }} />
            </div>

            {seriesInfo && (
              <nav className="mt-16 border-t border-stone-200 pt-8">
                <Link
                  to="/blog/series/$seriesId"
                  params={{ seriesId: seriesInfo.series.id }}
                  className="text-sm text-stone-400 transition-colors hover:text-green-700"
                >
                  {seriesInfo.series.title}
                </Link>
                <div className="mt-4 grid grid-cols-2 gap-3">
                  {prevChapter ? (
                    <Link
                      to="/blog/$slug"
                      params={{ slug: prevChapter.slug }}
                      className="group rounded-2xl border border-stone-200 bg-white p-4 no-underline transition-all duration-200 hover:-translate-y-0.5 hover:border-green-900/30 hover:no-underline"
                    >
                      <span className="text-xs text-stone-400 uppercase">Previous</span>
                      <div className="mt-1 text-sm text-stone-600 transition-colors group-hover:text-green-700">
                        <span className="text-green-700">
                          {String(prevChapter.part).padStart(2, '0')}
                        </span>{' '}
                        {prevChapter.shortTitle}
                      </div>
                    </Link>
                  ) : (
                    <div />
                  )}
                  {nextChapter ? (
                    nextChapter.published ? (
                      <Link
                        to="/blog/$slug"
                        params={{ slug: nextChapter.slug }}
                        className="group rounded-2xl border border-stone-200 bg-white p-4 text-right no-underline transition-all duration-200 hover:-translate-y-0.5 hover:border-green-900/30 hover:no-underline"
                      >
                        <span className="text-xs text-stone-400 uppercase">Next</span>
                        <div className="mt-1 text-sm text-stone-600 transition-colors group-hover:text-green-700">
                          {nextChapter.shortTitle}{' '}
                          <span className="text-green-700">
                            {String(nextChapter.part).padStart(2, '0')}
                          </span>
                        </div>
                      </Link>
                    ) : (
                      <div className="rounded-2xl border border-stone-200/50 bg-white/50 p-4 text-right opacity-50">
                        <span className="text-xs text-stone-400 uppercase">Next</span>
                        <div className="mt-1 text-sm text-stone-400">
                          {nextChapter.shortTitle} - Coming soon
                        </div>
                      </div>
                    )
                  ) : (
                    <div />
                  )}
                </div>
              </nav>
            )}
          </article>
        </main>
      </div>
    )
  }

  // Product post (dark theme)
  return (
    <>
      <header className="px-4 pt-24 pb-12 sm:px-6">
        <div className="mx-auto max-w-3xl">
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
              <Link to="/product/blog" className="transition-colors hover:text-neutral-200">
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
                  to="/product/blog"
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
                  href="https://github.com/aspectrr/deer.sh"
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
        <article className="mx-auto max-w-3xl">
          <header className="mb-12">
            <div className="flex items-center gap-3">
              <h1 className="font-logo text-2xl tracking-tight md:text-3xl">
                <span className="text-green-800">#</span> {frontmatter.title}
              </h1>
            </div>

            <div className="mt-2 font-mono text-sm text-neutral-500">
              {new Date(frontmatter.pubDate).toLocaleDateString('en-us', {
                year: 'numeric',
                month: 'long',
                day: 'numeric',
              })}
            </div>

            {frontmatter.author && (
              <div className="mt-6 flex items-center gap-4">
                {frontmatter.authorImage ? (
                  <img
                    src={frontmatter.authorImage}
                    alt={frontmatter.author}
                    className="h-12 w-12 rounded-full border border-neutral-700 object-cover"
                  />
                ) : (
                  <div className="flex h-12 w-12 items-center justify-center rounded-full border border-neutral-700 bg-neutral-800 font-mono text-lg text-green-800">
                    {frontmatter.author.charAt(0)}
                  </div>
                )}
                <div>
                  <div className="font-medium text-neutral-200">{frontmatter.author}</div>
                  <div className="mt-1 flex gap-3">
                    {frontmatter.authorEmail && (
                      <a
                        href={`mailto:${frontmatter.authorEmail}`}
                        className="font-mono text-xs text-neutral-500 transition-colors hover:text-green-800"
                      >
                        email
                      </a>
                    )}
                    {frontmatter.authorPhone && (
                      <a
                        href={`tel:${frontmatter.authorPhone}`}
                        className="font-mono text-xs text-neutral-500 transition-colors hover:text-green-800"
                      >
                        text
                      </a>
                    )}
                    {frontmatter.authorDiscord && isSafeUrl(frontmatter.authorDiscord) && (
                      <a
                        href={frontmatter.authorDiscord}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="font-mono text-xs text-neutral-500 transition-colors hover:text-green-800"
                      >
                        discord
                      </a>
                    )}
                    <ShareButton title={frontmatter.title} consulting={false} />
                  </div>
                </div>
              </div>
            )}
          </header>

          <div className="blog-prose max-w-none">
            <Content components={{ YouTube, BlogInlineCTA, pre: BlogPreBlock }} />
          </div>

          <BlogCTA />

          {seriesInfo && (
            <nav className="mt-16 border-t border-neutral-800 pt-8">
              <Link
                to="/blog/series/$seriesId"
                params={{ seriesId: seriesInfo.series.id }}
                className="font-mono text-xs text-neutral-500 transition-colors hover:text-green-800"
              >
                {seriesInfo.series.title}
              </Link>
              <div className="mt-4 grid grid-cols-2 gap-3">
                {prevChapter ? (
                  <Link
                    to="/blog/$slug"
                    params={{ slug: prevChapter.slug }}
                    className="group rounded-lg border border-neutral-800 p-3 no-underline transition-colors hover:border-green-900/30 hover:no-underline"
                  >
                    <span className="font-mono text-[10px] text-neutral-600 uppercase">
                      Previous
                    </span>
                    <div className="mt-1 font-mono text-sm text-neutral-400 transition-colors group-hover:text-green-800">
                      <span className="text-green-800">
                        {String(prevChapter.part).padStart(2, '0')}
                      </span>{' '}
                      {prevChapter.shortTitle}
                    </div>
                  </Link>
                ) : (
                  <div />
                )}
                {nextChapter ? (
                  nextChapter.published ? (
                    <Link
                      to="/blog/$slug"
                      params={{ slug: nextChapter.slug }}
                      className="group rounded-lg border border-neutral-800 p-3 text-right no-underline transition-colors hover:border-green-900/30 hover:no-underline"
                    >
                      <span className="font-mono text-[10px] text-neutral-600 uppercase">Next</span>
                      <div className="mt-1 font-mono text-sm text-neutral-400 transition-colors group-hover:text-green-800">
                        {nextChapter.shortTitle}{' '}
                        <span className="text-green-800">
                          {String(nextChapter.part).padStart(2, '0')}
                        </span>
                      </div>
                    </Link>
                  ) : (
                    <div className="rounded-lg border border-neutral-800/50 p-3 text-right opacity-50">
                      <span className="font-mono text-[10px] text-neutral-600 uppercase">Next</span>
                      <div className="mt-1 font-mono text-sm text-neutral-600">
                        {nextChapter.shortTitle} - Coming soon
                      </div>
                    </div>
                  )
                ) : (
                  <div />
                )}
              </div>
            </nav>
          )}
        </article>
      </main>
    </>
  )
}
