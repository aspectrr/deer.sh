import { createFileRoute, Link, notFound } from '@tanstack/react-router'
import { YouTube } from '~/components/youtube'

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
  for (const [path, mod] of Object.entries(modules)) {
    if (getSlug(path) === slug) {
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

  return (
    <main className="px-6 py-24">
      <article className="mx-auto max-w-3xl">
        <header className="mb-12">
          <Link
            to="/blog"
            className="font-mono text-sm text-neutral-500 transition-colors hover:text-blue-400"
          >
            <span className="text-blue-400">$</span> cd ..
          </Link>

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
          <Content components={{ YouTube }} />
        </div>
      </article>
    </main>
  )
}
