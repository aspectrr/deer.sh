import { createFileRoute, Link } from '@tanstack/react-router'
import { useState, useCallback, useRef } from 'react'
import { Menu, X, ChevronDown, ArrowRight } from 'lucide-react'
import { motion, AnimatePresence } from 'framer-motion'
import { useInView } from 'react-intersection-observer'
import { ScriptedDemo } from '~/components/landing/scripted-demo'
import { ArchitectureAnimation } from '~/components/landing/architecture-animation'
import type { DiagramPhase } from '~/lib/diagram-phases'
import { useAuth } from '~/lib/auth'
import { usePostHog } from '~/lib/posthog'
import { useReturningVisitor } from '~/lib/use-returning-visitor'

export const Route = createFileRoute('/_public/')({
  component: LandingPage,
})

const CopyIcon = () => (
  <svg className="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path
      strokeLinecap="round"
      strokeLinejoin="round"
      strokeWidth={2}
      d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"
    />
  </svg>
)

const CheckIcon = () => (
  <svg className="h-4 w-4 text-blue-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
  </svg>
)

function CopyButton({ command, method }: { command: string; method?: string }) {
  const posthog = usePostHog()
  const [copied, setCopied] = useState(false)
  const timeoutRef = useRef<ReturnType<typeof setTimeout>>(null)

  const handleCopy = useCallback(async () => {
    await navigator.clipboard.writeText(command)
    if (method) {
      posthog.capture('install_command_copied', { method })
    }
    if (timeoutRef.current) clearTimeout(timeoutRef.current)
    setCopied(true)
    timeoutRef.current = setTimeout(() => setCopied(false), 2000)
  }, [command, method, posthog])

  return (
    <button
      onClick={handleCopy}
      className="rounded p-2 text-neutral-600 transition-colors hover:bg-neutral-800 hover:text-neutral-300"
      aria-label="Copy to clipboard"
    >
      {copied ? <CheckIcon /> : <CopyIcon />}
    </button>
  )
}

function SecuritySection({
  children,
  className,
}: {
  children: React.ReactNode
  className?: string
}) {
  const { ref, inView } = useInView({ triggerOnce: true, threshold: 0.15 })
  return (
    <motion.section
      ref={ref}
      initial={{ opacity: 0, y: 20 }}
      animate={inView ? { opacity: 1, y: 0 } : {}}
      transition={{ duration: 0.5 }}
      className={className}
    >
      {children}
    </motion.section>
  )
}

const faqs = [
  {
    question: 'You want me to give an AI agent SSH access to my production VMs?',
    answer:
      'Not unrestricted SSH access. Fluid creates a dedicated fluid-readonly user with a restricted login shell. A client-side allowlist validates every command against ~50 permitted read-only commands (cat, ls, grep, ps, journalctl, etc.) before it is even sent. Server-side, the restricted shell blocks 50+ destructive patterns - sudo, rm, mv, chmod, wget, python, bash - at the OS level. Command substitution ($(...), backticks), output redirection, and subshells are all blocked. Even if the AI constructs something creative, the shell will not execute it.',
  },
  {
    question: 'What data leaves my environment?',
    answer:
      'Command output passes through a PII redactor before reaching the LLM. IP addresses, API keys (sk-..., Bearer tokens), AWS credentials (AKIA...), SSH private keys, and connection strings are replaced with deterministic tokens like [REDACTED_IP_1]. The same value always maps to the same token within a session, so the AI can reason about relationships without seeing the actual data. You can add custom patterns and allowlists. You can choose to use a different OpenAI-compatible endpoint within /settings',
  },
  {
    question: 'What happens if the AI hallucinates a destructive command?',
    answer:
      'It gets blocked by technical enforcement, not a system prompt. The client-side allowlist rejects the command before it touches SSH. If somehow bypassed, the server-side restricted shell - installed as the actual login shell for the fluid-readonly user - blocks it independently. Both layers parse pipelines, detect chained commands (;, &&, ||), and validate each segment. This is defense in depth: two independent enforcement layers, neither relying on LLM compliance.',
  },
  {
    question: 'Who else can see my infrastructure through this?',
    answer:
      'Nobody. Fluid runs on your infrastructure. The CLI runs on your workstation. The daemon runs on your sandbox host. Source VM access uses SSH from your network. For multi-daemon setups, there is a hosted control-plane option used for remote agent execution, sandbox management, and enterprise features. SOC2 compliant. We use anonymized and redacted telemetry to improve the product.',
  },
  {
    question: 'Does this open a new attack surface?',
    answer:
      'Fluid uses SSH certificates with 30-minute TTLs issued by a local CA, not persistent credentials. Certificates are scoped to a single principal (fluid-readonly) with port forwarding, agent forwarding, and X11 forwarding disabled. The daemon listens on gRPC :9091 for local CLI communication. No new ports are opened on your production VMs - Fluid uses standard SSH (port 22).',
  },
  {
    question: 'Can I audit everything it did after the fact?',
    answer:
      'Every tool call, every LLM request and response, and every session start/end is logged in JSONL format. Each entry contains a SHA-256 hash of the previous entry plus its own content, forming a tamper-evident chain. Run VerifyChain() on the log file - it will detect if any entry was modified, inserted, or deleted. Logs record tool name, arguments, results, duration, and timestamps. User input records length only (not content) for privacy.',
  },
]

function FAQSection() {
  const [openIndex, setOpenIndex] = useState<number | null>(null)
  const { ref, inView } = useInView({ triggerOnce: true, threshold: 0.1 })

  return (
    <motion.section
      ref={ref}
      initial={{ opacity: 0, y: 20 }}
      animate={inView ? { opacity: 1, y: 0 } : {}}
      transition={{ duration: 0.5 }}
      className="mt-24"
    >
      <h2 className="mb-6 text-xl text-neutral-200">FAQ</h2>
      <div className="border border-neutral-800">
        {faqs.map((faq, i) => (
          <div key={i} className="border-b border-neutral-800 last:border-b-0">
            <button
              onClick={() => setOpenIndex(openIndex === i ? null : i)}
              className="flex w-full items-center justify-between gap-4 px-4 py-3 text-left font-mono text-sm text-neutral-200 transition-colors hover:bg-neutral-900"
            >
              <span>{faq.question}</span>
              <ChevronDown
                className={`h-4 w-4 shrink-0 text-neutral-500 transition-transform duration-200 ${openIndex === i ? 'rotate-180' : ''}`}
              />
            </button>
            <AnimatePresence>
              {openIndex === i && (
                <motion.div
                  initial={{ height: 0, opacity: 0 }}
                  animate={{ height: 'auto', opacity: 1 }}
                  exit={{ height: 0, opacity: 0 }}
                  transition={{ duration: 0.2 }}
                  className="overflow-hidden"
                >
                  <div className="px-4 pb-4 font-mono text-xs leading-relaxed text-neutral-400">
                    {faq.answer}
                  </div>
                </motion.div>
              )}
            </AnimatePresence>
          </div>
        ))}
      </div>
    </motion.section>
  )
}

const installTabs = [
  {
    id: 'go',
    label: 'go',
    command: 'go install github.com/aspectrr/fluid.sh/fluid-cli/cmd/fluid@latest',
    method: 'go_install',
  },
  {
    id: 'curl',
    label: 'curl',
    command: 'curl -fsSL https://fluid.sh/install.sh | bash',
    method: 'curl',
  },
] as const

function LandingPage() {
  const { isAuthenticated } = useAuth()
  const isReturning = useReturningVisitor()
  const [activeTab, setActiveTab] = useState<string>('go')
  const [diagramPhase, setDiagramPhase] = useState<DiagramPhase>('idle')
  const [mobileOpen, setMobileOpen] = useState(false)

  const currentTab = installTabs.find((t) => t.id === activeTab)!

  return (
    <>
      <header className="px-4 py-24 sm:px-6">
        <div className="mx-auto max-w-2xl">
          <div className="mb-6 flex items-center justify-between">
            <div className="flex items-center gap-3">
              <h1 className="font-logo text-2xl tracking-tight md:text-3xl">
                <span className="text-blue-400">$</span> fluid.sh
              </h1>
            </div>
            <div className="hidden items-center gap-6 font-mono text-sm text-neutral-400 md:flex">
              <Link to="/docs/quickstart" className="transition-colors hover:text-neutral-200">
                Docs
              </Link>
              <Link to="/blog" className="transition-colors hover:text-neutral-200">
                Blog
              </Link>
              <Link to="/pricing" className="transition-colors hover:text-neutral-200">
                Pricing
              </Link>
              <a
                href="https://github.com/aspectrr/fluid.sh"
                target="_blank"
                rel="noopener noreferrer"
                className="transition-colors hover:text-neutral-200"
              >
                GitHub
              </a>
              <a
                href="https://discord.gg/4WGGXJWm8J"
                target="_blank"
                rel="noopener noreferrer"
                className="transition-colors hover:text-neutral-200"
              >
                Discord
              </a>
              {isAuthenticated ? (
                <Link
                  to="/dashboard"
                  className="rounded border border-neutral-700 px-3 py-1 text-neutral-300 transition-colors hover:border-neutral-500 hover:text-neutral-100"
                >
                  Dashboard
                </Link>
              ) : isReturning ? (
                <Link
                  to="/login"
                  className="rounded border border-neutral-700 px-3 py-1 text-neutral-300 transition-colors hover:border-neutral-500 hover:text-neutral-100"
                >
                  Login
                </Link>
              ) : (
                <Link
                  to="/register"
                  className="rounded border border-neutral-700 px-3 py-1 text-neutral-300 transition-colors hover:border-neutral-500 hover:text-neutral-100"
                >
                  Sign Up
                </Link>
              )}
            </div>
            <button
              className="text-neutral-400 hover:text-white md:hidden"
              onClick={() => setMobileOpen(!mobileOpen)}
            >
              {mobileOpen ? <X className="h-5 w-5" /> : <Menu className="h-5 w-5" />}
            </button>
          </div>

          {/* Mobile nav overlay */}
          {mobileOpen && (
            <div
              className="fixed inset-0 z-30 bg-black md:hidden"
              onClick={() => setMobileOpen(false)}
            >
              <nav
                className="flex flex-col gap-6 p-8 pt-20 font-mono text-lg text-neutral-300"
                onClick={(e) => e.stopPropagation()}
              >
                <Link
                  to="/docs/quickstart"
                  onClick={() => setMobileOpen(false)}
                  className="transition-colors hover:text-white"
                >
                  Docs
                </Link>
                <Link
                  to="/blog"
                  onClick={() => setMobileOpen(false)}
                  className="transition-colors hover:text-white"
                >
                  Blog
                </Link>
                <Link
                  to="/pricing"
                  onClick={() => setMobileOpen(false)}
                  className="transition-colors hover:text-white"
                >
                  Pricing
                </Link>
                <a
                  href="https://github.com/aspectrr/fluid.sh"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="transition-colors hover:text-white"
                  onClick={() => setMobileOpen(false)}
                >
                  GitHub
                </a>
                <a
                  href="https://discord.gg/4WGGXJWm8J"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="transition-colors hover:text-white"
                  onClick={() => setMobileOpen(false)}
                >
                  Discord
                </a>
                <Link
                  to={isAuthenticated ? '/dashboard' : isReturning ? '/login' : '/register'}
                  onClick={() => setMobileOpen(false)}
                  className="transition-colors hover:text-white"
                >
                  {isAuthenticated ? 'Dashboard' : isReturning ? 'Login' : 'Sign Up'}
                </Link>
              </nav>
            </div>
          )}
          <p className="font-logo mt-2 text-lg tracking-tight text-neutral-200">
            Claude Code for working on Linux Servers.
          </p>
          <p className="my-2 text-neutral-400">
            Read-only shell access. PII redaction. Tamper-evident audit logs. Fluid gets just the
            access it needs to debug and manage your servers - nothing more.
          </p>
          <div className="mt-4 flex items-center gap-3">
            <a
              href="#install"
              className="inline-flex items-center gap-2 bg-blue-500 px-5 py-2 font-mono text-sm text-white transition-colors hover:bg-blue-600"
            >
              Get Started <ArrowRight className="h-4 w-4" />
            </a>
            <Link
              to="/docs/quickstart"
              className="inline-flex items-center gap-2 border border-neutral-700 px-5 py-2 font-mono text-sm text-neutral-300 transition-colors hover:border-neutral-500 hover:text-neutral-100"
            >
              Read the Docs
            </Link>
          </div>
        </div>
        <div className="mx-auto mt-6 max-w-2xl">
          <ScriptedDemo onPhase={setDiagramPhase} />
          <ArchitectureAnimation phase={diagramPhase} />
        </div>
        <div className="mx-auto max-w-2xl">
          {/* Read-Only Shell */}
          <SecuritySection className="mt-16">
            <h3 className="font-logo text-lg tracking-tight">
              <span className="text-blue-400">&gt;</span> read-only shell
            </h3>
            <p className="mt-2 text-neutral-400">
              Client-side command allowlist validates every command before execution. Server-side
              restricted shell blocks destructive operations even if the allowlist is bypassed.
              Defense in depth - not just a system prompt.
            </p>
            <div className="mt-4 overflow-hidden border border-neutral-800 bg-neutral-900">
              <div className="flex items-center gap-2 border-b border-neutral-800 px-4 py-2">
                <span className="font-mono text-xs text-neutral-500">fluid-readonly-shell</span>
              </div>
              <div className="p-4 font-mono text-xs">
                <div className="text-green-400">allowed:</div>
                <div className="ml-4 text-neutral-400">
                  cat ls grep ps systemctl status journalctl df ss dig ...
                </div>
                <div className="mt-2 text-red-400">blocked:</div>
                <div className="ml-4 text-neutral-500 line-through">
                  sudo rm mv chmod wget curl python bash sh
                </div>
                <div className="mt-2 text-red-400">blocked patterns:</div>
                <div className="ml-4 text-neutral-500 line-through">
                  {'$(...) `...` >(...) > >> |&'}
                </div>
              </div>
            </div>
          </SecuritySection>

          {/* PII Redaction */}
          <SecuritySection className="mt-12">
            <h3 className="font-logo text-lg tracking-tight">
              <span className="text-blue-400">&gt;</span> pii redaction
            </h3>
            <p className="mt-2 text-neutral-400">
              Sensitive data is replaced with deterministic tokens before it reaches the LLM. IP
              addresses, API keys, AWS credentials, SSH private keys, connection strings - detected
              and redacted automatically.
            </p>
            <div className="mt-4 overflow-hidden border border-neutral-800 bg-neutral-900">
              <div className="flex items-center gap-2 border-b border-neutral-800 px-4 py-2">
                <span className="font-mono text-xs text-neutral-500">redactor output</span>
              </div>
              <div className="space-y-1 p-4 font-mono text-xs">
                <div>
                  <span className="text-neutral-500">upstream: </span>
                  <span className="text-blue-400">[REDACTED_IP_1]</span>
                  <span className="text-neutral-500">:3000</span>
                </div>
                <div>
                  <span className="text-neutral-500">api_key: </span>
                  <span className="text-blue-400">[REDACTED_KEY_1]</span>
                </div>
                <div>
                  <span className="text-neutral-500">db: </span>
                  <span className="text-blue-400">[REDACTED_SECRET_1]</span>
                </div>
                <div className="mt-2 text-neutral-600">
                  {'// categories: IP, KEY, SECRET, HOST, PATH'}
                </div>
              </div>
            </div>
          </SecuritySection>

          {/* Audit Trail */}
          <SecuritySection className="mt-12">
            <h3 className="font-logo text-lg tracking-tight">
              <span className="text-blue-400">&gt;</span> audit trail
            </h3>
            <p className="mt-2 text-neutral-400">
              Every tool call, LLM request, and session is logged to tamper-evident JSONL files.
              SHA-256 hash chains link each entry to the previous one. Run VerifyChain() to detect
              any tampering.
            </p>
            <div className="mt-4 overflow-hidden border border-neutral-800 bg-neutral-900">
              <div className="flex items-center gap-2 border-b border-neutral-800 px-4 py-2">
                <span className="font-mono text-xs text-neutral-500">audit.jsonl</span>
              </div>
              <div className="space-y-1 overflow-x-auto p-4 font-mono text-xs">
                <div className="whitespace-nowrap">
                  <span className="text-neutral-500">{'{"'}</span>
                  <span className="text-blue-400">seq</span>
                  <span className="text-neutral-500">{'":1,"'}</span>
                  <span className="text-blue-400">type</span>
                  <span className="text-neutral-500">{'":"tool_call","'}</span>
                  <span className="text-blue-400">tool</span>
                  <span className="text-neutral-500">{'":"run_source_command","'}</span>
                  <span className="text-blue-400">prev_hash</span>
                  <span className="text-neutral-500">{'":"0000...","'}</span>
                  <span className="text-blue-400">hash</span>
                  <span className="text-neutral-500">{'":"a3f2..."}'}</span>
                </div>
                <div className="whitespace-nowrap">
                  <span className="text-neutral-500">{'{"'}</span>
                  <span className="text-blue-400">seq</span>
                  <span className="text-neutral-500">{'":2,"'}</span>
                  <span className="text-blue-400">type</span>
                  <span className="text-neutral-500">{'":"llm_response","'}</span>
                  <span className="text-blue-400">prev_hash</span>
                  <span className="text-neutral-500">{'":"a3f2...","'}</span>
                  <span className="text-blue-400">hash</span>
                  <span className="text-neutral-500">{'":"b7c1..."}'}</span>
                </div>
              </div>
            </div>
          </SecuritySection>

          {/* Allowlists */}
          <SecuritySection className="mt-12">
            <h3 className="font-logo text-lg tracking-tight">
              <span className="text-blue-400">&gt;</span> allowlists
            </h3>
            <p className="mt-2 text-neutral-400">
              Explicit command allowlist you can inspect and customize. Subcommand restrictions for
              tools like systemctl - only status, show, list-units are permitted. No implicit trust.
            </p>
            <div className="mt-4 overflow-hidden border border-neutral-800 bg-neutral-900">
              <div className="flex items-center gap-2 border-b border-neutral-800 px-4 py-2">
                <span className="font-mono text-xs text-neutral-500">subcommand restrictions</span>
              </div>
              <div className="space-y-1 p-4 font-mono text-xs">
                <div>
                  <span className="text-neutral-200">systemctl </span>
                  <span className="text-green-400">
                    status show list-units is-active is-enabled
                  </span>
                </div>
                <div>
                  <span className="text-neutral-200">dpkg </span>
                  <span className="text-green-400">-l --list</span>
                </div>
                <div>
                  <span className="text-neutral-200">apt </span>
                  <span className="text-green-400">list</span>
                </div>
                <div>
                  <span className="text-neutral-200">rpm </span>
                  <span className="text-green-400">-qa -q</span>
                </div>
                <div className="mt-2 text-neutral-600">{'// all other subcommands blocked'}</div>
              </div>
            </div>
          </SecuritySection>

          <FAQSection />

          <h2 id="install" className="mt-24 mb-4 text-xl text-neutral-200">
            Installation
          </h2>
          <p className="my-2 text-neutral-400">
            This will install the <span className="text-blue-400">$</span>{' '}
            <span className="font-logo text-white">fluid.sh</span> terminal agent / mcp server meant
            to be installed on your local workstation.
          </p>
          {/* Install tabs */}
          <div className="mt-8 overflow-hidden rounded-lg bg-neutral-900">
            <div className="flex border-b border-neutral-800">
              {installTabs.map((tab) => (
                <button
                  key={tab.id}
                  onClick={() => setActiveTab(tab.id)}
                  className={`border-b-2 px-4 py-2 font-mono text-sm transition-colors ${
                    activeTab === tab.id
                      ? 'border-blue-400 text-blue-400'
                      : 'border-transparent text-neutral-500 hover:text-neutral-300'
                  }`}
                >
                  {tab.label}
                </button>
              ))}
            </div>
            <div className="px-5 py-4">
              <div className="flex items-center justify-between gap-4 font-mono text-sm">
                <div className="min-w-0 overflow-x-auto text-neutral-400">
                  <span className="text-blue-400 select-none">$ </span>
                  <span className="whitespace-nowrap">{currentTab.command}</span>
                </div>
                <CopyButton command={currentTab.command} method={currentTab.method} />
              </div>
              <div className="mt-4 border-t border-neutral-800 pt-4 font-mono text-sm text-neutral-500">
                <span className="text-blue-400 select-none">$ </span>fluid
              </div>
            </div>
          </div>
        </div>
      </header>
    </>
  )
}
