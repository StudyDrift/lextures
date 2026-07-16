import { useEffect, useRef, useState } from 'react'

interface TrustISOStatus {
  scopeStatement: string
  iso27001Status: string
  iso27701Status: string
  iso27001CertUrl?: string
  iso27001LastAudit?: string
  soaLastReview?: string
  soa?: { total: number; implemented: number; planned: number; excluded: number }
}

function isoStatusLabel(status: string): string {
  switch (status) {
    case 'certified': return 'Certified'
    case 'stage1': return 'Stage 1 complete'
    case 'in_progress': return 'In Progress'
    case 'not_started': return 'Not started'
    default: return status
  }
}
import { Link } from 'react-router-dom'
import { BrandLogo } from '../components/brand-logo'
import { MARKETING_LEGAL_URLS } from '../lib/marketing-site'
import {
  AI_SUBPROCESSOR_BYOK_NOTE,
  SUB_PROCESSORS,
  SUB_PROCESSORS_EFFECTIVE_DATE,
  type DpaStatus,
} from '../content/trust/sub-processors'
import { INCIDENTS, type IncidentSeverity, type IncidentStatus } from '../content/trust/incidents'

// ── Helpers ─────────────────────────────────────────────────────────────────

function dpaLabel(status: DpaStatus): string {
  switch (status) {
    case 'signed': return 'Signed'
    case 'in-review': return 'In Review'
    case 'not-applicable': return 'N/A'
  }
}

function dpaClass(status: DpaStatus): string {
  switch (status) {
    case 'signed': return 'bg-emerald-50 text-emerald-700 dark:bg-emerald-950 dark:text-emerald-300'
    case 'in-review': return 'bg-amber-50 text-amber-700 dark:bg-amber-950 dark:text-amber-300'
    case 'not-applicable': return 'bg-slate-100 text-slate-600 dark:bg-neutral-800 dark:text-neutral-400'
  }
}

function severityClass(s: IncidentSeverity): string {
  switch (s) {
    case 'critical': return 'bg-red-100 text-red-700 dark:bg-red-950 dark:text-red-300'
    case 'high': return 'bg-orange-100 text-orange-700 dark:bg-orange-950 dark:text-orange-300'
    case 'medium': return 'bg-amber-100 text-amber-700 dark:bg-amber-950 dark:text-amber-300'
    case 'low': return 'bg-blue-50 text-blue-700 dark:bg-blue-950 dark:text-blue-300'
  }
}

function statusClass(s: IncidentStatus): string {
  return s === 'resolved'
    ? 'bg-emerald-50 text-emerald-700 dark:bg-emerald-950 dark:text-emerald-300'
    : 'bg-amber-50 text-amber-700 dark:bg-amber-950 dark:text-amber-300'
}

function capitalize(s: string): string {
  return s.charAt(0).toUpperCase() + s.slice(1)
}

// ── Accordion section ────────────────────────────────────────────────────────

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  const [open, setOpen] = useState(true)
  const id = `section-${title.toLowerCase().replace(/\s+/g, '-')}`
  return (
    <div className="border border-slate-200 dark:border-neutral-800 rounded-lg overflow-hidden">
      <button
        type="button"
        aria-expanded={open}
        aria-controls={id}
        onClick={() => setOpen((v) => !v)}
        className="w-full flex items-center justify-between gap-3 px-5 py-4 text-start bg-white dark:bg-neutral-900 hover:bg-slate-50 dark:hover:bg-neutral-800 transition-colors"
      >
        <span className="text-base font-semibold text-slate-900 dark:text-neutral-50">{title}</span>
        <svg
          aria-hidden="true"
          className={`h-5 w-5 shrink-0 text-slate-400 transition-transform duration-200 ${open ? 'rotate-180' : ''}`}
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
          strokeWidth={2}
        >
          <path strokeLinecap="round" strokeLinejoin="round" d="M19 9l-7 7-7-7" />
        </svg>
      </button>
      {open ? (
        <div id={id} className="px-5 pb-5 pt-2 bg-white dark:bg-neutral-900">
          {children}
        </div>
      ) : null}
    </div>
  )
}

// ── Subscribe form ───────────────────────────────────────────────────────────

function SubscribeForm() {
  const [email, setEmail] = useState('')
  const [status, setStatus] = useState<'idle' | 'loading' | 'success' | 'error'>('idle')
  const inputRef = useRef<HTMLInputElement>(null)

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setStatus('loading')
    try {
      const res = await fetch('/api/v1/trust/sub-processor-updates/subscribe', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email }),
      })
      if (res.ok || res.status === 204) {
        setStatus('success')
        setEmail('')
      } else {
        setStatus('error')
      }
    } catch {
      setStatus('error')
    }
  }

  if (status === 'success') {
    return (
      <p role="status" className="text-sm text-emerald-700 dark:text-emerald-300">
        You're subscribed! You'll receive an email when the sub-processor list changes.
      </p>
    )
  }

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-3 sm:flex-row sm:items-end">
      <div className="flex-1">
        <label htmlFor="sub-email" className="block text-sm font-medium text-slate-700 dark:text-neutral-300 mb-1">
          Email address
        </label>
        <input
          id="sub-email"
          ref={inputRef}
          type="email"
          required
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          placeholder="you@example.com"
          className="w-full rounded-md border border-slate-300 dark:border-neutral-700 bg-white dark:bg-neutral-900 px-3 py-2 text-sm text-slate-900 dark:text-neutral-100 placeholder:text-slate-400 focus:outline-none focus:ring-2 focus:ring-indigo-500"
        />
      </div>
      <button
        type="submit"
        disabled={status === 'loading'}
        className="rounded-md bg-indigo-600 px-4 py-2 text-sm font-semibold text-white hover:bg-indigo-700 disabled:opacity-60 transition-colors"
      >
        {status === 'loading' ? 'Subscribing…' : 'Subscribe'}
      </button>
      {status === 'error' ? (
        <p role="alert" className="text-sm text-red-600 dark:text-red-400">
          Something went wrong. Please try again.
        </p>
      ) : null}
    </form>
  )
}

// ── Main page ────────────────────────────────────────────────────────────────

export default function TrustCenterPage() {
  const [isoStatus, setIsoStatus] = useState<TrustISOStatus | null>(null)

  useEffect(() => {
    document.title = 'Trust Center — Lextures'
    void fetch('/api/v1/trust/iso')
      .then((res) => (res.ok ? res.json() : null))
      .then((data) => {
        if (data && typeof data === 'object') {
          setIsoStatus(data as TrustISOStatus)
        }
      })
      .catch(() => {
        /* static fallback in certifications table */
      })
  }, [])

  return (
    <div className="min-h-dvh bg-slate-50 text-slate-900 dark:bg-neutral-950 dark:text-neutral-100">
      {/* Header */}
      <header className="border-b border-slate-200 bg-white px-4 py-4 dark:border-neutral-800 dark:bg-neutral-900 sm:px-6 print:hidden">
        <div className="mx-auto flex max-w-5xl flex-wrap items-center justify-between gap-3">
          <Link to="/login" className="inline-flex items-center gap-2 text-sm font-medium text-slate-700 dark:text-neutral-200">
            <BrandLogo className="h-7 w-auto" />
            <span className="sr-only">Lextures home</span>
          </Link>
          <nav aria-label="Legal" className="flex flex-wrap gap-3 text-sm">
            <a href={MARKETING_LEGAL_URLS.privacy} className="text-indigo-700 underline-offset-2 hover:underline dark:text-indigo-300">Privacy</a>
            <a href={MARKETING_LEGAL_URLS.terms} className="text-indigo-700 underline-offset-2 hover:underline dark:text-indigo-300">Terms</a>
            <Link to="/login" className="text-slate-600 hover:text-slate-900 dark:text-neutral-400 dark:hover:text-neutral-100">Sign in</Link>
          </nav>
        </div>
      </header>

      <main className="mx-auto max-w-5xl px-4 py-8 sm:px-6 lg:py-12">
        {/* Hero */}
        <div className="mb-10">
          <h1 className="text-3xl font-semibold tracking-tight text-slate-900 dark:text-neutral-50">Trust Center</h1>
          <p className="mt-2 text-base text-slate-600 dark:text-neutral-400">
            Security overview, sub-processor list, certifications, and incident history for Lextures.
          </p>
          <p className="mt-1 text-sm text-slate-500 dark:text-neutral-500">
            Questions? Email{' '}
            <a href="mailto:security@lextures.com" className="text-indigo-700 underline dark:text-indigo-300">
              security@lextures.com
            </a>
          </p>
        </div>

        <div className="flex flex-col gap-6">
          {/* Security Overview */}
          <Section title="Security Overview">
            <div className="prose-sm max-w-none text-slate-700 dark:text-neutral-300 space-y-4">
              <div>
                <h3 className="font-semibold text-slate-900 dark:text-neutral-100 mb-1">Infrastructure</h3>
                <p>
                  Lextures runs on Amazon Web Services (AWS) in the <strong>us-east-1</strong> region. All services are
                  deployed in a private VPC with public subnets limited to load balancers only. Application workloads run
                  on Amazon EKS (Kubernetes) with node auto-scaling.
                </p>
              </div>
              <div>
                <h3 className="font-semibold text-slate-900 dark:text-neutral-100 mb-1">Network Security</h3>
                <p>
                  All traffic is routed through Cloudflare for DDoS mitigation and WAF protection. Internal service
                  communication uses mTLS. Security groups restrict inbound access to the minimum required ports.
                </p>
              </div>
              <div>
                <h3 className="font-semibold text-slate-900 dark:text-neutral-100 mb-1">Encryption</h3>
                <p>
                  Data in transit is encrypted with TLS 1.2+. Data at rest (databases, object storage) is encrypted
                  using AES-256. Database encryption keys are managed via AWS KMS with automatic rotation.
                </p>
              </div>
              <div>
                <h3 className="font-semibold text-slate-900 dark:text-neutral-100 mb-1">Access Controls</h3>
                <p>
                  All engineer access to production systems requires MFA. Access follows the principle of least
                  privilege; production database access requires a time-limited approval workflow. Administrative
                  actions are logged in an immutable audit trail.
                </p>
              </div>
              <div>
                <h3 className="font-semibold text-slate-900 dark:text-neutral-100 mb-1">Vulnerability Management</h3>
                <p>
                  Dependency vulnerabilities are monitored via automated scanning on every CI build. Container images
                  are scanned for CVEs before deployment. Critical vulnerabilities are remediated within 24 hours.
                </p>
              </div>
              <div>
                <h3 className="font-semibold text-slate-900 dark:text-neutral-100 mb-1">Security Testing</h3>
                <p>
                  We conduct internal penetration testing on a quarterly basis. External penetration tests are
                  commissioned annually. Results inform our remediation roadmap.
                </p>
              </div>
              <div>
                <h3 className="font-semibold text-slate-900 dark:text-neutral-100 mb-1">Incident Response</h3>
                <p>
                  Lextures maintains a documented incident response plan. On detection, incidents are triaged within
                  1 hour, root cause identified within 24 hours, and affected customers notified within 72 hours in
                  accordance with GDPR Art. 33 obligations.
                </p>
              </div>
            </div>
          </Section>

          {/* Certifications & Compliance */}
          <Section title="Certifications &amp; Compliance">
            <div className="overflow-x-auto">
              <table className="min-w-full text-sm border-collapse" aria-label="Certification status">
                <caption className="sr-only">Current certification and compliance status</caption>
                <thead>
                  <tr className="border-b border-slate-200 dark:border-neutral-700">
                    <th scope="col" className="text-start py-2 pe-4 font-semibold text-slate-700 dark:text-neutral-300">Framework</th>
                    <th scope="col" className="text-start py-2 pe-4 font-semibold text-slate-700 dark:text-neutral-300">Status</th>
                    <th scope="col" className="text-start py-2 font-semibold text-slate-700 dark:text-neutral-300">Notes</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-100 dark:divide-neutral-800">
                  {[
                    { name: 'SOC 2 Type II', status: 'In Progress', notes: 'Evidence collection underway; report expected Q4 2026.' },
                    {
                      name: 'ISO 27001',
                      status: isoStatus ? isoStatusLabel(isoStatus.iso27001Status) : 'In Progress',
                      notes: isoStatus?.iso27001LastAudit
                        ? `Last audit: ${isoStatus.iso27001LastAudit}. SoA: ${isoStatus.soa?.implemented ?? 0}/${isoStatus.soa?.total ?? 93} controls implemented.`
                        : 'Gap assessment complete; certification target Q1 2027.',
                    },
                    {
                      name: 'ISO 27701 (PIMS)',
                      status: isoStatus ? isoStatusLabel(isoStatus.iso27701Status) : 'In Progress',
                      notes: 'Privacy extension mapped to GDPR Art. 25 (data protection by design).',
                    },
                    { name: 'FERPA', status: 'Aligned', notes: 'Data handling follows FERPA requirements. School agreements available on request.' },
                    { name: 'COPPA', status: 'Aligned', notes: 'Student data under 13 requires institutional agreement.' },
                    { name: 'GDPR', status: 'Aligned', notes: 'DPO designated. Data Processing Addendum (DPA) available.' },
                    { name: 'WCAG 2.1 AA', status: 'In Progress', notes: 'Key workflows pass axe-core; full audit planned 2026.' },
                  ].map((row) => (
                    <tr key={row.name}>
                      <td className="py-2 pe-4 font-medium text-slate-900 dark:text-neutral-100">{row.name}</td>
                      <td className="py-2 pe-4">
                        <span className={`inline-block rounded px-2 py-0.5 text-xs font-medium ${
                          row.status === 'Aligned'
                            ? 'bg-emerald-50 text-emerald-700 dark:bg-emerald-950 dark:text-emerald-300'
                            : 'bg-amber-50 text-amber-700 dark:bg-amber-950 dark:text-amber-300'
                        }`}>
                          {row.status}
                        </span>
                      </td>
                      <td className="py-2 text-slate-600 dark:text-neutral-400">{row.notes}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            {isoStatus?.scopeStatement ? (
              <p className="mt-4 text-sm text-slate-600 dark:text-neutral-400">
                <strong className="text-slate-800 dark:text-neutral-200">ISMS scope:</strong>{' '}
                {isoStatus.scopeStatement}
              </p>
            ) : null}
            {isoStatus?.iso27001CertUrl ? (
              <p className="mt-2 text-sm">
                <a
                  href={isoStatus.iso27001CertUrl}
                  className="text-indigo-700 underline dark:text-indigo-300"
                  rel="noopener noreferrer"
                >
                  ISO 27001 certificate
                </a>
              </p>
            ) : null}
            <p className="mt-4 text-sm text-slate-500 dark:text-neutral-500">
              To request a copy of our SOC 2 report (NDA required), email{' '}
              <a href="mailto:security@lextures.com" className="text-indigo-700 underline dark:text-indigo-300">
                security@lextures.com
              </a>
              .
            </p>
          </Section>

          {/* Sub-Processors */}
          <Section title="Sub-Processors">
            <div className="mb-3 flex flex-wrap items-baseline gap-x-4 gap-y-1 text-sm">
              <p className="text-slate-600 dark:text-neutral-400">
                Effective date: <span className="font-medium">{SUB_PROCESSORS_EFFECTIVE_DATE}</span>
              </p>
              <p className="text-slate-500 dark:text-neutral-500">
                In accordance with GDPR Art. 28 and FERPA, we disclose all third-party vendors who process customer data.
              </p>
            </div>
            <p className="mb-4 text-sm text-slate-600 dark:text-neutral-400" data-testid="ai-byok-subprocessor-note">
              {AI_SUBPROCESSOR_BYOK_NOTE}
            </p>
            <div className="overflow-x-auto">
              <table className="min-w-full text-sm border-collapse" aria-label="Sub-processor list">
                <caption className="sr-only">Complete list of Lextures sub-processors as of {SUB_PROCESSORS_EFFECTIVE_DATE}</caption>
                <thead>
                  <tr className="border-b border-slate-200 dark:border-neutral-700">
                    <th scope="col" className="text-start py-2 pe-3 font-semibold text-slate-700 dark:text-neutral-300 whitespace-nowrap">Vendor</th>
                    <th scope="col" className="text-start py-2 pe-3 font-semibold text-slate-700 dark:text-neutral-300">Purpose</th>
                    <th scope="col" className="text-start py-2 pe-3 font-semibold text-slate-700 dark:text-neutral-300">Data Categories</th>
                    <th scope="col" className="text-start py-2 pe-3 font-semibold text-slate-700 dark:text-neutral-300 whitespace-nowrap">HQ</th>
                    <th scope="col" className="text-start py-2 pe-3 font-semibold text-slate-700 dark:text-neutral-300 whitespace-nowrap">Data Region</th>
                    <th scope="col" className="text-start py-2 font-semibold text-slate-700 dark:text-neutral-300 whitespace-nowrap">DPA Status</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-100 dark:divide-neutral-800">
                  {SUB_PROCESSORS.map((sp) => (
                    <tr key={sp.name}>
                      <td className="py-2 pe-3 font-medium text-slate-900 dark:text-neutral-100 whitespace-nowrap">
                        <a href={sp.privacyUrl} target="_blank" rel="noreferrer" className="text-indigo-700 underline-offset-2 hover:underline dark:text-indigo-300">
                          {sp.name}
                        </a>
                      </td>
                      <td className="py-2 pe-3 text-slate-700 dark:text-neutral-300">{sp.service}</td>
                      <td className="py-2 pe-3 text-slate-600 dark:text-neutral-400">
                        {sp.dataCategories.join(', ')}
                      </td>
                      <td className="py-2 pe-3 text-slate-600 dark:text-neutral-400">{sp.headquarters}</td>
                      <td className="py-2 pe-3 text-slate-600 dark:text-neutral-400">{sp.dataRegion}</td>
                      <td className="py-2">
                        <span className={`inline-block rounded px-2 py-0.5 text-xs font-medium ${dpaClass(sp.dpaStatus)}`}>
                          {dpaLabel(sp.dpaStatus)}
                        </span>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            <div className="mt-6 border-t border-slate-100 dark:border-neutral-800 pt-4">
              <p className="text-sm font-medium text-slate-800 dark:text-neutral-200 mb-2">
                Get notified when this list changes
              </p>
              <SubscribeForm />
            </div>
          </Section>

          {/* Incident History */}
          <Section title="Incident History">
            {INCIDENTS.length === 0 ? (
              <p className="text-sm text-slate-500 dark:text-neutral-500">No incidents on record.</p>
            ) : (
              <div className="overflow-x-auto">
                <table className="min-w-full text-sm border-collapse" aria-label="Incident history">
                  <caption className="sr-only">History of security and availability incidents</caption>
                  <thead>
                    <tr className="border-b border-slate-200 dark:border-neutral-700">
                      <th scope="col" className="text-start py-2 pe-3 font-semibold text-slate-700 dark:text-neutral-300 whitespace-nowrap">Date</th>
                      <th scope="col" className="text-start py-2 pe-3 font-semibold text-slate-700 dark:text-neutral-300">Severity</th>
                      <th scope="col" className="text-start py-2 pe-3 font-semibold text-slate-700 dark:text-neutral-300">Summary</th>
                      <th scope="col" className="text-start py-2 pe-3 font-semibold text-slate-700 dark:text-neutral-300">Impact</th>
                      <th scope="col" className="text-start py-2 pe-3 font-semibold text-slate-700 dark:text-neutral-300 whitespace-nowrap">Resolved</th>
                      <th scope="col" className="text-start py-2 font-semibold text-slate-700 dark:text-neutral-300">Status</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-100 dark:divide-neutral-800">
                    {INCIDENTS.map((inc, i) => (
                      <tr key={i}>
                        <td className="py-2 pe-3 text-slate-700 dark:text-neutral-300 whitespace-nowrap">{inc.date}</td>
                        <td className="py-2 pe-3">
                          <span className={`inline-block rounded px-2 py-0.5 text-xs font-medium ${severityClass(inc.severity)}`}>
                            {capitalize(inc.severity)}
                          </span>
                        </td>
                        <td className="py-2 pe-3 text-slate-700 dark:text-neutral-300">{inc.summary}</td>
                        <td className="py-2 pe-3 text-slate-600 dark:text-neutral-400">{inc.impact}</td>
                        <td className="py-2 pe-3 text-slate-600 dark:text-neutral-400 whitespace-nowrap">
                          {inc.resolvedDate ?? '—'}
                        </td>
                        <td className="py-2">
                          <span className={`inline-block rounded px-2 py-0.5 text-xs font-medium ${statusClass(inc.status)}`}>
                            {capitalize(inc.status)}
                          </span>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </Section>

          {/* Contact & Disclosure */}
          <Section title="Contact &amp; Responsible Disclosure">
            <div className="space-y-4 text-sm text-slate-700 dark:text-neutral-300">
              <div>
                <p className="font-semibold text-slate-900 dark:text-neutral-100 mb-1">Security inquiries</p>
                <p>
                  For security questions, data processing inquiries, or to request a DPA, email{' '}
                  <a href="mailto:security@lextures.com" className="text-indigo-700 underline dark:text-indigo-300">
                    security@lextures.com
                  </a>
                  .
                </p>
              </div>
              <div>
                <p className="font-semibold text-slate-900 dark:text-neutral-100 mb-1">Responsible disclosure</p>
                <p>
                  We welcome vulnerability reports from security researchers. Please review our{' '}
                  <a href={MARKETING_LEGAL_URLS.security} className="text-indigo-700 underline dark:text-indigo-300">
                    responsible disclosure policy
                  </a>{' '}
                  before reporting. We commit to a 90-day disclosure timeline and will acknowledge valid reports
                  within 5 business days.
                </p>
              </div>
              <div>
                <p className="font-semibold text-slate-900 dark:text-neutral-100 mb-1">SOC 2 report</p>
                <p>
                  Our SOC 2 Type II report is available under NDA. To request a copy, email{' '}
                  <a href="mailto:security@lextures.com" className="text-indigo-700 underline dark:text-indigo-300">
                    security@lextures.com
                  </a>{' '}
                  with your organization name and intended use.
                </p>
              </div>
              <div>
                <p className="font-semibold text-slate-900 dark:text-neutral-100 mb-1">Privacy inquiries</p>
                <p>
                  For privacy-related requests (GDPR, FERPA, data subject rights), email{' '}
                  <a href="mailto:privacy@lextures.com" className="text-indigo-700 underline dark:text-indigo-300">
                    privacy@lextures.com
                  </a>
                  . See our{' '}
                  <a href={MARKETING_LEGAL_URLS.privacy} className="text-indigo-700 underline dark:text-indigo-300">
                    Privacy Policy
                  </a>{' '}
                  for full details.
                </p>
              </div>
            </div>
          </Section>
        </div>
      </main>

      <footer className="mt-12 border-t border-slate-200 dark:border-neutral-800 py-6 px-4 text-center text-xs text-slate-500 dark:text-neutral-500 print:hidden">
        <p>
          &copy; {new Date().getFullYear()} Lextures, Inc. &middot;{' '}
          <a href={MARKETING_LEGAL_URLS.privacy} className="underline-offset-2 hover:underline">Privacy Policy</a>
          {' '}&middot;{' '}
          <a href={MARKETING_LEGAL_URLS.terms} className="underline-offset-2 hover:underline">Terms of Service</a>
        </p>
      </footer>
    </div>
  )
}
