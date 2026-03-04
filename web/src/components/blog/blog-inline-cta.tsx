import { Link } from '@tanstack/react-router'
import { ArrowRight } from 'lucide-react'

export function BlogInlineCTA({ feature }: { feature: string }) {
  return (
    <aside className="my-8 border-l-2 border-blue-500 bg-neutral-900/50 py-3 pr-4 pl-4">
      <p className="font-mono text-sm text-neutral-300">{feature}</p>
      <Link
        to="/docs/quickstart"
        className="mt-3 inline-flex items-center gap-2 bg-blue-500 px-5 py-2 font-mono text-sm text-white transition-colors hover:bg-blue-600"
      >
        Get Started <ArrowRight className="h-4 w-4" />
      </Link>
    </aside>
  )
}
