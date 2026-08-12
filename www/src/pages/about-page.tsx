import { MarketingPageShell } from '../components/marketing-page-shell'
import { SITE_LINKS } from '../lib/site-links'
import { BRAND, VERIFIED_SAME_AS } from '../lib/schema/entity'
import { getActiveAuthors } from '../lib/authors'

/**
 * Entity home for Lextures (SEO.3 FR-18).
 * Canonical URL for Wikidata / directory official website.
 */
export function AboutPage() {
  const authors = getActiveAuthors()

  return (
    <MarketingPageShell>
      <main id="main-content">
        <section className="border-b border-slate-200 bg-white py-16 sm:py-20">
          <div className="mx-auto max-w-3xl px-4 sm:px-6 lg:px-8">
            <p className="text-[0.7rem] font-semibold uppercase tracking-[0.22em] text-teal-700">
              About
            </p>
            <h1 className="font-display mt-4 text-4xl font-semibold tracking-tight text-slate-900 sm:text-5xl">
              Who builds Lextures
            </h1>
            <p className="mt-5 text-lg leading-relaxed text-slate-600">
              {BRAND.description}
            </p>
          </div>
        </section>

        <section className="py-14 sm:py-16">
          <div className="mx-auto max-w-3xl space-y-10 px-4 sm:px-6 lg:px-8">
            <div>
              <h2 className="font-display text-2xl font-semibold text-slate-900">
                What Lextures is
              </h2>
              <p className="mt-3 text-[16px] leading-relaxed text-slate-600">
                Lextures is an adaptive learning management system for K–12 districts, higher
                education, and homeschool families. One platform covers IRT-routed quizzes, spaced
                repetition, gradebook with audit trails, roster sync, LTI 1.3, and a public course
                marketplace — instead of a patchwork of single-purpose tools.
              </p>
              <p className="mt-3 text-[16px] leading-relaxed text-slate-600">
                The software is open source under AGPL-3.0. Institutions can self-host on their own
                Postgres stack or run a hosted account. Homeschool learners can use{' '}
                <a href={SITE_LINKS.homeschool} className="text-teal-800 underline underline-offset-2">
                  self.lextures.com
                </a>{' '}
                or self-host for free.
              </p>
            </div>

            <div>
              <h2 className="font-display text-2xl font-semibold text-slate-900">
                Mission
              </h2>
              <p className="mt-3 text-[16px] leading-relaxed text-slate-600">
                Make adaptive practice, fair assessment, and accessible learning infrastructure
                available to every school and family — with transparent pricing, real accessibility
                documentation, and code you can inspect and run yourself.
              </p>
            </div>

            <div>
              <h2 className="font-display text-2xl font-semibold text-slate-900">
                Founding
              </h2>
              <dl className="mt-3 grid gap-3 text-[15px] text-slate-600 sm:grid-cols-2">
                <div>
                  <dt className="font-medium text-slate-900">Legal name</dt>
                  <dd>{BRAND.legalName}</dd>
                </div>
                <div>
                  <dt className="font-medium text-slate-900">Founded</dt>
                  <dd>
                    <time dateTime={BRAND.foundingDate}>
                      {new Date(BRAND.foundingDate + 'T00:00:00').toLocaleDateString('en-US', {
                        year: 'numeric',
                        month: 'long',
                      })}
                    </time>
                  </dd>
                </div>
                <div>
                  <dt className="font-medium text-slate-900">Founder</dt>
                  <dd>
                    <a
                      href="/authors/chase-willden"
                      className="text-teal-800 underline underline-offset-2"
                    >
                      Chase Willden
                    </a>
                  </dd>
                </div>
                <div>
                  <dt className="font-medium text-slate-900">Ownership</dt>
                  <dd>Independently owned; no outside institutional investors disclosed.</dd>
                </div>
              </dl>
            </div>

            <div>
              <h2 className="font-display text-2xl font-semibold text-slate-900">
                People
              </h2>
              <ul className="mt-4 space-y-4">
                {authors.map(a => (
                  <li key={a.slug} className="rounded-xl border border-slate-200 bg-slate-50/80 p-4">
                    <a
                      href={`/authors/${a.slug}`}
                      className="font-semibold text-slate-900 no-underline hover:underline"
                    >
                      {a.name}
                    </a>
                    <p className="text-sm text-slate-500">{a.jobTitle}</p>
                    <p className="mt-2 text-sm leading-relaxed text-slate-600">{a.bio}</p>
                  </li>
                ))}
              </ul>
              <p className="mt-3 text-sm">
                <a href="/authors" className="text-teal-800 underline underline-offset-2">
                  All authors
                </a>
              </p>
            </div>

            <div>
              <h2 className="font-display text-2xl font-semibold text-slate-900">
                Profiles & contact
              </h2>
              <p className="mt-3 text-[15px] leading-relaxed text-slate-600">
                Official profiles we own or claim (also listed in Organization schema{' '}
                <code className="text-sm">sameAs</code>):
              </p>
              <ul className="mt-3 list-disc space-y-1 pl-5 text-[15px] text-slate-700">
                {VERIFIED_SAME_AS.map(url => (
                  <li key={url}>
                    <a
                      href={url}
                      className="text-teal-800 underline underline-offset-2"
                      target="_blank"
                      rel="noopener noreferrer"
                    >
                      {url.replace(/^https?:\/\//, '')}
                    </a>
                  </li>
                ))}
              </ul>
              <p className="mt-4 text-[15px] text-slate-600">
                Press:{' '}
                <a href={SITE_LINKS.press} className="text-teal-800 underline underline-offset-2">
                  media resources
                </a>
                .{' '}
                Sales:{' '}
                <a
                  href={`mailto:${SITE_LINKS.institutionInquiryEmail}`}
                  className="text-teal-800 underline underline-offset-2"
                >
                  {SITE_LINKS.institutionInquiryEmail}
                </a>
                . Security:{' '}
                <a href={SITE_LINKS.security} className="text-teal-800 underline underline-offset-2">
                  security page
                </a>
                . Accessibility:{' '}
                <a
                  href={SITE_LINKS.accessibility}
                  className="text-teal-800 underline underline-offset-2"
                >
                  conformance statement
                </a>
                . Request a demo:{' '}
                <a href="/request-information" className="text-teal-800 underline underline-offset-2">
                  request information
                </a>
                .
              </p>
            </div>
          </div>
        </section>
      </main>
    </MarketingPageShell>
  )
}
