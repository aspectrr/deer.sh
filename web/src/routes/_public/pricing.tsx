import { createFileRoute, Link } from '@tanstack/react-router'
import { useState, useCallback, useRef } from 'react'
import { Menu, X } from 'lucide-react'
import { useAuth } from '~/lib/auth'
import { useReturningVisitor } from '~/lib/use-returning-visitor'

function CopyButton({ text }: { text: string }) {
  const [copied, setCopied] = useState(false)
  const timeoutRef = useRef<ReturnType<typeof setTimeout>>(null)

  const handleCopy = useCallback(async () => {
    await navigator.clipboard.writeText(text)
    if (timeoutRef.current) clearTimeout(timeoutRef.current)
    setCopied(true)
    timeoutRef.current = setTimeout(() => setCopied(false), 2000)
  }, [text])

  return (
    <button
      onClick={handleCopy}
      className="rounded p-1 text-neutral-600 transition-colors hover:bg-neutral-800 hover:text-neutral-300"
      aria-label="Copy to clipboard"
    >
      {copied ? (
        <svg
          className="h-3.5 w-3.5 text-blue-400"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
        </svg>
      ) : (
        <svg className="h-3.5 w-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"
          />
        </svg>
      )}
    </button>
  )
}

export const Route = createFileRoute('/_public/pricing')({
  component: PricingPage,
})

const tiers = [
  {
    name: 'Open Source',
    price: 'Free',
    description: 'Self-hosted on your infrastructure',
    cta: null,
    install: 'curl -fsSL https://fluid.sh/install.sh | bash',
    features: [
      'Self-hosted daemon on your infrastructure',
      'Unlimited sandbox hosts',
      'Local Audit logging',
      'Read-only shell with allowlists',
      'PII Scrubbing from LLM requests',
      'CLI agent + TUI + MCP server',
      'Full read-only source VM access',
      'Ansible playbook generation',
      'Community support (GitHub + Discord)',
    ],
  },
  {
    name: 'Pay As You Go',
    price: 'Usage-based',
    description: 'Multi-host orchestration via control plane',
    cta: { label: 'Get Started', to: '/register' },
    features: [
      'Everything in Open Source, plus:',
      'Multi-host orchestration via control plane',
      'Source VM image caching',
      'Web dashboard for monitoring/approval',
      'PostgreSQL-backed shared state',
      'OAuth + team authentication',
    ],
    pricing: [
      '$50/mo per concurrent sandbox (1 free)',
      '$5/mo per source VM (3 free)',
      '$10/mo per daemon (1 free)',
    ],
  },
  {
    name: 'Enterprise',
    price: 'Contact us',
    description: 'Custom limits, compliance, dedicated support',
    cta: { label: 'Contact Sales', href: 'mailto:cpfeifer@madcactus.org' },
    features: [
      'Encrypted snapshots at rest',
      'Network isolation with allowlists',
      'RBAC',
      'Hosted Audit logging',
      'Secrets scrubbing for sandboxes',
      'Scoped daemon credentials',
    ],
  },
] as const

function PricingPage() {
  const { isAuthenticated } = useAuth()
  const isReturning = useReturningVisitor()
  const [mobileOpen, setMobileOpen] = useState(false)

  return (
    <>
      <header className="px-4 pt-24 pb-12 sm:px-6">
        <div className="mx-auto max-w-2xl">
          <div className="mb-6 flex items-center justify-between">
            <Link
              to="/"
              className="font-logo text-2xl tracking-tight text-white no-underline hover:no-underline md:text-3xl"
            >
              <span className="text-blue-400">$</span> fluid.sh
            </Link>
            <div className="hidden items-center gap-6 font-mono text-sm text-neutral-400 md:flex">
              <Link to="/docs/quickstart" className="transition-colors hover:text-neutral-200">
                Docs
              </Link>
              <Link to="/blog" className="transition-colors hover:text-neutral-200">
                Blog
              </Link>
              <Link to="/pricing" className="text-neutral-200">
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
        </div>
      </header>

      <main className="px-4 pb-24 sm:px-6">
        <div className="mx-auto max-w-2xl">
          <h1 className="font-logo text-2xl tracking-tight md:text-3xl">
            <span className="text-blue-400">#</span> Pricing
          </h1>
          <p className="mt-2 text-neutral-400">
            Start free with the open-source CLI. Scale with the control plane.
          </p>

          <div className="mt-12 space-y-4">
            {tiers.map((tier) => (
              <div
                key={tier.name}
                className="rounded-lg border border-neutral-800 bg-neutral-900/50 p-6"
              >
                <div className="flex items-baseline justify-between">
                  <h2 className="font-mono text-sm text-neutral-200">{tier.name}</h2>
                  <span className="font-mono text-xs text-blue-400">{tier.price}</span>
                </div>
                <p className="mt-1 text-sm text-neutral-500">{tier.description}</p>

                <ul className="mt-4 space-y-2">
                  {tier.features.map((feature) => (
                    <li key={feature} className="flex items-start gap-2 font-mono text-xs">
                      <span className="mt-0.5 text-blue-400">-</span>
                      <span className="text-neutral-400">{feature}</span>
                    </li>
                  ))}
                </ul>

                {'pricing' in tier && tier.pricing && (
                  <div className="mt-4 border-t border-neutral-800 pt-4">
                    {tier.pricing.map((line) => (
                      <div key={line} className="font-mono text-xs text-neutral-500">
                        {line}
                      </div>
                    ))}
                  </div>
                )}

                {'install' in tier && tier.install && (
                  <div className="mt-4 overflow-hidden rounded border border-neutral-800 bg-neutral-950">
                    <div className="flex items-center justify-between gap-2 px-3 py-2">
                      <div className="min-w-0 overflow-x-auto font-mono text-xs text-neutral-400">
                        <span className="text-blue-400 select-none">$ </span>
                        <span className="whitespace-nowrap">{tier.install}</span>
                      </div>
                      <CopyButton text={tier.install} />
                    </div>
                  </div>
                )}

                {tier.cta && (
                  <div className="mt-6">
                    {'to' in tier.cta ? (
                      <Link
                        to={tier.cta.to}
                        className="inline-block rounded border border-blue-500/30 bg-blue-500/10 px-4 py-2 font-mono text-xs text-blue-400 no-underline transition-colors hover:border-blue-500/50 hover:bg-blue-500/20 hover:no-underline"
                      >
                        {tier.cta.label}
                      </Link>
                    ) : (
                      <a
                        href={tier.cta.href}
                        className="inline-block rounded border border-neutral-700 px-4 py-2 font-mono text-xs text-neutral-400 transition-colors hover:border-neutral-500 hover:text-neutral-200"
                      >
                        {tier.cta.label}
                      </a>
                    )}
                  </div>
                )}
              </div>
            ))}
          </div>
        </div>
      </main>
    </>
  )
}
