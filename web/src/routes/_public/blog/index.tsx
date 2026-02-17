import { createFileRoute, Link } from '@tanstack/react-router'

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

function getPosts() {
  return Object.entries(modules)
    .map(([path, mod]) => ({
      slug: getSlug(path),
      ...mod.frontmatter,
    }))
    .sort((a, b) => new Date(b.pubDate).valueOf() - new Date(a.pubDate).valueOf())
}

export const Route = createFileRoute('/_public/blog/')({
  component: BlogIndex,
})

function BlogIndex() {
  const posts = getPosts()

  return (
    <>
      <header className="px-6 pt-24 pb-12">
        <div className="mx-auto max-w-2xl">
          <div className="mb-6 flex items-center justify-between">
            <Link
              to="/"
              className="font-logo text-2xl tracking-tight no-underline hover:no-underline md:text-3xl"
            >
              <span className="text-blue-400">$</span>
              <span className="text-white"> fluid.sh</span>
            </Link>
            <div className="flex gap-6 font-mono text-sm text-neutral-400">
              <Link to="/blog" className="text-neutral-200">
                Blog
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
            </div>
          </div>
        </div>
      </header>

      <main className="px-6 pb-24">
        <div className="mx-auto max-w-2xl space-y-3">
          {posts.map((post) => (
            <Link
              key={post.slug}
              to="/blog/$slug"
              params={{ slug: post.slug }}
              className="group block rounded-lg border border-neutral-800 bg-neutral-900/50 p-4 no-underline transition-all duration-300 hover:border-blue-500/30 hover:no-underline"
            >
              <div className="flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
                <h2 className="font-mono text-sm text-neutral-200 transition-colors group-hover:text-blue-400">
                  {post.title}
                </h2>
                <time
                  dateTime={new Date(post.pubDate).toISOString()}
                  className="font-mono text-xs whitespace-nowrap text-neutral-600"
                >
                  {new Date(post.pubDate).toLocaleDateString('en-us', {
                    year: 'numeric',
                    month: 'short',
                    day: 'numeric',
                  })}
                </time>
              </div>
              <p className="mt-2 line-clamp-2 text-sm text-neutral-500">{post.description}</p>
              {post.author && (
                <div className="mt-3 flex items-center gap-2">
                  {post.authorImage ? (
                    <img
                      src={post.authorImage}
                      alt={post.author}
                      className="h-5 w-5 rounded-full border border-neutral-700"
                    />
                  ) : (
                    <div className="flex h-5 w-5 items-center justify-center rounded-full border border-neutral-700 bg-neutral-800 font-mono text-[10px] text-blue-400">
                      {post.author.charAt(0)}
                    </div>
                  )}
                  <span className="font-mono text-xs text-neutral-600">{post.author}</span>
                </div>
              )}
            </Link>
          ))}
        </div>
      </main>
    </>
  )
}
