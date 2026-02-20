import { createFileRoute } from '@tanstack/react-router'
import { useState } from 'react'
import { useMutation } from '@tanstack/react-query'
import { axios } from '~/lib/axios'
import { Input } from '~/components/ui/input'
import { Label } from '~/components/ui/label'
import { Button } from '~/components/ui/button'
// import { ModelSelector } from '~/components/model-selector'

export const Route = createFileRoute('/_app/billing/calculator')({
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
  // const [estimatedTokens, setEstimatedTokens] = useState(0)
  // const [tokenModel, setTokenModel] = useState('anthropic/claude-sonnet-4')

  const calculate = useMutation({
    mutationFn: async () => {
      const res = await axios.post('/v1/billing/calculator', {
        concurrent_sandboxes: sandboxes,
        source_vms: sourceVMs,
        agent_hosts: agentHosts,
        // estimated_tokens: estimatedTokens,
        // model: tokenModel,
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
            <p className="text-muted-foreground text-[10px]">
              Concurrent Sandboxes Running at any given time
            </p>
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
            <p className="text-muted-foreground text-[10px]">
              Source VMs that sandboxes can be created from
            </p>
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
            <Label className="text-xs">Fluid Daemons</Label>
            <p className="text-muted-foreground text-[10px]">
              Fluid daemons that run on sandbox hosts, manage sandbox creation and management
            </p>
            <Input
              type="number"
              min={0}
              value={agentHosts}
              onChange={(e) => setAgentHosts(parseInt(e.target.value) || 0)}
              className="bg-background text-xs"
            />
            <p className="text-muted-foreground text-[10px]">Free tier: 1</p>
          </div>

          {/*<div className="space-y-1">
            <Label className="text-xs">Estimated LLM Output Tokens / Month</Label>
            <Input
              type="number"
              min={0}
              value={estimatedTokens}
              onChange={(e) => setEstimatedTokens(parseInt(e.target.value) || 0)}
              className="bg-background text-xs"
            />
            <p className="text-muted-foreground text-[10px]">
              Average user is probably 100k-200k/month
            </p>
          </div>

          <div className="space-y-1">
            <Label className="text-xs">Agent Model</Label>
            <ModelSelector value={tokenModel} onChange={setTokenModel} />
          </div>
*/}
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
                label="Concurrent Sandboxes"
                detail={`${sandboxes} sandboxes (1 free) x $50.00/mo`}
                cost={calculate.data.sandbox_cost}
              />
              <CostLine
                label="Source VMs"
                detail={`${sourceVMs} VMs (3 free) x $5.00/mo`}
                cost={calculate.data.source_vm_cost}
              />
              <CostLine
                label="Fluid Daemons"
                detail={`${agentHosts} daemons (1 free) x $10.00/mo`}
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
                <ul className="mt-6 space-y-1">
                  <li className="text-[10px] text-white">Want custom limits?</li>
                  <li className="text-[10px] text-white">
                    Want sensitive data to be scrubbed from LLM inputs?
                  </li>
                  <li className="text-[10px] text-white">
                    Want a dedicated Slack channel with 24/7 support?
                  </li>
                </ul>
                <p className="mt-3 text-[10px] text-white">
                  Contact Collin about your{' '}
                  <a href="mailto:cpfeifer@madcactus.org" className="text-blue-400 underline">
                    enterprise needs
                  </a>
                  .
                </p>
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
              <p>Concurrent Sandbox: $50.00/month</p>
              <p>Source VM: $5.00/month</p>
              <p>Fluid Daemon: $10.00/month</p>
              {/*<p>Tokens: pass-through cost + 5% upcharge (100k/mo free)</p>*/}
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
