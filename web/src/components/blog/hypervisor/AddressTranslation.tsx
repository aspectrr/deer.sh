import { useState } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { ChevronLeft, ChevronRight } from 'lucide-react'

const REDUCED_MOTION =
  typeof window !== 'undefined' && window.matchMedia('(prefers-reduced-motion: reduce)').matches

const BLUE = '#60a5fa'
const GREEN = '#4ade80'
const AMBER = '#fbbf24'
const PURPLE = '#a855f7'
const TEXT_MUTED = '#737373'
const BORDER = '#262626'
const CARD_BG = '#171717'

interface Stage {
  label: string
  address: string
  fill: string
  stroke: string
}

interface Step {
  title: string
  description: string
  stages: Stage[]
  activeArrow: number
}

const STEPS: Step[] = [
  {
    title: 'Guest Virtual Address (GVA)',
    description:
      "A process inside the VM accesses a virtual address. The guest kernel's page tables (managed by the guest OS at EL1) will translate this to what the guest thinks is a physical address.",
    stages: [{ label: 'GVA', address: '0xFFFF8000_1234_5000', fill: '#1e3a5f', stroke: BLUE }],
    activeArrow: -1,
  },
  {
    title: 'Stage 1: GVA to IPA',
    description:
      "The guest page tables translate the Guest Virtual Address to an Intermediate Physical Address (IPA). The guest kernel thinks this is a real physical address - it has no idea there's another translation layer. This is the same page table walk a normal OS does on bare metal.",
    stages: [
      { label: 'GVA', address: '0xFFFF8000_1234_5000', fill: '#1e3a5f', stroke: BLUE },
      { label: 'IPA', address: '0x00000000_8000_3000', fill: '#1a2e1a', stroke: GREEN },
    ],
    activeArrow: 0,
  },
  {
    title: 'Stage 2: IPA to PA',
    description:
      "The hypervisor's stage-2 page tables (configured at EL2) translate the Intermediate Physical Address to the real Host Physical Address. This is the translation that Hypervisor.framework manages via hv_vm_map(). The guest never sees real physical addresses.",
    stages: [
      { label: 'GVA', address: '0xFFFF8000_1234_5000', fill: '#1e3a5f', stroke: BLUE },
      { label: 'IPA', address: '0x00000000_8000_3000', fill: '#1a2e1a', stroke: GREEN },
      { label: 'PA', address: '0x00000001_0A00_F000', fill: '#2a2000', stroke: AMBER },
    ],
    activeArrow: 1,
  },
  {
    title: 'Physical Memory Access',
    description:
      'The CPU finally accesses real physical RAM at the Host Physical Address. Two full page table walks happened: the guest walk (GVA->IPA) and the host walk (IPA->PA). On a TLB miss this means up to 24 memory accesses on ARM64 - 4 levels in each table, and each guest table entry needs its own IPA->PA translation.',
    stages: [
      { label: 'GVA', address: '0xFFFF8000_1234_5000', fill: '#1e3a5f', stroke: BLUE },
      { label: 'IPA', address: '0x00000000_8000_3000', fill: '#1a2e1a', stroke: GREEN },
      { label: 'PA', address: '0x00000001_0A00_F000', fill: '#2a2000', stroke: AMBER },
    ],
    activeArrow: 2,
  },
  {
    title: 'What Hypervisor.framework Controls',
    description:
      'When you call hv_vm_map(guest_memory_ptr, ipa, size, flags), you\'re populating the stage-2 page table. You\'re saying "when the guest accesses IPA X, map it to host virtual address Y." The framework translates your host pointer to a PA internally. Any IPA not mapped causes a data abort exit - which is how MMIO works.',
    stages: [
      {
        label: "Your mmap'd buffer",
        address: 'host_ptr (userspace)',
        fill: '#1e1e3f',
        stroke: PURPLE,
      },
      { label: 'hv_vm_map()', address: 'IPA 0x8000_0000', fill: '#1a2e1a', stroke: GREEN },
      { label: 'Guest accesses', address: 'GPA 0x8000_0000', fill: '#2a2000', stroke: AMBER },
    ],
    activeArrow: -1,
  },
]

const BOX_WIDTH = 260
const BOX_HEIGHT = 48
const BOX_GAP = 60
const START_X = 100
const START_Y = 30

const duration = REDUCED_MOTION ? 0 : 0.35

export function AddressTranslation() {
  const [step, setStep] = useState(0)
  const current = STEPS[step]

  const svgHeight = current.stages.length * (BOX_HEIGHT + BOX_GAP) + 20

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

      {/* SVG */}
      <div style={{ overflowX: 'auto' }}>
        <svg
          viewBox={`0 0 460 ${svgHeight}`}
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
              id="addr-arrow"
              markerWidth={8}
              markerHeight={6}
              refX={8}
              refY={3}
              orient="auto"
            >
              <polygon points="0,0 8,3 0,6" fill={TEXT_MUTED} />
            </marker>
          </defs>

          <AnimatePresence mode="popLayout">
            {current.stages.map((stage, i) => {
              const y = START_Y + i * (BOX_HEIGHT + BOX_GAP)
              return (
                <motion.g
                  key={`${step}-${stage.label}`}
                  initial={{ opacity: 0, y: -10 }}
                  animate={{ opacity: 1, y: 0 }}
                  exit={{ opacity: 0, y: 10 }}
                  transition={{ duration, delay: i * 0.08 }}
                >
                  {/* Box */}
                  <rect
                    x={START_X}
                    y={y}
                    width={BOX_WIDTH}
                    height={BOX_HEIGHT}
                    rx={4}
                    fill={stage.fill}
                    stroke={stage.stroke}
                    strokeWidth={1.5}
                  />
                  <text
                    x={START_X + 12}
                    y={y + 20}
                    fill={stage.stroke}
                    fontSize={11}
                    fontFamily="ui-monospace, SFMono-Regular, monospace"
                    fontWeight={600}
                  >
                    {stage.label}
                  </text>
                  <text
                    x={START_X + 12}
                    y={y + 36}
                    fill={TEXT_MUTED}
                    fontSize={9}
                    fontFamily="ui-monospace, SFMono-Regular, monospace"
                  >
                    {stage.address}
                  </text>

                  {/* Arrow to next box */}
                  {i < current.stages.length - 1 && (
                    <g>
                      <line
                        x1={START_X + BOX_WIDTH / 2}
                        y1={y + BOX_HEIGHT}
                        x2={START_X + BOX_WIDTH / 2}
                        y2={y + BOX_HEIGHT + BOX_GAP - 4}
                        stroke={i === current.activeArrow ? stage.stroke : TEXT_MUTED}
                        strokeWidth={i === current.activeArrow ? 2 : 1}
                        strokeDasharray={i === current.activeArrow ? undefined : '4,4'}
                        markerEnd="url(#addr-arrow)"
                      />
                      {i === current.activeArrow && (
                        <text
                          x={START_X + BOX_WIDTH / 2 + 12}
                          y={y + BOX_HEIGHT + BOX_GAP / 2 + 3}
                          fill={stage.stroke}
                          fontSize={8}
                          fontFamily="ui-monospace, SFMono-Regular, monospace"
                        >
                          {i === 0 ? 'Stage 1 (guest page tables)' : 'Stage 2 (EL2 / hv_vm_map)'}
                        </text>
                      )}
                    </g>
                  )}
                </motion.g>
              )
            })}
          </AnimatePresence>
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
