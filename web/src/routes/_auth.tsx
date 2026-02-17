import { createFileRoute, Outlet, Navigate } from '@tanstack/react-router'
import { useAuth } from '~/lib/auth'

export const Route = createFileRoute('/_auth')({
  component: AuthLayout,
})

function AuthLayout() {
  const { isAuthenticated, isLoading } = useAuth()

  if (isLoading) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-black">
        <div className="text-muted-foreground text-sm">Loading...</div>
      </div>
    )
  }

  if (isAuthenticated) {
    return <Navigate to="/dashboard" />
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-black">
      <div className="w-full max-w-sm">
        <div className="mb-8 text-center">
          <h1 className="font-logo text-3xl font-extrabold text-white">fluid.sh</h1>
          <p className="text-muted-foreground mt-2 text-xs">
            Autonomous AI agents for infrastructure
          </p>
        </div>
        <div className="border-border bg-card border p-6">
          <Outlet />
        </div>
      </div>
    </div>
  )
}
