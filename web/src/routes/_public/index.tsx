import { createFileRoute, Link } from '@tanstack/react-router'
import { useState } from 'react'
import { Menu, X, ArrowRight, CheckCircle, ChevronDown } from 'lucide-react'
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
    id: 'pipeline-setup',
    title: 'Pipeline Setup',
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
    id: 'elastic-implementation',
    title: 'Elastic Implementation',
    tagline: 'end-to-end elastic, built for your stack',
    description:
      'Custom Elasticsearch implementation tailored to your data model and search requirements. From index design to client integration.',
    details: [
      'Custom index mapping and analyzer design',
      'Relevance tuning and scoring configuration',
      'Search template and query DSL development',
      'Elasticsearch client integration (Java, Python, Go, Node)',
      'Data modeling for documents and nested objects',
    ],
  },
  {
    id: 'fully-managed-elastic',
    title: 'Fully Managed Elastic',
    tagline: 'we run it, you use it',
    description:
      'Complete Elasticsearch operations - monitoring, scaling, maintenance, and incident response. You focus on search, we handle the infrastructure.',
    details: [
      '24/7 cluster health monitoring and alerting',
      'Capacity planning and horizontal scaling',
      'Scheduled maintenance and patch windows',
      'Incident response with documented runbooks',
      'Monthly performance and cost reports',
    ],
  },
  {
    id: 'elastic-consulting',
    title: 'Elastic Consulting',
    tagline: 'expert guidance, no fluff',
    description:
      'Architecture reviews, performance audits, and strategic guidance for teams that need an expert perspective on their ELK deployment.',
    details: [
      'Architecture review and written assessment',
      'Index strategy and mapping audit',
      'Query performance profiling and optimization',
      'Developer pairing and knowledge transfer',
      'Custom training for your engineering team',
    ],
  },
  {
    id: 'elastic-upgrades',
    title: 'Elastic Upgrades',
    tagline: 'stay current, stay stable',
    description:
      'Major and minor version upgrades with zero downtime. We handle deprecation warnings, breaking changes, and post-upgrade validation.',
    details: [
      'Pre-upgrade compatibility and deprecation audit',
      'Breaking change remediation across plugins and clients',
      'Rolling upgrade execution with zero downtime',
      'Post-upgrade performance validation',
      'Rollback plan and recovery documentation',
    ],
  },
  {
    id: 'opensearch',
    title: 'OpenSearch',
    tagline: 'open source, enterprise ready',
    description:
      'OpenSearch cluster setup, security configuration, and Dashboards deployment. Self-hosted or AWS OpenSearch Service.',
    details: [
      'OpenSearch cluster setup and node sizing',
      'Security plugin configuration (RBAC, TLS, audit logging)',
      'OpenSearch Dashboards setup and customization',
      'ISM index policy design',
      'AWS OpenSearch Service and self-hosted deployment',
    ],
  },
  {
    id: 'ai-rag-implementation',
    title: 'AI / RAG Implementation',
    tagline: 'elastic as your ai backbone',
    description:
      'Vector search and RAG pipelines powered by Elasticsearch. Dense vectors, hybrid search, and LLM integration for AI applications.',
    details: [
      'Dense vector index design and kNN configuration',
      'Embedding pipeline setup (OpenAI, Cohere, local models)',
      'Hybrid search tuning (BM25 + kNN combined scoring)',
      'Elasticsearch inference API integration',
      'RAG pipeline wiring to LLM APIs',
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
  { src: '/images/logos/951515-249938965.png', name: '951515' },
]

// 4 copies per half = 8 total; mr-24 on each item makes slot width exact so -50% loops seamlessly
const marqueeItems = Array.from({ length: 8 }, () => logos).flat()

function LogoMarquee() {
  return (
    <div className="w-full overflow-hidden border-y border-stone-200 py-12">
      <div
        className="flex items-center"
        style={{
          width: 'max-content',
          animation: 'marquee 70s linear infinite',
        }}
      >
        {marqueeItems.map((logo, i) => (
          <div key={i} className="mr-24 flex h-20 w-56 shrink-0 items-center justify-center">
            <img
              src={logo.src}
              alt={logo.name}
              className="max-h-full max-w-full object-contain opacity-85 grayscale transition-opacity duration-300 hover:opacity-100"
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
    <div className="font-inter min-h-screen bg-stone-50">
      {/* Nav */}
      <header className="px-4 pt-8 pb-0 sm:px-6">
        <div className="mx-auto max-w-2xl">
          <div className="mb-0 flex items-center justify-between">
            <Link
              to="/"
              className="font-logo text-2xl tracking-tight text-stone-900 no-underline hover:no-underline md:text-3xl"
            >
              🦌 <span className="text-green-700">deer.sh</span>
            </Link>
            <div className="hidden items-center gap-6 text-sm text-stone-500 md:flex">
              <a href="#services" className="transition-colors hover:text-stone-800">
                Services
              </a>
              <a href="#case-studies" className="transition-colors hover:text-stone-800">
                Case Studies
              </a>
              <Link to="/product" className="transition-colors hover:text-stone-800">
                Product
              </Link>
              <Link to="/blog" className="transition-colors hover:text-stone-800">
                Blog
              </Link>
              <a
                href="#contact"
                className="inline-flex items-center gap-1 rounded-full border border-green-900/40 bg-green-900/10 px-4 py-1.5 text-green-700 transition-colors hover:border-green-900/60 hover:bg-green-900/20"
              >
                Get in Touch
              </a>
            </div>
            <button
              className="text-stone-500 hover:text-stone-800 md:hidden"
              onClick={() => setMobileOpen(!mobileOpen)}
            >
              {mobileOpen ? <X className="h-5 w-5" /> : <Menu className="h-5 w-5" />}
            </button>
          </div>

          {mobileOpen && (
            <div
              className="fixed inset-0 z-30 bg-stone-900 md:hidden"
              onClick={() => setMobileOpen(false)}
            >
              <nav
                className="flex flex-col gap-6 p-8 pt-20 text-lg text-stone-300"
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
                <Link
                  to="/blog"
                  onClick={() => setMobileOpen(false)}
                  className="transition-colors hover:text-white"
                >
                  Blog
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
            <h1 className="font-logo text-3xl font-bold tracking-tight text-stone-900 md:text-4xl">
              Expert ELK Stack Consulting
            </h1>
            <p className="mt-4 leading-relaxed text-stone-700">
              Cluster setup, pipeline architecture, and system integration - done right. We work
              alongside your team to build ELK infrastructure that handles production scale without
              the months of trial and error.
            </p>
            <div className="mt-6 flex items-center gap-3">
              <a
                href="#contact"
                className="inline-flex items-center gap-2 rounded-full bg-stone-800 px-6 py-2.5 text-sm text-white transition-colors hover:bg-stone-900"
              >
                Contact <ArrowRight className="h-4 w-4" />
              </a>
              <a
                href="#services"
                className="inline-flex items-center gap-2 rounded-full border border-stone-300 px-6 py-2.5 text-sm text-stone-600 transition-colors hover:border-stone-400 hover:text-stone-800"
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
            className="mt-12 grid grid-cols-3 divide-x divide-stone-200 rounded-2xl border border-stone-200"
          >
            {[
              { value: '10+', label: 'Universities' },
              { value: '< 2 weeks', label: 'Avg. time to production' },
              { value: '100%', label: 'Documented handoffs' },
            ].map((stat, i) => (
              <div key={i} className="p-4 text-center">
                <div className="text-xl font-bold text-amber-800">{stat.value}</div>
                <div className="mt-1 text-xs text-stone-400">{stat.label}</div>
              </div>
            ))}
          </motion.div>
        </div>
      </section>

      {/* Services */}
      <section id="services" className="border-t border-stone-200 px-4 py-16 sm:px-6">
        <div className="mx-auto max-w-2xl">
          <FadeIn>
            <h2 className="font-logo text-xl tracking-tight text-stone-900">Services</h2>
            <p className="mt-2 text-sm text-stone-500">
              Eight core engagements. Scoped to your needs, priced by outcome.
            </p>
          </FadeIn>

          <div className="mt-8 space-y-3">
            {services.map((svc) => (
              <FadeIn key={svc.id}>
                <div className="rounded-2xl border border-stone-200 bg-white transition-transform duration-200 hover:-translate-y-0.5">
                  <button
                    onClick={() => setExpandedService(expandedService === svc.id ? null : svc.id)}
                    className="w-full px-5 py-4 text-left"
                  >
                    <div className="flex items-start justify-between gap-4">
                      <div>
                        <div className="text-sm font-semibold text-stone-900">{svc.title}</div>
                        <div className="mt-0.5 text-xs text-green-700 italic">{svc.tagline}</div>
                      </div>
                      <ChevronDown
                        className={`mt-0.5 h-4 w-4 shrink-0 text-stone-400 transition-transform duration-200 ${expandedService === svc.id ? 'rotate-180' : ''}`}
                      />
                    </div>
                    <p className="mt-2 text-sm leading-relaxed text-stone-500">{svc.description}</p>
                  </button>

                  {expandedService === svc.id && (
                    <div className="border-t border-stone-200 px-5 py-4">
                      <ul className="space-y-1.5">
                        {svc.details.map((d) => (
                          <li key={d} className="flex items-start gap-2 text-sm">
                            <CheckCircle className="mt-0.5 h-3.5 w-3.5 shrink-0 text-green-700" />
                            <span className="text-stone-700">{d}</span>
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

      {/* Customers */}
      <section id="case-studies" className="border-t border-stone-200 py-16">
        <div className="mx-auto max-w-2xl px-4 sm:px-6">
          <FadeIn>
            <h2 className="font-logo text-xl tracking-tight text-stone-900">Customers</h2>
            <p className="mt-2 text-sm text-stone-500">
              Our engineers have worked with these organizations.
            </p>
          </FadeIn>
        </div>
        <div className="mt-8">
          <LogoMarquee />
        </div>
      </section>

      {/* How We Work */}
      <section className="border-t border-stone-200 px-4 py-16 sm:px-6">
        <div className="mx-auto max-w-2xl">
          <FadeIn>
            <h2 className="font-logo text-xl tracking-tight text-stone-900">How We Work</h2>
            <p className="mt-2 text-sm text-stone-500">
              A repeatable process that gets to production fast.
            </p>
          </FadeIn>

          <div className="mt-8 space-y-3">
            {process.map((p) => (
              <FadeIn key={p.step}>
                <div className="flex gap-5 rounded-2xl border border-stone-200 bg-white px-5 py-5 transition-transform duration-200 hover:-translate-y-0.5">
                  <div className="mt-1.5 h-1.5 w-1.5 shrink-0 rounded-full bg-green-700"></div>
                  <div>
                    <div className="font-semibold text-stone-900">{p.title}</div>
                    <p className="mt-1 text-sm leading-relaxed text-stone-500">{p.description}</p>
                  </div>
                </div>
              </FadeIn>
            ))}
          </div>
        </div>
      </section>

      {/* Contact / CTA */}
      <section id="contact" className="border-t border-stone-200 px-4 py-20 sm:px-6">
        <div className="mx-auto max-w-2xl">
          <FadeIn>
            <h2 className="font-logo text-xl tracking-tight text-stone-900">Get in Touch</h2>
            <p className="mt-2 leading-relaxed text-stone-700">
              Ready to modernize your logging infrastructure? Tell us what you're working with and
              what's not working. We'll respond within one business day.
            </p>
            <div className="mt-6 flex flex-col gap-3 sm:flex-row sm:items-center">
              <a
                href="mailto:cpfeifer@madcactus.org"
                className="inline-flex items-center gap-2 rounded-full bg-stone-800 px-7 py-3 text-sm text-white transition-colors hover:bg-stone-900"
              >
                Contact <ArrowRight className="h-4 w-4" />
              </a>
              <a
                href="#contact"
                className="text-sm text-stone-400 underline underline-offset-2 transition-colors hover:text-stone-600"
              >
                or book a 30-min discovery call
              </a>
            </div>

            <div className="mt-12 rounded-2xl border border-stone-200 bg-white p-5">
              <div className="mb-3 text-xs font-medium tracking-wide text-stone-400 uppercase">
                what to include
              </div>
              <ul className="space-y-1.5">
                {[
                  'Current ELK version and deployment method',
                  'Data volume (events/sec or GB/day)',
                  'What is broken or missing today',
                  'Timeline and any compliance constraints',
                ].map((item) => (
                  <li key={item} className="flex items-start gap-2 text-sm">
                    <span className="mt-0.5 text-green-700">-</span>
                    <span className="text-stone-700">{item}</span>
                  </li>
                ))}
              </ul>
            </div>
          </FadeIn>
        </div>
      </section>
    </div>
  )
}
