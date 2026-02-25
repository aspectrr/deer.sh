import { useRef, useCallback, useState } from 'react'
import { motion, useAnimation } from 'framer-motion'
import { RotateCcw, Play } from 'lucide-react'

const REDUCED_MOTION =
  typeof window !== 'undefined' && window.matchMedia('(prefers-reduced-motion: reduce)').matches

const BLUE = '#60a5fa'
const TEXT_MUTED = '#737373'
const BORDER = '#262626'
const CARD_BG = '#171717'
const DOT_FILL = '#525252'
const GREEN = '#4ade80'

// Node positions
const NODES = [
  { x: 20, label: 'Laptop', sublabel: 'your device' },
  { x: 240, label: 'Firewall', sublabel: 'school network' },
  { x: 460, label: 'VPN Server', sublabel: '198.51.100.1' },
  { x: 680, label: 'YouTube', sublabel: '142.250.80.46' },
]

const NODE_W = 160
const NODE_H = 70
const NODE_Y = 75
const LINE_Y = NODE_Y + NODE_H / 2

function WindowDots({ x, y }: { x: number; y: number }) {
  return (
    <>
      <rect x={x} y={y} width={4} height={4} rx={1} fill={DOT_FILL} />
      <rect x={x + 8} y={y} width={4} height={4} rx={1} fill={DOT_FILL} />
      <rect x={x + 16} y={y} width={4} height={4} rx={1} fill={DOT_FILL} />
    </>
  )
}

export function PacketJourney() {
  const controls = useAnimation()
  const [hasPlayed, setHasPlayed] = useState(false)
  const isAnimatingRef = useRef(false)

  const animate = useCallback(async () => {
    if (isAnimatingRef.current) return
    isAnimatingRef.current = true
    setHasPlayed(true)

    if (REDUCED_MOTION) {
      await controls.start('instant')
      isAnimatingRef.current = false
      return
    }

    await controls.start('hidden')
    // Phase 1: encrypting at laptop
    await controls.start('encrypting')
    // Phase 2: dot travels laptop -> firewall
    await controls.start('travel1')
    // Phase 3: firewall inspects
    await controls.start('firewall')
    // Phase 4: dot travels firewall -> vpn
    await controls.start('travel2')
    // Phase 5: decapsulating at vpn
    await controls.start('decapsulate')
    // Phase 6: green dot vpn -> youtube
    await controls.start('travel3')
    // Phase 7: youtube responds
    await controls.start('done')

    isAnimatingRef.current = false
  }, [controls])

  const replay = () => {
    isAnimatingRef.current = false
    animate()
  }

  // Line endpoints
  const line1Start = NODES[0].x + NODE_W
  const line1End = NODES[1].x
  const line2Start = NODES[1].x + NODE_W
  const line2End = NODES[2].x
  const line3Start = NODES[2].x + NODE_W
  const line3End = NODES[3].x

  return (
    <div
      style={{
        background: 'black',
        border: `1px solid ${BORDER}`,
        padding: '1.5rem',
        marginTop: '1.5rem',
        marginBottom: '1.5rem',
        position: 'relative',
        overflowX: 'auto',
      }}
    >
      {/* Top-right button */}
      <div style={{ position: 'absolute', top: 12, right: 12, zIndex: 1 }}>
        {hasPlayed ? (
          <button
            onClick={replay}
            style={{
              background: 'none',
              border: 'none',
              color: TEXT_MUTED,
              cursor: 'pointer',
              padding: 4,
            }}
            title="Replay animation"
          >
            <RotateCcw size={14} />
          </button>
        ) : null}
      </div>

      {/* Send Packet button (centered, shown before first play) */}
      {!hasPlayed && (
        <div
          style={{
            position: 'absolute',
            top: 0,
            left: 0,
            right: 0,
            bottom: 0,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            zIndex: 2,
          }}
        >
          <button
            onClick={animate}
            style={{
              background: 'none',
              border: `1px solid ${BLUE}`,
              color: BLUE,
              cursor: 'pointer',
              padding: '0.5rem 1rem',
              fontFamily: 'ui-monospace, SFMono-Regular, monospace',
              fontSize: '0.8rem',
              display: 'flex',
              alignItems: 'center',
              gap: '0.5rem',
            }}
          >
            <Play size={14} /> Send Packet
          </button>
        </div>
      )}

      <svg
        viewBox="0 0 880 220"
        style={{
          width: '100%',
          minWidth: 500,
          display: 'block',
          opacity: hasPlayed ? 1 : 0.3,
          transition: 'opacity 0.3s',
        }}
      >
        {/* Glow filter */}
        {!REDUCED_MOTION && (
          <defs>
            <filter id="glow-journey">
              <feGaussianBlur stdDeviation="3" result="blur" />
              <feMerge>
                <feMergeNode in="blur" />
                <feMergeNode in="SourceGraphic" />
              </feMerge>
            </filter>
          </defs>
        )}

        {/* Connection lines (static) */}
        <line
          x1={line1Start}
          y1={LINE_Y}
          x2={line1End}
          y2={LINE_Y}
          stroke={BORDER}
          strokeWidth={1}
        />
        <line
          x1={line2Start}
          y1={LINE_Y}
          x2={line2End}
          y2={LINE_Y}
          stroke={BORDER}
          strokeWidth={1}
        />
        <line
          x1={line3Start}
          y1={LINE_Y}
          x2={line3End}
          y2={LINE_Y}
          stroke={BORDER}
          strokeWidth={1}
        />

        {/* Node boxes */}
        {NODES.map((node, i) => (
          <g key={node.label}>
            {/* Node highlight for firewall during inspection */}
            <motion.rect
              x={node.x}
              y={NODE_Y}
              width={NODE_W}
              height={NODE_H}
              fill={CARD_BG}
              stroke={BORDER}
              strokeWidth={1}
              variants={{
                hidden: { stroke: BORDER },
                firewall: i === 1 ? { stroke: '#fbbf24', transition: { duration: 0.3 } } : {},
                decapsulate:
                  i === 2
                    ? { stroke: BLUE, transition: { duration: 0.3 } }
                    : i === 1
                      ? { stroke: BORDER }
                      : {},
                done: i === 3 ? { stroke: GREEN, transition: { duration: 0.3 } } : {},
                instant: {
                  stroke: i === 3 ? GREEN : i === 2 ? BLUE : i === 1 ? '#fbbf24' : BORDER,
                },
              }}
              initial="hidden"
              animate={controls}
            />
            <WindowDots x={node.x + 10} y={NODE_Y + 10} />
            <text
              x={node.x + NODE_W / 2}
              y={NODE_Y + 32}
              textAnchor="middle"
              fill="#e5e5e5"
              fontSize={11}
              fontFamily="ui-monospace, SFMono-Regular, monospace"
              fontWeight={500}
            >
              {node.label}
            </text>
            <text
              x={node.x + NODE_W / 2}
              y={NODE_Y + 48}
              textAnchor="middle"
              fill={TEXT_MUTED}
              fontSize={9}
              fontFamily="ui-monospace, SFMono-Regular, monospace"
            >
              {node.sublabel}
            </text>
          </g>
        ))}

        {/* "encrypting..." label at laptop */}
        <motion.text
          x={NODES[0].x + NODE_W / 2}
          y={NODE_Y - 10}
          textAnchor="middle"
          fill={BLUE}
          fontSize={9}
          fontFamily="ui-monospace, SFMono-Regular, monospace"
          variants={{
            hidden: { opacity: 0 },
            encrypting: { opacity: [0, 1, 1, 0], transition: { duration: 1.2 } },
            instant: { opacity: 0 },
          }}
          initial="hidden"
          animate={controls}
        >
          encrypting...
        </motion.text>

        {/* "reads outer IP" label at firewall */}
        <motion.text
          x={NODES[1].x + NODE_W / 2}
          y={NODE_Y - 10}
          textAnchor="middle"
          fill="#fbbf24"
          fontSize={9}
          fontFamily="ui-monospace, SFMono-Regular, monospace"
          variants={{
            hidden: { opacity: 0 },
            firewall: { opacity: [0, 1, 1, 0], transition: { duration: 1.2 } },
            instant: { opacity: 0 },
          }}
          initial="hidden"
          animate={controls}
        >
          reads outer IP: pass
        </motion.text>

        {/* "decapsulating..." label at VPN */}
        <motion.text
          x={NODES[2].x + NODE_W / 2}
          y={NODE_Y - 10}
          textAnchor="middle"
          fill={BLUE}
          fontSize={9}
          fontFamily="ui-monospace, SFMono-Regular, monospace"
          variants={{
            hidden: { opacity: 0 },
            decapsulate: { opacity: [0, 1, 1, 0], transition: { duration: 1.2 } },
            instant: { opacity: 0 },
          }}
          initial="hidden"
          animate={controls}
        >
          decapsulating...
        </motion.text>

        {/* "200 OK" label at YouTube */}
        <motion.text
          x={NODES[3].x + NODE_W / 2}
          y={NODE_Y - 10}
          textAnchor="middle"
          fill={GREEN}
          fontSize={9}
          fontFamily="ui-monospace, SFMono-Regular, monospace"
          fontWeight={600}
          variants={{
            hidden: { opacity: 0 },
            done: { opacity: 1, transition: { duration: 0.5 } },
            instant: { opacity: 1 },
          }}
          initial="hidden"
          animate={controls}
        >
          200 OK
        </motion.text>

        {/* Traveling dot - segment 1: laptop -> firewall (blue, encrypted) */}
        {!REDUCED_MOTION && (
          <motion.circle
            cx={line1Start}
            cy={LINE_Y}
            r={4}
            fill={BLUE}
            filter="url(#glow-journey)"
            variants={{
              hidden: { opacity: 0, cx: line1Start },
              travel1: {
                opacity: [0, 1, 1, 0],
                cx: [line1Start, (line1Start + line1End) / 2, line1End - 10, line1End],
                transition: { duration: 1 },
              },
            }}
            initial="hidden"
            animate={controls}
          />
        )}

        {/* Traveling dot - segment 2: firewall -> vpn (blue, still encrypted) */}
        {!REDUCED_MOTION && (
          <motion.circle
            cx={line2Start}
            cy={LINE_Y}
            r={4}
            fill={BLUE}
            filter="url(#glow-journey)"
            variants={{
              hidden: { opacity: 0, cx: line2Start },
              travel2: {
                opacity: [0, 1, 1, 0],
                cx: [line2Start, (line2Start + line2End) / 2, line2End - 10, line2End],
                transition: { duration: 1 },
              },
            }}
            initial="hidden"
            animate={controls}
          />
        )}

        {/* Traveling dot - segment 3: vpn -> youtube (green, decrypted) */}
        {!REDUCED_MOTION && (
          <motion.circle
            cx={line3Start}
            cy={LINE_Y}
            r={4}
            fill={GREEN}
            filter="url(#glow-journey)"
            variants={{
              hidden: { opacity: 0, cx: line3Start },
              travel3: {
                opacity: [0, 1, 1, 0],
                cx: [line3Start, (line3Start + line3End) / 2, line3End - 10, line3End],
                transition: { duration: 1 },
              },
            }}
            initial="hidden"
            animate={controls}
          />
        )}

        {/* Line labels */}
        <motion.text
          x={(line1Start + line1End) / 2}
          y={LINE_Y + 20}
          textAnchor="middle"
          fill={TEXT_MUTED}
          fontSize={8}
          fontFamily="ui-monospace, SFMono-Regular, monospace"
          variants={{
            hidden: { opacity: 0 },
            travel1: { opacity: 1, transition: { delay: 0.3 } },
            instant: { opacity: 1 },
          }}
          initial="hidden"
          animate={controls}
        >
          encrypted
        </motion.text>

        <motion.text
          x={(line2Start + line2End) / 2}
          y={LINE_Y + 20}
          textAnchor="middle"
          fill={TEXT_MUTED}
          fontSize={8}
          fontFamily="ui-monospace, SFMono-Regular, monospace"
          variants={{
            hidden: { opacity: 0 },
            travel2: { opacity: 1, transition: { delay: 0.3 } },
            instant: { opacity: 1 },
          }}
          initial="hidden"
          animate={controls}
        >
          encrypted
        </motion.text>

        <motion.text
          x={(line3Start + line3End) / 2}
          y={LINE_Y + 20}
          textAnchor="middle"
          fill={TEXT_MUTED}
          fontSize={8}
          fontFamily="ui-monospace, SFMono-Regular, monospace"
          variants={{
            hidden: { opacity: 0 },
            travel3: { opacity: 1, transition: { delay: 0.3 } },
            instant: { opacity: 1 },
          }}
          initial="hidden"
          animate={controls}
        >
          HTTPS
        </motion.text>
      </svg>
    </div>
  )
}
