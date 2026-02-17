import { useEffect, useRef, useCallback } from 'react'
import { motion, useAnimation } from 'framer-motion'
import { useInView } from 'react-intersection-observer'
import { RotateCcw } from 'lucide-react'

const REDUCED_MOTION =
  typeof window !== 'undefined' && window.matchMedia('(prefers-reduced-motion: reduce)').matches

export function DaemonConnectionAnimation() {
  const controls = useAnimation()
  const [ref, inView] = useInView({ threshold: 0.3, triggerOnce: false })
  const hasPlayedRef = useRef(false)

  const animate = useCallback(async () => {
    // Reset
    await controls.start('hidden')
    // Sequence
    await controls.start('daemon')
    await controls.start('line1')
    await controls.start('cp')
    await controls.start('line2')
    await controls.start('dashboard')
    await controls.start('pulse')
    hasPlayedRef.current = true
  }, [controls])

  useEffect(() => {
    if (inView && !hasPlayedRef.current) {
      if (REDUCED_MOTION) {
        controls.start('instant')
        hasPlayedRef.current = true
      } else {
        animate()
      }
    }
  }, [inView, animate, controls])

  const replay = () => {
    hasPlayedRef.current = false
    animate()
  }

  return (
    <div ref={ref} className="border-border relative mb-6 overflow-x-auto border bg-black p-6">
      <button
        onClick={replay}
        className="text-muted-foreground hover:text-foreground absolute top-3 right-3 p-1 transition-colors"
        title="Replay animation"
      >
        <RotateCcw className="h-3 w-3" />
      </button>

      <svg viewBox="0 0 800 200" className="mx-auto w-full max-w-[800px]" style={{ minWidth: 500 }}>
        {/* Glow filter */}
        {!REDUCED_MOTION && (
          <defs>
            <filter id="glow">
              <feGaussianBlur stdDeviation="3" result="blur" />
              <feMerge>
                <feMergeNode in="blur" />
                <feMergeNode in="SourceGraphic" />
              </feMerge>
            </filter>
          </defs>
        )}

        {/* Daemon node */}
        <motion.g
          variants={{
            hidden: { opacity: 0 },
            daemon: { opacity: 1, transition: { duration: 0.5 } },
            instant: { opacity: 1 },
          }}
          initial="hidden"
          animate={controls}
        >
          <rect
            x="20"
            y="60"
            width="180"
            height="80"
            fill="#171717"
            stroke="#262626"
            strokeWidth="1"
          />
          {/* Title bar dots */}
          <circle cx="35" cy="75" r="2.5" fill="#404040" />
          <circle cx="45" cy="75" r="2.5" fill="#404040" />
          <circle cx="55" cy="75" r="2.5" fill="#404040" />
          <text
            x="110"
            y="106"
            textAnchor="middle"
            fill="#e5e5e5"
            fontSize="11"
            fontFamily="monospace"
          >
            fluid-daemon
          </text>
          <text
            x="110"
            y="122"
            textAnchor="middle"
            fill="#a3a3a3"
            fontSize="9"
            fontFamily="monospace"
          >
            :9091
          </text>
        </motion.g>

        {/* Line 1: daemon -> control plane */}
        <motion.line
          x1="200"
          y1="100"
          x2="310"
          y2="100"
          stroke="#3B82F6"
          strokeWidth="1.5"
          variants={{
            hidden: { pathLength: 0, opacity: 0 },
            line1: { pathLength: 1, opacity: 1, transition: { duration: 1.5 } },
            instant: { pathLength: 1, opacity: 1 },
          }}
          initial="hidden"
          animate={controls}
        />
        {/* Line 1 label */}
        <motion.text
          x="255"
          y="90"
          textAnchor="middle"
          fill="#a3a3a3"
          fontSize="8"
          fontFamily="monospace"
          variants={{
            hidden: { opacity: 0 },
            line1: { opacity: 1, transition: { delay: 0.5 } },
            instant: { opacity: 1 },
          }}
          initial="hidden"
          animate={controls}
        >
          gRPC stream
        </motion.text>

        {/* Spinner dot on line 1 */}
        {!REDUCED_MOTION && (
          <motion.circle
            cx="200"
            cy="100"
            r="3"
            fill="#3B82F6"
            filter="url(#glow)"
            variants={{
              hidden: { opacity: 0, cx: 200 },
              line1: {
                opacity: [0, 1, 1, 0],
                cx: [200, 250, 300, 310],
                transition: { duration: 1.5 },
              },
              instant: { opacity: 0 },
            }}
            initial="hidden"
            animate={controls}
          />
        )}

        {/* Control Plane node */}
        <motion.g
          variants={{
            hidden: { opacity: 0.2 },
            cp: { opacity: 1, transition: { duration: 0.5 } },
            instant: { opacity: 1 },
          }}
          initial="hidden"
          animate={controls}
        >
          <rect
            x="310"
            y="60"
            width="180"
            height="80"
            fill="#171717"
            stroke="#262626"
            strokeWidth="1"
          />
          <circle cx="325" cy="75" r="2.5" fill="#404040" />
          <circle cx="335" cy="75" r="2.5" fill="#404040" />
          <circle cx="345" cy="75" r="2.5" fill="#404040" />
          <text
            x="400"
            y="106"
            textAnchor="middle"
            fill="#e5e5e5"
            fontSize="11"
            fontFamily="monospace"
          >
            Control Plane
          </text>
          <text
            x="400"
            y="122"
            textAnchor="middle"
            fill="#a3a3a3"
            fontSize="9"
            fontFamily="monospace"
          >
            REST + gRPC
          </text>
        </motion.g>

        {/* Connected badge */}
        <motion.g
          variants={{
            hidden: { opacity: 0 },
            cp: { opacity: 1, transition: { delay: 0.3 } },
            instant: { opacity: 1 },
          }}
          initial="hidden"
          animate={controls}
        >
          <rect
            x="355"
            y="148"
            width="90"
            height="20"
            fill="none"
            stroke="#4ade80"
            strokeWidth="1"
            opacity="0.3"
          />
          <text
            x="400"
            y="162"
            textAnchor="middle"
            fill="#4ade80"
            fontSize="9"
            fontFamily="monospace"
          >
            Connected
          </text>
        </motion.g>

        {/* Line 2: control plane -> dashboard */}
        <motion.line
          x1="490"
          y1="100"
          x2="600"
          y2="100"
          stroke="#3B82F6"
          strokeWidth="1.5"
          variants={{
            hidden: { pathLength: 0, opacity: 0 },
            line2: { pathLength: 1, opacity: 1, transition: { duration: 1.5 } },
            instant: { pathLength: 1, opacity: 1 },
          }}
          initial="hidden"
          animate={controls}
        />
        {/* Line 2 label */}
        <motion.text
          x="545"
          y="90"
          textAnchor="middle"
          fill="#a3a3a3"
          fontSize="8"
          fontFamily="monospace"
          variants={{
            hidden: { opacity: 0 },
            line2: { opacity: 1, transition: { delay: 0.5 } },
            instant: { opacity: 1 },
          }}
          initial="hidden"
          animate={controls}
        >
          REST API
        </motion.text>

        {/* Spinner dot on line 2 */}
        {!REDUCED_MOTION && (
          <motion.circle
            cx="490"
            cy="100"
            r="3"
            fill="#3B82F6"
            filter="url(#glow)"
            variants={{
              hidden: { opacity: 0, cx: 490 },
              line2: {
                opacity: [0, 1, 1, 0],
                cx: [490, 540, 590, 600],
                transition: { duration: 1.5 },
              },
              instant: { opacity: 0 },
            }}
            initial="hidden"
            animate={controls}
          />
        )}

        {/* Dashboard node */}
        <motion.g
          variants={{
            hidden: { opacity: 0.2 },
            dashboard: { opacity: 1, transition: { duration: 0.5 } },
            instant: { opacity: 1 },
          }}
          initial="hidden"
          animate={controls}
        >
          <rect
            x="600"
            y="60"
            width="180"
            height="80"
            fill="#171717"
            stroke="#262626"
            strokeWidth="1"
          />
          <circle cx="615" cy="75" r="2.5" fill="#404040" />
          <circle cx="625" cy="75" r="2.5" fill="#404040" />
          <circle cx="635" cy="75" r="2.5" fill="#404040" />
          <text
            x="690"
            y="106"
            textAnchor="middle"
            fill="#e5e5e5"
            fontSize="11"
            fontFamily="monospace"
          >
            Dashboard
          </text>
          <text
            x="690"
            y="122"
            textAnchor="middle"
            fill="#a3a3a3"
            fontSize="9"
            fontFamily="monospace"
          >
            Web UI
          </text>
        </motion.g>

        {/* Pulse effect on all connections after complete */}
        {!REDUCED_MOTION && (
          <>
            <motion.line
              x1="200"
              y1="100"
              x2="310"
              y2="100"
              stroke="#3B82F6"
              strokeWidth="1"
              variants={{
                hidden: { opacity: 0 },
                pulse: {
                  opacity: [0.2, 0.6, 0.2],
                  transition: { duration: 2, repeat: Infinity },
                },
                instant: { opacity: 0.3 },
              }}
              initial="hidden"
              animate={controls}
            />
            <motion.line
              x1="490"
              y1="100"
              x2="600"
              y2="100"
              stroke="#3B82F6"
              strokeWidth="1"
              variants={{
                hidden: { opacity: 0 },
                pulse: {
                  opacity: [0.2, 0.6, 0.2],
                  transition: { duration: 2, repeat: Infinity, delay: 1 },
                },
                instant: { opacity: 0.3 },
              }}
              initial="hidden"
              animate={controls}
            />
          </>
        )}
      </svg>
    </div>
  )
}
