import { createFileRoute } from '@tanstack/react-router'
import { useQuery, useMutation } from '@tanstack/react-query'
import { axios } from '~/lib/axios'
import { Button } from '~/components/ui/button'
import { Zap, CreditCard } from 'lucide-react'

export const Route = createFileRoute('/_app/billing')({
  component: BillingPage,
})

// TODO: get org slug from context/route params
const orgSlug = 'default'

interface BillingData {
  plan: string
  status: string
  usage: {
    sandboxes: number
    source_vms: number
    sandbox_hosts: number
    tokens_used: number
  }
  limits: {
    sandboxes: number
    source_vms: number
    sandbox_hosts: number
  }
}

function BillingPage() {
  const { data: billing, isLoading } = useQuery({
    queryKey: ['billing', orgSlug],
    queryFn: async () => {
      const res = await axios.get(`/v1/orgs/${orgSlug}/billing`)
      return res.data as BillingData
    },
  })

  const subscribe = useMutation({
    mutationFn: async () => {
      const res = await axios.post(`/v1/orgs/${orgSlug}/billing/subscribe`)
      return res.data as { checkout_url: string }
    },
    onSuccess: (data) => {
      window.location.href = data.checkout_url
    },
  })

  const portal = useMutation({
    mutationFn: async () => {
      const res = await axios.post(`/v1/orgs/${orgSlug}/billing/portal`)
      return res.data as { portal_url: string }
    },
    onSuccess: (data) => {
      window.location.href = data.portal_url
    },
  })

  const usage = billing?.usage || { sandboxes: 0, source_vms: 0, sandbox_hosts: 0, tokens_used: 0 }
  const limits = billing?.limits || { sandboxes: 1, source_vms: 3, sandbox_hosts: 1 }
  const plan = billing?.plan || 'free'

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-sm font-medium text-white">Billing</h2>
        <p className="text-muted-foreground text-xs">Manage your plan and usage</p>
      </div>

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
                  ? '1 sandbox, 3 source VMs, 1 agent host'
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
          current={usage.sandboxes}
          limit={limits.sandboxes}
          color="blue"
        />
        <UsageMeter
          label="Source VMs"
          current={usage.source_vms}
          limit={limits.source_vms}
          color="green"
        />
        <UsageMeter
          label="Sandbox Hosts"
          current={usage.sandbox_hosts}
          limit={limits.sandbox_hosts}
          color="amber"
        />
        <UsageMeter label="Tokens" current={usage.tokens_used} limit={100000} color="purple" />
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

      {isLoading && <p className="text-muted-foreground text-xs">Loading billing data...</p>}
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
  limit: number
  color: string
}) {
  const pct = limit > 0 ? Math.min((current / limit) * 100, 100) : 0
  const colorMap: Record<string, string> = {
    blue: 'bg-blue-400',
    green: 'bg-green-400',
    amber: 'bg-amber-400',
    purple: 'bg-purple-400',
  }

  const formatNumber = (n: number) => {
    if (n >= 1000) return `${(n / 1000).toFixed(n >= 10000 ? 0 : 1)}k`
    return n.toString()
  }

  return (
    <div className="border-border border bg-neutral-900/50 p-4">
      <div className="flex items-center justify-between text-xs">
        <span className="text-muted-foreground">{label}</span>
        <span className="text-white">
          {formatNumber(current)} / {formatNumber(limit)}
        </span>
      </div>
      <div className="mt-2 h-1.5 w-full bg-neutral-800">
        <div
          className={`h-full ${colorMap[color] || 'bg-blue-400'} transition-all`}
          style={{ width: `${pct}%` }}
        />
      </div>
    </div>
  )
}
