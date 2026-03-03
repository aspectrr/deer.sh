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

interface Instruction {
  addr: string
  asm: string
  hex: string
}

interface Step {
  title: string
  description: string
  guestPC: number
  phase: 'running' | 'trapped' | 'handling' | 'resuming'
  exitReason?: string
  ec?: string
  handlerAction?: string
}

const GUEST_CODE: Instruction[] = [
  { addr: '0x80000000', asm: 'mov x0, #1', hex: '0xD2800020' },
  { addr: '0x80000004', asm: 'mov x1, #2', hex: '0xD2800041' },
  { addr: '0x80000008', asm: 'hvc #0', hex: '0xD4000002' },
  { addr: '0x8000000c', asm: 'mov x2, #3', hex: '0xD2800062' },
  { addr: '0x80000010', asm: 'smc #0', hex: '0xD4000003' },
  { addr: '0x80000014', asm: 'wfi', hex: '0xD503207F' },
  { addr: '0x80000018', asm: 'mov x3, #4', hex: '0xD2800083' },
  { addr: '0x8000001c', asm: 'brk #0', hex: '0xD4200000' },
]

const STEPS: Step[] = [
  {
    title: 'Guest Starts Executing',
    description:
      'The VMM calls hv_vcpu_run(). The guest begins executing at PC = 0x80000000. Instructions run natively on the CPU until something causes a trap.',
    guestPC: 0,
    phase: 'running',
  },
  {
    title: 'mov x0, #1 / mov x1, #2',
    description:
      'These are normal instructions - they execute at full speed without any VM exit. The hypervisor never sees them. Register writes happen directly in hardware.',
    guestPC: 1,
    phase: 'running',
  },
  {
    title: 'HVC #0 - Trap!',
    description:
      'The HVC (Hypervisor Call) instruction causes an immediate VM exit. The CPU saves guest state and returns control to the VMM. hv_vcpu_run() returns.',
    guestPC: 2,
    phase: 'trapped',
    exitReason: 'EXCEPTION',
    ec: '0x16 (HVC from AArch64)',
  },
  {
    title: 'VMM Handles HVC',
    description:
      'The VMM reads exit.reason = EXCEPTION and decodes the syndrome. EC = 0x16 means HVC. It reads x0 = 1, x1 = 2 (the call arguments). The VMM can implement custom hypercalls here. PC must be advanced by 4 bytes.',
    guestPC: 2,
    phase: 'handling',
    exitReason: 'EXCEPTION',
    ec: '0x16 (HVC)',
    handlerAction: 'Read args from x0-x3, advance PC += 4',
  },
  {
    title: 'Resume - Guest Continues',
    description:
      'VMM sets PC = 0x8000000c (past the HVC) and calls hv_vcpu_run() again. Guest resumes at mov x2, #3.',
    guestPC: 3,
    phase: 'resuming',
  },
  {
    title: 'SMC #0 - Trap!',
    description:
      'SMC (Secure Monitor Call) also causes a VM exit. On real hardware this would go to EL3, but in a VM it traps to the hypervisor instead. The VMM typically returns "not supported" (-1 in x0).',
    guestPC: 4,
    phase: 'trapped',
    exitReason: 'EXCEPTION',
    ec: '0x17 (SMC from AArch64)',
  },
  {
    title: 'VMM Handles SMC',
    description:
      'EC = 0x17 means SMC. Most hypervisors return PSCI_NOT_SUPPORTED or -1 for unknown SMC calls. Linux uses SMC for PSCI (power management). Set x0 = -1, advance PC += 4.',
    guestPC: 4,
    phase: 'handling',
    exitReason: 'EXCEPTION',
    ec: '0x17 (SMC)',
    handlerAction: 'Set x0 = -1 (not supported), advance PC += 4',
  },
  {
    title: 'WFI - Trap!',
    description:
      'WFI (Wait For Interrupt) tells the CPU to idle. In a VM this traps to the hypervisor, because the VMM needs to decide what to do: wait for a timer, yield the host CPU, or just resume immediately.',
    guestPC: 5,
    phase: 'trapped',
    exitReason: 'EXCEPTION',
    ec: '0x01 (WFI/WFE)',
  },
  {
    title: 'VMM Handles WFI',
    description:
      'For WFI, the VMM can sleep until the next timer interrupt or just advance past it. In our simple VMM we just advance PC += 4 and resume. A production VMM would check pending interrupts first.',
    guestPC: 5,
    phase: 'handling',
    exitReason: 'EXCEPTION',
    ec: '0x01 (WFI)',
    handlerAction: 'Advance PC += 4 (or sleep until timer)',
  },
  {
    title: 'BRK #0 - Trap!',
    description:
      'BRK is a software breakpoint. EC = 0x3C. Unlike the others, we do NOT advance PC past BRK - the VMM should halt the VM. This is how we signal "guest is done" in our test programs.',
    guestPC: 7,
    phase: 'trapped',
    exitReason: 'EXCEPTION',
    ec: '0x3C (BRK from AArch64)',
  },
  {
    title: 'VMM Halts',
    description:
      'BRK means the guest wants to stop. The VMM breaks out of the run loop, reads final register state, cleans up the VM. Note: PC was NOT advanced - if we resumed, BRK would trap again infinitely.',
    guestPC: 7,
    phase: 'handling',
    exitReason: 'EXCEPTION',
    ec: '0x3C (BRK)',
    handlerAction: 'Break out of run loop. Do NOT advance PC.',
  },
]

const duration = REDUCED_MOTION ? 0 : 0.3

export function VmExitLoop() {
  const [step, setStep] = useState(0)
  const current = STEPS[step]

  const phaseColor = {
    running: GREEN,
    trapped: RED,
    handling: AMBER,
    resuming: BLUE,
  }[current.phase]

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

      {/* Split layout: guest code left, exit info right */}
      <div style={{ display: 'flex', gap: '1rem', flexWrap: 'wrap' }}>
        {/* Guest code listing */}
        <div style={{ flex: '1 1 220px', minWidth: 220 }}>
          <div
            style={{
              fontSize: '0.7rem',
              color: TEXT_MUTED,
              marginBottom: '0.5rem',
              textTransform: 'uppercase',
              letterSpacing: '0.05em',
            }}
          >
            Guest Code
          </div>
          <div
            style={{
              background: '#0a0a0a',
              border: `1px solid ${BORDER}`,
              padding: '0.5rem',
            }}
          >
            {GUEST_CODE.map((inst, i) => {
              const isPC = i === current.guestPC
              const isPast = i < current.guestPC
              return (
                <motion.div
                  key={i}
                  animate={{
                    backgroundColor: isPC ? phaseColor + '22' : 'transparent',
                    borderColor: isPC ? phaseColor : 'transparent',
                  }}
                  transition={{ duration: duration / 2 }}
                  style={{
                    display: 'flex',
                    gap: '0.5rem',
                    padding: '0.2rem 0.3rem',
                    fontSize: '0.7rem',
                    borderLeft: `2px solid transparent`,
                    opacity: isPast && !isPC ? 0.4 : 1,
                  }}
                >
                  <span style={{ color: TEXT_MUTED, minWidth: 80 }}>{inst.addr}</span>
                  <span
                    style={{ color: isPC ? phaseColor : '#a3a3a3', fontWeight: isPC ? 600 : 400 }}
                  >
                    {inst.asm}
                  </span>
                  {isPC && (
                    <span style={{ color: phaseColor, marginLeft: 'auto', fontSize: '0.6rem' }}>
                      {'<- PC'}
                    </span>
                  )}
                </motion.div>
              )
            })}
          </div>
        </div>

        {/* Exit info panel */}
        <div style={{ flex: '1 1 200px', minWidth: 200 }}>
          <div
            style={{
              fontSize: '0.7rem',
              color: TEXT_MUTED,
              marginBottom: '0.5rem',
              textTransform: 'uppercase',
              letterSpacing: '0.05em',
            }}
          >
            VMM State
          </div>
          <AnimatePresence mode="wait">
            <motion.div
              key={step}
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              exit={{ opacity: 0 }}
              transition={{ duration: duration / 2 }}
              style={{
                background: '#0a0a0a',
                border: `1px solid ${BORDER}`,
                padding: '0.75rem',
                display: 'flex',
                flexDirection: 'column',
                gap: '0.5rem',
              }}
            >
              {/* Phase badge */}
              <div
                style={{
                  display: 'inline-flex',
                  alignSelf: 'flex-start',
                  padding: '0.15rem 0.5rem',
                  background: phaseColor + '22',
                  border: `1px solid ${phaseColor}`,
                  color: phaseColor,
                  fontSize: '0.65rem',
                  fontWeight: 600,
                  textTransform: 'uppercase',
                  letterSpacing: '0.05em',
                }}
              >
                {current.phase}
              </div>

              {current.exitReason && (
                <div style={{ fontSize: '0.7rem' }}>
                  <span style={{ color: TEXT_MUTED }}>exit.reason: </span>
                  <span style={{ color: RED }}>{current.exitReason}</span>
                </div>
              )}
              {current.ec && (
                <div style={{ fontSize: '0.7rem' }}>
                  <span style={{ color: TEXT_MUTED }}>EC: </span>
                  <span style={{ color: AMBER }}>{current.ec}</span>
                </div>
              )}
              {current.handlerAction && (
                <div style={{ fontSize: '0.7rem' }}>
                  <span style={{ color: TEXT_MUTED }}>action: </span>
                  <span style={{ color: GREEN }}>{current.handlerAction}</span>
                </div>
              )}
              {!current.exitReason && !current.handlerAction && (
                <div style={{ fontSize: '0.7rem', color: TEXT_MUTED }}>
                  Guest executing natively...
                </div>
              )}
            </motion.div>
          </AnimatePresence>
        </div>
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
