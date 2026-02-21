import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { useQuery, useMutation } from '@tanstack/react-query'
import { axios } from '~/lib/axios'
import { useOrg } from '~/lib/org'
import { Button } from '~/components/ui/button'
import { Zap, CreditCard, Building2, Calculator } from 'lucide-react'
export const Route = createFileRoute('/_app/billing/')({
  component: BillingPage,
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

function BillingPage() {
  const navigate = useNavigate()
  const { org } = useOrg()
  const orgSlug = org?.slug ?? ''

  const {
    data: billing,
    isLoading,
    isError,
    error,
  } = useQuery({
    queryKey: ['billing', orgSlug],
    queryFn: async () => {
      const res = await axios.get(`/v1/orgs/${encodeURIComponent(orgSlug)}/billing`)
      return res.data as BillingData
    },
    enabled: !!orgSlug,
  })

  const subscribe = useMutation({
    mutationFn: async () => {
      const res = await axios.post(`/v1/orgs/${encodeURIComponent(orgSlug)}/billing/subscribe`)
      return res.data as { checkout_url?: string; status?: string }
    },
    onSuccess: (data) => {
      if (data.checkout_url) {
        window.location.href = data.checkout_url
      } else {
        throw new Error('No checkout URL returned')
      }
    },
  })

  const portal = useMutation({
    mutationFn: async () => {
      const res = await axios.post(`/v1/orgs/${encodeURIComponent(orgSlug)}/billing/portal`)
      return res.data as { portal_url?: string; status?: string }
    },
    onSuccess: (data) => {
      if (data.portal_url) {
        window.location.href = data.portal_url
      }
    },
  })

  if (!org) {
    return <p className="text-muted-foreground text-xs">No organization selected.</p>
  }

  const plan = billing?.plan || 'free'
  const freeTier = billing?.free_tier
  const usage = billing?.usage

  if (isLoading) {
    return <p className="text-muted-foreground text-xs">Loading billing data...</p>
  }

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-sm font-medium text-white">Billing</h2>
        <p className="text-muted-foreground text-xs">Manage your plan and usage</p>
      </div>

      {isError && (
        <div className="border border-red-500/50 bg-red-950/30 p-4 text-xs text-red-400">
          Failed to load billing data: {error?.message || 'Unknown error'}
        </div>
      )}

      {subscribe.isError && (
        <div className="border border-red-500/50 bg-red-950/30 p-4 text-xs text-red-400">
          Failed to start checkout: {subscribe.error?.message || 'Unknown error'}
        </div>
      )}

      {portal.isError && (
        <div className="border border-red-500/50 bg-red-950/30 p-4 text-xs text-red-400">
          Failed to open billing portal: {portal.error?.message || 'Unknown error'}
        </div>
      )}

      {/* Current Plan */}
      <div className="border-border border bg-neutral-900/50 p-6">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <Zap className="h-5 w-5 text-purple-400" />
            <div>
              <h3 className="text-sm font-medium text-white">
                {plan === 'free' ? 'Free Plan' : 'Usage-Based Plan'}
              </h3>
              <p className="text-muted-foreground text-xs">
                {plan === 'free'
                  ? '1 sandbox, 3 source VMs, 1 fluid daemon'
                  : 'Pay only for what you use'}
              </p>
            </div>
          </div>
          {plan === 'free' && (
            <Button
              className="bg-blue-500 text-xs text-black hover:bg-blue-400"
              onClick={() => subscribe.mutate()}
              disabled={subscribe.isPending}
            >
              {subscribe.isPending ? 'Redirecting...' : 'Upgrade to Usage-Based'}
            </Button>
          )}
        </div>
      </div>

      {/* Usage meters */}
      <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-4">
        <UsageMeter
          label="Max Concurrent Sandboxes"
          current={usage?.max_concurrent_sandboxes ?? 0}
          limit={freeTier?.max_concurrent_sandboxes}
          color="blue"
        />
        <UsageMeter
          label="Source VMs"
          current={usage?.source_vms ?? 0}
          limit={freeTier?.max_source_vms}
          color="green"
        />
        <UsageMeter
          label="Fluid Daemons"
          current={usage?.agent_hosts ?? 0}
          limit={freeTier?.max_agent_hosts}
          color="amber"
        />
      </div>

      {/* Enterprise */}
      <div className="border-border border bg-neutral-900/50 p-6">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <Building2 className="h-5 w-5 text-amber-400" />
            <div>
              <h3 className="text-sm font-medium text-white">Enterprise</h3>
              <p className="text-muted-foreground text-xs">Custom limits, dedicated support, SLA</p>
            </div>
          </div>
          <Button variant="outline" className="text-xs" asChild>
            <a href="mailto:cpfeifer@madcactus.org">Contact Sales</a>
          </Button>
        </div>
      </div>

      {/* Cost Calculator */}
      <div className="border-border border bg-neutral-900/50 p-6">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <Calculator className="h-5 w-5 text-green-400" />
            <div>
              <h3 className="text-xs font-medium text-white">Cost Calculator</h3>
              <p className="text-muted-foreground text-[10px]">
                Estimate your monthly costs on the usage-based plan
              </p>
            </div>
          </div>
          <Button
            variant="outline"
            className="text-xs"
            onClick={() => navigate({ to: '/billing/calculator' })}
          >
            Open Calculator
          </Button>
        </div>
      </div>

      {/* Manage billing */}
      <div className="border-border border bg-neutral-900/50 p-6">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <CreditCard className="h-5 w-5 text-blue-400" />
            <div>
              <h3 className="text-xs font-medium text-white">Payment Method</h3>
              <p className="text-muted-foreground text-[10px]">
                {plan === 'free' ? 'No payment method on file' : 'Managed via Stripe'}
              </p>
            </div>
          </div>
          <Button
            variant="outline"
            className="text-xs"
            onClick={() => portal.mutate()}
            disabled={portal.isPending || plan === 'free'}
          >
            {portal.isPending ? 'Redirecting...' : 'Manage Billing'}
          </Button>
        </div>
      </div>
    </div>
  )
}

function UsageMeter({
  label,
  current,
  limit,
  color,
}: {
  label: string
  current: number
  limit?: number
  color: string
}) {
  const hasLimit = limit != null && limit > 0
  const pct = hasLimit ? Math.min((current / limit) * 100, 100) : 0
  const colorMap: Record<string, string> = {
    blue: 'bg-blue-400',
    green: 'bg-green-400',
    amber: 'bg-amber-400',
    purple: 'bg-purple-400',
  }

  const formatNumber = (n: number) => {
    if (n >= 1000) return `${(n / 1000).toFixed(n >= 10000 ? 0 : 1)}k`
    return n % 1 === 0 ? n.toString() : n.toFixed(1)
  }

  return (
    <div className="border-border border bg-neutral-900/50 p-4">
      <div className="flex items-center justify-between text-xs">
        <span className="text-muted-foreground">{label}</span>
        <span className="text-white">
          {formatNumber(current)}
          {hasLimit ? ` / ${formatNumber(limit)}` : ''}
        </span>
      </div>
      <div className="mt-2 h-1.5 w-full bg-neutral-800">
        <div
          className={`h-full ${colorMap[color] || 'bg-blue-400'} transition-all`}
          style={{ width: hasLimit ? `${pct}%` : '0%' }}
        />
      </div>
    </div>
  )
}
