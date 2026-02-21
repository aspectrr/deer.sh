import { useState } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { ChevronLeft, ChevronRight } from 'lucide-react'

const REDUCED_MOTION =
  typeof window !== 'undefined' && window.matchMedia('(prefers-reduced-motion: reduce)').matches

const BLUE = '#60a5fa'
const TEXT_MUTED = '#737373'
const BORDER = '#262626'
const CARD_BG = '#171717'

interface Layer {
  id: string
  label: string
  fill: string
  stroke: string
  detail: string
}

const LAYERS: Layer[] = [
  { id: 'data', label: 'Data', fill: '#1e3a5f', stroke: '#60a5fa', detail: '"GET /watch?v=..."' },
  { id: 'tcp', label: 'TCP :443', fill: '#2a1a4f', stroke: '#a855f7', detail: 'src:49152 dst:443' },
  {
    id: 'ip',
    label: 'IP (inner)',
    fill: '#1a2e1a',
    stroke: '#4ade80',
    detail: 'dst: 142.250.80.46',
  },
  {
    id: 'encrypted',
    label: 'Encrypted',
    fill: '#2a1a1a',
    stroke: '#f87171',
    detail: 'AES-256-GCM ciphertext',
  },
  {
    id: 'outer-ip',
    label: 'Outer IP',
    fill: '#2a2000',
    stroke: '#fbbf24',
    detail: 'dst: 198.51.100.1',
  },
]

interface Step {
  title: string
  description: string
  visibleLayers: string[]
  direction: 'encapsulate' | 'decapsulate'
}

const STEPS: Step[] = [
  {
    title: 'Original Data',
    description:
      'You type youtube.com in your browser. This generates an HTTP request - just bytes of data that need to get to YouTube.',
    visibleLayers: ['data'],
    direction: 'encapsulate',
  },
  {
    title: 'Add TCP Header',
    description:
      'TCP wraps the data with port numbers: your random source port and destination port 443 (HTTPS). This tells the receiving machine which application should handle the data.',
    visibleLayers: ['data', 'tcp'],
    direction: 'encapsulate',
  },
  {
    title: 'Add IP Header',
    description:
      'IP wraps everything with addresses: your laptop\'s IP and YouTube\'s IP (142.250.80.46). This is the "envelope" that routers read to forward the packet.',
    visibleLayers: ['data', 'tcp', 'ip'],
    direction: 'encapsulate',
  },
  {
    title: 'Encrypt the Payload',
    description:
      'The VPN client encrypts the entire original packet (data + TCP + IP) into an unreadable ciphertext blob. Anyone intercepting this sees random bytes.',
    visibleLayers: ['data', 'tcp', 'ip', 'encrypted'],
    direction: 'encapsulate',
  },
  {
    title: 'Add Outer IP Header',
    description:
      'A new IP header wraps the encrypted blob. This outer header points to the VPN server (198.51.100.1) instead of YouTube. This is what the firewall sees.',
    visibleLayers: ['data', 'tcp', 'ip', 'encrypted', 'outer-ip'],
    direction: 'encapsulate',
  },
  {
    title: 'Firewall Inspection',
    description:
      'The school firewall reads only the outer IP header. It sees traffic going to 198.51.100.1 (just some random IP). Not youtube.com. The encrypted payload is opaque. Packet passes through.',
    visibleLayers: ['data', 'tcp', 'ip', 'encrypted', 'outer-ip'],
    direction: 'decapsulate',
  },
  {
    title: 'VPN Server Strips Outer Layer',
    description:
      'The VPN server receives the packet, strips the outer IP header, and decrypts the payload. The original packet is fully restored.',
    visibleLayers: ['data', 'tcp', 'ip'],
    direction: 'decapsulate',
  },
  {
    title: 'Original Packet Restored',
    description:
      'The VPN server now forwards the original packet (dst: youtube.com) to its real destination. YouTube sees a request from the VPN server, not from your school.',
    visibleLayers: ['data', 'tcp', 'ip'],
    direction: 'decapsulate',
  },
]

const LAYER_HEIGHT = 36
const LAYER_GAP = 4
const BASE_X = 60
const BASE_WIDTH = 280

function getLayerGeometry(totalVisible: number, positionInStack: number) {
  // Outermost layer is widest, innermost is narrowest
  const nestLevel = totalVisible - 1 - positionInStack
  const padding = nestLevel * 20
  const x = BASE_X + padding
  const width = BASE_WIDTH - padding * 2
  const y = 20 + positionInStack * (LAYER_HEIGHT + LAYER_GAP)
  return { x, y, width }
}

const duration = REDUCED_MOTION ? 0 : 0.35

export function PacketEncapsulation() {
  const [step, setStep] = useState(0)
  const current = STEPS[step]

  const visibleLayers = current.visibleLayers
    .map((id) => LAYERS.find((l) => l.id === id)!)
    .filter(Boolean)

  // Reverse so outermost layer renders first (back), innermost renders last (front)
  const orderedLayers = [...visibleLayers].reverse()

  const svgHeight = visibleLayers.length * (LAYER_HEIGHT + LAYER_GAP) + 40

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

      {/* SVG layers */}
      <div style={{ overflowX: 'auto' }}>
        <svg
          viewBox={`0 0 400 ${svgHeight}`}
          style={{
            width: '100%',
            maxWidth: 500,
            minWidth: 320,
            display: 'block',
            margin: '0 auto',
          }}
        >
          <AnimatePresence mode="popLayout">
            {orderedLayers.map((layer) => {
              const positionInStack = visibleLayers.indexOf(layer)
              const { x, y, width } = getLayerGeometry(visibleLayers.length, positionInStack)
              return (
                <motion.g
                  key={layer.id}
                  initial={{ opacity: 0, x: current.direction === 'encapsulate' ? -20 : 20 }}
                  animate={{ opacity: 1, x: 0 }}
                  exit={{ opacity: 0, x: current.direction === 'decapsulate' ? 20 : -20 }}
                  transition={{ duration }}
                >
                  <rect
                    x={x}
                    y={y}
                    width={width}
                    height={LAYER_HEIGHT}
                    rx={3}
                    fill={layer.fill}
                    stroke={layer.stroke}
                    strokeWidth={1.5}
                  />
                  <text
                    x={x + 10}
                    y={y + 15}
                    fill={layer.stroke}
                    fontSize={10}
                    fontFamily="ui-monospace, SFMono-Regular, monospace"
                    fontWeight={600}
                  >
                    {layer.label}
                  </text>
                  <text
                    x={x + 10}
                    y={y + 28}
                    fill={TEXT_MUTED}
                    fontSize={8}
                    fontFamily="ui-monospace, SFMono-Regular, monospace"
                  >
                    {layer.detail}
                  </text>
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
