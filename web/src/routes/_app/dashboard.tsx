import { createFileRoute } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { useAuth } from '~/lib/auth'
import { useOrg } from '~/lib/org'
import { axios } from '~/lib/axios'
import { Box, Server, Monitor, Zap } from 'lucide-react'
import { StepTracker } from '~/components/docs/step-tracker'
import { quickstartSteps } from '~/components/docs/quickstart-steps'

export const Route = createFileRoute('/_app/dashboard')({
  component: DashboardPage,
})

interface FreeTier {
  max_concurrent_sandboxes: number
  max_source_vms: number
  max_agent_hosts: number
}

interface UsageSummary {
  max_concurrent_sandboxes: number
  source_vms: number
  agent_hosts: number
}

interface BillingData {
  plan: string
  status: string
  free_tier?: FreeTier
  usage?: UsageSummary
}

function DashboardPage() {
  const { user } = useAuth()
  const { org } = useOrg()
  const orgSlug = org?.slug ?? ''

  const { data: billing, isLoading } = useQuery({
    queryKey: ['billing', orgSlug],
    queryFn: async () => {
      const res = await axios.get(`/v1/orgs/${encodeURIComponent(orgSlug)}/billing`)
      return res.data as BillingData
    },
    enabled: !!orgSlug,
  })

  const plan = billing?.plan || 'free'
  const freeTier = billing?.free_tier
  const usage = billing?.usage
  const loading = isLoading || !orgSlug

  const stats = loading
    ? [
        {
          label: 'Active Sandboxes',
          value: '--',
          icon: Box,
          color: 'text-blue-400',
          borderColor: 'border-blue-500/30',
        },
        {
          label: 'Source VMs',
          value: '--',
          icon: Server,
          color: 'text-green-400',
          borderColor: 'border-green-500/30',
        },
        {
          label: 'Agent Hosts',
          value: '--',
          icon: Monitor,
          color: 'text-amber-400',
          borderColor: 'border-amber-500/30',
        },
        {
          label: 'Current Plan',
          value: '--',
          icon: Zap,
          color: 'text-purple-400',
          borderColor: 'border-purple-500/30',
        },
      ]
    : plan === 'free'
      ? [
          {
            label: 'Active Sandboxes',
            value: `${usage?.max_concurrent_sandboxes ?? 0} / ${freeTier?.max_concurrent_sandboxes ?? 1}`,
            icon: Box,
            color: 'text-blue-400',
            borderColor: 'border-blue-500/30',
          },
          {
            label: 'Source VMs',
            value: `${usage?.source_vms ?? 0} / ${freeTier?.max_source_vms ?? 3}`,
            icon: Server,
            color: 'text-green-400',
            borderColor: 'border-green-500/30',
          },
          {
            label: 'Agent Hosts',
            value: `${usage?.agent_hosts ?? 0} / ${freeTier?.max_agent_hosts ?? 1}`,
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
      : [
          {
            label: 'Active Sandboxes',
            value: `${usage?.max_concurrent_sandboxes ?? 0}`,
            icon: Box,
            color: 'text-blue-400',
            borderColor: 'border-blue-500/30',
          },
          {
            label: 'Source VMs',
            value: `${usage?.source_vms ?? 0}`,
            icon: Server,
            color: 'text-green-400',
            borderColor: 'border-green-500/30',
          },
          {
            label: 'Agent Hosts',
            value: `${usage?.agent_hosts ?? 0}`,
            icon: Monitor,
            color: 'text-amber-400',
            borderColor: 'border-amber-500/30',
          },
          {
            label: 'Current Plan',
            value: 'Usage-Based',
            icon: Zap,
            color: 'text-purple-400',
            borderColor: 'border-purple-500/30',
          },
        ]

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
