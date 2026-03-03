import { createFileRoute, Link, notFound } from '@tanstack/react-router'
import { YouTube } from '~/components/youtube'
import { BlogPreBlock } from '~/components/blog/blog-pre-block'
import { findSeriesForSlug } from '~/lib/blog-series'

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

const modules = import.meta.glob<BlogModule>('/src/content/blog/*.{md,mdx}', { eager: true })

function getSlug(path: string) {
  const filename = path.split('/').pop() || ''
  return filename.replace(/\.(md|mdx)$/, '')
}

function findPost(slug: string) {
  const now = new Date()
  for (const [path, mod] of Object.entries(modules)) {
    if (getSlug(path) === slug) {
      if (new Date(mod.frontmatter.pubDate) > now) return null
      return mod
    }
  }
  return null
}

export const Route = createFileRoute('/_public/blog/$slug')({
  component: BlogPost,
})

function BlogPost() {
  const { slug } = Route.useParams()
  const post = findPost(slug)

  if (!post) {
    throw notFound()
  }

  const { frontmatter, default: Content } = post
  const seriesInfo = findSeriesForSlug(slug)

  // Determine prev/next chapter availability
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
      // Check if published
      let published = false
      for (const [path, mod] of Object.entries(modules)) {
        if (getSlug(path) === next.slug && new Date(mod.frontmatter.pubDate) <= now) {
          published = true
        }
      }
      nextChapter = { ...next, published }
    }
  }

  return (
    <main className="px-4 py-24 sm:px-6">
      <article className="mx-auto max-w-3xl">
        <header className="mb-12">
          {seriesInfo ? (
            <Link
              to="/blog/series/hypervisor"
              className="font-mono text-sm text-neutral-500 transition-colors hover:text-blue-400"
            >
              <span className="text-blue-400">$</span> cd /series
            </Link>
          ) : (
            <Link
              to="/blog"
              className="font-mono text-sm text-neutral-500 transition-colors hover:text-blue-400"
            >
              <span className="text-blue-400">$</span> cd ..
            </Link>
          )}

          <div className="mt-8 flex items-center gap-3">
            <h1 className="font-logo text-2xl tracking-tight md:text-3xl">
              <span className="text-blue-400">#</span> {frontmatter.title}
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
                <div className="flex h-12 w-12 items-center justify-center rounded-full border border-neutral-700 bg-neutral-800 font-mono text-lg text-blue-400">
                  {frontmatter.author.charAt(0)}
                </div>
              )}
              <div>
                <div className="font-medium text-neutral-200">{frontmatter.author}</div>
                <div className="mt-1 flex gap-3">
                  {frontmatter.authorEmail && (
                    <a
                      href={`mailto:${frontmatter.authorEmail}`}
                      className="font-mono text-xs text-neutral-500 transition-colors hover:text-blue-400"
                    >
                      email
                    </a>
                  )}
                  {frontmatter.authorPhone && (
                    <a
                      href={`tel:${frontmatter.authorPhone}`}
                      className="font-mono text-xs text-neutral-500 transition-colors hover:text-blue-400"
                    >
                      text
                    </a>
                  )}
                  {frontmatter.authorDiscord && (
                    <a
                      href={frontmatter.authorDiscord}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="font-mono text-xs text-neutral-500 transition-colors hover:text-blue-400"
                    >
                      discord
                    </a>
                  )}
                </div>
              </div>
            </div>
          )}
        </header>

        <div className="blog-prose max-w-none">
          <Content components={{ YouTube, pre: BlogPreBlock }} />
        </div>

        {seriesInfo && (
          <nav className="mt-16 border-t border-neutral-800 pt-8">
            <Link
              to="/blog/series/hypervisor"
              className="font-mono text-xs text-neutral-500 transition-colors hover:text-blue-400"
            >
              {seriesInfo.series.title}
            </Link>
            <div className="mt-4 grid grid-cols-2 gap-3">
              {prevChapter ? (
                <Link
                  to="/blog/$slug"
                  params={{ slug: prevChapter.slug }}
                  className="group rounded-lg border border-neutral-800 p-3 no-underline transition-colors hover:border-blue-500/30 hover:no-underline"
                >
                  <span className="font-mono text-[10px] text-neutral-600 uppercase">Previous</span>
                  <div className="mt-1 font-mono text-sm text-neutral-400 transition-colors group-hover:text-blue-400">
                    <span className="text-blue-400">
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
                    className="group rounded-lg border border-neutral-800 p-3 text-right no-underline transition-colors hover:border-blue-500/30 hover:no-underline"
                  >
                    <span className="font-mono text-[10px] text-neutral-600 uppercase">Next</span>
                    <div className="mt-1 font-mono text-sm text-neutral-400 transition-colors group-hover:text-blue-400">
                      {nextChapter.shortTitle}{' '}
                      <span className="text-blue-400">
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
  )
}
