import { useState, useEffect } from 'react'
import { motion } from 'framer-motion'
import type { DiagramPhase } from '~/lib/diagram-phases'

// Check reduced motion preference
const prefersReducedMotion =
  typeof window !== 'undefined' && window.matchMedia('(prefers-reduced-motion: reduce)').matches

const fast = prefersReducedMotion ? 0 : 0.4
const med = prefersReducedMotion ? 0 : 0.6

// Colors - matched to docs architecture-diagram.tsx (TierBox + Arrow)
const BLUE = '#60a5fa'
const TEXT_MUTED = '#737373' // neutral-500
const BORDER = '#262626' // neutral-800
const LINE_COLOR = '#404040' // neutral-700
const CARD_BG = '#171717' // neutral-900
const DOT_FILL = '#525252' // neutral-600

const SPINNER_FRAMES = ['⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏']

// -------------------------------------------------------------------
// Terminal window dots (reusable)
// -------------------------------------------------------------------
function WindowDots({ x, y }: { x: number; y: number }) {
  return (
    <>
      <rect x={x} y={y} width={4} height={4} rx={1} fill={DOT_FILL} />
      <rect x={x + 8} y={y} width={4} height={4} rx={1} fill={DOT_FILL} />
      <rect x={x + 16} y={y} width={4} height={4} rx={1} fill={DOT_FILL} />
    </>
  )
}

// -------------------------------------------------------------------
// Arrow - solid line with triangle arrowhead, matches docs Arrow style
// -------------------------------------------------------------------
function Arrow({
  x1,
  y1,
  x2,
  y2,
  opacity,
  label,
}: {
  x1: number
  y1: number
  x2: number
  y2: number
  opacity: number
  label?: string
}) {
  const dx = x2 - x1
  const dy = y2 - y1
  const len = Math.sqrt(dx * dx + dy * dy)
  // Unit vector along the line
  const ux = dx / len
  const uy = dy / len
  // Perpendicular vector
  const px = -uy
  const py = ux

  // Arrowhead at end point
  const headLen = 5
  const headWidth = 3
  const tipX = x2
  const tipY = y2
  const baseX = x2 - ux * headLen
  const baseY = y2 - uy * headLen
  const leftX = baseX + px * headWidth
  const leftY = baseY + py * headWidth
  const rightX = baseX - px * headWidth
  const rightY = baseY - py * headWidth

  // Label position: midpoint, offset perpendicular to line
  const mx = (x1 + x2) / 2
  const my = (y1 + y2) / 2

  return (
    <motion.g animate={{ opacity }} transition={{ duration: fast }}>
      <line x1={x1} y1={y1} x2={baseX} y2={baseY} stroke={LINE_COLOR} strokeWidth={1} />
      <polygon points={`${tipX},${tipY} ${leftX},${leftY} ${rightX},${rightY}`} fill={LINE_COLOR} />
      {label && (
        <text
          x={mx + 8}
          y={my}
          textAnchor="start"
          dominantBaseline="central"
          fill={TEXT_MUTED}
          fontSize={10}
          fontFamily="ui-monospace, SFMono-Regular, monospace"
        >
          {label}
        </text>
      )}
    </motion.g>
  )
}

// -------------------------------------------------------------------
// Node box - terminal-window style, matches docs TierBox
// -------------------------------------------------------------------
function NodeBox({
  x,
  y,
  w,
  h,
  label,
  sublabel,
  subtitle,
  opacity,
  highlight = false,
  borderColor,
  labelColor = '#ffffff',
}: {
  x: number
  y: number
  w: number
  h: number
  label: string
  sublabel?: string
  subtitle?: string
  opacity: number
  highlight?: boolean
  borderColor?: string
  labelColor?: string
}) {
  const stroke = highlight ? 'rgba(96,165,250,0.4)' : (borderColor ?? BORDER)
  const fill = highlight ? 'rgba(96,165,250,0.05)' : CARD_BG

  return (
    <motion.g animate={{ opacity }} transition={{ duration: fast }}>
      <rect x={x} y={y} width={w} height={h} rx={0} fill={fill} stroke={stroke} strokeWidth={1} />
      {/* Window dots */}
      <WindowDots x={x + 10} y={y + 10} />
      {/* Sublabel next to dots */}
      {sublabel && (
        <text
          x={x + 34}
          y={y + 13}
          dominantBaseline="central"
          fill={TEXT_MUTED}
          fontSize={9}
          fontFamily="ui-monospace, SFMono-Regular, monospace"
        >
          {sublabel}
        </text>
      )}
      {/* Title */}
      <text
        x={x + 12}
        y={y + 32}
        dominantBaseline="central"
        fill={labelColor}
        fontSize={12}
        fontFamily="ui-monospace, SFMono-Regular, monospace"
        fontWeight={500}
      >
        {label}
      </text>
      {/* Subtitle */}
      {subtitle && (
        <text
          x={x + 12}
          y={y + 46}
          dominantBaseline="central"
          fill={TEXT_MUTED}
          fontSize={10}
          fontFamily="ui-monospace, SFMono-Regular, monospace"
        >
          {subtitle}
        </text>
      )}
    </motion.g>
  )
}

// -------------------------------------------------------------------
// Playbook card (appears during create-playbook phase)
// -------------------------------------------------------------------
type YamlLine =
  | { type: 'plain'; indent: number; text: string }
  | { type: 'kv'; indent: number; key: string; value?: string }

const YAML_HEADER: YamlLine[] = [
  { type: 'plain', indent: 0, text: '---' },
  { type: 'kv', indent: 0, key: '- name:', value: ' fix-nginx-502' },
  { type: 'kv', indent: 1, key: 'hosts:', value: ' web-prod-01' },
  { type: 'kv', indent: 1, key: 'become:', value: ' true' },
  { type: 'kv', indent: 1, key: 'tasks:' },
]

const YAML_TASK1: YamlLine[] = [
  { type: 'kv', indent: 2, key: '- name:', value: ' Restart app' },
  { type: 'kv', indent: 3, key: 'systemd:' },
  { type: 'kv', indent: 4, key: 'name:', value: ' app' },
  { type: 'kv', indent: 4, key: 'state:', value: ' restarted' },
]

const YAML_TASK2: YamlLine[] = [
  { type: 'kv', indent: 2, key: '- name:', value: ' Verify nginx' },
  { type: 'kv', indent: 3, key: 'uri:' },
  { type: 'kv', indent: 4, key: 'url:', value: ' localhost/health' },
]

function YamlLineText({
  line,
  baseX,
  y,
  visible,
  delay,
}: {
  line: YamlLine
  baseX: number
  y: number
  visible: boolean
  delay: number
}) {
  const x = baseX + line.indent * 12

  return (
    <motion.g
      initial={{ opacity: 0 }}
      animate={{ opacity: visible ? 1 : 0 }}
      transition={{ duration: fast, delay: prefersReducedMotion ? 0 : delay }}
    >
      {line.type === 'kv' ? (
        <text x={x} y={y} fontSize={10} fontFamily="ui-monospace, SFMono-Regular, monospace">
          <tspan fill={BLUE}>{line.key}</tspan>
          {line.value && <tspan fill="#ffffff">{line.value}</tspan>}
        </text>
      ) : (
        <text
          x={x}
          y={y}
          fill={TEXT_MUTED}
          fontSize={10}
          fontFamily="ui-monospace, SFMono-Regular, monospace"
        >
          {line.text}
        </text>
      )}
    </motion.g>
  )
}

function PlaybookCard({ phase }: { phase: DiagramPhase }) {
  const showCard = [
    'create-playbook',
    'add-task-1',
    'add-task-2',
    'destroy-sandbox',
    'done',
    'cleanup',
  ].includes(phase)
  const showHeader = showCard
  const showTask1 = ['add-task-1', 'add-task-2', 'destroy-sandbox', 'done', 'cleanup'].includes(
    phase
  )
  const showTask2 = ['add-task-2', 'destroy-sandbox', 'done', 'cleanup'].includes(phase)

  const cardX = 340
  const cardY = 10
  const cardW = 250
  const cardH = 280
  const baseX = cardX + 10
  const startY = cardY + 52
  const lineH = 14
  const stagger = 0.12

  let lineIdx = 0

  return (
    <motion.g animate={{ opacity: showCard ? 1 : 0 }} transition={{ duration: med }}>
      {/* Card background */}
      <rect
        x={cardX}
        y={cardY}
        width={cardW}
        height={cardH}
        rx={0}
        fill="rgba(96,165,250,0.05)"
        stroke="rgba(96,165,250,0.4)"
        strokeWidth={1}
      />

      {/* Window dots + sublabel */}
      <WindowDots x={cardX + 10} y={cardY + 10} />
      <text
        x={cardX + 34}
        y={cardY + 13}
        dominantBaseline="central"
        fill={TEXT_MUTED}
        fontSize={9}
        fontFamily="ui-monospace, SFMono-Regular, monospace"
      >
        playbook
      </text>

      {/* Title */}
      <text
        x={cardX + 12}
        y={cardY + 30}
        dominantBaseline="central"
        fill="#ffffff"
        fontSize={11}
        fontFamily="ui-monospace, SFMono-Regular, monospace"
        fontWeight={500}
      >
        fix-nginx-502.yml
      </text>
      <line
        x1={cardX + 8}
        y1={cardY + 42}
        x2={cardX + cardW - 8}
        y2={cardY + 42}
        stroke={BORDER}
        strokeWidth={0.5}
      />

      {/* YAML header lines */}
      {YAML_HEADER.map((line, i) => {
        const idx = lineIdx++
        return (
          <YamlLineText
            key={`h-${i}`}
            line={line}
            baseX={baseX}
            y={startY + idx * lineH}
            visible={showHeader}
            delay={idx * stagger}
          />
        )
      })}

      {/* Task 1 lines */}
      {YAML_TASK1.map((line, i) => {
        const idx = lineIdx++
        return (
          <YamlLineText
            key={`t1-${i}`}
            line={line}
            baseX={baseX}
            y={startY + idx * lineH}
            visible={showTask1}
            delay={i * stagger}
          />
        )
      })}

      {/* Task 2 lines */}
      {YAML_TASK2.map((line, i) => {
        const idx = lineIdx++
        return (
          <YamlLineText
            key={`t2-${i}`}
            line={line}
            baseX={baseX}
            y={startY + idx * lineH}
            visible={showTask2}
            delay={i * stagger}
          />
        )
      })}
    </motion.g>
  )
}

// -------------------------------------------------------------------
// Braille spinner with label (tool name, "Thinking...", etc.)
// -------------------------------------------------------------------
function BrailleSpinner({
  x,
  y,
  active,
  label,
}: {
  x: number
  y: number
  active: boolean
  label: string
}) {
  const [frame, setFrame] = useState(0)

  useEffect(() => {
    if (!active) return
    if (prefersReducedMotion) return
    const id = setInterval(() => {
      setFrame((f) => (f + 1) % SPINNER_FRAMES.length)
    }, 300)
    return () => {
      clearInterval(id)
      setFrame(0)
    }
  }, [active])

  return (
    <motion.g animate={{ opacity: active ? 1 : 0 }} transition={{ duration: fast }}>
      <text
        x={x}
        y={y}
        dominantBaseline="central"
        fill={BLUE}
        fontSize={11}
        fontFamily="ui-monospace, SFMono-Regular, monospace"
      >
        {SPINNER_FRAMES[frame]}
      </text>
      <text
        x={x + 14}
        y={y}
        dominantBaseline="central"
        fill={TEXT_MUTED}
        fontSize={10}
        fontFamily="ui-monospace, SFMono-Regular, monospace"
        fontStyle="italic"
      >
        {label}
      </text>
    </motion.g>
  )
}

// -------------------------------------------------------------------
// Main component
// -------------------------------------------------------------------
export function ArchitectureAnimation({ phase }: { phase: DiagramPhase }) {
  // Derive visibility/activity from phase
  const isIdle = phase === 'idle' || phase === 'reset'

  const cliActive = !isIdle
  const daemonActive = !isIdle && phase !== 'user-input'

  const readSourcePhases = ['read-source-1', 'read-source-2', 'read-source-3', 'read-source-4']
  const sourceActive = readSourcePhases.includes(phase)

  const sandboxVisible = [
    'creating-sandbox',
    'sandbox-cmd-1',
    'sandbox-cmd-2',
    'sandbox-cmd-3',
    'thinking-3',
    'create-playbook',
    'add-task-1',
    'add-task-2',
    'done',
  ].includes(phase)
  const sandboxDestroying = phase === 'destroy-sandbox'

  const isThinking = phase === 'thinking-1' || phase === 'thinking-2' || phase === 'thinking-3'

  // Tool name shown next to sandbox box during sandbox commands
  const sandboxToolActive = ['sandbox-cmd-1', 'sandbox-cmd-2', 'sandbox-cmd-3'].includes(phase)
  const sandboxToolName = phase === 'creating-sandbox' ? 'create_sandbox' : 'run_command'
  const sandboxCreating = phase === 'creating-sandbox'

  // Tool name shown next to source VM box during read phases
  const sourceToolName = phase === 'read-source-3' ? 'read_source_file' : 'run_source_command'

  // CLI -> Daemon connection
  const cliDaemonActive = daemonActive
  // Daemon -> Source connection
  const daemonSourceActive = sourceActive

  const playbookActive = [
    'create-playbook',
    'add-task-1',
    'add-task-2',
    'destroy-sandbox',
    'done',
    'cleanup',
  ].includes(phase)

  // Layout coordinates
  const cli = { x: 60, y: 10, w: 200, h: 70 }
  const daemon = { x: 30, y: 110, w: 260, h: 195 }
  const sandbox = { x: 55, y: 210, w: 190, h: 65 }
  const source = { x: 45, y: 340, w: 190, h: 65 }

  return (
    <div className="mt-6">
      <svg viewBox="0 0 600 420" className="h-full w-full" style={{ maxHeight: 440 }}>
        {/* Background */}
        <rect width="600" height="420" fill="transparent" />

        {/* CLI -> Daemon arrow */}
        <Arrow x1={160} y1={80} x2={160} y2={110} opacity={cliDaemonActive ? 0.8 : 0.2} />

        {/* Daemon -> Source arrow */}
        <Arrow
          x1={140}
          y1={305}
          x2={140}
          y2={340}
          opacity={daemonSourceActive || daemonActive ? (sourceActive ? 0.8 : 0.3) : 0.15}
          label={sourceActive ? 'read-only' : undefined}
        />

        {/* CLI -> Playbook arrow (horizontal) */}
        <Arrow x1={260} y1={45} x2={340} y2={45} opacity={playbookActive ? 0.6 : 0} />

        {/* CLI node */}
        <NodeBox
          {...cli}
          label="Fluid CLI/Web"
          sublabel="client"
          opacity={cliActive ? 1 : 0.3}
          labelColor={cliActive ? '#ffffff' : TEXT_MUTED}
        />

        {/* Braille spinner inside CLI box */}
        <BrailleSpinner x={cli.x + 12} y={cli.y + 52} active={isThinking} label="Thinking..." />

        {/* Daemon container - terminal-window style */}
        <motion.g animate={{ opacity: daemonActive ? 1 : 0.3 }} transition={{ duration: fast }}>
          <rect
            x={daemon.x}
            y={daemon.y}
            width={daemon.w}
            height={daemon.h}
            rx={0}
            fill={daemonActive ? 'rgba(96,165,250,0.05)' : CARD_BG}
            stroke={daemonActive ? 'rgba(96,165,250,0.4)' : BORDER}
            strokeWidth={1}
          />
          {/* Window dots + sublabel */}
          <WindowDots x={daemon.x + 10} y={daemon.y + 10} />
          <text
            x={daemon.x + 34}
            y={daemon.y + 13}
            dominantBaseline="central"
            fill={TEXT_MUTED}
            fontSize={9}
            fontFamily="ui-monospace, SFMono-Regular, monospace"
          >
            fluid-daemon
          </text>
          {/* Title */}
          <text
            x={daemon.x + 12}
            y={daemon.y + 32}
            dominantBaseline="central"
            fill={daemonActive ? '#ffffff' : TEXT_MUTED}
            fontSize={12}
            fontFamily="ui-monospace, SFMono-Regular, monospace"
            fontWeight={500}
          >
            sandbox-host-1
          </text>
        </motion.g>

        {/* Source VM node */}
        <NodeBox
          {...source}
          label="web-prod-01"
          sublabel="source VM"
          opacity={sourceActive ? 1 : daemonActive ? 0.5 : 0.2}
          labelColor={sourceActive ? '#ffffff' : TEXT_MUTED}
        />

        {/* Tool spinner inside source VM box */}
        <BrailleSpinner
          x={source.x + 12}
          y={source.y + 50}
          active={sourceActive}
          label={sourceToolName}
        />

        {/* Sandbox node - inside daemon container */}
        <motion.g
          animate={{
            opacity: sandboxDestroying ? 0 : sandboxVisible ? 1 : 0,
            scale: sandboxVisible ? 1 : 0.8,
          }}
          transition={{ duration: med }}
          style={{
            transformOrigin: `${sandbox.x + sandbox.w / 2}px ${sandbox.y + sandbox.h / 2}px`,
          }}
        >
          <NodeBox
            {...sandbox}
            label="sbx-a1b2c3"
            sublabel="sandbox"
            opacity={1}
            labelColor="#ffffff"
          />
        </motion.g>

        {/* Tool spinner inside sandbox box */}
        <BrailleSpinner
          x={sandbox.x + 12}
          y={sandbox.y + 50}
          active={sandboxToolActive || sandboxCreating}
          label={sandboxToolName}
        />

        {/* Playbook card */}
        <PlaybookCard phase={phase} />
      </svg>
    </div>
  )
}
