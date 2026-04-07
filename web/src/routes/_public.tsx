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
            🦌 <span className="text-green-800">deer.sh</span>
          </Link>
          <div className="flex flex-wrap justify-center gap-4">
            <a href="/#services" className="transition-colors hover:text-neutral-400">
              Services
            </a>
            <a href="/#case-studies" className="transition-colors hover:text-neutral-400">
              Customers
            </a>
            <a href="/#contact" className="transition-colors hover:text-neutral-400">
              Contact
            </a>
            <Link to="/product" className="transition-colors hover:text-neutral-400">
              Product
            </Link>
            <Link to="/docs/quickstart" className="transition-colors hover:text-neutral-400">
              Docs
            </Link>
          </div>
          <span className="font-mono">&copy; {new Date().getFullYear()} Deer.sh</span>
        </div>
      </footer>
    </div>
  )
}
