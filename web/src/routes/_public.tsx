import { createFileRoute, Link, Outlet } from '@tanstack/react-router'

export const Route = createFileRoute('/_public')({
  component: PublicLayout,
})

function PublicLayout() {
  return (
    <div className="min-h-screen bg-black text-white">
      <Outlet />
      <footer className="border-t border-neutral-800 px-6 py-8">
        <div className="mx-auto flex max-w-2xl flex-col items-center gap-4 text-xs text-neutral-600 sm:flex-row sm:justify-between">
          <Link
            to="/"
            className="font-logo no-underline transition-colors hover:text-neutral-400 hover:no-underline"
          >
            <span className="text-blue-400">$</span> fluid.sh
          </Link>
          <div className="flex flex-wrap justify-center gap-4 font-mono">
            <Link to="/docs/quickstart" className="transition-colors hover:text-neutral-400">
              Docs
            </Link>
            <Link to="/blog" className="transition-colors hover:text-neutral-400">
              Blog
            </Link>
            <Link to="/pricing" className="transition-colors hover:text-neutral-400">
              Pricing
            </Link>
            <a
              href="https://github.com/aspectrr/fluid.sh"
              target="_blank"
              rel="noopener noreferrer"
              className="transition-colors hover:text-neutral-400"
            >
              GitHub
            </a>
            <a
              href="https://discord.gg/4WGGXJWm8J"
              target="_blank"
              rel="noopener noreferrer"
              className="transition-colors hover:text-neutral-400"
            >
              Discord
            </a>
          </div>
          <span className="font-mono">&copy; {new Date().getFullYear()} Fluid.sh</span>
        </div>
      </footer>
    </div>
  )
}
