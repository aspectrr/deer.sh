import { createRootRoute, Outlet } from '@tanstack/react-router'
import { Toaster } from 'sonner'
import { AuthProvider } from '~/lib/auth'

export const Route = createRootRoute({
  component: RootComponent,
})

function RootComponent() {
  return (
    <AuthProvider>
      <Outlet />
      <Toaster
        theme="dark"
        toastOptions={{
          style: {
            background: '#171717',
            border: '1px solid #262626',
            color: '#e5e5e5',
            fontFamily: "'JetBrains Mono Variable', monospace",
            fontSize: '0.75rem',
          },
        }}
      />
    </AuthProvider>
  )
}
