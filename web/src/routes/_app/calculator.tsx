import { createFileRoute } from '@tanstack/react-router'
import { useState } from 'react'
import { useMutation } from '@tanstack/react-query'
import { axios } from '~/lib/axios'
import { Input } from '~/components/ui/input'
import { Label } from '~/components/ui/label'
import { Button } from '~/components/ui/button'
import { ModelSelector } from '~/components/model-selector'

export const Route = createFileRoute('/_app/calculator')({
  component: CalculatorPage,
})

interface TokenBreakdown {
  free_tokens: number
  billable_tokens: number
  cost_per_1k: number
  markup: number
}

interface CalculatorResult {
  sandbox_cost: number
  source_vm_cost: number
  agent_host_cost: number
  token_cost: number
  token_breakdown?: TokenBreakdown
  total_monthly: number
  currency: string
}

function CalculatorPage() {
  const [sandboxes, setSandboxes] = useState(2)
  const [sourceVMs, setSourceVMs] = useState(5)
  const [agentHosts, setAgentHosts] = useState(2)
  const [hours, setHours] = useState(720)
  const [estimatedTokens, setEstimatedTokens] = useState(0)
  const [tokenModel, setTokenModel] = useState('anthropic/claude-sonnet-4')

  const calculate = useMutation({
    mutationFn: async () => {
      const res = await axios.post('/v1/billing/calculator', {
        concurrent_sandboxes: sandboxes,
        source_vms: sourceVMs,
        agent_hosts: agentHosts,
        hours_per_month: hours,
        estimated_tokens: estimatedTokens,
        model: tokenModel,
      })
      return res.data as CalculatorResult
    },
  })

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-sm font-medium text-white">Cost Calculator</h2>
        <p className="text-muted-foreground text-xs">
          Estimate your monthly costs on the usage-based plan
        </p>
      </div>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        {/* Inputs */}
        <div className="border-border space-y-4 border bg-neutral-900/50 p-6">
          <h3 className="text-xs font-medium text-white">Configuration</h3>

          <div className="space-y-1">
            <Label className="text-xs">Concurrent Sandboxes</Label>
            <Label className="text-xs">Concurrent Sandboxes Running at any given time</Label>
            <Input
              type="number"
              min={0}
              value={sandboxes}
              onChange={(e) => setSandboxes(parseInt(e.target.value) || 0)}
              className="bg-background text-xs"
            />
            <p className="text-muted-foreground text-[10px]">Free tier: 1</p>
          </div>

          <div className="space-y-1">
            <Label className="text-xs">Source VMs</Label>
            <Input
              type="number"
              min={0}
              value={sourceVMs}
              onChange={(e) => setSourceVMs(parseInt(e.target.value) || 0)}
              className="bg-background text-xs"
            />
            <p className="text-muted-foreground text-[10px]">Free tier: 3</p>
          </div>

          <div className="space-y-1">
            <Label className="text-xs">Agent Hosts</Label>
            <Input
              type="number"
              min={0}
              value={agentHosts}
              onChange={(e) => setAgentHosts(parseInt(e.target.value) || 0)}
              className="bg-background text-xs"
            />
            <p className="text-muted-foreground text-[10px]">Free tier: 1</p>
          </div>

          <div className="space-y-1">
            <Label className="text-xs">Hours per Month</Label>
            <Input
              type="number"
              min={0}
              value={hours}
              onChange={(e) => setHours(parseInt(e.target.value) || 0)}
              className="bg-background text-xs"
            />
            <p className="text-muted-foreground text-[10px]">730 hours = 24/7</p>
          </div>

          <div className="space-y-1">
            <Label className="text-xs">Estimated Tokens / Month</Label>
            <Input
              type="number"
              min={0}
              value={estimatedTokens}
              onChange={(e) => setEstimatedTokens(parseInt(e.target.value) || 0)}
              className="bg-background text-xs"
            />
            <p className="text-muted-foreground text-[10px]">Free tier: 100,000 tokens/month</p>
          </div>

          <div className="space-y-1">
            <Label className="text-xs">Agent Model</Label>
            <ModelSelector value={tokenModel} onChange={setTokenModel} />
          </div>

          <Button
            className="w-full bg-blue-500 text-xs text-black hover:bg-blue-400"
            onClick={() => calculate.mutate()}
            disabled={calculate.isPending}
          >
            {calculate.isPending ? 'Calculating...' : 'Calculate'}
          </Button>
        </div>

        {/* Results */}
        <div className="border-border space-y-4 border bg-neutral-900/50 p-6">
          <h3 className="text-xs font-medium text-white">Estimated Monthly Cost</h3>

          {calculate.data ? (
            <div className="space-y-3">
              <CostLine
                label="Sandbox hours"
                detail={`${sandboxes} x ${hours}h x $0.05/h`}
                cost={calculate.data.sandbox_cost}
              />
              <CostLine
                label="Source VMs"
                detail={`${sourceVMs} x $5.00/mo`}
                cost={calculate.data.source_vm_cost}
              />
              <CostLine
                label="Agent hosts"
                detail={`${agentHosts} x $10.00/mo`}
                cost={calculate.data.agent_host_cost}
              />
              {calculate.data.token_breakdown && (
                <CostLine
                  label="Agent tokens"
                  detail={`${calculate.data.token_breakdown.billable_tokens.toLocaleString()} billable (${calculate.data.token_breakdown.free_tokens.toLocaleString()} free)`}
                  cost={calculate.data.token_cost}
                />
              )}
              <div className="border-border border-t pt-3">
                <div className="flex items-center justify-between">
                  <span className="text-xs font-medium text-white">Total</span>
                  <span className="text-lg font-medium text-blue-400">
                    ${calculate.data.total_monthly.toFixed(2)}
                  </span>
                </div>
                <p className="text-muted-foreground mt-1 text-[10px]">per month (USD)</p>
              </div>
            </div>
          ) : (
            <div className="text-muted-foreground flex items-center justify-center py-12 text-xs">
              Adjust the configuration and click Calculate
            </div>
          )}

          <div className="border-border border-t pt-4">
            <h4 className="text-[10px] font-medium text-white">Unit Prices</h4>
            <div className="text-muted-foreground mt-2 space-y-1 text-[10px]">
              <p>Sandbox: $0.05/hour</p>
              <p>Source VM: $5.00/month</p>
              <p>Agent Host: $10.00/month</p>
              <p>Tokens: pass-through cost + 0.5% (100k/mo free)</p>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

function CostLine({ label, detail, cost }: { label: string; detail: string; cost: number }) {
  return (
    <div className="flex items-center justify-between">
      <div>
        <span className="text-xs text-white">{label}</span>
        <p className="text-muted-foreground text-[10px]">{detail}</p>
      </div>
      <span className="text-xs text-white">${cost.toFixed(2)}</span>
    </div>
  )
}
