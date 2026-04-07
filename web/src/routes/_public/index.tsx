import { createFileRoute, Link } from '@tanstack/react-router'
import { useState } from 'react'
import { Menu, X, ArrowRight, CheckCircle } from 'lucide-react'
import { motion } from 'framer-motion'
import { useInView } from 'react-intersection-observer'

export const Route = createFileRoute('/_public/')({
  component: ConsultingHomePage,
})

function FadeIn({ children, className }: { children: React.ReactNode; className?: string }) {
  const { ref, inView } = useInView({ triggerOnce: true, threshold: 0.1 })
  return (
    <motion.div
      ref={ref}
      initial={{ opacity: 0, y: 20 }}
      animate={inView ? { opacity: 1, y: 0 } : {}}
      transition={{ duration: 0.5 }}
      className={className}
    >
      {children}
    </motion.div>
  )
}

const services = [
  {
    id: 'cluster-setup',
    title: 'Cluster Setup',
    tagline: 'production-ready from day one',
    description:
      'End-to-end Elasticsearch cluster provisioning - sizing, shard strategy, index lifecycle management, and cross-cluster replication. We get your cluster right the first time.',
    details: [
      'Node sizing and hardware recommendations',
      'Index lifecycle management (ILM) policies',
      'Cross-cluster replication and failover',
      'Security hardening and TLS configuration',
      'Snapshot and restore strategy',
    ],
  },
  {
    id: 'pipeline-architecture',
    title: 'Pipeline Architecture',
    tagline: 'data flows that actually work',
    description:
      'Logstash and Beats pipeline design for your specific data sources. We architect pipelines that are maintainable, observable, and built to handle production load.',
    details: [
      'Logstash pipeline design and optimization',
      'Beats configuration for servers, containers, and cloud',
      'Dead letter queue setup and error handling',
      'Parsing complex log formats (JSON, multiline, custom)',
      'Kafka and message queue integration',
    ],
  },
  {
    id: 'pipeline-debugging',
    title: 'Pipeline Debugging',
    tagline: 'find it fast, fix it right',
    description:
      'Broken ingestion, missing documents, mapping conflicts, slow queries - we diagnose and resolve ELK issues that your team has been chasing for weeks.',
    details: [
      'Ingestion pipeline failure diagnosis',
      'Mapping conflict resolution',
      'Query performance analysis',
      'Memory pressure and GC investigation',
      'Hot shard and split-brain remediation',
    ],
  },
  {
    id: 'system-integration',
    title: 'System Integration',
    tagline: 'connect elk to everything',
    description:
      'Integrate ELK with your existing infrastructure - SIEMs, monitoring platforms, ticketing systems, cloud providers. We handle the connective tissue.',
    details: [
      'SIEM and security tooling integration',
      'AWS, GCP, and Azure log ingestion',
      'Alerting and incident routing setup',
      'Kibana dashboard and visualization design',
      'APM and distributed tracing configuration',
    ],
  },
]

const logos = [
  { src: '/images/logos/Indiana-University-Logo-465862039.png', name: 'Indiana University' },
  { src: '/images/logos/omnisoc-768w-3770841716.png', name: 'OmniSOC' },
  { src: '/images/logos/internet2-logo-png_seeklogo-72798-1654782020.png', name: 'Internet2' },
  { src: '/images/logos/Purdue-Logo-PNG-HD-2002129623.png', name: 'Purdue' },
  { src: '/images/logos/Clemson-University-Emblem-2654677447.png', name: 'Clemson University' },
  { src: '/images/logos/Virginia-Tech-Logo-PNG-Photo-2687094516.png', name: 'Virginia Tech' },
  {
    src: '/images/logos/lehigh-mountain-hawks-ncaa-logo-sticker-ncaa142-5215-4a50fb-2570742203.png',
    name: 'Lehigh University',
  },
  {
    src: '/images/logos/Santa_Clara_University_Logo-588491151.png',
    name: 'Santa Clara University',
  },
  { src: '/images/logos/ucsc-logo-png-4-4148969120.png', name: 'UC Santa Cruz' },
  { src: '/images/logos/UCSD-Seal-Logo-3371647571.png', name: 'UC San Diego' },
  { src: '/images/logos/951515-249938965.png', name: '' },
]

// 4 copies per half = 8 total; mr-12 on each item makes slot width exact (160+48=208px)
// so -50% = exactly 4 sets, seamless on any screen width
const marqueeItems = Array.from({ length: 8 }, () => logos).flat()

function LogoMarquee() {
  return (
    <div className="w-full overflow-hidden border-y border-neutral-800 py-12">
      <div
        className="flex items-center"
        style={{
          width: 'max-content',
          animation: 'marquee 40s linear infinite',
        }}
      >
        {marqueeItems.map((logo, i) => (
          <div key={i} className="mr-16 flex h-12 w-40 shrink-0 items-center justify-center">
            <img
              src={logo.src}
              alt={logo.name}
              className="max-h-full max-w-full object-contain opacity-40 grayscale transition-opacity duration-300 hover:opacity-70"
            />
          </div>
        ))}
      </div>
    </div>
  )
}

const process = [
  {
    step: '01',
    title: 'Discovery',
    description:
      'We start with a call to understand your current setup, pain points, and goals. No templates - just questions specific to your environment.',
  },
  {
    step: '02',
    title: 'Architecture Review',
    description:
      'We audit your existing ELK configuration, identify gaps, and produce a written assessment with prioritized recommendations.',
  },
  {
    step: '03',
    title: 'Implementation',
    description:
      'We do the work. Changes are tested in isolated environments before touching production. Every change is documented.',
  },
  {
    step: '04',
    title: 'Handoff',
    description:
      'Your team gets runbooks, architecture diagrams, and a walkthrough of everything we built. We leave you self-sufficient.',
  },
]

function ConsultingHomePage() {
  const [mobileOpen, setMobileOpen] = useState(false)
  const [expandedService, setExpandedService] = useState<string | null>(null)

  return (
    <>
      {/* Nav */}
      <header className="px-4 pt-8 pb-0 sm:px-6">
        <div className="mx-auto max-w-2xl">
          <div className="mb-0 flex items-center justify-between">
            <Link
              to="/"
              className="font-logo text-2xl tracking-tight text-white no-underline hover:no-underline md:text-3xl"
            >
              🦌 <span className="text-green-800">deer.sh</span>
            </Link>
            <div className="hidden items-center gap-6 font-mono text-sm text-neutral-400 md:flex">
              <a href="#services" className="transition-colors hover:text-neutral-200">
                Services
              </a>
              <a href="#case-studies" className="transition-colors hover:text-neutral-200">
                Case Studies
              </a>
              <Link to="/product" className="transition-colors hover:text-neutral-200">
                Product
              </Link>
              <a
                href="#contact"
                className="inline-flex items-center gap-1 rounded border border-green-900/40 bg-green-900/10 px-3 py-1 text-green-800 transition-colors hover:border-green-900/60 hover:bg-green-900/20"
              >
                Get in Touch
              </a>
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
                <a
                  href="#services"
                  onClick={() => setMobileOpen(false)}
                  className="transition-colors hover:text-white"
                >
                  Services
                </a>
                <a
                  href="#case-studies"
                  onClick={() => setMobileOpen(false)}
                  className="transition-colors hover:text-white"
                >
                  Case Studies
                </a>
                <Link
                  to="/product"
                  onClick={() => setMobileOpen(false)}
                  className="transition-colors hover:text-white"
                >
                  Product
                </Link>
                <a
                  href="#contact"
                  onClick={() => setMobileOpen(false)}
                  className="transition-colors hover:text-white"
                >
                  Get in Touch
                </a>
              </nav>
            </div>
          )}
        </div>
      </header>

      {/* Hero */}
      <section className="px-4 pt-20 pb-20 sm:px-6">
        <div className="mx-auto max-w-2xl">
          <motion.div
            initial={{ opacity: 0, y: 16 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.6 }}
          >
            <div className="mb-3 inline-block border border-green-900/30 px-2 py-0.5 font-mono text-xs text-green-800">
              ELK Stack Consulting
            </div>
            <h1 className="font-logo text-3xl tracking-tight text-neutral-100 md:text-4xl">
              Expert ELK Stack Consulting for Higher Education
            </h1>
            <p className="mt-4 leading-relaxed text-neutral-400">
              Cluster setup, pipeline architecture, and system integration - done right. We work
              alongside your team to build ELK infrastructure that handles production scale without
              the months of trial and error.
            </p>
            <div className="mt-6 flex items-center gap-3">
              <a
                href="#contact"
                className="inline-flex items-center gap-2 bg-green-900 px-5 py-2 font-mono text-sm text-white transition-colors hover:bg-green-950"
              >
                Book a Consultation <ArrowRight className="h-4 w-4" />
              </a>
              <a
                href="#services"
                className="inline-flex items-center gap-2 border border-neutral-700 px-5 py-2 font-mono text-sm text-neutral-300 transition-colors hover:border-neutral-500 hover:text-neutral-100"
              >
                See Our Work
              </a>
            </div>
          </motion.div>

          {/* Stats row */}
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            transition={{ duration: 0.6, delay: 0.3 }}
            className="mt-12 grid grid-cols-3 border border-neutral-800"
          >
            {[
              { value: '3+', label: 'Universities' },
              { value: '<2wk', label: 'Avg. time to production' },
              { value: '100%', label: 'Documented handoffs' },
            ].map((stat, i) => (
              <div
                key={i}
                className={`p-4 text-center ${i < 2 ? 'border-r border-neutral-800' : ''}`}
              >
                <div className="font-logo text-xl text-green-800">{stat.value}</div>
                <div className="mt-1 font-mono text-xs text-neutral-500">{stat.label}</div>
              </div>
            ))}
          </motion.div>
        </div>
      </section>

      {/* Services */}
      <section id="services" className="border-t border-neutral-800/60 px-4 py-16 sm:px-6">
        <div className="mx-auto max-w-2xl">
          <FadeIn>
            <h2 className="font-logo text-xl tracking-tight text-neutral-200">
              <span className="text-green-800">#</span> Services
            </h2>
            <p className="mt-2 text-sm text-neutral-500">
              Four core engagements. Scoped to your needs, priced by outcome.
            </p>
          </FadeIn>

          <div className="mt-8 space-y-3">
            {services.map((svc) => (
              <FadeIn key={svc.id}>
                <div className="border border-neutral-800 bg-neutral-900/40">
                  <button
                    onClick={() => setExpandedService(expandedService === svc.id ? null : svc.id)}
                    className="w-full px-5 py-4 text-left"
                  >
                    <div className="flex items-start justify-between gap-4">
                      <div>
                        <div className="font-mono text-sm text-neutral-200">{svc.title}</div>
                        <div className="mt-0.5 font-mono text-xs text-green-800">{svc.tagline}</div>
                      </div>
                      <span className="mt-0.5 shrink-0 font-mono text-xs text-neutral-600">
                        {expandedService === svc.id ? '[-]' : '[+]'}
                      </span>
                    </div>
                    <p className="mt-2 text-xs leading-relaxed text-neutral-500">
                      {svc.description}
                    </p>
                  </button>

                  {expandedService === svc.id && (
                    <div className="border-t border-neutral-800 px-5 py-4">
                      <ul className="space-y-1.5">
                        {svc.details.map((d) => (
                          <li key={d} className="flex items-start gap-2 font-mono text-xs">
                            <CheckCircle className="mt-0.5 h-3 w-3 shrink-0 text-green-800" />
                            <span className="text-neutral-400">{d}</span>
                          </li>
                        ))}
                      </ul>
                    </div>
                  )}
                </div>
              </FadeIn>
            ))}
          </div>
        </div>
      </section>

      {/* Case Studies / Customers */}
      <section id="case-studies" className="border-t border-neutral-800/60 py-16">
        <div className="mx-auto max-w-2xl px-4 sm:px-6">
          <FadeIn>
            <h2 className="font-logo text-xl tracking-tight text-neutral-200">
              <span className="text-green-800">#</span> Customers
            </h2>
            <p className="mt-2 text-sm text-neutral-500">
              Our engineers have worked with these organizations.
            </p>
          </FadeIn>
        </div>
        <div className="mt-8">
          <LogoMarquee />
        </div>
      </section>

      {/* How We Work */}
      <section className="border-t border-neutral-800/60 px-4 py-16 sm:px-6">
        <div className="mx-auto max-w-2xl">
          <FadeIn>
            <h2 className="font-logo text-xl tracking-tight text-neutral-200">
              <span className="text-green-800">#</span> How We Work
            </h2>
            <p className="mt-2 text-sm text-neutral-500">
              A repeatable process that gets to production fast.
            </p>
          </FadeIn>

          <div className="mt-8 space-y-0 border border-neutral-800">
            {process.map((p, i) => (
              <FadeIn key={p.step}>
                <div
                  className={`flex gap-5 px-5 py-5 ${i < process.length - 1 ? 'border-b border-neutral-800' : ''}`}
                >
                  <div className="shrink-0 pt-0.5 font-mono text-xs text-green-800">{p.step}</div>
                  <div>
                    <div className="font-mono text-sm text-neutral-200">{p.title}</div>
                    <p className="mt-1 text-xs leading-relaxed text-neutral-500">{p.description}</p>
                  </div>
                </div>
              </FadeIn>
            ))}
          </div>
        </div>
      </section>

      {/* Contact / CTA */}
      <section id="contact" className="border-t border-neutral-800/60 px-4 py-20 sm:px-6">
        <div className="mx-auto max-w-2xl">
          <FadeIn>
            <h2 className="font-logo text-xl tracking-tight text-neutral-200">
              <span className="text-green-800">#</span> Get in Touch
            </h2>
            <p className="mt-2 leading-relaxed text-neutral-400">
              Ready to modernize your logging infrastructure? Tell us what you're working with and
              what's not working. We'll respond within one business day.
            </p>
            <div className="mt-6 flex flex-col gap-3 sm:flex-row sm:items-center">
              <a
                href="mailto:hello@deer.sh"
                className="inline-flex items-center gap-2 bg-green-900 px-6 py-3 font-mono text-sm text-white transition-colors hover:bg-green-950"
              >
                hello@deer.sh <ArrowRight className="h-4 w-4" />
              </a>
              <span className="font-mono text-xs text-neutral-600">
                or book a 30-min discovery call
              </span>
            </div>

            <div className="mt-12 border border-neutral-800 bg-neutral-900/40 p-5">
              <div className="mb-3 font-mono text-xs text-neutral-500">what to include</div>
              <ul className="space-y-1.5">
                {[
                  'Current ELK version and deployment method',
                  'Data volume (events/sec or GB/day)',
                  'What is broken or missing today',
                  'Timeline and any compliance constraints',
                ].map((item) => (
                  <li key={item} className="flex items-start gap-2 font-mono text-xs">
                    <span className="mt-0.5 text-green-800">-</span>
                    <span className="text-neutral-400">{item}</span>
                  </li>
                ))}
              </ul>
            </div>
          </FadeIn>
        </div>
      </section>
    </>
  )
}
