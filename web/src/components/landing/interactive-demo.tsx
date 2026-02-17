import { useEffect, useRef, useState } from 'react'
import { ScriptedDemoEngine } from '~/lib/scripted-demo'
import '@xterm/xterm/css/xterm.css'

export function InteractiveDemo() {
  const containerRef = useRef<HTMLDivElement>(null)
  const engineRef = useRef<ScriptedDemoEngine | null>(null)
  const [status, setStatus] = useState('demo')

  useEffect(() => {
    if (!containerRef.current || engineRef.current) return

    engineRef.current = new ScriptedDemoEngine(containerRef.current, (s: string) => {
      setStatus(s)
    })

    return () => {
      engineRef.current?.destroy()
      engineRef.current = null
    }
  }, [])

  return (
    <div className="relative">
      <div className="flex items-center justify-between rounded-t-lg bg-blue-500 px-3 py-1.5 font-mono text-xs text-white">
        <div className="flex items-center gap-3">
          <span className="font-bold">fluid.sh</span>
          <span className="opacity-75">demo</span>
        </div>
        <div className="flex items-center gap-3">
          <span className="text-green-200">{status}</span>
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
