import { useEffect, useRef, useState } from 'react'
import { ScriptedDemoEngine } from '~/lib/scripted-demo'
import type { DiagramPhase } from '~/lib/diagram-phases'
import '@xterm/xterm/css/xterm.css'

interface ScriptedDemoProps {
  onPhase?: (phase: DiagramPhase) => void
}

export function ScriptedDemo({ onPhase }: ScriptedDemoProps) {
  const containerRef = useRef<HTMLDivElement>(null)
  const engineRef = useRef<ScriptedDemoEngine | null>(null)
  const [visible, setVisible] = useState(false)
  const onPhaseRef = useRef(onPhase)
  useEffect(() => {
    onPhaseRef.current = onPhase
  }, [onPhase])

  useEffect(() => {
    // Trigger fade-in on mount
    requestAnimationFrame(() => setVisible(true))
  }, [])

  useEffect(() => {
    if (!containerRef.current || engineRef.current) return

    // Delay engine init until after fade-in animation completes
    const timer = setTimeout(() => {
      if (!containerRef.current || engineRef.current) return
      engineRef.current = new ScriptedDemoEngine(containerRef.current, (phase) =>
        onPhaseRef.current?.(phase)
      )
    }, 700)

    return () => {
      clearTimeout(timer)
      engineRef.current?.destroy()
      engineRef.current = null
    }
  }, [])

  return (
    <div
      className="relative rounded-2xl"
      style={{
        opacity: visible ? 1 : 0,
        transform: visible ? 'translateY(0)' : 'translateY(20px)',
        transition: 'opacity 700ms ease-out, transform 700ms ease-out',
      }}
    >
      <div className="flex items-center gap-2 rounded-t-2xl bg-neutral-800 px-4 py-2.5">
        <div className="flex items-center gap-1.5">
          <div className="h-3 w-3 rounded-full bg-neutral-600" />
          <div className="h-3 w-3 rounded-full bg-neutral-600" />
          <div className="h-3 w-3 rounded-full bg-neutral-600" />
        </div>
        <span className="ml-1 font-mono text-xs font-medium text-neutral-400">Fluid</span>
      </div>
      <div
        ref={containerRef}
        className="overflow-hidden rounded-b-2xl border border-t-0 border-neutral-800"
        style={{ height: 400, background: '#000000' }}
      />
    </div>
  )
}
