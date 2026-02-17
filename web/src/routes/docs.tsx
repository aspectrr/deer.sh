import { createFileRoute, Outlet } from '@tanstack/react-router'
import { DocsSidebar } from '~/components/docs/docs-sidebar'
import { DocsHeader } from '~/components/docs/docs-header'

export const Route = createFileRoute('/docs')({
  component: DocsLayout,
})

function DocsLayout() {
  return (
    <div className="flex min-h-screen flex-col bg-black">
      <DocsHeader />
      <div className="flex flex-1">
        <DocsSidebar />
        <main className="flex-1 overflow-auto">
          <Outlet />
        </main>
      </div>
    </div>
  )
}
