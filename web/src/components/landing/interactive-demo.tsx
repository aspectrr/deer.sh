import { useEffect, useRef, useState } from 'react'
import { DemoEngine } from '~/lib/demo-engine'
import '@xterm/xterm/css/xterm.css'

export function InteractiveDemo() {
  const containerRef = useRef<HTMLDivElement>(null)
  const engineRef = useRef<DemoEngine | null>(null)
  const [status, setStatus] = useState('connecting...')

  useEffect(() => {
    if (!containerRef.current || engineRef.current) return

    const wsUrl = import.meta.env.VITE_DEMO_WS_URL || 'ws://localhost:8090/ws/demo'

    engineRef.current = new DemoEngine(containerRef.current, wsUrl, (s: string) => {
      setStatus(s)
    })

    return () => {
      engineRef.current?.destroy()
      engineRef.current = null
    }
  }, [])

  const statusClass =
    status === 'connected'
      ? 'text-green-200'
      : status === 'connecting' || status === 'connecting...'
        ? 'opacity-75'
        : 'text-red-200'

  return (
    <div className="relative">
      <div className="flex items-center justify-between rounded-t-lg bg-blue-500 px-3 py-1.5 font-mono text-xs text-white">
        <div className="flex items-center gap-3">
          <span className="font-bold">fluid.sh</span>
          <span className="opacity-75">demo</span>
        </div>
        <div className="flex items-center gap-3">
          <span className={statusClass}>{status}</span>
        </div>
      </div>
      <div
        ref={containerRef}
        className="overflow-hidden rounded-b-lg border border-t-0 border-neutral-800"
        style={{ height: 400, background: '#000000' }}
      />
    </div>
  )
}
