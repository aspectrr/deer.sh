import { cn } from '~/lib/utils'

interface TierBoxProps {
  label: string
  sublabel: string
  items: string[]
  highlight?: boolean
}

function TierBox({ label, sublabel, items, highlight }: TierBoxProps) {
  return (
    <div
      className={cn(
        'min-w-50 border p-4',
        highlight ? 'border-blue-400/40 bg-blue-400/5' : 'border-border bg-neutral-900'
      )}
    >
      {/* Terminal title bar */}
      <div className="mb-3 flex items-center gap-1.5">
        <div className="h-1.5 w-1.5 bg-neutral-600" />
        <div className="h-1.5 w-1.5 bg-neutral-600" />
        <div className="h-1.5 w-1.5 bg-neutral-600" />
        <span className="ml-2 text-[10px] text-neutral-500">{sublabel}</span>
      </div>
      <p className="mb-2 text-xs font-medium text-white">{label}</p>
      <ul className="space-y-1">
        {items.map((item) => (
          <li key={item} className="text-[10px] text-neutral-500">
            <span className="mr-1 text-blue-400">-</span>
            {item}
          </li>
        ))}
      </ul>
    </div>
  )
}

function Arrow({ label }: { label: string }) {
  return (
    <div className="flex flex-col items-center justify-center px-2 py-4 md:px-4 md:py-0">
      <div className="mb-1 text-[10px] text-neutral-500">{label}</div>
      <div className="relative hidden h-px w-12 bg-neutral-700 md:block">
        <div className="-top-0.7 absolute right-0 border-t-[3px] border-r-0 border-b-[3px] border-l-[5px] border-transparent border-l-neutral-700" />
      </div>
      <div className="relative h-6 w-px bg-neutral-700 md:hidden">
        <div className="absolute bottom-0 -left-0.75 border-t-[5px] border-r-[3px] border-b-0 border-l-[3px] border-transparent border-t-neutral-700" />
      </div>
    </div>
  )
}

export function ArchitectureDiagram() {
  return (
    <div className="border-border mb-6 overflow-x-auto border bg-black p-6">
      <div className="flex min-w-fit flex-col items-center justify-center gap-0 md:flex-row">
        <TierBox
          label="Fluid CLI"
          sublabel="tier 1"
          items={['Direct libvirt', 'SQLite state', 'Single machine']}
        />
        <Arrow label="gRPC" />
        <TierBox
          label="fluid-daemon"
          sublabel="tier 2"
          highlight
          items={['gRPC server', 'Sandbox lifecycle', 'Image cache']}
        />
        <Arrow label="gRPC stream" />
        <TierBox
          label="Control Plane"
          sublabel="tier 3"
          items={['REST API', 'Multi-host orchestrator', 'PostgreSQL']}
        />
      </div>
    </div>
  )
}
