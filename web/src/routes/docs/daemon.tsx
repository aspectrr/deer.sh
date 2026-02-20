import { createFileRoute } from '@tanstack/react-router'
import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { StepTracker } from '~/components/docs/step-tracker'
import { TerminalBlock } from '~/components/docs/terminal-block'
import { Callout } from '~/components/docs/callout'
import { PrevNext } from '~/components/docs/prev-next'
import { DaemonConnectionStatus } from '~/components/daemon-connection-status'
import { daemonSetupSteps } from '~/components/docs/daemon-setup-steps'
import { useAuth } from '~/lib/auth'
import { axios } from '~/lib/axios'

export const Route = createFileRoute('/docs/daemon')({
  component: DaemonPage,
})

function DaemonPage() {
  const [showSource, setShowSource] = useState(false)
  const { isAuthenticated } = useAuth()
  const { data: orgsData } = useQuery({
    queryKey: ['orgs'],
    queryFn: async () => {
      const res = await axios.get('/v1/orgs')
      return res.data as { organizations: Array<{ slug: string }> }
    },
    enabled: isAuthenticated,
  })
  const orgSlug = orgsData?.organizations[0]?.slug

  return (
    <div className="mx-auto max-w-2xl px-6 py-8">
      <h1 className="mb-1 text-lg font-medium text-white">Daemon Setup</h1>
      <p className="text-muted-foreground mb-8 text-xs">
        Install and configure fluid-daemon on each sandbox host.
      </p>

      <Callout type="info" title="One daemon per sandbox host">
        The daemon runs on each machine where you want to run sandboxes. Each daemon can connect to
        multiple libvirt hosts over SSH for source VM access. Multiple daemons are typically needed
        for heavily NATed enterprise networks or separate data centers.
      </Callout>

      {/* Alternative install methods (collapsed) */}
      <div className="mb-6 space-y-2">
        <button
          onClick={() => setShowSource(!showSource)}
          className="text-xs text-blue-400 transition-colors hover:text-blue-300"
        >
          {showSource ? 'Hide' : 'Show'} build from source
        </button>
        {showSource && (
          <div className="border-border border bg-neutral-900/30 p-4">
            <TerminalBlock
              lines={[
                { command: 'git clone https://github.com/aspectrr/fluid.sh.git' },
                {
                  command:
                    'cd fluid.sh/fluid-daemon && go build -o /usr/local/bin/fluid-daemon ./cmd/fluid-daemon',
                },
              ]}
            />
          </div>
        )}
      </div>

      <StepTracker storageKey="daemon" steps={daemonSetupSteps} />

      {orgSlug && <DaemonConnectionStatus orgSlug={orgSlug} />}

      <PrevNext />
    </div>
  )
}
