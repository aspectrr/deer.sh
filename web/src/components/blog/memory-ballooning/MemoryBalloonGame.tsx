import { useState, useRef, useCallback } from 'react'

const BLUE = '#60a5fa'
const GREEN = '#4ade80'
const RED = '#f87171'
const AMBER = '#fbbf24'
const PURPLE = '#a855f7'
const TEXT_MUTED = '#737373'
const BORDER = '#262626'
const CARD_BG = '#171717'
const MONO = 'ui-monospace, SFMono-Regular, monospace'
const DOT_FILL = '#525252'

const TOTAL_RAM = 8192

interface VM {
  id: string
  name: string
  color: string
  fill: string
  used: number
  allocated: number
  minAlloc: number
  baseGPA: number
  baseHPA: number
}

const INITIAL_VMS: VM[] = [
  {
    id: 'web',
    name: 'web-server',
    color: BLUE,
    fill: '#1e3a5f',
    used: 1200,
    allocated: 2048,
    minAlloc: 512,
    baseGPA: 0x00100000,
    baseHPA: 0x8f000000,
  },
  {
    id: 'db',
    name: 'database',
    color: GREEN,
    fill: '#1a2e1a',
    used: 2800,
    allocated: 3072,
    minAlloc: 1024,
    baseGPA: 0x00200000,
    baseHPA: 0xa0000000,
  },
  {
    id: 'dev',
    name: 'dev-env',
    color: PURPLE,
    fill: '#2a1a4f',
    used: 800,
    allocated: 2048,
    minAlloc: 256,
    baseGPA: 0x00300000,
    baseHPA: 0xc0000000,
  },
]

interface LogEntry {
  id: number
  text: string
  color: string
}

function WindowDots({ x, y }: { x: number; y: number }) {
  return (
    <>
      <rect x={x} y={y} width={4} height={4} rx={1} fill={DOT_FILL} />
      <rect x={x + 8} y={y} width={4} height={4} rx={1} fill={DOT_FILL} />
      <rect x={x + 16} y={y} width={4} height={4} rx={1} fill={DOT_FILL} />
    </>
  )
}

function hexAddr(base: number, offsetMB: number): string {
  return '0x' + (base + offsetMB * 0x1000).toString(16).padStart(8, '0')
}

function getAddrEntries(vm: VM): { gpa: string; hpa: string; reclaimed: boolean }[] {
  const entries = []
  const chunkSize = Math.floor(vm.allocated / 4) || 256
  for (let i = 0; i < 4; i++) {
    const offsetMB = i * chunkSize
    entries.push({
      gpa: hexAddr(vm.baseGPA, offsetMB),
      hpa: hexAddr(vm.baseHPA, offsetMB),
      reclaimed: offsetMB >= vm.allocated,
    })
  }
  return entries
}

// Lever layout constants
const LEVER_TRACK_H = 140
const LEVER_HANDLE_W = 36
const LEVER_HANDLE_H = 18
const LEVER_TRACK_W = 4

function Lever({
  x,
  y,
  vm,
  maxAlloc,
  onAllocChange,
}: {
  x: number
  y: number
  vm: VM
  maxAlloc: number
  onAllocChange: (vmId: string, newAlloc: number) => void
}) {
  const dragging = useRef(false)
  const svgRef = useRef<SVGSVGElement | null>(null)

  // Position: top = maxAlloc, bottom = minAlloc
  const range = maxAlloc - vm.minAlloc
  const fraction = range > 0 ? (vm.allocated - vm.minAlloc) / range : 0
  const handleY = y + LEVER_TRACK_H - fraction * LEVER_TRACK_H - LEVER_HANDLE_H / 2

  // Convert screen Y to allocation using SVG coordinate transform
  const pointerToAlloc = useCallback(
    (clientY: number): number => {
      if (!svgRef.current || range <= 0) return vm.allocated
      const ctm = svgRef.current.getScreenCTM()
      if (!ctm) return vm.allocated

      // Convert screen Y to SVG Y via the inverse of the CTM
      const svgY = (clientY - ctm.f) / ctm.d

      // Map: top of track (y) = maxAlloc, bottom (y + LEVER_TRACK_H) = minAlloc
      const frac = 1 - Math.max(0, Math.min(1, (svgY - y) / LEVER_TRACK_H))
      const raw = vm.minAlloc + frac * range
      // Snap to 64MB increments
      return Math.round(raw / 64) * 64
    },
    [vm.allocated, vm.minAlloc, range, y]
  )

  const onPointerDown = useCallback((e: React.PointerEvent) => {
    e.preventDefault()
    ;(e.target as SVGElement).setPointerCapture(e.pointerId)
    dragging.current = true
    // Walk up to find the SVG element
    let el: Element | null = e.target as Element
    while (el && el.tagName !== 'svg') el = el.parentElement
    svgRef.current = el as SVGSVGElement | null
  }, [])

  const onPointerMove = useCallback(
    (e: React.PointerEvent) => {
      if (!dragging.current) return
      e.preventDefault()
      const newAlloc = pointerToAlloc(e.clientY)
      if (newAlloc !== vm.allocated) {
        onAllocChange(vm.id, newAlloc)
      }
    },
    [vm.id, vm.allocated, pointerToAlloc, onAllocChange]
  )

  const onPointerUp = useCallback((e: React.PointerEvent) => {
    dragging.current = false
    ;(e.target as SVGElement).releasePointerCapture(e.pointerId)
  }, [])

  return (
    <g>
      {/* Track */}
      <rect
        x={x + LEVER_HANDLE_W / 2 - LEVER_TRACK_W / 2}
        y={y}
        width={LEVER_TRACK_W}
        height={LEVER_TRACK_H}
        rx={2}
        fill="#333"
      />
      {/* Filled portion */}
      <rect
        x={x + LEVER_HANDLE_W / 2 - LEVER_TRACK_W / 2}
        y={handleY + LEVER_HANDLE_H / 2}
        width={LEVER_TRACK_W}
        height={Math.max(0, y + LEVER_TRACK_H - (handleY + LEVER_HANDLE_H / 2))}
        rx={2}
        fill={vm.color}
        opacity={0.4}
      />
      {/* Handle */}
      <rect
        x={x}
        y={handleY}
        width={LEVER_HANDLE_W}
        height={LEVER_HANDLE_H}
        rx={4}
        fill={vm.fill}
        stroke={vm.color}
        strokeWidth={1.5}
        style={{ cursor: 'grab', touchAction: 'none' }}
        onPointerDown={onPointerDown}
        onPointerMove={onPointerMove}
        onPointerUp={onPointerUp}
        onPointerCancel={onPointerUp}
      />
      {/* Grip lines */}
      {[0, 4, 8].map((offset) => (
        <line
          key={offset}
          x1={x + 10}
          y1={handleY + 5 + offset}
          x2={x + LEVER_HANDLE_W - 10}
          y2={handleY + 5 + offset}
          stroke={vm.color}
          strokeWidth={0.5}
          opacity={0.5}
          style={{ pointerEvents: 'none' }}
        />
      ))}
      {/* Labels */}
      <text
        x={x + LEVER_HANDLE_W / 2}
        y={y - 6}
        textAnchor="middle"
        fill={TEXT_MUTED}
        fontSize={7}
        fontFamily={MONO}
      >
        max
      </text>
      <text
        x={x + LEVER_HANDLE_W / 2}
        y={y + LEVER_TRACK_H + 12}
        textAnchor="middle"
        fill={TEXT_MUTED}
        fontSize={7}
        fontFamily={MONO}
      >
        min
      </text>
    </g>
  )
}

export function MemoryBalloonGame() {
  const [vms, setVms] = useState<VM[]>(INITIAL_VMS)
  const [log, setLog] = useState<LogEntry[]>([
    { id: 0, text: '[host] ready - 8192MB total, 1024MB free', color: TEXT_MUTED },
  ])
  const logIdRef = useRef(1)

  const addLog = useCallback((text: string, color: string) => {
    const id = logIdRef.current++
    setLog((prev) => [{ id, text, color }, ...prev].slice(0, 8))
  }, [])

  const handleAllocChange = useCallback(
    (vmId: string, newAlloc: number) => {
      setVms((prev) => {
        const vmIndex = prev.findIndex((v) => v.id === vmId)
        if (vmIndex === -1) return prev
        const vm = prev[vmIndex]
        const delta = newAlloc - vm.allocated

        if (delta === 0) return prev

        const next = [...prev]
        // Compute free from current state, not stale closure
        const currentFree = TOTAL_RAM - prev.reduce((sum, v) => sum + v.allocated, 0)

        if (delta > 0) {
          // Requesting more memory
          let remaining = delta

          // First use free host RAM
          const fromFree = Math.min(remaining, currentFree)
          remaining -= fromFree

          // If still need more, reclaim from other VMs (prefer most unused)
          if (remaining > 0) {
            const others = next
              .map((v, i) => ({
                ...v,
                idx: i,
                unused: v.allocated - v.used,
                reclaimable: v.allocated - Math.max(v.used, v.minAlloc),
              }))
              .filter((v) => v.id !== vmId && v.reclaimable > 0)
              .sort((a, b) => b.unused - a.unused)

            for (const donor of others) {
              if (remaining <= 0) break
              const take = Math.min(remaining, donor.reclaimable)
              const snappedTake = Math.round(take / 64) * 64
              if (snappedTake <= 0) continue
              next[donor.idx] = {
                ...next[donor.idx],
                allocated: next[donor.idx].allocated - snappedTake,
              }
              remaining -= snappedTake
              addLog(`[host] reclaimed ${snappedTake}MB from ${donor.name}`, RED)
            }

            // Cap if we couldn't get enough
            if (remaining > 0) {
              const actualDelta = delta - remaining
              if (actualDelta <= 0) return prev
              next[vmIndex] = { ...next[vmIndex], allocated: vm.allocated + actualDelta }
              addLog(
                `[balloon] ${vm.name}: inflate +${actualDelta}MB (${vm.allocated} -> ${vm.allocated + actualDelta})`,
                vm.color
              )
              return next
            }
          }

          next[vmIndex] = { ...next[vmIndex], allocated: newAlloc }
          addLog(
            `[balloon] ${vm.name}: inflate +${delta}MB (${vm.allocated} -> ${newAlloc})`,
            vm.color
          )
        } else {
          // Releasing memory - cannot go below used or minAlloc
          const clamped = Math.max(newAlloc, vm.used, vm.minAlloc)
          if (clamped === vm.allocated) return prev
          const released = vm.allocated - clamped
          next[vmIndex] = { ...next[vmIndex], allocated: clamped }
          addLog(
            `[balloon] ${vm.name}: deflate -${released}MB (${vm.allocated} -> ${clamped})`,
            AMBER
          )
        }

        return next
      })
    },
    [addLog]
  )

  // SVG layout
  const VB_W = 720
  const VB_H = 500
  const VM_COL_W = 210
  const VM_COL_GAP = 25
  const VM_START_X = (VB_W - (3 * VM_COL_W + 2 * VM_COL_GAP)) / 2
  const VM_Y = 10
  const VM_BOX_H = 260
  const HOST_BAR_Y = VM_Y + VM_BOX_H + 20
  const HOST_BAR_H = 32
  const LOG_Y = HOST_BAR_Y + HOST_BAR_H + 20
  const LOG_H = 120

  const currentFree = TOTAL_RAM - vms.reduce((sum, vm) => sum + vm.allocated, 0)

  // Precompute host bar segments for stable rendering
  const barW = VB_W - 2 * VM_START_X
  const segments: { x: number; w: number; color: string; label: string }[] = []
  let segOffset = 0
  for (const vm of vms) {
    const w = (vm.allocated / TOTAL_RAM) * barW
    segments.push({ x: VM_START_X + segOffset, w, color: vm.color, label: `${vm.allocated}` })
    segOffset += w
  }
  if (currentFree > 0) {
    const freeW = (currentFree / TOTAL_RAM) * barW
    segments.push({ x: VM_START_X + segOffset, w: freeW, color: '#444', label: `${currentFree}` })
  }

  return (
    <div
      style={{
        background: 'black',
        border: `1px solid ${BORDER}`,
        padding: '1rem 1rem 0.5rem',
        marginTop: '1.5rem',
        marginBottom: '1.5rem',
        fontFamily: MONO,
      }}
    >
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          marginBottom: '0.5rem',
          padding: '0 0.25rem',
        }}
      >
        <span style={{ color: AMBER, fontSize: '0.75rem', fontWeight: 600 }}>
          Memory Balloon Simulator
        </span>
        <span style={{ color: TEXT_MUTED, fontSize: '0.7rem' }}>
          Drag levers up to allocate more memory
        </span>
      </div>

      <div style={{ overflowX: 'auto' }}>
        <svg
          viewBox={`0 0 ${VB_W} ${VB_H}`}
          style={{
            width: '100%',
            minWidth: 500,
            display: 'block',
          }}
        >
          {/* VM Columns */}
          {vms.map((vm, i) => {
            const colX = VM_START_X + i * (VM_COL_W + VM_COL_GAP)
            const addrEntries = getAddrEntries(vm)

            // Memory bar inside VM box
            const memBarX = colX + 12
            const memBarY = VM_Y + 30
            const memBarW = 80
            const memBarH = 120
            const usedFraction = vm.used / (vm.allocated || 1)
            const usedH = Math.max(0, usedFraction * memBarH)

            // Balloon
            const balloonCX = memBarX + memBarW / 2
            const balloonCY = memBarY + memBarH + 28
            const balloonRX = 12 + (vm.allocated / TOTAL_RAM) * 25
            const balloonRY = 8 + (vm.allocated / TOTAL_RAM) * 16

            // Lever position
            const leverX = colX + 110
            const leverY = VM_Y + 36

            // Address table
            const addrX = colX + 110
            const addrY = VM_Y + 36 + LEVER_TRACK_H + 24

            // Max this VM could have
            const othersReclaimable = vms
              .filter((v) => v.id !== vm.id)
              .reduce((sum, v) => sum + Math.max(0, v.allocated - Math.max(v.used, v.minAlloc)), 0)
            const maxAlloc = Math.min(
              TOTAL_RAM - vms.filter((v) => v.id !== vm.id).reduce((sum, v) => sum + v.minAlloc, 0),
              vm.allocated + currentFree + othersReclaimable
            )

            return (
              <g key={vm.id}>
                {/* VM box */}
                <rect
                  x={colX}
                  y={VM_Y}
                  width={VM_COL_W}
                  height={VM_BOX_H}
                  rx={4}
                  fill={CARD_BG}
                  stroke={BORDER}
                  strokeWidth={1}
                />
                {/* Window header */}
                <rect x={colX} y={VM_Y} width={VM_COL_W} height={20} rx={4} fill={CARD_BG} />
                <rect x={colX} y={VM_Y + 16} width={VM_COL_W} height={4} fill={CARD_BG} />
                <line
                  x1={colX}
                  y1={VM_Y + 20}
                  x2={colX + VM_COL_W}
                  y2={VM_Y + 20}
                  stroke={BORDER}
                  strokeWidth={1}
                />
                <WindowDots x={colX + 8} y={VM_Y + 8} />
                <text
                  x={colX + 34}
                  y={VM_Y + 14}
                  fill={vm.color}
                  fontSize={9}
                  fontFamily={MONO}
                  fontWeight={500}
                >
                  {vm.name}
                </text>

                {/* Memory bar */}
                <rect
                  x={memBarX}
                  y={memBarY}
                  width={memBarW}
                  height={memBarH}
                  rx={2}
                  fill="#111"
                  stroke={BORDER}
                  strokeWidth={1}
                />
                {/* Allocated portion (full bar background) */}
                <rect
                  x={memBarX}
                  y={memBarY}
                  width={memBarW}
                  height={memBarH}
                  rx={2}
                  fill={vm.fill}
                  opacity={0.3}
                />
                {/* Used portion */}
                <rect
                  x={memBarX}
                  y={memBarY + memBarH - usedH}
                  width={memBarW}
                  height={usedH}
                  rx={2}
                  fill={vm.color}
                  opacity={0.6}
                />
                {/* Memory label */}
                <text
                  x={memBarX + memBarW / 2}
                  y={memBarY + memBarH + 14}
                  textAnchor="middle"
                  fill={vm.color}
                  fontSize={8}
                  fontFamily={MONO}
                  fontWeight={500}
                >
                  {vm.used}MB / {vm.allocated}MB
                </text>

                {/* Balloon */}
                <ellipse
                  cx={balloonCX}
                  cy={balloonCY}
                  rx={balloonRX}
                  ry={balloonRY}
                  fill={vm.color}
                  opacity={0.15}
                  stroke={vm.color}
                  strokeWidth={1}
                  strokeDasharray="3 2"
                />
                <text
                  x={balloonCX}
                  y={balloonCY + 3}
                  textAnchor="middle"
                  fill={vm.color}
                  fontSize={7}
                  fontFamily={MONO}
                  opacity={0.6}
                >
                  balloon
                </text>

                {/* Lever */}
                <Lever
                  x={leverX}
                  y={leverY}
                  vm={vm}
                  maxAlloc={maxAlloc}
                  onAllocChange={handleAllocChange}
                />

                {/* Address mappings (compact) */}
                <text x={addrX} y={addrY} fill={TEXT_MUTED} fontSize={6.5} fontFamily={MONO}>
                  GPA {'->'} HPA
                </text>
                {addrEntries.slice(0, 3).map((entry, j) => {
                  const ey = addrY + 10 + j * 11
                  const isReclaimed = j * Math.floor(vm.allocated / 4) >= vm.allocated
                  return (
                    <g key={j}>
                      <text
                        x={addrX}
                        y={ey}
                        fill={isReclaimed ? RED : vm.color}
                        fontSize={6}
                        fontFamily={MONO}
                        opacity={isReclaimed ? 0.5 : 0.8}
                      >
                        {isReclaimed ? `${entry.gpa} [reclaimed]` : `${entry.gpa} ${entry.hpa}`}
                      </text>
                    </g>
                  )
                })}
              </g>
            )
          })}

          {/* Host RAM Bar */}
          <text x={VM_START_X} y={HOST_BAR_Y - 4} fill={TEXT_MUTED} fontSize={9} fontFamily={MONO}>
            Host Physical RAM ({TOTAL_RAM}MB)
          </text>
          <rect
            x={VM_START_X}
            y={HOST_BAR_Y}
            width={barW}
            height={HOST_BAR_H}
            rx={3}
            fill="#111"
            stroke={BORDER}
            strokeWidth={1}
          />
          {/* Colored segments for each VM + free */}
          {segments.map((seg, i) => (
            <g key={i}>
              <rect
                x={seg.x}
                y={HOST_BAR_Y}
                width={Math.max(0, seg.w)}
                height={HOST_BAR_H}
                rx={i === 0 ? 3 : 0}
                fill={seg.color}
                opacity={seg.color === '#444' ? 0.3 : 0.25}
              />
              {seg.w > 40 && (
                <text
                  x={seg.x + seg.w / 2}
                  y={HOST_BAR_Y + HOST_BAR_H / 2 + 3}
                  textAnchor="middle"
                  fill={seg.color === '#444' ? TEXT_MUTED : seg.color}
                  fontSize={8}
                  fontFamily={MONO}
                  fontWeight={500}
                >
                  {seg.label}MB
                </text>
              )}
            </g>
          ))}

          {/* Event Log */}
          <rect
            x={VM_START_X}
            y={LOG_Y}
            width={barW}
            height={LOG_H}
            rx={4}
            fill={CARD_BG}
            stroke={BORDER}
            strokeWidth={1}
          />
          {/* Log header */}
          <rect x={VM_START_X} y={LOG_Y} width={barW} height={18} rx={4} fill={CARD_BG} />
          <rect x={VM_START_X} y={LOG_Y + 14} width={barW} height={4} fill={CARD_BG} />
          <line
            x1={VM_START_X}
            y1={LOG_Y + 18}
            x2={VM_START_X + barW}
            y2={LOG_Y + 18}
            stroke={BORDER}
            strokeWidth={1}
          />
          <WindowDots x={VM_START_X + 8} y={LOG_Y + 7} />
          <text x={VM_START_X + 34} y={LOG_Y + 13} fill={TEXT_MUTED} fontSize={8} fontFamily={MONO}>
            event log
          </text>
          {/* Log entries */}
          {log.slice(0, 6).map((entry, i) => (
            <text
              key={entry.id}
              x={VM_START_X + 10}
              y={LOG_Y + 32 + i * 14}
              fill={entry.color}
              fontSize={8}
              fontFamily={MONO}
              opacity={1 - i * 0.1}
            >
              {entry.text.length > 80 ? entry.text.slice(0, 80) + '...' : entry.text}
            </text>
          ))}
        </svg>
      </div>
    </div>
  )
}
