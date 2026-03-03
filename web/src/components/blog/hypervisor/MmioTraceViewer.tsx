import { useState, useEffect, useCallback } from 'react'
import { motion, AnimatePresence } from 'framer-motion'

const REDUCED_MOTION =
  typeof window !== 'undefined' && window.matchMedia('(prefers-reduced-motion: reduce)').matches

const BLUE = '#60a5fa'
const GREEN = '#4ade80'
const AMBER = '#fbbf24'
const RED = '#f87171'
const PURPLE = '#a855f7'
const TEXT_MUTED = '#737373'
const BORDER = '#262626'
const CARD_BG = '#171717'

interface TraceEntry {
  cycle: number
  op: 'W' | 'R'
  device: string
  offset: string
  size: number
  value: string
  detail: string
}

const TRACE: TraceEntry[] = [
  {
    cycle: 1,
    op: 'W',
    device: 'echo',
    offset: '0x000',
    size: 4,
    value: '0x41424344',
    detail: 'Write "ABCD" to echo device',
  },
  {
    cycle: 2,
    op: 'R',
    device: 'echo',
    offset: '0x000',
    size: 4,
    value: '0x41424344',
    detail: 'Read back "ABCD" from echo',
  },
  {
    cycle: 3,
    op: 'R',
    device: 'counter',
    offset: '0x000',
    size: 4,
    value: '0x00000001',
    detail: 'Counter read #1',
  },
  {
    cycle: 4,
    op: 'R',
    device: 'counter',
    offset: '0x000',
    size: 4,
    value: '0x00000002',
    detail: 'Counter read #2 (incremented)',
  },
  {
    cycle: 5,
    op: 'W',
    device: 'uart',
    offset: '0x000',
    size: 1,
    value: '0x48',
    detail: "Write 'H' to UART DR",
  },
  {
    cycle: 6,
    op: 'W',
    device: 'uart',
    offset: '0x000',
    size: 1,
    value: '0x65',
    detail: "Write 'e' to UART DR",
  },
  {
    cycle: 7,
    op: 'W',
    device: 'uart',
    offset: '0x000',
    size: 1,
    value: '0x6C',
    detail: "Write 'l' to UART DR",
  },
  {
    cycle: 8,
    op: 'R',
    device: 'uart',
    offset: '0x018',
    size: 4,
    value: '0x00000000',
    detail: 'Read UART FR (TX not full)',
  },
  {
    cycle: 9,
    op: 'W',
    device: 'echo',
    offset: '0x000',
    size: 4,
    value: '0xDEADBEEF',
    detail: 'Write 0xDEADBEEF to echo',
  },
  {
    cycle: 10,
    op: 'R',
    device: 'echo',
    offset: '0x000',
    size: 4,
    value: '0xDEADBEEF',
    detail: 'Read back 0xDEADBEEF',
  },
]

const DEVICE_COLORS: Record<string, string> = {
  echo: BLUE,
  counter: PURPLE,
  uart: GREEN,
}

const DEVICE_ADDRS: Record<string, string> = {
  echo: '0x1000_0000',
  counter: '0x1000_1000',
  uart: '0x0900_0000',
}

const duration = REDUCED_MOTION ? 0 : 0.25

export function MmioTraceViewer() {
  const [visibleCount, setVisibleCount] = useState(0)
  const [running, setRunning] = useState(false)

  const advance = useCallback(() => {
    setVisibleCount((c) => {
      if (c >= TRACE.length) {
        setRunning(false)
        return c
      }
      return c + 1
    })
  }, [])

  useEffect(() => {
    if (!running) return
    const interval = setInterval(advance, 600)
    return () => clearInterval(interval)
  }, [running, advance])

  const visibleEntries = TRACE.slice(0, visibleCount)
  const latestEntry = visibleEntries.length > 0 ? visibleEntries[visibleEntries.length - 1] : null

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
      {/* Header */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          marginBottom: '1rem',
        }}
      >
        <span style={{ color: BLUE, fontSize: '0.75rem' }}>MMIO Trace</span>
        <div style={{ display: 'flex', gap: '0.5rem' }}>
          <button
            onClick={() => {
              if (visibleCount >= TRACE.length) {
                setVisibleCount(0)
              }
              setRunning(true)
            }}
            style={{
              background: 'none',
              border: `1px solid ${GREEN}`,
              color: GREEN,
              cursor: 'pointer',
              padding: '0.25rem 0.75rem',
              fontSize: '0.75rem',
              fontFamily: 'ui-monospace, SFMono-Regular, monospace',
            }}
          >
            {visibleCount >= TRACE.length ? 'Replay' : running ? 'Running...' : 'Play'}
          </button>
          {running && (
            <button
              onClick={() => setRunning(false)}
              style={{
                background: 'none',
                border: `1px solid ${RED}`,
                color: RED,
                cursor: 'pointer',
                padding: '0.25rem 0.75rem',
                fontSize: '0.75rem',
                fontFamily: 'ui-monospace, SFMono-Regular, monospace',
              }}
            >
              Pause
            </button>
          )}
        </div>
      </div>

      {/* Device legend */}
      <div
        style={{
          display: 'flex',
          gap: '1rem',
          marginBottom: '1rem',
          flexWrap: 'wrap',
        }}
      >
        {Object.entries(DEVICE_ADDRS).map(([name, addr]) => (
          <div
            key={name}
            style={{ fontSize: '0.7rem', display: 'flex', alignItems: 'center', gap: '0.3rem' }}
          >
            <div
              style={{
                width: 8,
                height: 8,
                borderRadius: '50%',
                background: DEVICE_COLORS[name],
              }}
            />
            <span style={{ color: DEVICE_COLORS[name] }}>{name}</span>
            <span style={{ color: TEXT_MUTED }}>@ {addr}</span>
          </div>
        ))}
      </div>

      {/* Trace log */}
      <div
        style={{
          background: '#0a0a0a',
          border: `1px solid ${BORDER}`,
          padding: '0.5rem',
          maxHeight: 260,
          overflowY: 'auto',
          minHeight: 100,
        }}
      >
        {visibleEntries.length === 0 && (
          <div
            style={{ color: TEXT_MUTED, fontSize: '0.7rem', textAlign: 'center', padding: '1rem' }}
          >
            Press Play to start the MMIO trace...
          </div>
        )}
        <AnimatePresence>
          {visibleEntries.map((entry, i) => {
            const color = DEVICE_COLORS[entry.device] || TEXT_MUTED
            const isLatest = i === visibleEntries.length - 1
            return (
              <motion.div
                key={entry.cycle}
                initial={{ opacity: 0, height: 0 }}
                animate={{ opacity: 1, height: 'auto' }}
                transition={{ duration }}
                style={{
                  display: 'flex',
                  gap: '0.5rem',
                  padding: '0.2rem 0.3rem',
                  fontSize: '0.7rem',
                  background: isLatest ? color + '11' : 'transparent',
                  borderLeft: `2px solid ${isLatest ? color : 'transparent'}`,
                }}
              >
                <span style={{ color: TEXT_MUTED, minWidth: 20, textAlign: 'right' }}>
                  {entry.cycle}
                </span>
                <span
                  style={{
                    color: entry.op === 'W' ? AMBER : GREEN,
                    fontWeight: 600,
                    minWidth: 14,
                  }}
                >
                  {entry.op}
                </span>
                <span style={{ color, minWidth: 50 }}>{entry.device}</span>
                <span style={{ color: TEXT_MUTED, minWidth: 40 }}>+{entry.offset}</span>
                <span style={{ color: TEXT_MUTED, minWidth: 16 }}>{entry.size}B</span>
                <span style={{ color: '#e5e5e5' }}>{entry.value}</span>
              </motion.div>
            )
          })}
        </AnimatePresence>
      </div>

      {/* Current entry detail */}
      {latestEntry && (
        <div
          style={{
            marginTop: '0.75rem',
            padding: '0.5rem 0.75rem',
            background: '#0a0a0a',
            border: `1px solid ${BORDER}`,
            fontSize: '0.75rem',
            color: '#a3a3a3',
          }}
        >
          <span style={{ color: DEVICE_COLORS[latestEntry.device], fontWeight: 600 }}>
            [{latestEntry.device}]
          </span>{' '}
          {latestEntry.detail}
        </div>
      )}
    </div>
  )
}
