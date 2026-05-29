import { useEffect } from 'react'
import { Header } from '../components/header'
import { LegalNav } from '../components/legal-nav'
import { SiteFooter } from '../components/site-footer'
import {
  COORDINATED_DISCLOSURE_DAYS,
  IN_SCOPE_ITEMS,
  OUT_OF_SCOPE_ITEMS,
  PATCH_SLA_ROWS,
  PGP_FINGERPRINT,
  PGP_KEY_URL,
  SECURITY_CONTACT_EMAIL,
} from '../content/security/disclosure-policy'

export function SecurityPage() {
  useEffect(() => {
    document.title = 'Security & Responsible Disclosure — Lextures'
  }, [])

  return (
    <div className="relative min-h-screen overflow-x-hidden bg-stone-50 text-slate-700">
      <Header />

      <main id="main-content" className="mx-auto max-w-5xl px-4 py-8 sm:px-6 lg:py-12">
        <LegalNav />

        <div className="mb-10">
          <h1 className="font-display text-3xl font-normal tracking-tight text-stone-900 sm:text-4xl">
            Security &amp; Responsible Disclosure
          </h1>
          <p className="mt-2 text-base leading-relaxed text-stone-600">
            How to report vulnerabilities, our safe harbor commitment, and coordinated disclosure timeline.
          </p>
        </div>

        <div className="flex flex-col gap-8 text-sm leading-relaxed text-stone-700">
          <section aria-labelledby="report-heading">
            <h2 id="report-heading" className="text-lg font-semibold text-stone-900">
              Report a vulnerability
            </h2>
            <p className="mt-2">
              Email{' '}
              <a href={`mailto:${SECURITY_CONTACT_EMAIL}`} className="text-accent underline underline-offset-2">
                {SECURITY_CONTACT_EMAIL}
              </a>{' '}
              with reproduction steps, affected components, and any safe proof-of-concept. We acknowledge valid reports
              within <strong>2 business days</strong>.
            </p>
            <p className="mt-2">
              <strong>PGP fingerprint:</strong>{' '}
              <code className="rounded bg-stone-100 px-1.5 py-0.5 text-xs">{PGP_FINGERPRINT}</code>
            </p>
            <p className="mt-2">
              <a href={PGP_KEY_URL} className="text-accent underline underline-offset-2" rel="noopener noreferrer" target="_blank">
                Download public key on keys.openpgp.org
              </a>
            </p>
            <p className="mt-2 text-stone-500">
              Repository policy: see{' '}
              <a
                href="https://github.com/StudyDrift/lextures/blob/main/SECURITY.md"
                className="text-accent underline underline-offset-2"
                rel="noopener noreferrer"
                target="_blank"
              >
                SECURITY.md
              </a>{' '}
              at the repo root.
            </p>
          </section>

          <section aria-labelledby="safe-harbor-heading">
            <h2 id="safe-harbor-heading" className="text-lg font-semibold text-stone-900">
              Safe harbor
            </h2>
            <p className="mt-2">
              Good-faith security research conducted within the scope below will not result in legal action from Lextures,
              provided you avoid privacy violations, service disruption, and access to other users&apos; data beyond what is
              necessary to demonstrate the issue.
            </p>
          </section>

          <section aria-labelledby="scope-heading">
            <h2 id="scope-heading" className="text-lg font-semibold text-stone-900">
              Scope
            </h2>
            <h3 className="mt-3 font-medium text-stone-800">In scope</h3>
            <ul className="mt-1 list-disc space-y-1 ps-5">
              {IN_SCOPE_ITEMS.map((item) => (
                <li key={item}>{item}</li>
              ))}
            </ul>
            <h3 className="mt-4 font-medium text-stone-800">Out of scope</h3>
            <ul className="mt-1 list-disc space-y-1 ps-5">
              {OUT_OF_SCOPE_ITEMS.map((item) => (
                <li key={item}>{item}</li>
              ))}
            </ul>
          </section>

          <section aria-labelledby="disclosure-heading">
            <h2 id="disclosure-heading" className="text-lg font-semibold text-stone-900">
              Coordinated disclosure
            </h2>
            <p className="mt-2">
              We use a <strong>{COORDINATED_DISCLOSURE_DAYS}-day</strong> coordinated disclosure window from acknowledgment,
              aligned with industry practice (Google Project Zero / CERT/CC). Critical issues may be patched sooner with
              reporter agreement.
            </p>
          </section>

          <section aria-labelledby="sla-heading">
            <h2 id="sla-heading" className="text-lg font-semibold text-stone-900">
              Patch targets (CVSS 3.1)
            </h2>
            <div className="mt-3 overflow-x-auto">
              <table className="min-w-full border-collapse text-sm" aria-label="Patch SLA by severity">
                <thead>
                  <tr className="border-b border-stone-200">
                    <th scope="col" className="py-2 pe-4 text-start font-semibold">Severity</th>
                    <th scope="col" className="py-2 text-start font-semibold">Target</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-stone-100">
                  {PATCH_SLA_ROWS.map((row) => (
                    <tr key={row.severity}>
                      <td className="py-2 pe-4 font-medium text-stone-900">{row.severity}</td>
                      <td className="py-2">{row.days}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </section>

          <section aria-labelledby="bounty-heading">
            <h2 id="bounty-heading" className="text-lg font-semibold text-stone-900">
              Bug bounty
            </h2>
            <p className="mt-2">
              Invite-only rewards may be offered for valid critical and high findings. Program details will be posted here
              when a HackerOne or Bugcrowd program launches.
            </p>
          </section>
        </div>
      </main>

      <SiteFooter />
    </div>
  )
}
