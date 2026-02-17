import { createFileRoute, Outlet } from '@tanstack/react-router'

export const Route = createFileRoute('/_public')({
  component: PublicLayout,
})

function PublicLayout() {
  return (
    <div className="min-h-screen bg-black text-white">
      <Outlet />
      <footer className="border-t border-neutral-800 px-6 py-8">
        <div className="mx-auto flex max-w-2xl items-center justify-between text-xs text-neutral-600">
          <span className="font-logo">fluid.sh</span>
          <div className="flex gap-4 font-mono">
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
        </div>
      </footer>
    </div>
  )
}
