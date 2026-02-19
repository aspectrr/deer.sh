import { createFileRoute, Link } from '@tanstack/react-router'
import { useState, useEffect, useRef, useCallback } from 'react'
import { ScriptedDemo } from '~/components/landing/scripted-demo'
import { ArchitectureAnimation } from '~/components/landing/architecture-animation'
import type { DiagramPhase } from '~/lib/diagram-phases'
import { useAuth } from '~/lib/auth'

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

function CopyButton({ command }: { command: string }) {
  const [copied, setCopied] = useState(false)
  const timeoutRef = useRef<ReturnType<typeof setTimeout>>(null)

  const handleCopy = useCallback(async () => {
    await navigator.clipboard.writeText(command)
    if (timeoutRef.current) clearTimeout(timeoutRef.current)
    setCopied(true)
    timeoutRef.current = setTimeout(() => setCopied(false), 2000)
  }, [command])

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

const installTabs = [
  {
    id: 'go',
    label: 'go',
    command: 'go install github.com/aspectrr/fluid.sh/fluid/cmd/fluid@latest',
  },
  {
    id: 'curl',
    label: 'curl',
    command: 'curl -fsSL https://fluid.sh/install.sh | bash',
  },
] as const

function LandingPage() {
  const { isAuthenticated } = useAuth()
  const [activeTab, setActiveTab] = useState<string>('go')
  const [diagramPhase, setDiagramPhase] = useState<DiagramPhase>('idle')
  const workflowRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const el = workflowRef.current
    if (!el) return

    const steps = el.querySelectorAll<HTMLElement>('.workflow-step')
    const progress = el.querySelector<HTMLElement>('#workflow-progress')
    if (steps.length === 0) return

    const observer = new IntersectionObserver(
      (entries) => {
        entries.forEach((entry) => {
          if (entry.isIntersecting) {
            steps.forEach((step, index) => {
              setTimeout(() => {
                step.classList.remove('opacity-0', 'translate-y-4')
                step.classList.add('opacity-100', 'translate-y-0')

                const dot = step.querySelector('.workflow-dot')
                const num = step.querySelector('.workflow-dot span')
                dot?.classList.remove('border-neutral-700')
                dot?.classList.add('border-blue-500', 'shadow-[0_0_10px_2px_rgba(59,130,246,0.3)]')
                num?.classList.remove('text-neutral-500')
                num?.classList.add('text-blue-400')

                if (progress) {
                  progress.style.width = `${((index + 1) / steps.length) * 100}%`
                }
              }, index * 250)
            })
            observer.disconnect()
          }
        })
      },
      { threshold: 0.5 }
    )

    observer.observe(steps[0])
    return () => observer.disconnect()
  }, [])

  const currentTab = installTabs.find((t) => t.id === activeTab)!

  return (
    <>
      <header className="px-6 py-24">
        <div className="mx-auto max-w-2xl">
          <div className="mb-6 flex items-center justify-between">
            <div className="flex items-center gap-3">
              <h1 className="font-logo text-2xl tracking-tight md:text-3xl">
                <span className="text-blue-400">$</span> fluid.sh
              </h1>
            </div>
            <div className="flex items-center gap-6 font-mono text-sm text-neutral-400">
              <Link to="/blog" className="transition-colors hover:text-neutral-200">
                Blog
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
              <Link
                to={isAuthenticated ? '/dashboard' : '/login'}
                className="rounded border border-neutral-700 px-3 py-1 text-neutral-300 transition-colors hover:border-neutral-500 hover:text-neutral-100"
              >
                {isAuthenticated ? 'Dashboard' : 'Login'}
              </Link>
            </div>
          </div>
          <p className="mt-2 text-neutral-400">Claude Code for Linux Servers</p>
          <p className="my-2 text-neutral-400">
            Fluid enables System Administrators, SREs, Platform Engineers, and DevOps Engineers to
            diagnose issues on servers, run investigations, and fix problems safely with microVM
            sandboxes.
          </p>
        </div>
        <div className="mx-auto mt-6 max-w-2xl">
          <ScriptedDemo onPhase={setDiagramPhase} />
          <ArchitectureAnimation phase={diagramPhase} />
        </div>
        <div className="mx-auto max-w-2xl">
          <p className="mt-6 text-neutral-400">Fluid works in four phases.</p>

          <h4 className="font-logo mt-2 text-lg tracking-tight">
            <span className="text-blue-400">&gt;</span> read
          </h4>
          <p className="mt-2 text-neutral-400">
            Debug production VMs with Fluid's read-only mode. Let Fluid investigate an issue by
            querying log files, reading systemctl changes, or accessing config.
          </p>
          <h4 className="font-logo mt-2 text-lg tracking-tight">
            <span className="text-blue-400">&gt;</span> edit
          </h4>
          <p className="mt-2 text-neutral-400">
            Edit VM sandboxes with Fluid's edit mode. After Fluid has some context on the system, it
            will create a sandbox by cloning the VM. Fluid will then make changes, edit files, and
            iterate within the sandbox until the issue is resolved.
          </p>
          <h4 className="font-logo mt-2 text-lg tracking-tight">
            <span className="text-blue-400">&gt;</span> ansible
          </h4>
          <p className="mt-2 text-neutral-400">
            Once Fluid has fixed the issue on the sandbox, it will begin to create an Ansible
            playbook to reconstruct the fix on production.
          </p>
          <h4 className="font-logo mt-2 text-lg tracking-tight">
            <span className="text-blue-400">&gt;</span> cleanup
          </h4>
          <p className="mt-2 text-neutral-400">
            After the changes are made, open sandboxes will be deleted when the Fluid CLI is closed
            or you can ask Fluid to delete them manually.
          </p>

          <h2 className="mt-24 mb-4 text-xl text-neutral-200">Installation</h2>
          <p className="my-2 text-neutral-400">
            This will install the <span className="text-blue-400">$</span>{' '}
            <span className="font-logo text-white">fluid.sh</span> terminal agent meant to be
            installed on your local workstation. It will take you through daemon setup and
            configuration.
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
                <div className="text-neutral-400">
                  <span className="text-blue-400 select-none">$ </span>
                  {currentTab.command}
                </div>
                <CopyButton command={currentTab.command} />
              </div>
              <div className="mt-4 border-t border-neutral-800 pt-4 font-mono text-sm text-neutral-500">
                <span className="text-blue-400 select-none">$ </span>fluid
              </div>
            </div>
          </div>
        </div>
      </header>
      {/*
      <main className="px-6 pb-24">
        <div className="mx-auto max-w-2xl space-y-16">
          <section ref={workflowRef}>
            <h2 className="mb-6 text-xl text-neutral-200">Built for where you already work</h2>
            <div className="mb-12 grid grid-cols-1 gap-3 md:grid-cols-2">
              <FeatureCard
                iconString="[~]"
                title="Sandbox Isolation"
                description="Clone VMs instantly. Test changes in isolation before touching production."
              />
              <FeatureCard
                iconString="ls"
                title="Context-Aware"
                description="Fluid explores your host first - OS, packages, CLI tools - then adapts."
              />
              <FeatureCard
                iconString=">>>"
                title="Full Audit Trail"
                description="Every command logged. Every change tracked. Review before production."
              />
              <FeatureCard
                iconString=".yaml"
                title="Ansible Playbooks"
                description="Auto-generates playbooks from sandbox work. Reproducible infrastructure."
              />
            </div>
          </section>
        </div>
      </main>*/}
    </>
  )
}
