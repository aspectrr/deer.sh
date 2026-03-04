import { Link } from '@tanstack/react-router'

export function BlogInlineCTA({ feature }: { feature: string }) {
  return (
    <aside className="my-8 border-l-2 border-blue-500 bg-neutral-900/50 py-3 pr-4 pl-4">
      <p className="font-mono text-sm text-neutral-300">{feature}</p>
      <Link
        to="/docs/quickstart"
        className="mt-2 inline-block font-mono text-sm text-blue-400 transition-colors hover:text-blue-300"
      >
        Try it on your own server -&gt;
      </Link>
    </aside>
  )
}
