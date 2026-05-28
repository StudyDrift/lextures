import { useEffect } from 'react'
import { Link } from 'react-router-dom'
import { BrandLogo } from '../components/brand-logo'
import {
  COORDINATED_DISCLOSURE_DAYS,
  IN_SCOPE_ITEMS,
  OUT_OF_SCOPE_ITEMS,
  PATCH_SLA_ROWS,
  PGP_FINGERPRINT,
  PGP_KEY_URL,
  SECURITY_CONTACT_EMAIL,
} from '../content/security/disclosure-policy'

export default function SecurityPage() {
  useEffect(() => {
    document.title = 'Security & Responsible Disclosure — Lextures'
  }, [])

  return (
    <div className="min-h-dvh bg-slate-50 text-slate-900 dark:bg-neutral-950 dark:text-neutral-100">
      <header className="border-b border-slate-200 bg-white px-4 py-4 dark:border-neutral-800 dark:bg-neutral-900 sm:px-6 print:hidden">
        <div className="mx-auto flex max-w-5xl flex-wrap items-center justify-between gap-3">
          <Link to="/login" className="inline-flex items-center gap-2 text-sm font-medium text-slate-700 dark:text-neutral-200">
            <BrandLogo className="h-7 w-auto" />
            <span className="sr-only">Lextures home</span>
          </Link>
          <nav aria-label="Legal" className="flex flex-wrap gap-3 text-sm">
            <Link to="/trust" className="text-indigo-700 underline-offset-2 hover:underline dark:text-indigo-300">Trust</Link>
            <Link to="/privacy" className="text-indigo-700 underline-offset-2 hover:underline dark:text-indigo-300">Privacy</Link>
            <Link to="/terms" className="text-indigo-700 underline-offset-2 hover:underline dark:text-indigo-300">Terms</Link>
            <Link to="/login" className="text-slate-600 hover:text-slate-900 dark:text-neutral-400 dark:hover:text-neutral-100">Sign in</Link>
          </nav>
        </div>
      </header>

      <main id="main-content" className="mx-auto max-w-5xl px-4 py-8 sm:px-6 lg:py-12">
        <div className="mb-10">
          <h1 className="text-3xl font-semibold tracking-tight text-slate-900 dark:text-neutral-50">
            Security &amp; Responsible Disclosure
          </h1>
          <p className="mt-2 text-base text-slate-600 dark:text-neutral-400">
            How to report vulnerabilities, our safe harbor commitment, and coordinated disclosure timeline.
          </p>
        </div>

        <div className="flex flex-col gap-8 text-sm text-slate-700 dark:text-neutral-300">
          <section aria-labelledby="report-heading">
            <h2 id="report-heading" className="text-lg font-semibold text-slate-900 dark:text-neutral-100">
              Report a vulnerability
            </h2>
            <p className="mt-2">
              Email{' '}
              <a href={`mailto:${SECURITY_CONTACT_EMAIL}`} className="text-indigo-700 underline dark:text-indigo-300">
                {SECURITY_CONTACT_EMAIL}
              </a>{' '}
              with reproduction steps, affected components, and any safe proof-of-concept. We acknowledge valid reports
              within <strong>2 business days</strong>.
            </p>
            <p className="mt-2">
              <strong>PGP fingerprint:</strong>{' '}
              <code className="rounded bg-slate-100 px-1.5 py-0.5 text-xs dark:bg-neutral-800">{PGP_FINGERPRINT}</code>
            </p>
            <p className="mt-2">
              <a href={PGP_KEY_URL} className="text-indigo-700 underline dark:text-indigo-300" rel="noopener noreferrer" target="_blank">
                Download public key on keys.openpgp.org
              </a>
            </p>
            <p className="mt-2 text-slate-500 dark:text-neutral-500">
              Repository policy: see{' '}
              <a
                href="https://github.com/lextures/lextures/blob/main/SECURITY.md"
                className="text-indigo-700 underline dark:text-indigo-300"
                rel="noopener noreferrer"
                target="_blank"
              >
                SECURITY.md
              </a>{' '}
              at the repo root.
            </p>
          </section>

          <section aria-labelledby="safe-harbor-heading">
            <h2 id="safe-harbor-heading" className="text-lg font-semibold text-slate-900 dark:text-neutral-100">
              Safe harbor
            </h2>
            <p className="mt-2">
              Good-faith security research conducted within the scope below will not result in legal action from Lextures,
              provided you avoid privacy violations, service disruption, and access to other users&apos; data beyond what is
              necessary to demonstrate the issue.
            </p>
          </section>

          <section aria-labelledby="scope-heading">
            <h2 id="scope-heading" className="text-lg font-semibold text-slate-900 dark:text-neutral-100">
              Scope
            </h2>
            <h3 className="mt-3 font-medium text-slate-800 dark:text-neutral-200">In scope</h3>
            <ul className="mt-1 list-disc ps-5 space-y-1">
              {IN_SCOPE_ITEMS.map((item) => (
                <li key={item}>{item}</li>
              ))}
            </ul>
            <h3 className="mt-4 font-medium text-slate-800 dark:text-neutral-200">Out of scope</h3>
            <ul className="mt-1 list-disc ps-5 space-y-1">
              {OUT_OF_SCOPE_ITEMS.map((item) => (
                <li key={item}>{item}</li>
              ))}
            </ul>
          </section>

          <section aria-labelledby="disclosure-heading">
            <h2 id="disclosure-heading" className="text-lg font-semibold text-slate-900 dark:text-neutral-100">
              Coordinated disclosure
            </h2>
            <p className="mt-2">
              We use a <strong>{COORDINATED_DISCLOSURE_DAYS}-day</strong> coordinated disclosure window from acknowledgment,
              aligned with industry practice (Google Project Zero / CERT/CC). Critical issues may be patched sooner with
              reporter agreement.
            </p>
          </section>

          <section aria-labelledby="sla-heading">
            <h2 id="sla-heading" className="text-lg font-semibold text-slate-900 dark:text-neutral-100">
              Patch targets (CVSS 3.1)
            </h2>
            <div className="mt-3 overflow-x-auto">
              <table className="min-w-full border-collapse text-sm" aria-label="Patch SLA by severity">
                <thead>
                  <tr className="border-b border-slate-200 dark:border-neutral-700">
                    <th scope="col" className="py-2 pe-4 text-start font-semibold">Severity</th>
                    <th scope="col" className="py-2 text-start font-semibold">Target</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-100 dark:divide-neutral-800">
                  {PATCH_SLA_ROWS.map((row) => (
                    <tr key={row.severity}>
                      <td className="py-2 pe-4 font-medium text-slate-900 dark:text-neutral-100">{row.severity}</td>
                      <td className="py-2">{row.days}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </section>

          <section aria-labelledby="bounty-heading">
            <h2 id="bounty-heading" className="text-lg font-semibold text-slate-900 dark:text-neutral-100">
              Bug bounty
            </h2>
            <p className="mt-2">
              Invite-only rewards may be offered for valid critical and high findings. Program details will be posted here
              when a HackerOne or Bugcrowd program launches.
            </p>
          </section>
        </div>
      </main>

      <footer className="mt-12 border-t border-slate-200 dark:border-neutral-800 py-6 px-4 text-center text-xs text-slate-500 dark:text-neutral-500 print:hidden">
        <p>
          &copy; {new Date().getFullYear()} Lextures, Inc. &middot;{' '}
          <Link to="/trust" className="underline-offset-2 hover:underline">Trust Center</Link>
        </p>
      </footer>
    </div>
  )
}
