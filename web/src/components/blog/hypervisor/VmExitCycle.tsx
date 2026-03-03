import { useState, useEffect, useCallback } from 'react'
import { motion } from 'framer-motion'

const REDUCED_MOTION =
  typeof window !== 'undefined' && window.matchMedia('(prefers-reduced-motion: reduce)').matches

const BLUE = '#60a5fa'
const GREEN = '#4ade80'
const AMBER = '#fbbf24'
const RED = '#f87171'
const TEXT_MUTED = '#737373'
const BORDER = '#262626'
const CARD_BG = '#171717'

interface Phase {
  id: string
  label: string
  detail: string
  color: string
}

const PHASES: Phase[] = [
  { id: 'entry', label: 'VM Entry', detail: 'Restore guest state, enter EL1', color: GREEN },
  { id: 'exec', label: 'Guest Executes', detail: 'Running guest code at EL0/EL1', color: BLUE },
  { id: 'trap', label: 'VM Exit (Trap)', detail: 'HVC, MMIO, system reg, timer...', color: RED },
  {
    id: 'handle',
    label: 'VMM Handles',
    detail: 'Decode exit, emulate, update state',
    color: AMBER,
  },
]

const duration = REDUCED_MOTION ? 0 : 0.6

export function VmExitCycle() {
  const [activePhase, setActivePhase] = useState(0)
  const [running, setRunning] = useState(false)

  const advance = useCallback(() => {
    setActivePhase((p) => (p + 1) % PHASES.length)
  }, [])

  useEffect(() => {
    if (!running) return
    const interval = setInterval(advance, 1500)
    return () => clearInterval(interval)
  }, [running, advance])

  const cx = 230
  const cy = 140
  const r = 100

  return (
    <div
      style={{
        background: CARD_BG,
        border: `1px solid ${BORDER}`,
        padding: '1.5rem',
        marginTop: '1.5rem',
        marginBottom: '1.5rem',
        fontFamily: 'ui-monospace, SFMono-Regular, monospace',
      }}
    >
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          marginBottom: '1rem',
        }}
      >
        <span style={{ color: BLUE, fontSize: '0.75rem' }}>VM Exit/Entry Cycle</span>
        <button
          onClick={() => {
            if (!running) {
              setRunning(true)
            } else {
              setRunning(false)
            }
          }}
          style={{
            background: 'none',
            border: `1px solid ${BLUE}`,
            color: BLUE,
            cursor: 'pointer',
            padding: '0.25rem 0.75rem',
            fontSize: '0.75rem',
            fontFamily: 'ui-monospace, SFMono-Regular, monospace',
          }}
        >
          {running ? 'Pause' : 'Play'}
        </button>
      </div>

      <div style={{ overflowX: 'auto' }}>
        <svg
          viewBox="0 0 460 280"
          style={{
            width: '100%',
            maxWidth: 540,
            minWidth: 320,
            display: 'block',
            margin: '0 auto',
          }}
        >
          <defs>
            <marker
              id="cycle-arrow"
              markerWidth={8}
              markerHeight={6}
              refX={8}
              refY={3}
              orient="auto"
            >
              <polygon points="0,0 8,3 0,6" fill={TEXT_MUTED} />
            </marker>
          </defs>

          {/* Draw circular path with arrows */}
          <circle
            cx={cx}
            cy={cy}
            r={r}
            fill="none"
            stroke={BORDER}
            strokeWidth={1}
            strokeDasharray="4,4"
          />

          {/* Phase nodes positioned around the circle */}
          {PHASES.map((phase, i) => {
            const angle = (i / PHASES.length) * Math.PI * 2 - Math.PI / 2
            const px = cx + r * Math.cos(angle)
            const py = cy + r * Math.sin(angle)
            const isActive = i === activePhase
            const boxW = 120
            const boxH = 42

            // Arrow to next node
            const arrowX = cx + (r + 2) * Math.cos(angle + ((Math.PI * 2) / PHASES.length) * 0.5)
            const arrowY = cy + (r + 2) * Math.sin(angle + ((Math.PI * 2) / PHASES.length) * 0.5)

            return (
              <g key={phase.id}>
                {/* Connecting arc segment */}
                <motion.circle
                  cx={arrowX}
                  cy={arrowY}
                  r={3}
                  fill={isActive ? phase.color : TEXT_MUTED}
                  animate={{ opacity: isActive ? 1 : 0.3 }}
                  transition={{ duration: duration / 2 }}
                />

                {/* Phase box */}
                <motion.g
                  animate={{
                    opacity: isActive ? 1 : 0.4,
                    scale: isActive ? 1.05 : 1,
                  }}
                  transition={{ duration: duration / 2 }}
                  style={{ transformOrigin: `${px}px ${py}px` }}
                >
                  <rect
                    x={px - boxW / 2}
                    y={py - boxH / 2}
                    width={boxW}
                    height={boxH}
                    rx={4}
                    fill={isActive ? phase.color + '22' : '#0a0a0a'}
                    stroke={isActive ? phase.color : BORDER}
                    strokeWidth={isActive ? 2 : 1}
                  />
                  <text
                    x={px}
                    y={py - 4}
                    textAnchor="middle"
                    fill={isActive ? phase.color : TEXT_MUTED}
                    fontSize={10}
                    fontFamily="ui-monospace, SFMono-Regular, monospace"
                    fontWeight={600}
                  >
                    {phase.label}
                  </text>
                  <text
                    x={px}
                    y={py + 10}
                    textAnchor="middle"
                    fill={TEXT_MUTED}
                    fontSize={7}
                    fontFamily="ui-monospace, SFMono-Regular, monospace"
                  >
                    {phase.detail}
                  </text>
                </motion.g>
              </g>
            )
          })}

          {/* Center label */}
          <text
            x={cx}
            y={cy - 6}
            textAnchor="middle"
            fill="#e5e5e5"
            fontSize={10}
            fontFamily="ui-monospace, SFMono-Regular, monospace"
            fontWeight={600}
          >
            hv_vcpu_run()
          </text>
          <text
            x={cx}
            y={cy + 8}
            textAnchor="middle"
            fill={TEXT_MUTED}
            fontSize={8}
            fontFamily="ui-monospace, SFMono-Regular, monospace"
          >
            loop
          </text>
        </svg>
      </div>

      {/* Current phase description */}
      <div
        style={{
          marginTop: '1rem',
          padding: '0.75rem',
          background: '#0a0a0a',
          border: `1px solid ${BORDER}`,
          fontSize: '0.8rem',
          lineHeight: 1.5,
          color: '#a3a3a3',
        }}
      >
        <span style={{ color: PHASES[activePhase].color, fontWeight: 600 }}>
          {PHASES[activePhase].label}:
        </span>{' '}
        {activePhase === 0 &&
          "The VMM calls hv_vcpu_run(). The CPU restores the guest's register state (x0-x30, PC, CPSR, system registers) and switches to EL1. Guest code resumes exactly where it left off."}
        {activePhase === 1 &&
          'The guest CPU executes instructions natively at full speed. No emulation, no interpretation. This continues until something triggers a trap - the guest is running on real hardware.'}
        {activePhase === 2 &&
          'Something causes the CPU to exit back to the host. The exit reason is written to the hv_vcpu_exit_t struct: an HVC instruction, an unmapped memory access (MMIO), a system register access, or a timer firing.'}
        {activePhase === 3 &&
          'The VMM reads the exit reason and handles it: emulate the MMIO device, respond to the HVC call, advance the PC past the instruction, update registers. Then loop back to VM entry.'}
      </div>

      {/* Step indicator */}
      <div
        style={{
          display: 'flex',
          justifyContent: 'center',
          gap: '0.375rem',
          marginTop: '0.75rem',
        }}
      >
        {PHASES.map((phase, i) => (
          <button
            key={i}
            onClick={() => {
              setActivePhase(i)
              setRunning(false)
            }}
            style={{
              width: 6,
              height: 6,
              borderRadius: '50%',
              background: i === activePhase ? phase.color : BORDER,
              border: 'none',
              cursor: 'pointer',
              padding: 0,
            }}
          />
        ))}
      </div>
    </div>
  )
}
