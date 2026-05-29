import { useEffect } from 'react'
import { Header } from '../components/header'
import { LegalNav } from '../components/legal-nav'
import { SiteFooter } from '../components/site-footer'
import { ConformanceBadge } from '../lib/conformance-ui'
import { SITE_LINKS } from '../lib/site-links'
import {
  EN301549_SOFTWARE_CRITERIA,
  EN301549_SUPPORT_CRITERIA,
  FPC_CRITERIA,
  SEC508_SOFTWARE_CRITERIA,
  SEC508_SUPPORT_CRITERIA,
  WCAG_CRITERIA,
  type ConformanceLevel,
} from '../lib/vpat-data'

const EVAL_DATE = 'May 27, 2026'
const PRODUCT_VERSION = '1.0'

function WcagTable({ criteria }: { criteria: typeof WCAG_CRITERIA }) {
  return (
    <div className="overflow-x-auto rounded-xl border border-stone-200/90">
      <table className="min-w-full border-collapse text-sm" aria-label="WCAG success criteria conformance">
        <caption className="sr-only">WCAG 2.1 success criteria with conformance level and notes</caption>
        <thead className="bg-stone-50">
          <tr className="border-b border-stone-200">
            <th scope="col" className="px-4 py-3 text-start text-xs font-semibold uppercase tracking-wide text-stone-600 whitespace-nowrap">SC</th>
            <th scope="col" className="px-4 py-3 text-start text-xs font-semibold uppercase tracking-wide text-stone-600">Title</th>
            <th scope="col" className="px-4 py-3 text-start text-xs font-semibold uppercase tracking-wide text-stone-600 whitespace-nowrap">Conformance</th>
            <th scope="col" className="px-4 py-3 text-start text-xs font-semibold uppercase tracking-wide text-stone-600">Remarks</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-stone-100 bg-white">
          {criteria.map((c) => (
            <tr key={c.sc} className="hover:bg-stone-50/80">
              <td className="px-4 py-3 font-mono text-xs font-medium text-stone-700 whitespace-nowrap">{c.sc}</td>
              <td className="px-4 py-3 text-stone-800">{c.title}</td>
              <td className="px-4 py-3 whitespace-nowrap"><ConformanceBadge level={c.conformance} /></td>
              <td className="px-4 py-3 text-stone-600">{c.notes}</td>
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
    <div className="overflow-x-auto rounded-xl border border-stone-200/90">
      <table className="min-w-full border-collapse text-sm" aria-label={label}>
        <caption className="sr-only">{label}</caption>
        <thead className="bg-stone-50">
          <tr className="border-b border-stone-200">
            <th scope="col" className="px-4 py-3 text-start text-xs font-semibold uppercase tracking-wide text-stone-600 whitespace-nowrap">{idLabel}</th>
            <th scope="col" className="px-4 py-3 text-start text-xs font-semibold uppercase tracking-wide text-stone-600">Criterion</th>
            <th scope="col" className="px-4 py-3 text-start text-xs font-semibold uppercase tracking-wide text-stone-600 whitespace-nowrap">Conformance</th>
            <th scope="col" className="px-4 py-3 text-start text-xs font-semibold uppercase tracking-wide text-stone-600">Remarks</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-stone-100 bg-white">
          {rows.map((r) => (
            <tr key={String(r[idKey])} className="hover:bg-stone-50/80">
              <td className="px-4 py-3 font-mono text-xs font-medium text-stone-700 whitespace-nowrap">{String(r[idKey])}</td>
              <td className="px-4 py-3 text-stone-800">{String(r[titleKey])}</td>
              <td className="px-4 py-3 whitespace-nowrap"><ConformanceBadge level={r.conformance} /></td>
              <td className="px-4 py-3 text-stone-600">{r.notes}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

export function VpatPage() {
  useEffect(() => {
    document.title = 'VPAT — Accessibility Conformance Report — Lextures'
  }, [])

  const levelA = WCAG_CRITERIA.filter((c) => c.level === 'A')
  const levelAA = WCAG_CRITERIA.filter((c) => c.level === 'AA')

  return (
    <div className="relative min-h-screen overflow-x-hidden bg-stone-50 text-slate-700">
      <a
        href="#main-content"
        className="sr-only focus:not-sr-only focus:absolute focus:left-4 focus:top-4 focus:z-50 focus:rounded-lg focus:bg-white focus:px-4 focus:py-2 focus:text-sm focus:font-medium focus:shadow-lg focus:ring-2 focus:ring-accent"
      >
        Skip to main content
      </a>

      <Header />

      <main id="main-content" className="mx-auto max-w-5xl px-4 py-8 sm:px-6 lg:py-12">
        <LegalNav />

        <div className="mb-8">
          <p className="text-sm font-medium uppercase tracking-wide text-accent">VPAT® 2.5 INT (International Edition)</p>
          <h1 className="font-display mt-1 text-3xl font-normal tracking-tight text-stone-900 sm:text-4xl">
            Accessibility Conformance Report
          </h1>
          <p className="mt-2 text-base leading-relaxed text-stone-600">
            Based on VPAT® Version 2.5 as published by the Information Technology Industry Council (ITI).
            Covers WCAG 2.1, Revised Section 508, and EN 301 549 v3.2.1.
          </p>
        </div>

        <section aria-labelledby="product-info-heading" className="mb-8">
          <h2 id="product-info-heading" className="mb-3 text-xl font-semibold text-stone-900">
            Product Information
          </h2>
          <div className="rounded-xl border border-stone-200/90 bg-white p-5 shadow-sm">
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
                  <dt className="text-xs font-semibold uppercase tracking-wide text-stone-500">{label}</dt>
                  <dd className="mt-1 text-sm text-stone-900">
                    {label === 'Contact / Accessibility Support' ? (
                      <a href="mailto:accessibility@lextures.com" className="text-accent underline underline-offset-2">
                        {value}
                      </a>
                    ) : value}
                  </dd>
                </div>
              ))}
            </dl>
          </div>
        </section>

        <section aria-labelledby="download-heading" className="mb-8">
          <h2 id="download-heading" className="mb-3 text-xl font-semibold text-stone-900">
            Download This Report
          </h2>
          <div className="flex flex-wrap gap-3">
            <a
              href="/vpat/VPAT_2.5_INT_Lextures_2026-05.md"
              download
              className="btn-secondary"
              aria-label="Download VPAT source document (Markdown)"
            >
              Download VPAT (Markdown source)
            </a>
          </div>
          <p className="mt-2 text-xs text-stone-500">
            DOCX and PDF versions are available on request at{' '}
            <a href="mailto:accessibility@lextures.com" className="text-accent underline underline-offset-2">accessibility@lextures.com</a>.
          </p>
        </section>

        <nav aria-label="Report sections" className="mb-10 rounded-xl border border-stone-200/90 bg-white p-5 shadow-sm">
          <h2 className="mb-3 text-base font-semibold text-stone-900">Contents</h2>
          <ol className="list-decimal space-y-1 ps-5 text-sm text-accent">
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

        <div className="mb-8 rounded-xl border border-amber-200 bg-amber-50 p-4 text-sm text-amber-800">
          <strong>Note:</strong> Entries marked &quot;Partially Supports&quot; represent known gaps actively being remediated. Each entry includes a remarks field with the specific gap and remediation timeline. This report was last evaluated on {EVAL_DATE} and is updated with each major release.
        </div>

        <section aria-labelledby="wcag-level-a" className="mb-10">
          <h2 id="wcag-level-a" className="mb-1 text-xl font-semibold text-stone-900">
            WCAG 2.1 Report — Level A Success Criteria
          </h2>
          <p className="mb-3 text-sm text-stone-600">
            Table 1: Success Criteria, Level A. Applies to web content per EN 301 549 Chapter 9 and Section 508 Chapter 5.
          </p>
          <WcagTable criteria={levelA} />
        </section>

        <section aria-labelledby="wcag-level-aa" className="mb-10">
          <h2 id="wcag-level-aa" className="mb-1 text-xl font-semibold text-stone-900">
            WCAG 2.1 Report — Level AA Success Criteria
          </h2>
          <p className="mb-3 text-sm text-stone-600">
            Table 2: Success Criteria, Level AA. Applies to web content per EN 301 549 Chapter 9 and Section 508 Chapter 5.
          </p>
          <WcagTable criteria={levelAA} />
        </section>

        <section aria-labelledby="sec508-fpc" className="mb-10">
          <h2 id="sec508-fpc" className="mb-1 text-xl font-semibold text-stone-900">
            Revised Section 508 — Chapter 3: Functional Performance Criteria
          </h2>
          <p className="mb-3 text-sm text-stone-600">
            36 CFR Part 1194 Appendix C. Evaluates whether users with specific functional limitations can accomplish tasks.
          </p>
          <GenericTable rows={FPC_CRITERIA} idKey="id" idLabel="Criterion" titleKey="title" label="Section 508 Functional Performance Criteria" />
        </section>

        <section aria-labelledby="sec508-software" className="mb-10">
          <h2 id="sec508-software" className="mb-1 text-xl font-semibold text-stone-900">
            Revised Section 508 — Chapter 5: Software
          </h2>
          <p className="mb-3 text-sm text-stone-600">
            36 CFR Part 1194 Appendix C, Chapter 5. Non-WCAG criteria for software interoperability and user preferences.
            Section 508 501.1 incorporates WCAG 2.0 Level A and AA by reference; see the WCAG tables above for those criteria.
          </p>
          <GenericTable rows={SEC508_SOFTWARE_CRITERIA} idKey="id" idLabel="Criterion" titleKey="title" label="Section 508 Chapter 5 Software criteria" />
        </section>

        <section aria-labelledby="sec508-support" className="mb-10">
          <h2 id="sec508-support" className="mb-1 text-xl font-semibold text-stone-900">
            Revised Section 508 — Chapter 6: Support Documentation and Services
          </h2>
          <p className="mb-3 text-sm text-stone-600">
            36 CFR Part 1194 Appendix C, Chapter 6. Covers accessibility of product documentation and support services.
          </p>
          <GenericTable rows={SEC508_SUPPORT_CRITERIA} idKey="id" idLabel="Criterion" titleKey="title" label="Section 508 Chapter 6 Support Documentation criteria" />
        </section>

        <section aria-labelledby="en301549-web" className="mb-10">
          <h2 id="en301549-web" className="mb-1 text-xl font-semibold text-stone-900">
            EN 301 549 — Chapter 9: Web
          </h2>
          <p className="mb-3 text-sm text-stone-600">
            EN 301 549 v3.2.1 Chapter 9 maps directly to WCAG 2.1 Level A and AA success criteria. Refer to the WCAG 2.1 Level A and Level AA tables above for full conformance details. All WCAG 2.1 Level A and AA criteria documented above apply under Chapter 9.
          </p>
          <div className="rounded-xl border border-stone-200/90 bg-white p-4 text-sm text-stone-700">
            EN 301 549 Clauses 9.1.1.1 – 9.4.1.3 incorporate WCAG 2.1 success criteria by reference. See the WCAG tables above.
          </div>
        </section>

        <section aria-labelledby="en301549-software" className="mb-10">
          <h2 id="en301549-software" className="mb-1 text-xl font-semibold text-stone-900">
            EN 301 549 — Chapter 11: Software
          </h2>
          <p className="mb-3 text-sm text-stone-600">
            EN 301 549 v3.2.1 Chapter 11 criteria beyond those incorporated from WCAG. Applies to non-web software components; criteria that duplicate WCAG are addressed in the WCAG tables above.
          </p>
          <GenericTable rows={EN301549_SOFTWARE_CRITERIA} idKey="clause" idLabel="Clause" titleKey="title" label="EN 301 549 Chapter 11 Software criteria" />
        </section>

        <section aria-labelledby="en301549-support" className="mb-10">
          <h2 id="en301549-support" className="mb-1 text-xl font-semibold text-stone-900">
            EN 301 549 — Chapter 12: Documentation and Support Services
          </h2>
          <p className="mb-3 text-sm text-stone-600">
            EN 301 549 v3.2.1 Chapter 12. Accessibility of product documentation and support communication.
          </p>
          <GenericTable rows={EN301549_SUPPORT_CRITERIA} idKey="clause" idLabel="Clause" titleKey="title" label="EN 301 549 Chapter 12 Documentation and Support criteria" />
        </section>

        <section aria-labelledby="legal-disclaimer" className="mb-10">
          <h2 id="legal-disclaimer" className="mb-3 text-xl font-semibold text-stone-900">
            Legal Disclaimer
          </h2>
          <div className="space-y-3 text-sm leading-relaxed text-stone-700">
            <p>
              The information provided in this Accessibility Conformance Report is accurate and true to the best of Lextures&apos; knowledge and belief. This report is provided for informational purposes only. Lextures does not warrant that use of the product will be uninterrupted or error-free.
            </p>
            <p>
              VPAT® is a registered trademark of the Information Technology Industry Council (ITI). Use of this template does not imply ITI endorsement of the product.
            </p>
            <p>
              This report covers Version {PRODUCT_VERSION} of Lextures as evaluated on {EVAL_DATE}. Subsequent releases may alter conformance status. Archived versions of this report are retained at{' '}
              <a href={SITE_LINKS.accessibilityVpat} className="text-accent underline underline-offset-2">
                {SITE_LINKS.accessibilityVpat}
              </a>
              .
            </p>
          </div>
        </section>

        <section aria-labelledby="accessibility-support" className="mb-10">
          <h2 id="accessibility-support" className="mb-3 text-xl font-semibold text-stone-900">
            Contact Accessibility Support
          </h2>
          <div className="space-y-3 text-sm leading-relaxed text-stone-700">
            <p>
              If you encounter an accessibility barrier, need an accommodation, or have questions about this report:
            </p>
            <ul className="list-disc space-y-1 ps-6">
              <li>
                <a
                  href="mailto:accessibility@lextures.com?subject=Accessibility%20accommodation%20request"
                  className="text-accent underline underline-offset-2"
                >
                  Contact Accessibility Support
                </a>
                {' '}— pre-fills subject: &quot;Accessibility accommodation request&quot;
              </li>
              <li>We aim to respond to accessibility inquiries within 2 business days.</li>
              <li>For urgent accommodation needs, also contact your institution&apos;s IT or accessibility services office.</li>
            </ul>
          </div>
        </section>
      </main>

      <SiteFooter />
    </div>
  )
}
