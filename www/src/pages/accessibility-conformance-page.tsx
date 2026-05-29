import { useEffect } from 'react'
import { Header } from '../components/header'
import { LegalNav } from '../components/legal-nav'
import { SiteFooter } from '../components/site-footer'
import { ConformanceBadge } from '../lib/conformance-ui'
import { WCAG_CRITERIA } from '../lib/vpat-data'
import { SITE_LINKS } from '../lib/site-links'

export function AccessibilityConformancePage() {
  useEffect(() => {
    document.title = 'Accessibility Conformance Statement — Lextures'
  }, [])

  const levelA = WCAG_CRITERIA.filter((c) => c.level === 'A')
  const levelAA = WCAG_CRITERIA.filter((c) => c.level === 'AA')

  return (
    <div className="relative min-h-screen overflow-x-hidden bg-stone-50 text-slate-700">
      <Header />

      <main id="main-content" className="mx-auto max-w-5xl px-4 py-8 sm:px-6 lg:py-12">
        <LegalNav />

        <div className="mb-10">
          <h1 className="font-display text-3xl font-normal tracking-tight text-stone-900 sm:text-4xl">
            Accessibility Conformance Statement
          </h1>
          <p className="mt-2 text-base leading-relaxed text-stone-600">
            Lextures strives to conform to the{' '}
            <a
              href="https://www.w3.org/TR/WCAG21/"
              className="text-accent underline underline-offset-2"
              target="_blank"
              rel="noreferrer"
            >
              Web Content Accessibility Guidelines (WCAG) 2.1
            </a>{' '}
            at Level AA, as required by Section 508 of the Rehabilitation Act (36 CFR Part 1194) and
            EN 301 549 for public-sector procurement.
          </p>
          <p className="mt-2 text-sm text-stone-500">
            Last updated: May 27, 2026. This statement is reviewed and updated annually.
            For the full VPAT (Voluntary Product Accessibility Template) covering Section 508 and EN 301 549, see the{' '}
            <a href={SITE_LINKS.accessibilityVpat} className="text-accent underline underline-offset-2">
              Accessibility Conformance Report (VPAT)
            </a>
            . Questions? Email{' '}
            <a href="mailto:accessibility@lextures.com" className="text-accent underline underline-offset-2">
              accessibility@lextures.com
            </a>
            .
          </p>
        </div>

        <div className="mb-8 rounded-xl border border-stone-200/90 bg-white p-5 shadow-sm">
          <h2 className="mb-3 text-lg font-semibold text-stone-900">Conformance Summary</h2>
          <dl className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
            {[
              { label: 'Standard', value: 'WCAG 2.1 AA' },
              { label: 'Conformance Level', value: 'AA (target)' },
              { label: 'Evaluation Method', value: 'Axe-core automated + manual review' },
              { label: 'Applicable Laws', value: 'Section 508 / EN 301 549 / ADA Title II' },
            ].map(({ label, value }) => (
              <div key={label}>
                <dt className="text-xs font-medium uppercase tracking-wide text-stone-500">{label}</dt>
                <dd className="mt-1 text-sm font-medium text-stone-900">{value}</dd>
              </div>
            ))}
          </dl>
        </div>

        <div className="mb-8 rounded-xl border border-amber-200 bg-amber-50 p-4 text-sm text-amber-800">
          <strong>Status note:</strong> Lextures is an active conformance program (plan 10.7). The
          items marked &quot;Partially Supports&quot; represent known gaps that are actively being remediated.
          Automated axe-core checks run on every pull request to prevent regressions.
        </div>

        <section aria-labelledby="level-a-heading" className="mb-10">
          <h2 id="level-a-heading" className="mb-3 text-xl font-semibold text-stone-900">
            WCAG 2.1 Level A Success Criteria
          </h2>
          <CriteriaTable criteria={levelA} />
        </section>

        <section aria-labelledby="level-aa-heading" className="mb-10">
          <h2 id="level-aa-heading" className="mb-3 text-xl font-semibold text-stone-900">
            WCAG 2.1 Level AA Success Criteria
          </h2>
          <CriteriaTable criteria={levelAA} />
        </section>

        <section aria-labelledby="contact-heading" className="mb-10">
          <h2 id="contact-heading" className="mb-3 text-xl font-semibold text-stone-900">
            Feedback &amp; Assistance
          </h2>
          <div className="space-y-3 text-sm leading-relaxed text-stone-700">
            <p>If you encounter an accessibility barrier on Lextures, please contact us:</p>
            <ul className="list-disc space-y-1 ps-6">
              <li>
                Email:{' '}
                <a href="mailto:accessibility@lextures.com" className="text-accent underline underline-offset-2">
                  accessibility@lextures.com
                </a>
              </li>
              <li>We aim to respond to accessibility inquiries within 2 business days.</li>
              <li>
                For urgent accommodation needs, please also contact your institution&apos;s IT or
                accessibility services office.
              </li>
            </ul>
          </div>
        </section>
      </main>

      <SiteFooter />
    </div>
  )
}

function CriteriaTable({ criteria }: { criteria: typeof WCAG_CRITERIA }) {
  return (
    <div className="overflow-x-auto rounded-xl border border-stone-200/90">
      <table className="min-w-full border-collapse text-sm" aria-label="WCAG success criteria conformance">
        <caption className="sr-only">WCAG 2.1 success criteria with conformance level and notes</caption>
        <thead className="bg-stone-50">
          <tr className="border-b border-stone-200">
            <th scope="col" className="px-4 py-3 text-start text-xs font-semibold uppercase tracking-wide text-stone-600 whitespace-nowrap">
              SC
            </th>
            <th scope="col" className="px-4 py-3 text-start text-xs font-semibold uppercase tracking-wide text-stone-600">
              Title
            </th>
            <th scope="col" className="px-4 py-3 text-start text-xs font-semibold uppercase tracking-wide text-stone-600 whitespace-nowrap">
              Conformance
            </th>
            <th scope="col" className="px-4 py-3 text-start text-xs font-semibold uppercase tracking-wide text-stone-600">
              Notes
            </th>
          </tr>
        </thead>
        <tbody className="divide-y divide-stone-100 bg-white">
          {criteria.map((c) => (
            <tr key={c.sc} className="hover:bg-stone-50/80">
              <td className="px-4 py-3 font-mono text-xs font-medium text-stone-700 whitespace-nowrap">{c.sc}</td>
              <td className="px-4 py-3 text-stone-800">{c.title}</td>
              <td className="px-4 py-3 whitespace-nowrap">
                <ConformanceBadge level={c.conformance} />
              </td>
              <td className="px-4 py-3 text-stone-600">{c.notes}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
