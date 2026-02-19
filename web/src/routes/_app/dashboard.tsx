import { createFileRoute } from '@tanstack/react-router'
import { useAuth } from '~/lib/auth'
import { Box, Server, Monitor, Zap } from 'lucide-react'
import { StepTracker } from '~/components/docs/step-tracker'
import { quickstartSteps } from '~/components/docs/quickstart-steps'

export const Route = createFileRoute('/_app/dashboard')({
  component: DashboardPage,
})

const stats = [
  {
    label: 'Active Sandboxes',
    value: '0 / 1',
    icon: Box,
    color: 'text-blue-400',
    borderColor: 'border-blue-500/30',
  },
  {
    label: 'Source VMs',
    value: '0 / 3',
    icon: Server,
    color: 'text-green-400',
    borderColor: 'border-green-500/30',
  },
  {
    label: 'Agent Hosts',
    value: '0 / 1',
    icon: Monitor,
    color: 'text-amber-400',
    borderColor: 'border-amber-500/30',
  },
  {
    label: 'Current Plan',
    value: 'Free',
    icon: Zap,
    color: 'text-purple-400',
    borderColor: 'border-purple-500/30',
  },
]

function DashboardPage() {
  const { user } = useAuth()

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-sm font-medium text-white">Dashboard</h2>
        <p className="text-muted-foreground text-xs">Welcome back, {user?.display_name}</p>
      </div>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        <div className="border-border border bg-neutral-900/50 p-6">
          <h3 className="mb-4 text-xs font-medium text-white">Getting Started</h3>
          <StepTracker
            storageKey="dashboard-quickstart"
            steps={quickstartSteps}
            progressEndpoint="/v1/docs-progress"
          />
        </div>

        <div className="space-y-6">
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            {stats.map((stat) => (
              <div
                key={stat.label}
                className={`border-border hover:${stat.borderColor} border bg-neutral-900/50 p-4 transition-all duration-300`}
              >
                <div className="flex items-center justify-between">
                  <span className="text-muted-foreground text-xs">{stat.label}</span>
                  <stat.icon className={`h-4 w-4 ${stat.color}`} />
                </div>
                <div className="mt-2 text-lg font-medium text-white">{stat.value}</div>
              </div>
            ))}
          </div>

          <div className="border-border border bg-neutral-900/50 p-6">
            <h3 className="text-xs font-medium text-white">Recent Activity</h3>
            <div className="text-muted-foreground mt-4 flex items-center justify-center py-8 text-xs">
              <div className="text-center">
                <p>No recent activity</p>
                <p className="mt-1 text-[10px]">Create a sandbox to get started</p>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
