import { useState } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { ChevronLeft, ChevronRight } from 'lucide-react'

const REDUCED_MOTION =
  typeof window !== 'undefined' && window.matchMedia('(prefers-reduced-motion: reduce)').matches

const BLUE = '#60a5fa'
const GREEN = '#4ade80'
const AMBER = '#fbbf24'
const RED = '#f87171'
const TEXT_MUTED = '#737373'
const BORDER = '#262626'
const CARD_BG = '#171717'

interface ELLevel {
  label: string
  sublabel: string
  fill: string
  stroke: string
  y: number
}

const LEVELS: ELLevel[] = [
  { label: 'EL3', sublabel: 'Secure Monitor', fill: '#2a1a1a', stroke: RED, y: 20 },
  { label: 'EL2', sublabel: 'Hypervisor', fill: '#2a2000', stroke: AMBER, y: 90 },
  { label: 'EL1', sublabel: 'OS Kernel', fill: '#1a2e1a', stroke: GREEN, y: 160 },
  { label: 'EL0', sublabel: 'User Applications', fill: '#1e3a5f', stroke: BLUE, y: 230 },
]

interface Step {
  title: string
  description: string
  highlights: Record<string, string>
  annotations: Record<string, string>
}

const STEPS: Step[] = [
  {
    title: 'The Exception Level Hierarchy',
    description:
      'ARM CPUs have four privilege levels called Exception Levels (EL0-EL3). Higher numbers mean more privilege. Code at each level can only access resources allowed at that level or below. Moving between levels requires explicit transitions via exceptions.',
    highlights: {},
    annotations: {},
  },
  {
    title: 'EL0 - User Space',
    description:
      'EL0 is the least privileged level. Normal applications run here - your browser, your editor, your shell. Code at EL0 cannot touch hardware directly, access page tables, or configure interrupts. It must ask the kernel via system calls (SVC instruction), which trap to EL1.',
    highlights: { EL0: BLUE },
    annotations: { EL0: 'Your apps run here' },
  },
  {
    title: 'EL1 - OS Kernel',
    description:
      'EL1 runs the operating system kernel. It manages page tables, handles interrupts, schedules processes, and controls hardware. Linux, macOS, and Windows kernels all run at EL1. Inside a VM, the guest kernel also runs at EL1 - but its "hardware" access is actually trapped by EL2.',
    highlights: { EL1: GREEN },
    annotations: { EL1: 'Linux / guest kernel' },
  },
  {
    title: 'EL2 - Hypervisor',
    description:
      "EL2 is the hypervisor level. It controls the second stage of address translation (IPA -> PA), traps sensitive guest operations, and manages VM entry/exit. On a Mac, Apple's Hypervisor.framework provides EL2 access from userspace - you don't write EL2 code directly, but you configure what EL2 does via the framework's API.",
    highlights: { EL2: AMBER },
    annotations: { EL2: 'Hypervisor.framework' },
  },
  {
    title: 'EL3 - Secure Monitor',
    description:
      "EL3 is the highest privilege level, running the Secure Monitor firmware. It mediates transitions between Secure and Non-secure worlds (TrustZone). On Apple Silicon, EL3 is Apple's firmware - you never touch it. PSCI (Power State Coordination Interface) calls from the guest go through EL3 via SMC instructions.",
    highlights: { EL3: RED },
    annotations: { EL3: 'Apple firmware' },
  },
  {
    title: 'Guest Execution Model',
    description:
      "In a VM, the guest thinks it has the whole machine. Guest apps run at EL0, guest kernel at EL1. But the hypervisor at EL2 intercepts privileged operations: the guest kernel's page table changes go through stage-2 translation, sensitive system register accesses trap to EL2, and instructions like HVC/SMC cause VM exits.",
    highlights: { EL0: BLUE, EL1: GREEN, EL2: AMBER },
    annotations: { EL0: 'Guest app', EL1: 'Guest kernel', EL2: 'Traps & translates' },
  },
]

const duration = REDUCED_MOTION ? 0 : 0.35

export function ExceptionLevelDiagram() {
  const [step, setStep] = useState(0)
  const current = STEPS[step]

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
      {/* Step header */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          marginBottom: '1rem',
        }}
      >
        <span style={{ color: BLUE, fontSize: '0.75rem' }}>
          Step {step + 1} / {STEPS.length}
        </span>
        <span style={{ color: '#e5e5e5', fontSize: '0.875rem', fontWeight: 500 }}>
          {current.title}
        </span>
        <div style={{ display: 'flex', gap: '0.5rem' }}>
          <button
            onClick={() => setStep((s) => Math.max(0, s - 1))}
            disabled={step === 0}
            style={{
              background: 'none',
              border: `1px solid ${step === 0 ? BORDER : BLUE}`,
              color: step === 0 ? TEXT_MUTED : BLUE,
              cursor: step === 0 ? 'default' : 'pointer',
              padding: '0.25rem',
              display: 'flex',
              alignItems: 'center',
            }}
          >
            <ChevronLeft size={14} />
          </button>
          <button
            onClick={() => setStep((s) => Math.min(STEPS.length - 1, s + 1))}
            disabled={step === STEPS.length - 1}
            style={{
              background: 'none',
              border: `1px solid ${step === STEPS.length - 1 ? BORDER : BLUE}`,
              color: step === STEPS.length - 1 ? TEXT_MUTED : BLUE,
              cursor: step === STEPS.length - 1 ? 'default' : 'pointer',
              padding: '0.25rem',
              display: 'flex',
              alignItems: 'center',
            }}
          >
            <ChevronRight size={14} />
          </button>
        </div>
      </div>

      {/* SVG diagram */}
      <div style={{ overflowX: 'auto' }}>
        <svg
          viewBox="0 0 460 300"
          style={{
            width: '100%',
            maxWidth: 540,
            minWidth: 320,
            display: 'block',
            margin: '0 auto',
          }}
        >
          {/* Privilege arrow */}
          <text
            x={18}
            y={160}
            fill={TEXT_MUTED}
            fontSize={9}
            fontFamily="ui-monospace, SFMono-Regular, monospace"
            textAnchor="middle"
            transform="rotate(-90, 18, 160)"
          >
            privilege
          </text>
          <line x1={18} y1={55} x2={18} y2={32} stroke={TEXT_MUTED} strokeWidth={1} />
          <polygon points="14,32 22,32 18,24" fill={TEXT_MUTED} />

          {LEVELS.map((level) => {
            const isHighlighted = level.label in current.highlights
            const highlightColor = current.highlights[level.label]
            const annotation = current.annotations[level.label]
            const strokeColor = isHighlighted ? highlightColor : level.stroke
            const opacity = Object.keys(current.highlights).length === 0 || isHighlighted ? 1 : 0.3

            return (
              <motion.g key={level.label} animate={{ opacity }} transition={{ duration }}>
                <rect
                  x={40}
                  y={level.y}
                  width={300}
                  height={50}
                  rx={4}
                  fill={level.fill}
                  stroke={strokeColor}
                  strokeWidth={isHighlighted ? 2 : 1}
                />
                <text
                  x={55}
                  y={level.y + 22}
                  fill={strokeColor}
                  fontSize={13}
                  fontFamily="ui-monospace, SFMono-Regular, monospace"
                  fontWeight={700}
                >
                  {level.label}
                </text>
                <text
                  x={55}
                  y={level.y + 40}
                  fill={TEXT_MUTED}
                  fontSize={10}
                  fontFamily="ui-monospace, SFMono-Regular, monospace"
                >
                  {level.sublabel}
                </text>
                <AnimatePresence>
                  {annotation && (
                    <motion.g
                      initial={{ opacity: 0, x: 10 }}
                      animate={{ opacity: 1, x: 0 }}
                      exit={{ opacity: 0, x: 10 }}
                      transition={{ duration }}
                    >
                      <line
                        x1={345}
                        y1={level.y + 25}
                        x2={365}
                        y2={level.y + 25}
                        stroke={strokeColor}
                        strokeWidth={1}
                        strokeDasharray="3,3"
                      />
                      <text
                        x={372}
                        y={level.y + 29}
                        fill={strokeColor}
                        fontSize={9}
                        fontFamily="ui-monospace, SFMono-Regular, monospace"
                      >
                        {annotation}
                      </text>
                    </motion.g>
                  )}
                </AnimatePresence>
              </motion.g>
            )
          })}

          {/* Transition arrows between levels */}
          {[
            { from: 230, to: 160, label: 'SVC', x: 310 },
            { from: 160, to: 90, label: 'HVC / trap', x: 310 },
            { from: 90, to: 20, label: 'SMC', x: 310 },
          ].map(({ from, to, label, x }) => (
            <g key={label} opacity={Object.keys(current.highlights).length === 0 ? 0.5 : 0.2}>
              <line
                x1={x}
                y1={from}
                x2={x}
                y2={to + 50}
                stroke={TEXT_MUTED}
                strokeWidth={1}
                strokeDasharray="3,3"
              />
              <text
                x={x + 6}
                y={(from + to + 50) / 2}
                fill={TEXT_MUTED}
                fontSize={8}
                fontFamily="ui-monospace, SFMono-Regular, monospace"
              >
                {label}
              </text>
            </g>
          ))}
        </svg>
      </div>

      {/* Description */}
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
        {current.description}
      </div>

      {/* Progress dots */}
      <div
        style={{
          display: 'flex',
          justifyContent: 'center',
          gap: '0.375rem',
          marginTop: '0.75rem',
        }}
      >
        {STEPS.map((_, i) => (
          <button
            key={i}
            onClick={() => setStep(i)}
            style={{
              width: 6,
              height: 6,
              borderRadius: '50%',
              background: i === step ? BLUE : BORDER,
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
