import { useEffect } from 'react'
import { Link } from 'react-router-dom'
import { BrandLogo } from '../components/brand-logo'
import {
  type ConformanceLevel,
  EN301549_SOFTWARE_CRITERIA,
  EN301549_SUPPORT_CRITERIA,
  FPC_CRITERIA,
  SEC508_SOFTWARE_CRITERIA,
  SEC508_SUPPORT_CRITERIA,
  WCAG_CRITERIA,
} from '../lib/vpat-data'

const EVAL_DATE = 'May 27, 2026'
const PRODUCT_VERSION = '1.0'

function conformanceBadgeClass(level: ConformanceLevel): string {
  switch (level) {
    case 'Supports':
      return 'bg-emerald-50 text-emerald-700 dark:bg-emerald-950 dark:text-emerald-300'
    case 'Partially Supports':
      return 'bg-amber-50 text-amber-700 dark:bg-amber-950 dark:text-amber-300'
    case 'Does Not Support':
      return 'bg-red-50 text-red-700 dark:bg-red-950 dark:text-red-300'
    case 'Not Applicable':
      return 'bg-slate-100 text-slate-600 dark:bg-neutral-800 dark:text-neutral-400'
  }
}

function ConformanceBadge({ level }: { level: ConformanceLevel }) {
  return (
    <span className={`inline-block rounded px-2 py-0.5 text-xs font-medium ${conformanceBadgeClass(level)}`}>
      {level}
    </span>
  )
}

function WcagTable({ criteria }: { criteria: typeof WCAG_CRITERIA }) {
  return (
    <div className="overflow-x-auto rounded-lg border border-slate-200 dark:border-neutral-800">
      <table className="min-w-full text-sm border-collapse" aria-label="WCAG success criteria conformance">
        <caption className="sr-only">WCAG 2.1 success criteria with conformance level and notes</caption>
        <thead className="bg-slate-50 dark:bg-neutral-900">
          <tr className="border-b border-slate-200 dark:border-neutral-700">
            <th scope="col" className="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wide text-slate-600 dark:text-neutral-400 whitespace-nowrap">SC</th>
            <th scope="col" className="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wide text-slate-600 dark:text-neutral-400">Title</th>
            <th scope="col" className="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wide text-slate-600 dark:text-neutral-400 whitespace-nowrap">Conformance</th>
            <th scope="col" className="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wide text-slate-600 dark:text-neutral-400">Remarks</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-slate-100 bg-white dark:divide-neutral-800 dark:bg-neutral-900">
          {criteria.map((c) => (
            <tr key={c.sc} className="hover:bg-slate-50 dark:hover:bg-neutral-800/50">
              <td className="px-4 py-3 font-mono text-xs font-medium text-slate-700 dark:text-neutral-300 whitespace-nowrap">{c.sc}</td>
              <td className="px-4 py-3 text-slate-800 dark:text-neutral-200">{c.title}</td>
              <td className="px-4 py-3 whitespace-nowrap"><ConformanceBadge level={c.conformance} /></td>
              <td className="px-4 py-3 text-slate-600 dark:text-neutral-400">{c.notes}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

function GenericTable<T extends { conformance: ConformanceLevel; notes: string }>({
  rows,
  idKey,
  idLabel,
  titleKey,
  label,
}: {
  rows: T[]
  idKey: keyof T
  idLabel: string
  titleKey: keyof T
  label: string
}) {
  return (
    <div className="overflow-x-auto rounded-lg border border-slate-200 dark:border-neutral-800">
      <table className="min-w-full text-sm border-collapse" aria-label={label}>
        <caption className="sr-only">{label}</caption>
        <thead className="bg-slate-50 dark:bg-neutral-900">
          <tr className="border-b border-slate-200 dark:border-neutral-700">
            <th scope="col" className="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wide text-slate-600 dark:text-neutral-400 whitespace-nowrap">{idLabel}</th>
            <th scope="col" className="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wide text-slate-600 dark:text-neutral-400">Criterion</th>
            <th scope="col" className="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wide text-slate-600 dark:text-neutral-400 whitespace-nowrap">Conformance</th>
            <th scope="col" className="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wide text-slate-600 dark:text-neutral-400">Remarks</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-slate-100 bg-white dark:divide-neutral-800 dark:bg-neutral-900">
          {rows.map((r) => (
            <tr key={String(r[idKey])} className="hover:bg-slate-50 dark:hover:bg-neutral-800/50">
              <td className="px-4 py-3 font-mono text-xs font-medium text-slate-700 dark:text-neutral-300 whitespace-nowrap">{String(r[idKey])}</td>
              <td className="px-4 py-3 text-slate-800 dark:text-neutral-200">{String(r[titleKey])}</td>
              <td className="px-4 py-3 whitespace-nowrap"><ConformanceBadge level={r.conformance} /></td>
              <td className="px-4 py-3 text-slate-600 dark:text-neutral-400">{r.notes}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

export default function VpatPage() {
  useEffect(() => {
    document.title = 'VPAT — Accessibility Conformance Report — Lextures'
  }, [])

  const levelA = WCAG_CRITERIA.filter((c) => c.level === 'A')
  const levelAA = WCAG_CRITERIA.filter((c) => c.level === 'AA')

  return (
    <div className="min-h-dvh bg-slate-50 text-slate-900 dark:bg-neutral-950 dark:text-neutral-100">
      <a
        href="#main-content"
        className="sr-only focus:not-sr-only focus:absolute focus:left-4 focus:top-4 focus:z-50 focus:rounded focus:bg-white focus:px-4 focus:py-2 focus:text-sm focus:font-medium focus:shadow-lg focus:ring-2 focus:ring-indigo-500 dark:focus:bg-neutral-900"
      >
        Skip to main content
      </a>

      <header className="border-b border-slate-200 bg-white px-4 py-4 dark:border-neutral-800 dark:bg-neutral-900 sm:px-6 print:hidden">
        <div className="mx-auto flex max-w-5xl flex-wrap items-center justify-between gap-3">
          <Link to="/login" className="inline-flex items-center gap-2 text-sm font-medium text-slate-700 dark:text-neutral-200">
            <BrandLogo className="h-7 w-auto" />
            <span className="sr-only">Lextures home</span>
          </Link>
          <nav aria-label="Legal" className="flex flex-wrap gap-3 text-sm">
            <Link to="/accessibility" className="text-indigo-700 underline-offset-2 hover:underline dark:text-indigo-300">Conformance Statement</Link>
            <Link to="/privacy" className="text-indigo-700 underline-offset-2 hover:underline dark:text-indigo-300">Privacy</Link>
            <Link to="/terms" className="text-indigo-700 underline-offset-2 hover:underline dark:text-indigo-300">Terms</Link>
            <Link to="/trust" className="text-indigo-700 underline-offset-2 hover:underline dark:text-indigo-300">Trust</Link>
            <Link to="/login" className="text-slate-600 hover:text-slate-900 dark:text-neutral-400 dark:hover:text-neutral-100">Sign in</Link>
          </nav>
        </div>
      </header>

      <main id="main-content" className="mx-auto max-w-5xl px-4 py-8 sm:px-6 lg:py-12">

        {/* Title and product info */}
        <div className="mb-8">
          <p className="text-sm font-medium uppercase tracking-wide text-indigo-600 dark:text-indigo-400">
            VPAT® 2.5 INT (International Edition)
          </p>
          <h1 className="mt-1 text-3xl font-semibold tracking-tight text-slate-900 dark:text-neutral-50">
            Accessibility Conformance Report
          </h1>
          <p className="mt-2 text-base text-slate-600 dark:text-neutral-400">
            Based on VPAT® Version 2.5 as published by the Information Technology Industry Council (ITI).
            Covers WCAG 2.1, Revised Section 508, and EN 301 549 v3.2.1.
          </p>
        </div>

        {/* Product information */}
        <section aria-labelledby="product-info-heading" className="mb-8">
          <h2 id="product-info-heading" className="mb-3 text-xl font-semibold text-slate-900 dark:text-neutral-50">
            Product Information
          </h2>
          <div className="rounded-lg border border-slate-200 bg-white p-5 dark:border-neutral-800 dark:bg-neutral-900">
            <dl className="grid grid-cols-1 gap-x-6 gap-y-4 sm:grid-cols-2">
              {[
                { label: 'Product Name', value: 'Lextures' },
                { label: 'Product Version', value: PRODUCT_VERSION },
                { label: 'Date of Evaluation', value: EVAL_DATE },
                { label: 'Report Version', value: '1.0' },
                { label: 'Product Description', value: 'Cloud-based learning management system (LMS) for K-12 and higher education. Delivered as a single-page web application (SPA).' },
                { label: 'Contact / Accessibility Support', value: 'accessibility@lextures.com' },
                { label: 'Notes', value: 'This report covers the web application at app.lextures.com. Mobile-specific testing is planned for a future VPAT revision.' },
                { label: 'Evaluation Methods', value: 'Automated axe-core scan on every pull request; manual screen-reader testing with VoiceOver (macOS) and NVDA (Windows); keyboard-only navigation walkthrough.' },
              ].map(({ label, value }) => (
                <div key={label} className={label === 'Product Description' || label === 'Evaluation Methods' || label === 'Notes' ? 'sm:col-span-2' : ''}>
                  <dt className="text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-neutral-400">{label}</dt>
                  <dd className="mt-1 text-sm text-slate-900 dark:text-neutral-100">
                    {label === 'Contact / Accessibility Support' ? (
                      <a href="mailto:accessibility@lextures.com" className="text-indigo-700 underline dark:text-indigo-300">
                        {value}
                      </a>
                    ) : value}
                  </dd>
                </div>
              ))}
            </dl>
          </div>
        </section>

        {/* Download links */}
        <section aria-labelledby="download-heading" className="mb-8">
          <h2 id="download-heading" className="mb-3 text-xl font-semibold text-slate-900 dark:text-neutral-50">
            Download This Report
          </h2>
          <div className="flex flex-wrap gap-3">
            <a
              href="/vpat/VPAT_2.5_INT_Lextures_2026-05.md"
              download
              className="inline-flex items-center gap-2 rounded-lg border border-slate-300 bg-white px-4 py-2 text-sm font-medium text-slate-700 transition hover:bg-slate-50 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-200 dark:hover:bg-neutral-800"
              aria-label="Download VPAT source document (Markdown)"
            >
              <svg aria-hidden="true" className="h-4 w-4" fill="none" stroke="currentColor" strokeWidth="1.5" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" d="M3 16.5v2.25A2.25 2.25 0 0 0 5.25 21h13.5A2.25 2.25 0 0 0 21 18.75V16.5M16.5 12 12 16.5m0 0L7.5 12m4.5 4.5V3" />
              </svg>
              Download VPAT (Markdown source)
            </a>
          </div>
          <p className="mt-2 text-xs text-slate-500 dark:text-neutral-500">
            DOCX and PDF versions are available on request at{' '}
            <a href="mailto:accessibility@lextures.com" className="text-indigo-700 underline dark:text-indigo-300">accessibility@lextures.com</a>.
          </p>
        </section>

        {/* Table of contents */}
        <nav aria-label="Report sections" className="mb-10 rounded-lg border border-slate-200 bg-white p-5 dark:border-neutral-800 dark:bg-neutral-900">
          <h2 className="mb-3 text-base font-semibold text-slate-900 dark:text-neutral-50">Contents</h2>
          <ol className="list-decimal pl-5 text-sm text-indigo-700 space-y-1 dark:text-indigo-300">
            <li><a href="#wcag-level-a" className="underline-offset-2 hover:underline">WCAG 2.1 — Level A Success Criteria</a></li>
            <li><a href="#wcag-level-aa" className="underline-offset-2 hover:underline">WCAG 2.1 — Level AA Success Criteria</a></li>
            <li><a href="#sec508-fpc" className="underline-offset-2 hover:underline">Revised Section 508 — Chapter 3: Functional Performance Criteria</a></li>
            <li><a href="#sec508-software" className="underline-offset-2 hover:underline">Revised Section 508 — Chapter 5: Software</a></li>
            <li><a href="#sec508-support" className="underline-offset-2 hover:underline">Revised Section 508 — Chapter 6: Support Documentation and Services</a></li>
            <li><a href="#en301549-web" className="underline-offset-2 hover:underline">EN 301 549 — Chapter 9: Web</a></li>
            <li><a href="#en301549-software" className="underline-offset-2 hover:underline">EN 301 549 — Chapter 11: Software</a></li>
            <li><a href="#en301549-support" className="underline-offset-2 hover:underline">EN 301 549 — Chapter 12: Documentation and Support</a></li>
            <li><a href="#legal-disclaimer" className="underline-offset-2 hover:underline">Legal Disclaimer</a></li>
            <li><a href="#accessibility-support" className="underline-offset-2 hover:underline">Contact Accessibility Support</a></li>
          </ol>
        </nav>

        {/* Status note */}
        <div className="mb-8 rounded-lg border border-amber-200 bg-amber-50 p-4 text-sm text-amber-800 dark:border-amber-900/60 dark:bg-amber-950/40 dark:text-amber-300">
          <strong>Note:</strong> Entries marked "Partially Supports" represent known gaps actively being remediated. Each entry includes a remarks field with the specific gap and remediation timeline. This report was last evaluated on {EVAL_DATE} and is updated with each major release.
        </div>

        {/* WCAG 2.1 Level A */}
        <section aria-labelledby="wcag-level-a" className="mb-10">
          <h2 id="wcag-level-a" className="mb-1 text-xl font-semibold text-slate-900 dark:text-neutral-50">
            WCAG 2.1 Report — Level A Success Criteria
          </h2>
          <p className="mb-3 text-sm text-slate-600 dark:text-neutral-400">
            Table 1: Success Criteria, Level A. Applies to web content per EN 301 549 Chapter 9 and Section 508 Chapter 5.
          </p>
          <WcagTable criteria={levelA} />
        </section>

        {/* WCAG 2.1 Level AA */}
        <section aria-labelledby="wcag-level-aa" className="mb-10">
          <h2 id="wcag-level-aa" className="mb-1 text-xl font-semibold text-slate-900 dark:text-neutral-50">
            WCAG 2.1 Report — Level AA Success Criteria
          </h2>
          <p className="mb-3 text-sm text-slate-600 dark:text-neutral-400">
            Table 2: Success Criteria, Level AA. Applies to web content per EN 301 549 Chapter 9 and Section 508 Chapter 5.
          </p>
          <WcagTable criteria={levelAA} />
        </section>

        {/* Section 508 – Chapter 3: FPC */}
        <section aria-labelledby="sec508-fpc" className="mb-10">
          <h2 id="sec508-fpc" className="mb-1 text-xl font-semibold text-slate-900 dark:text-neutral-50">
            Revised Section 508 — Chapter 3: Functional Performance Criteria
          </h2>
          <p className="mb-3 text-sm text-slate-600 dark:text-neutral-400">
            36 CFR Part 1194 Appendix C. Evaluates whether users with specific functional limitations can accomplish tasks.
          </p>
          <GenericTable
            rows={FPC_CRITERIA}
            idKey="id"
            idLabel="Criterion"
            titleKey="title"
            label="Section 508 Functional Performance Criteria"
          />
        </section>

        {/* Section 508 – Chapter 5: Software */}
        <section aria-labelledby="sec508-software" className="mb-10">
          <h2 id="sec508-software" className="mb-1 text-xl font-semibold text-slate-900 dark:text-neutral-50">
            Revised Section 508 — Chapter 5: Software
          </h2>
          <p className="mb-3 text-sm text-slate-600 dark:text-neutral-400">
            36 CFR Part 1194 Appendix C, Chapter 5. Non-WCAG criteria for software interoperability and user preferences.
            Section 508 501.1 incorporates WCAG 2.0 Level A and AA by reference; see the WCAG tables above for those criteria.
          </p>
          <GenericTable
            rows={SEC508_SOFTWARE_CRITERIA}
            idKey="id"
            idLabel="Criterion"
            titleKey="title"
            label="Section 508 Chapter 5 Software criteria"
          />
        </section>

        {/* Section 508 – Chapter 6: Support Documentation */}
        <section aria-labelledby="sec508-support" className="mb-10">
          <h2 id="sec508-support" className="mb-1 text-xl font-semibold text-slate-900 dark:text-neutral-50">
            Revised Section 508 — Chapter 6: Support Documentation and Services
          </h2>
          <p className="mb-3 text-sm text-slate-600 dark:text-neutral-400">
            36 CFR Part 1194 Appendix C, Chapter 6. Covers accessibility of product documentation and support services.
          </p>
          <GenericTable
            rows={SEC508_SUPPORT_CRITERIA}
            idKey="id"
            idLabel="Criterion"
            titleKey="title"
            label="Section 508 Chapter 6 Support Documentation criteria"
          />
        </section>

        {/* EN 301 549 – Chapter 9: Web */}
        <section aria-labelledby="en301549-web" className="mb-10">
          <h2 id="en301549-web" className="mb-1 text-xl font-semibold text-slate-900 dark:text-neutral-50">
            EN 301 549 — Chapter 9: Web
          </h2>
          <p className="mb-3 text-sm text-slate-600 dark:text-neutral-400">
            EN 301 549 v3.2.1 Chapter 9 maps directly to WCAG 2.1 Level A and AA success criteria. Refer to the WCAG 2.1 Level A and Level AA tables above for full conformance details. All WCAG 2.1 Level A and AA criteria documented above apply under Chapter 9.
          </p>
          <div className="rounded-lg border border-slate-200 bg-white p-4 text-sm text-slate-700 dark:border-neutral-800 dark:bg-neutral-900 dark:text-neutral-300">
            EN 301 549 Clauses 9.1.1.1 – 9.4.1.3 incorporate WCAG 2.1 success criteria by reference. See the WCAG tables above.
          </div>
        </section>

        {/* EN 301 549 – Chapter 11: Software */}
        <section aria-labelledby="en301549-software" className="mb-10">
          <h2 id="en301549-software" className="mb-1 text-xl font-semibold text-slate-900 dark:text-neutral-50">
            EN 301 549 — Chapter 11: Software
          </h2>
          <p className="mb-3 text-sm text-slate-600 dark:text-neutral-400">
            EN 301 549 v3.2.1 Chapter 11 criteria beyond those incorporated from WCAG. Applies to non-web software components; criteria that duplicate WCAG are addressed in the WCAG tables above.
          </p>
          <GenericTable
            rows={EN301549_SOFTWARE_CRITERIA}
            idKey="clause"
            idLabel="Clause"
            titleKey="title"
            label="EN 301 549 Chapter 11 Software criteria"
          />
        </section>

        {/* EN 301 549 – Chapter 12: Documentation and Support */}
        <section aria-labelledby="en301549-support" className="mb-10">
          <h2 id="en301549-support" className="mb-1 text-xl font-semibold text-slate-900 dark:text-neutral-50">
            EN 301 549 — Chapter 12: Documentation and Support Services
          </h2>
          <p className="mb-3 text-sm text-slate-600 dark:text-neutral-400">
            EN 301 549 v3.2.1 Chapter 12. Accessibility of product documentation and support communication.
          </p>
          <GenericTable
            rows={EN301549_SUPPORT_CRITERIA}
            idKey="clause"
            idLabel="Clause"
            titleKey="title"
            label="EN 301 549 Chapter 12 Documentation and Support criteria"
          />
        </section>

        {/* Legal Disclaimer */}
        <section aria-labelledby="legal-disclaimer" className="mb-10">
          <h2 id="legal-disclaimer" className="mb-3 text-xl font-semibold text-slate-900 dark:text-neutral-50">
            Legal Disclaimer
          </h2>
          <div className="space-y-3 text-sm text-slate-700 dark:text-neutral-300">
            <p>
              The information provided in this Accessibility Conformance Report is accurate and true to the best of Lextures' knowledge and belief. This report is provided for informational purposes only. Lextures does not warrant that use of the product will be uninterrupted or error-free.
            </p>
            <p>
              VPAT® is a registered trademark of the Information Technology Industry Council (ITI). Use of this template does not imply ITI endorsement of the product.
            </p>
            <p>
              This report covers Version {PRODUCT_VERSION} of Lextures as evaluated on {EVAL_DATE}. Subsequent releases may alter conformance status. Archived versions of this report are retained at{' '}
              <Link to="/accessibility/vpat" className="text-indigo-700 underline dark:text-indigo-300">
                /accessibility/vpat
              </Link>
              .
            </p>
          </div>
        </section>

        {/* Accessibility Support Contact */}
        <section aria-labelledby="accessibility-support" className="mb-10">
          <h2 id="accessibility-support" className="mb-3 text-xl font-semibold text-slate-900 dark:text-neutral-50">
            Contact Accessibility Support
          </h2>
          <div className="space-y-3 text-sm text-slate-700 dark:text-neutral-300">
            <p>
              If you encounter an accessibility barrier, need an accommodation, or have questions about this report:
            </p>
            <ul className="list-disc pl-6 space-y-1">
              <li>
                <a
                  href="mailto:accessibility@lextures.com?subject=Accessibility%20accommodation%20request"
                  className="text-indigo-700 underline dark:text-indigo-300"
                >
                  Contact Accessibility Support
                </a>
                {' '}— pre-fills subject: "Accessibility accommodation request"
              </li>
              <li>We aim to respond to accessibility inquiries within 2 business days.</li>
              <li>For urgent accommodation needs, also contact your institution's IT or accessibility services office.</li>
            </ul>
          </div>
        </section>

      </main>

      <footer className="mt-12 border-t border-slate-200 px-4 py-6 text-center text-xs text-slate-500 dark:border-neutral-800 dark:text-neutral-500 print:hidden">
        <p>
          &copy; {new Date().getFullYear()} Lextures, Inc. &middot;{' '}
          <Link to="/accessibility" className="underline-offset-2 hover:underline">Accessibility Statement</Link>
          {' '}&middot;{' '}
          <Link to="/privacy" className="underline-offset-2 hover:underline">Privacy Policy</Link>
          {' '}&middot;{' '}
          <Link to="/terms" className="underline-offset-2 hover:underline">Terms of Service</Link>
          {' '}&middot;{' '}
          <Link to="/trust" className="underline-offset-2 hover:underline">Trust Center</Link>
        </p>
      </footer>
    </div>
  )
}
