import { useEffect } from 'react'
import { Link } from 'react-router-dom'
import { BrandLogo } from '../components/brand-logo'
import { type ConformanceLevel, WCAG_CRITERIA } from '../lib/vpat-data'

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

export default function AccessibilityConformancePage() {
  useEffect(() => {
    document.title = 'Accessibility Conformance Statement — Lextures'
  }, [])

  const levelA = WCAG_CRITERIA.filter((c) => c.level === 'A')
  const levelAA = WCAG_CRITERIA.filter((c) => c.level === 'AA')

  return (
    <div className="min-h-dvh bg-slate-50 text-slate-900 dark:bg-neutral-950 dark:text-neutral-100">
      <header className="border-b border-slate-200 bg-white px-4 py-4 dark:border-neutral-800 dark:bg-neutral-900 sm:px-6 print:hidden">
        <div className="mx-auto flex max-w-5xl flex-wrap items-center justify-between gap-3">
          <Link to="/login" className="inline-flex items-center gap-2 text-sm font-medium text-slate-700 dark:text-neutral-200">
            <BrandLogo className="h-7 w-auto" />
            <span className="sr-only">Lextures home</span>
          </Link>
          <nav aria-label="Legal" className="flex flex-wrap gap-3 text-sm">
            <Link to="/accessibility/vpat" className="text-indigo-700 underline-offset-2 hover:underline dark:text-indigo-300">VPAT</Link>
            <Link to="/privacy" className="text-indigo-700 underline-offset-2 hover:underline dark:text-indigo-300">Privacy</Link>
            <Link to="/terms" className="text-indigo-700 underline-offset-2 hover:underline dark:text-indigo-300">Terms</Link>
            <Link to="/trust" className="text-indigo-700 underline-offset-2 hover:underline dark:text-indigo-300">Trust</Link>
            <Link to="/login" className="text-slate-600 hover:text-slate-900 dark:text-neutral-400 dark:hover:text-neutral-100">Sign in</Link>
          </nav>
        </div>
      </header>

      <main id="main-content" className="mx-auto max-w-5xl px-4 py-8 sm:px-6 lg:py-12">
        <div className="mb-10">
          <h1 className="text-3xl font-semibold tracking-tight text-slate-900 dark:text-neutral-50">
            Accessibility Conformance Statement
          </h1>
          <p className="mt-2 text-base text-slate-600 dark:text-neutral-400">
            Lextures strives to conform to the{' '}
            <a
              href="https://www.w3.org/TR/WCAG21/"
              className="text-indigo-700 underline underline-offset-2 dark:text-indigo-300"
              target="_blank"
              rel="noreferrer"
            >
              Web Content Accessibility Guidelines (WCAG) 2.1
            </a>{' '}
            at Level AA, as required by Section 508 of the Rehabilitation Act (36 CFR Part 1194) and
            EN 301 549 for public-sector procurement.
          </p>
          <p className="mt-2 text-sm text-slate-500 dark:text-neutral-500">
            Last updated: May 27, 2026. This statement is reviewed and updated annually.
            For the full VPAT (Voluntary Product Accessibility Template) covering Section 508 and EN 301 549, see the{' '}
            <Link to="/accessibility/vpat" className="text-indigo-700 underline dark:text-indigo-300">
              Accessibility Conformance Report (VPAT)
            </Link>
            . Questions? Email{' '}
            <a href="mailto:accessibility@lextures.com" className="text-indigo-700 underline dark:text-indigo-300">
              accessibility@lextures.com
            </a>
            .
          </p>
        </div>

        <div className="mb-8 rounded-lg border border-slate-200 bg-white p-5 dark:border-neutral-800 dark:bg-neutral-900">
          <h2 className="mb-3 text-lg font-semibold text-slate-900 dark:text-neutral-50">
            Conformance Summary
          </h2>
          <dl className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
            {[
              { label: 'Standard', value: 'WCAG 2.1 AA' },
              { label: 'Conformance Level', value: 'AA (target)' },
              { label: 'Evaluation Method', value: 'Axe-core automated + manual review' },
              { label: 'Applicable Laws', value: 'Section 508 / EN 301 549 / ADA Title II' },
            ].map(({ label, value }) => (
              <div key={label}>
                <dt className="text-xs font-medium uppercase tracking-wide text-slate-500 dark:text-neutral-400">
                  {label}
                </dt>
                <dd className="mt-1 text-sm font-medium text-slate-900 dark:text-neutral-100">{value}</dd>
              </div>
            ))}
          </dl>
        </div>

        <div className="mb-8 rounded-lg border border-amber-200 bg-amber-50 p-4 text-sm text-amber-800 dark:border-amber-900/60 dark:bg-amber-950/40 dark:text-amber-300">
          <strong>Status note:</strong> Lextures is an active conformance program (plan 10.7). The
          items marked "Partially Supports" represent known gaps that are actively being remediated.
          Automated axe-core checks run on every pull request to prevent regressions.
        </div>

        <section aria-labelledby="level-a-heading" className="mb-10">
          <h2 id="level-a-heading" className="mb-3 text-xl font-semibold text-slate-900 dark:text-neutral-50">
            WCAG 2.1 Level A Success Criteria
          </h2>
          <CriteriaTable criteria={levelA} />
        </section>

        <section aria-labelledby="level-aa-heading" className="mb-10">
          <h2 id="level-aa-heading" className="mb-3 text-xl font-semibold text-slate-900 dark:text-neutral-50">
            WCAG 2.1 Level AA Success Criteria
          </h2>
          <CriteriaTable criteria={levelAA} />
        </section>

        <section aria-labelledby="contact-heading" className="mb-10">
          <h2 id="contact-heading" className="mb-3 text-xl font-semibold text-slate-900 dark:text-neutral-50">
            Feedback &amp; Assistance
          </h2>
          <div className="space-y-3 text-sm text-slate-700 dark:text-neutral-300">
            <p>
              If you encounter an accessibility barrier on Lextures, please contact us:
            </p>
            <ul className="list-disc pl-6 space-y-1">
              <li>
                Email:{' '}
                <a href="mailto:accessibility@lextures.com" className="text-indigo-700 underline dark:text-indigo-300">
                  accessibility@lextures.com
                </a>
              </li>
              <li>We aim to respond to accessibility inquiries within 2 business days.</li>
              <li>
                For urgent accommodation needs, please also contact your institution's IT or
                accessibility services office.
              </li>
            </ul>
          </div>
        </section>
      </main>

      <footer className="mt-12 border-t border-slate-200 px-4 py-6 text-center text-xs text-slate-500 dark:border-neutral-800 dark:text-neutral-500 print:hidden">
        <p>
          &copy; {new Date().getFullYear()} Lextures, Inc. &middot;{' '}
          <Link to="/accessibility/vpat" className="underline-offset-2 hover:underline">
            VPAT
          </Link>
          {' '}&middot;{' '}
          <Link to="/privacy" className="underline-offset-2 hover:underline">
            Privacy Policy
          </Link>{' '}
          &middot;{' '}
          <Link to="/terms" className="underline-offset-2 hover:underline">
            Terms of Service
          </Link>{' '}
          &middot;{' '}
          <Link to="/trust" className="underline-offset-2 hover:underline">
            Trust Center
          </Link>
        </p>
      </footer>
    </div>
  )
}

function CriteriaTable({ criteria }: { criteria: typeof WCAG_CRITERIA }) {
  return (
    <div className="overflow-x-auto rounded-lg border border-slate-200 dark:border-neutral-800">
      <table
        className="min-w-full text-sm border-collapse"
        aria-label="WCAG success criteria conformance"
      >
        <caption className="sr-only">
          WCAG 2.1 success criteria with conformance level and notes
        </caption>
        <thead className="bg-slate-50 dark:bg-neutral-900">
          <tr className="border-b border-slate-200 dark:border-neutral-700">
            <th scope="col" className="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wide text-slate-600 dark:text-neutral-400 whitespace-nowrap">
              SC
            </th>
            <th scope="col" className="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wide text-slate-600 dark:text-neutral-400">
              Title
            </th>
            <th scope="col" className="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wide text-slate-600 dark:text-neutral-400 whitespace-nowrap">
              Conformance
            </th>
            <th scope="col" className="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wide text-slate-600 dark:text-neutral-400">
              Notes
            </th>
          </tr>
        </thead>
        <tbody className="divide-y divide-slate-100 bg-white dark:divide-neutral-800 dark:bg-neutral-900">
          {criteria.map((c) => (
            <tr key={c.sc} className="hover:bg-slate-50 dark:hover:bg-neutral-800/50">
              <td className="px-4 py-3 font-mono text-xs font-medium text-slate-700 dark:text-neutral-300 whitespace-nowrap">
                {c.sc}
              </td>
              <td className="px-4 py-3 text-slate-800 dark:text-neutral-200">{c.title}</td>
              <td className="px-4 py-3 whitespace-nowrap">
                <span
                  className={`inline-block rounded px-2 py-0.5 text-xs font-medium ${conformanceBadgeClass(c.conformance)}`}
                >
                  {c.conformance}
                </span>
              </td>
              <td className="px-4 py-3 text-slate-600 dark:text-neutral-400">{c.notes}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
