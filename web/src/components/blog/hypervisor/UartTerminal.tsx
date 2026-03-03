import { useState, useEffect, useCallback, useRef } from 'react'
import { motion } from 'framer-motion'

const REDUCED_MOTION =
  typeof window !== 'undefined' && window.matchMedia('(prefers-reduced-motion: reduce)').matches

const GREEN = '#4ade80'
const AMBER = '#fbbf24'
const TEXT_MUTED = '#737373'
const BORDER = '#262626'
const CARD_BG = '#171717'

interface UartTerminalProps {
  lines?: string[]
  title?: string
  typingSpeed?: number
}

const DEFAULT_LINES = [
  'PL011 UART initialized at 0x09000000',
  'Hello from the VM!',
  'Guest UART write: 0x48 (H)',
  'Guest UART write: 0x65 (e)',
  'Guest UART write: 0x6C (l)',
  'Guest UART write: 0x6C (l)',
  'Guest UART write: 0x6F (o)',
  'Guest UART write: 0x20 ( )',
  'Guest UART write: 0x66 (f)',
  'Guest UART write: 0x72 (r)',
  'Guest UART write: 0x6F (o)',
  'Guest UART write: 0x6D (m)',
  '',
  'Output: Hello from the VM!',
  'Guest halted via HVC #0',
]

export function UartTerminal({
  lines = DEFAULT_LINES,
  title = 'PL011 UART Output',
  typingSpeed = 50,
}: UartTerminalProps) {
  const [displayedChars, setDisplayedChars] = useState(0)
  const [running, setRunning] = useState(false)
  const termRef = useRef<HTMLDivElement>(null)

  // Flatten all lines into a single char stream for typing effect
  const fullText = lines.join('\n')
  const totalChars = fullText.length

  const advance = useCallback(() => {
    setDisplayedChars((c) => {
      if (c >= totalChars) {
        setRunning(false)
        return c
      }
      return c + 1
    })
  }, [totalChars])

  useEffect(() => {
    if (!running) return
    const interval = setInterval(advance, REDUCED_MOTION ? 0 : typingSpeed)
    return () => clearInterval(interval)
  }, [running, advance, typingSpeed])

  // Auto-scroll to bottom
  useEffect(() => {
    if (termRef.current) {
      termRef.current.scrollTop = termRef.current.scrollHeight
    }
  }, [displayedChars])

  const visibleText = fullText.slice(0, displayedChars)
  const showCursor = running || displayedChars < totalChars

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
      {/* Title bar */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          marginBottom: '0.75rem',
        }}
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
          {/* Fake window dots */}
          <div style={{ display: 'flex', gap: '0.3rem' }}>
            <div style={{ width: 8, height: 8, borderRadius: '50%', background: '#f87171' }} />
            <div style={{ width: 8, height: 8, borderRadius: '50%', background: AMBER }} />
            <div style={{ width: 8, height: 8, borderRadius: '50%', background: GREEN }} />
          </div>
          <span style={{ color: TEXT_MUTED, fontSize: '0.75rem' }}>{title}</span>
        </div>
        <button
          onClick={() => {
            if (displayedChars >= totalChars) {
              setDisplayedChars(0)
            }
            setRunning(true)
          }}
          style={{
            background: 'none',
            border: `1px solid ${GREEN}`,
            color: GREEN,
            cursor: 'pointer',
            padding: '0.2rem 0.6rem',
            fontSize: '0.7rem',
            fontFamily: 'ui-monospace, SFMono-Regular, monospace',
          }}
        >
          {displayedChars >= totalChars ? 'Replay' : running ? 'Typing...' : 'Start'}
        </button>
      </div>

      {/* Terminal */}
      <div
        ref={termRef}
        style={{
          background: '#0c0c0c',
          border: `1px solid ${BORDER}`,
          padding: '0.75rem',
          minHeight: 180,
          maxHeight: 300,
          overflowY: 'auto',
          fontSize: '0.75rem',
          lineHeight: 1.6,
          color: GREEN,
          whiteSpace: 'pre-wrap',
          wordBreak: 'break-all',
        }}
      >
        {visibleText}
        {showCursor && (
          <motion.span
            animate={{ opacity: [1, 0] }}
            transition={{ duration: 0.8, repeat: Infinity, repeatType: 'reverse' }}
            style={{ color: GREEN }}
          >
            _
          </motion.span>
        )}
        {displayedChars === 0 && !running && (
          <span style={{ color: TEXT_MUTED }}>Press Start to begin UART output simulation...</span>
        )}
      </div>

      {/* Progress bar */}
      <div
        style={{
          marginTop: '0.5rem',
          height: 2,
          background: BORDER,
          position: 'relative',
        }}
      >
        <div
          style={{
            height: '100%',
            width: `${(displayedChars / totalChars) * 100}%`,
            background: GREEN,
            transition: 'width 0.05s linear',
          }}
        />
      </div>
    </div>
  )
}
