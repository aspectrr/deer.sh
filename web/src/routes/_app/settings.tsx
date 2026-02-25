import { createFileRoute, Outlet, Link, useLocation } from '@tanstack/react-router'
import { cn } from '~/lib/utils'

export const Route = createFileRoute('/_app/settings')({
  component: SettingsLayout,
})

const settingsTabs = [
  { to: '/settings' as const, label: 'Profile', exact: true },
  { to: '/settings/organization' as const, label: 'Organization' },
  { to: '/settings/members' as const, label: 'Members' },
  { to: '/settings/hosts' as const, label: 'Hosts' },
]

function SettingsLayout() {
  const location = useLocation()

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-sm font-medium text-white">Settings</h1>
      </div>

      <div className="flex gap-1 border-b border-neutral-800">
        {settingsTabs.map((tab) => {
          const isActive = tab.exact
            ? location.pathname === tab.to || location.pathname === tab.to + '/'
            : location.pathname.startsWith(tab.to)
          return (
            <Link
              key={tab.to}
              to={tab.to}
              className={cn(
                '-mb-px border-b-2 px-3 py-2 text-xs transition-colors',
                isActive
                  ? 'border-blue-400 text-blue-400'
                  : 'border-transparent text-neutral-500 hover:text-neutral-300'
              )}
            >
              {tab.label}
            </Link>
          )
        })}
      </div>

      <Outlet />
    </div>
  )
}
