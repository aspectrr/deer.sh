import { useState } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { ChevronLeft, ChevronRight } from 'lucide-react'

const REDUCED_MOTION =
  typeof window !== 'undefined' && window.matchMedia('(prefers-reduced-motion: reduce)').matches

const BLUE = '#60a5fa'
const GREEN = '#4ade80'
const RED = '#f87171'
const AMBER = '#fbbf24'
const PURPLE = '#a855f7'
const TEXT_MUTED = '#737373'
const BORDER = '#262626'
const CARD_BG = '#171717'
const MONO = 'ui-monospace, SFMono-Regular, monospace'

const duration = REDUCED_MOTION ? 0 : 0.35

interface Step {
  title: string
  description: string
  id: string
}

const STEPS: Step[] = [
  {
    id: 'request',
    title: 'Process Requests Memory',
    description:
      'nginx calls malloc() and the CPU generates a guest virtual address (GVA). This is the address the process sees - it has no idea it is running inside a VM.',
  },
  {
    id: 'guest-pt',
    title: 'Guest Page Table Lookup',
    description:
      'The guest OS translates the GVA to a guest physical address (GPA) using its own page tables. The guest kernel thinks GPA 0x0012a000 is real RAM, but it is just another layer of indirection.',
  },
  {
    id: 'ept',
    title: 'Hypervisor EPT/NPT',
    description:
      'The CPU hardware (Intel EPT or AMD NPT) translates the GPA to a host physical address (HPA). This second translation happens in silicon - no VM exit required. The hypervisor controls which host pages each VM can access.',
  },
  {
    id: 'tlb',
    title: 'TLB Caches the Chain',
    description:
      'The TLB (Translation Lookaside Buffer) caches the full GVA-to-HPA mapping. On a TLB hit, the CPU skips both page table walks entirely and goes straight from virtual address to physical RAM in one cycle.',
  },
  {
    id: 'tlb-miss',
    title: 'TLB Miss: The Expensive Path',
    description:
      'On a TLB miss with nested paging, the CPU must walk two page table hierarchies. A 4-level guest table times a 4-level host table means up to 24 memory accesses for a single translation. This is why hugepages matter for VMs - fewer levels to walk.',
  },
]

// Layout constants
const VB_W = 540
const VB_H = 360
const BAR_X = 100
const BAR_W = 280
const BAR_H = 36
const BAR_RX = 3
const LABEL_X = 10

// Vertical positions for 4 layers
const LAYER_Y = [20, 110, 200, 290]

// Layer definitions
const LAYERS = [
  { label: 'Process (nginx)', addr: 'GVA 0x7fff1234', color: BLUE, fill: '#1e3a5f' },
  { label: 'Guest Page Table', addr: 'GPA 0x0012a000', color: GREEN, fill: '#1a2e1a' },
  { label: 'EPT / NPT', addr: 'HPA 0x8f3a1000', color: AMBER, fill: '#2a2000' },
  { label: 'Physical RAM', addr: 'Host DIMM Slot', color: PURPLE, fill: '#2a1a4f' },
]

function Arrow({
  x,
  y1,
  y2,
  color,
  dashed = false,
  label,
}: {
  x: number
  y1: number
  y2: number
  color: string
  dashed?: boolean
  label?: string
}) {
  const midY = (y1 + y2) / 2
  return (
    <motion.g
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      exit={{ opacity: 0 }}
      transition={{ duration }}
    >
      <line
        x1={x}
        y1={y1}
        x2={x}
        y2={y2 - 6}
        stroke={color}
        strokeWidth={1.5}
        strokeDasharray={dashed ? '4 3' : undefined}
        markerEnd={`url(#arrow-${color.replace('#', '')})`}
      />
      {label && (
        <text x={x + 8} y={midY + 3} fill={color} fontSize={8} fontFamily={MONO} fontWeight={500}>
          {label}
        </text>
      )}
    </motion.g>
  )
}

function LayerBar({
  y,
  layer,
  highlight = false,
}: {
  y: number
  layer: (typeof LAYERS)[number]
  highlight?: boolean
}) {
  return (
    <motion.g
      initial={{ opacity: 0, x: -10 }}
      animate={{ opacity: 1, x: 0 }}
      exit={{ opacity: 0, x: 10 }}
      transition={{ duration }}
    >
      <rect
        x={BAR_X}
        y={y}
        width={BAR_W}
        height={BAR_H}
        rx={BAR_RX}
        fill={layer.fill}
        stroke={highlight ? layer.color : layer.color}
        strokeWidth={highlight ? 2 : 1.5}
      />
      <text
        x={BAR_X + LABEL_X}
        y={y + 14}
        fill={layer.color}
        fontSize={10}
        fontFamily={MONO}
        fontWeight={600}
      >
        {layer.label}
      </text>
      <text x={BAR_X + LABEL_X} y={y + 27} fill={TEXT_MUTED} fontSize={9} fontFamily={MONO}>
        {layer.addr}
      </text>
    </motion.g>
  )
}

function StepRequest() {
  return (
    <g>
      <AnimatePresence mode="wait">
        <LayerBar key="process" y={LAYER_Y[0]} layer={LAYERS[0]} highlight />
      </AnimatePresence>
      <text
        x={BAR_X + BAR_W + 12}
        y={LAYER_Y[0] + 22}
        fill={TEXT_MUTED}
        fontSize={8}
        fontFamily={MONO}
      >
        malloc() called
      </text>
    </g>
  )
}

function StepGuestPT() {
  const arrowX = BAR_X + BAR_W / 2
  const labelX = BAR_X + BAR_W + 12
  const midY = (LAYER_Y[0] + BAR_H + LAYER_Y[1]) / 2
  return (
    <g>
      <LayerBar y={LAYER_Y[0]} layer={LAYERS[0]} />
      <Arrow x={arrowX} y1={LAYER_Y[0] + BAR_H} y2={LAYER_Y[1]} color={GREEN} />
      {/* Right-side annotation */}
      <motion.text
        x={labelX}
        y={midY}
        fill={GREEN}
        fontSize={8}
        fontFamily={MONO}
        fontWeight={500}
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        transition={{ duration }}
      >
        Guest Page Tables (CR3)
      </motion.text>
      <motion.text
        x={labelX}
        y={midY + 12}
        fill={TEXT_MUTED}
        fontSize={7}
        fontFamily={MONO}
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        transition={{ duration }}
      >
        translate GVA → GPA
      </motion.text>
      <LayerBar y={LAYER_Y[1]} layer={LAYERS[1]} highlight />
    </g>
  )
}

function StepEPT() {
  const arrowX = BAR_X + BAR_W / 2
  const labelX = BAR_X + BAR_W
  const midY12 = (LAYER_Y[1] + BAR_H + LAYER_Y[2]) / 2
  return (
    <g>
      <LayerBar y={LAYER_Y[0]} layer={LAYERS[0]} />
      <Arrow x={arrowX} y1={LAYER_Y[0] + BAR_H} y2={LAYER_Y[1]} color={GREEN} />
      <LayerBar y={LAYER_Y[1]} layer={LAYERS[1]} />
      <Arrow x={arrowX} y1={LAYER_Y[1] + BAR_H} y2={LAYER_Y[2]} color={AMBER} />
      {/* Right-side annotation */}
      <motion.text
        x={labelX}
        y={midY12}
        fill={AMBER}
        fontSize={8}
        fontFamily={MONO}
        fontWeight={500}
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        transition={{ duration }}
      >
        Extended Page Tables (EPT)
      </motion.text>
      <motion.text
        x={labelX}
        y={midY12 + 12}
        fill={TEXT_MUTED}
        fontSize={7}
        fontFamily={MONO}
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        transition={{ duration }}
      >
        hardware walk GPA → HPA
      </motion.text>
      <LayerBar y={LAYER_Y[2]} layer={LAYERS[2]} highlight />
      <Arrow x={arrowX} y1={LAYER_Y[2] + BAR_H} y2={LAYER_Y[3]} color={PURPLE} />
      <LayerBar y={LAYER_Y[3]} layer={LAYERS[3]} />
    </g>
  )
}

function StepTLB() {
  const arrowX = BAR_X + BAR_W / 2
  const fastX = BAR_X + BAR_W + 30
  return (
    <g>
      <LayerBar y={LAYER_Y[0]} layer={LAYERS[0]} />
      {/* Dimmed middle layers */}
      <g opacity={0.3}>
        <LayerBar y={LAYER_Y[1]} layer={LAYERS[1]} />
        <LayerBar y={LAYER_Y[2]} layer={LAYERS[2]} />
        <Arrow x={arrowX} y1={LAYER_Y[0] + BAR_H} y2={LAYER_Y[1]} color={GREEN} />
        <Arrow x={arrowX} y1={LAYER_Y[1] + BAR_H} y2={LAYER_Y[2]} color={AMBER} />
        <Arrow x={arrowX} y1={LAYER_Y[2] + BAR_H} y2={LAYER_Y[3]} color={PURPLE} />
      </g>
      <LayerBar y={LAYER_Y[3]} layer={LAYERS[3]} />
      {/* TLB fast path */}
      <motion.line
        x1={fastX}
        y1={LAYER_Y[0] + BAR_H / 2}
        x2={fastX}
        y2={LAYER_Y[3] + BAR_H / 2 - 6}
        stroke={BLUE}
        strokeWidth={2}
        strokeDasharray="6 3"
        markerEnd="url(#arrow-60a5fa)"
        initial={{ pathLength: 0, opacity: 0 }}
        animate={{ pathLength: 1, opacity: 1 }}
        transition={{ duration: REDUCED_MOTION ? 0 : 0.6 }}
      />
      <motion.text
        x={fastX + 8}
        y={(LAYER_Y[0] + LAYER_Y[3]) / 2 + BAR_H / 2}
        fill={BLUE}
        fontSize={9}
        fontFamily={MONO}
        fontWeight={600}
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        transition={{ duration, delay: REDUCED_MOTION ? 0 : 0.3 }}
      >
        TLB hit
      </motion.text>
      <motion.text
        x={fastX + 8}
        y={(LAYER_Y[0] + LAYER_Y[3]) / 2 + BAR_H / 2 + 12}
        fill={TEXT_MUTED}
        fontSize={7}
        fontFamily={MONO}
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        transition={{ duration, delay: REDUCED_MOTION ? 0 : 0.3 }}
      >
        1 cycle
      </motion.text>
    </g>
  )
}

function StepTLBMiss() {
  // Show a 2D grid: guest levels x host levels
  const guestLevels = ['PML4', 'PDPT', 'PD', 'PT']
  const hostLevels = ['EPT4', 'EPT3', 'EPT2', 'EPT1']
  const cellW = 48
  const cellH = 24
  const gridX = 70
  const gridY = 30
  const labelOffsetX = gridX - 4
  const labelOffsetY = gridY - 4

  let accessCount = 0
  const cells: { gx: number; hy: number; num: number }[] = []
  for (let g = 0; g < 4; g++) {
    for (let h = 0; h < 4; h++) {
      accessCount++
      cells.push({ gx: g, hy: h, num: accessCount })
    }
    // +1 for reading the guest entry itself at each level
    if (g < 3) {
      accessCount++
      cells.push({ gx: g, hy: -1, num: accessCount })
    }
  }
  // Final data access
  accessCount++

  return (
    <g>
      {/* Column headers - host EPT levels */}
      {hostLevels.map((label, i) => (
        <text
          key={label}
          x={gridX + i * cellW + cellW / 2}
          y={labelOffsetY}
          textAnchor="middle"
          fill={AMBER}
          fontSize={8}
          fontFamily={MONO}
          fontWeight={500}
        >
          {label}
        </text>
      ))}
      {/* Row headers - guest levels */}
      {guestLevels.map((label, i) => (
        <text
          key={label}
          x={labelOffsetX}
          y={gridY + i * (cellH + 4) + cellH / 2 + 3}
          textAnchor="end"
          fill={GREEN}
          fontSize={8}
          fontFamily={MONO}
          fontWeight={500}
        >
          {label}
        </text>
      ))}
      {/* Grid cells */}
      {cells.map((cell, idx) => {
        if (cell.hy === -1) {
          // Guest entry read - spans full row as a small marker
          return (
            <motion.g
              key={`read-${idx}`}
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              transition={{
                duration: REDUCED_MOTION ? 0 : 0.03,
                delay: REDUCED_MOTION ? 0 : idx * 0.04,
              }}
            >
              <rect
                x={gridX + 4 * cellW + 8}
                y={gridY + cell.gx * (cellH + 4)}
                width={cellW - 4}
                height={cellH}
                rx={2}
                fill="#1a0a0a"
                stroke={RED}
                strokeWidth={1}
              />
              <text
                x={gridX + 4 * cellW + 8 + (cellW - 4) / 2}
                y={gridY + cell.gx * (cellH + 4) + cellH / 2 + 3}
                textAnchor="middle"
                fill={RED}
                fontSize={8}
                fontFamily={MONO}
                fontWeight={600}
              >
                #{cell.num}
              </text>
            </motion.g>
          )
        }
        return (
          <motion.g
            key={`cell-${idx}`}
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            transition={{
              duration: REDUCED_MOTION ? 0 : 0.03,
              delay: REDUCED_MOTION ? 0 : idx * 0.04,
            }}
          >
            <rect
              x={gridX + cell.hy * cellW}
              y={gridY + cell.gx * (cellH + 4)}
              width={cellW - 4}
              height={cellH}
              rx={2}
              fill="#1a0a0a"
              stroke={RED}
              strokeWidth={1}
              opacity={0.8}
            />
            <text
              x={gridX + cell.hy * cellW + (cellW - 4) / 2}
              y={gridY + cell.gx * (cellH + 4) + cellH / 2 + 3}
              textAnchor="middle"
              fill={RED}
              fontSize={8}
              fontFamily={MONO}
              fontWeight={600}
            >
              #{cell.num}
            </text>
          </motion.g>
        )
      })}
      {/* Read column header */}
      <text
        x={gridX + 4 * cellW + 8 + (cellW - 4) / 2}
        y={labelOffsetY}
        textAnchor="middle"
        fill={RED}
        fontSize={8}
        fontFamily={MONO}
        fontWeight={500}
      >
        Read
      </text>
      {/* Total count */}
      <motion.text
        x={VB_W / 2}
        y={gridY + 4 * (cellH + 4) + 24}
        textAnchor="middle"
        fill={RED}
        fontSize={11}
        fontFamily={MONO}
        fontWeight={600}
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        transition={{ duration, delay: REDUCED_MOTION ? 0 : 0.8 }}
      >
        Up to {accessCount} memory accesses for one translation
      </motion.text>
      <motion.text
        x={VB_W / 2}
        y={gridY + 4 * (cellH + 4) + 40}
        textAnchor="middle"
        fill={TEXT_MUTED}
        fontSize={9}
        fontFamily={MONO}
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        transition={{ duration, delay: REDUCED_MOTION ? 0 : 1 }}
      >
        vs. 4 accesses on bare metal
      </motion.text>
    </g>
  )
}

const STEP_RENDERERS = [StepRequest, StepGuestPT, StepEPT, StepTLB, StepTLBMiss]

export function MemoryAddressTranslation() {
  const [step, setStep] = useState(0)
  const current = STEPS[step]
  const StepRenderer = STEP_RENDERERS[step]

  return (
    <div
      style={{
        background: CARD_BG,
        border: `1px solid ${BORDER}`,
        padding: '1.5rem',
        marginTop: '1.5rem',
        marginBottom: '1.5rem',
        fontFamily: MONO,
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

      {/* SVG visualization */}
      <div style={{ overflowX: 'auto' }}>
        <svg
          viewBox={`0 0 ${VB_W} ${VB_H}`}
          style={{
            width: '100%',
            maxWidth: 600,
            minWidth: 320,
            display: 'block',
            margin: '0 auto',
          }}
        >
          {/* Arrow marker definitions */}
          <defs>
            {[BLUE, GREEN, AMBER, PURPLE, RED].map((color) => (
              <marker
                key={color}
                id={`arrow-${color.replace('#', '')}`}
                viewBox="0 0 10 10"
                refX="9"
                refY="5"
                markerWidth={6}
                markerHeight={6}
                orient="auto-start-reverse"
              >
                <path d="M 0 0 L 10 5 L 0 10 z" fill={color} />
              </marker>
            ))}
          </defs>
          <AnimatePresence mode="wait">
            <motion.g
              key={current.id}
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              exit={{ opacity: 0 }}
              transition={{ duration }}
            >
              <StepRenderer />
            </motion.g>
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
              width: 24,
              height: 24,
              borderRadius: '50%',
              background: 'transparent',
              border: 'none',
              cursor: 'pointer',
              padding: 0,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
            }}
          >
            <span
              style={{
                width: 6,
                height: 6,
                borderRadius: '50%',
                background: i === step ? BLUE : BORDER,
                display: 'block',
              }}
            />
          </button>
        ))}
      </div>
    </div>
  )
}
