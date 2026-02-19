import { createFileRoute, Outlet, Navigate, Link, useLocation } from '@tanstack/react-router'
import { useAuth } from '~/lib/auth'
import { OrgProvider, useOrg } from '~/lib/org'
import {
  LayoutDashboard,
  CreditCard,
  Settings,
  LogOut,
  ChevronDown,
  // Bot,
  // BookOpen,
} from 'lucide-react'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '~/components/ui/dropdown-menu'

export const Route = createFileRoute('/_app')({
  component: AppLayout,
})

const navItems = [
  { to: '/dashboard' as const, label: 'Dashboard', icon: LayoutDashboard },
  // { to: '/agents' as const, label: 'Agents', icon: Bot },
  // { to: '/playbooks' as const, label: 'Playbooks', icon: BookOpen },
  { to: '/billing' as const, label: 'Billing', icon: CreditCard },
  { to: '/settings' as const, label: 'Settings', icon: Settings },
]

function AppLayout() {
  const { user, isAuthenticated, isLoading, logout } = useAuth()

  if (isLoading) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-black">
        <div className="text-muted-foreground text-sm">Loading...</div>
      </div>
    )
  }

  if (!isAuthenticated) {
    return <Navigate to="/login" />
  }

  return (
    <OrgProvider>
      <AppShell user={user} logout={logout} />
    </OrgProvider>
  )
}

function AppShell({
  user,
  logout,
}: {
  user: { display_name: string; email: string } | null
  logout: () => Promise<void>
}) {
  const { organizations, isLoading: orgsLoading } = useOrg()
  const location = useLocation()

  if (orgsLoading) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-black">
        <div className="text-muted-foreground text-sm">Loading...</div>
      </div>
    )
  }

  if (organizations.length === 0) {
    return <Navigate to="/onboarding" />
  }

  return (
    <div className="flex min-h-screen bg-black">
      {/* Sidebar */}
      <aside className="border-border flex w-56 flex-col border-r">
        <div className="border-border flex items-center gap-2 border-b p-4">
          <Link to="/dashboard">
            <h1 className="font-logo text-lg font-extrabold text-white">fluid.sh</h1>
          </Link>
          <span className="text-muted-foreground text-sm">/</span>
          <Link
            to="/docs/quickstart"
            className="text-muted-foreground text-sm transition-colors hover:text-white"
          >
            docs
          </Link>
        </div>

        <nav className="flex-1 p-2">
          {navItems.map((item) => {
            const isActive = location.pathname.startsWith(item.to)
            return (
              <Link
                key={item.to}
                to={item.to}
                className={`flex items-center gap-2 px-3 py-2 text-xs transition-colors ${
                  isActive
                    ? 'bg-blue-500/10 text-blue-400'
                    : 'text-muted-foreground hover:text-foreground hover:bg-neutral-900'
                }`}
              >
                <item.icon className="h-3.5 w-3.5" />
                {item.label}
              </Link>
            )
          })}
        </nav>

        {/* User menu */}
        <div className="border-border border-t p-2">
          <DropdownMenu>
            <DropdownMenuTrigger className="flex w-full items-center gap-2 px-3 py-2 text-xs text-white hover:bg-neutral-900">
              <div className="bg-secondary flex h-6 w-6 items-center justify-center text-[10px] text-white">
                {user?.display_name?.charAt(0).toUpperCase() || '?'}
              </div>
              <span className="flex-1 truncate text-left">{user?.display_name}</span>
              <ChevronDown className="h-3 w-3" />
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="w-48">
              <div className="px-2 py-1.5">
                <p className="text-xs font-medium">{user?.display_name}</p>
                <p className="text-muted-foreground text-[10px]">{user?.email}</p>
              </div>
              <DropdownMenuSeparator />
              <DropdownMenuItem onClick={() => logout()} variant="destructive">
                <LogOut className="h-3.5 w-3.5" />
                Sign out
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </aside>

      {/* Main content */}
      <main className="flex-1 overflow-auto p-6">
        <Outlet />
      </main>
    </div>
  )
}
