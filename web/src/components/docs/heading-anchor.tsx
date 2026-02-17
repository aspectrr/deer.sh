import { Hash } from 'lucide-react'

function slugify(text: string): string {
  return text
    .toLowerCase()
    .replace(/[^\w\s-]/g, '')
    .replace(/\s+/g, '-')
    .replace(/-+/g, '-')
    .trim()
}

export function H2({ children }: { children: string }) {
  const id = slugify(children)
  return (
    <h2
      id={id}
      className="group mt-8 mb-3 flex scroll-mt-20 items-center gap-1.5 text-sm font-medium text-white"
    >
      {children}
      <a
        href={`#${id}`}
        className="text-muted-foreground opacity-0 transition-opacity group-hover:opacity-100"
      >
        <Hash className="h-3 w-3" />
      </a>
    </h2>
  )
}

export function H3({ children }: { children: string }) {
  const id = slugify(children)
  return (
    <h3
      id={id}
      className="group mt-6 mb-2 flex scroll-mt-20 items-center gap-1.5 text-xs font-medium text-neutral-200"
    >
      {children}
      <a
        href={`#${id}`}
        className="text-muted-foreground opacity-0 transition-opacity group-hover:opacity-100"
      >
        <Hash className="h-2.5 w-2.5" />
      </a>
    </h3>
  )
}
