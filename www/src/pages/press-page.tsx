import { MarketingPageShell } from '../components/marketing-page-shell'
import { SITE_LINKS } from '../lib/site-links'
import { BRAND, SOFTWARE_FEATURES } from '../lib/schema/entity'

const founded = new Date(`${BRAND.foundingDate}T00:00:00`).toLocaleDateString('en-US', {
  year: 'numeric',
  month: 'long',
})

export function PressPage() {
  return (
    <MarketingPageShell>
      <main id="main-content">
        <section className="border-b border-slate-200 bg-white py-16 sm:py-20">
          <div className="mx-auto max-w-4xl px-5 md:px-10">
            <p className="text-[0.7rem] font-semibold uppercase tracking-[0.22em] text-teal-700">Press room</p>
            <h1 className="font-display mt-4 text-4xl font-semibold tracking-tight text-slate-900 sm:text-5xl">Lextures press & media resources</h1>
            <p className="mt-5 max-w-3xl text-lg leading-relaxed text-slate-600">Company facts, approved language, brand files, and a direct contact for journalists. For a deadline or interview request, email <a className="text-teal-800 underline underline-offset-2" href={`mailto:${BRAND.pressEmail}`}>{BRAND.pressEmail}</a>.</p>
          </div>
        </section>

        <div className="mx-auto max-w-4xl space-y-14 px-5 py-14 md:px-10 md:py-16">
          <section aria-labelledby="boilerplate-heading">
            <h2 id="boilerplate-heading" className="font-display text-2xl font-semibold text-slate-900">Approved boilerplate</h2>
            <p className="mt-4 rounded-xl border border-slate-200 bg-slate-50 p-5 text-[16px] leading-relaxed text-slate-700">{BRAND.description}</p>
          </section>

          <section aria-labelledby="facts-heading">
            <h2 id="facts-heading" className="font-display text-2xl font-semibold text-slate-900">Fact sheet</h2>
            <div className="mt-4 overflow-x-auto rounded-xl border border-slate-200">
              <table className="w-full min-w-[34rem] border-collapse text-left text-sm">
                <caption className="sr-only">Lextures company facts</caption>
                <tbody className="divide-y divide-slate-200">
                  <tr><th className="w-44 bg-slate-50 px-4 py-3 font-semibold text-slate-900" scope="row">Company</th><td className="px-4 py-3 text-slate-700">{BRAND.legalName}</td></tr>
                  <tr><th className="bg-slate-50 px-4 py-3 font-semibold text-slate-900" scope="row">Founded</th><td className="px-4 py-3 text-slate-700">{founded}</td></tr>
                  <tr><th className="bg-slate-50 px-4 py-3 font-semibold text-slate-900" scope="row">Founder</th><td className="px-4 py-3 text-slate-700">Chase Willden</td></tr>
                  <tr><th className="bg-slate-50 px-4 py-3 font-semibold text-slate-900" scope="row">Product</th><td className="px-4 py-3 text-slate-700">Adaptive learning management system</td></tr>
                  <tr><th className="bg-slate-50 px-4 py-3 font-semibold text-slate-900" scope="row">Audiences</th><td className="px-4 py-3 text-slate-700">K–12, higher education, and homeschool</td></tr>
                  <tr><th className="bg-slate-50 px-4 py-3 font-semibold text-slate-900" scope="row">Availability</th><td className="px-4 py-3 text-slate-700">Hosted service or self-hosted under AGPL-3.0</td></tr>
                  <tr><th className="bg-slate-50 px-4 py-3 font-semibold text-slate-900" scope="row">Website</th><td className="px-4 py-3 text-slate-700"><a className="text-teal-800 underline underline-offset-2" href={BRAND.url}>{BRAND.url}</a></td></tr>
                </tbody>
              </table>
            </div>
            <h3 className="mt-7 font-display text-xl font-semibold text-slate-900">Product capabilities</h3>
            <ul className="mt-3 grid list-disc gap-x-8 gap-y-2 pl-5 text-sm leading-relaxed text-slate-700 sm:grid-cols-2">{SOFTWARE_FEATURES.map(feature => <li key={feature}>{feature}</li>)}</ul>
          </section>

          <section aria-labelledby="founder-heading">
            <h2 id="founder-heading" className="font-display text-2xl font-semibold text-slate-900">Founder</h2>
            <p className="mt-4 text-[16px] leading-relaxed text-slate-700">Chase Willden is the founder of Lextures. He builds adaptive learning systems using Item Response Theory, spaced repetition, and open-source LMS infrastructure for schools and homeschool families.</p>
            <p className="mt-3 text-sm text-slate-600"><a className="text-teal-800 underline underline-offset-2" href="/authors/chase-willden">Full biography and published work</a>. An approved high-resolution founder photograph is available from the press contact; no substitute or generated portrait should be used.</p>
          </section>

          <section aria-labelledby="assets-heading">
            <h2 id="assets-heading" className="font-display text-2xl font-semibold text-slate-900">Brand assets</h2>
            <p className="mt-4 text-[16px] leading-relaxed text-slate-700">Use the Lextures name and logo without altering proportions, colors, or clear space. Both files depict the Lextures sail mark and wordmark.</p>
            <div className="mt-5 flex flex-wrap gap-3">
              <a className="btn-secondary" href="/logo.svg" download="lextures-logo.svg">Download SVG logo</a>
              <a className="btn-secondary" href="/press/lextures-logo.png" download>Download PNG logo</a>
              <a className="btn-secondary" href="/press/lextures-fact-sheet.txt" download>Download fact sheet</a>
            </div>
          </section>

          <section aria-labelledby="coverage-heading">
            <h2 id="coverage-heading" className="font-display text-2xl font-semibold text-slate-900">Coverage & research</h2>
            <p className="mt-4 text-[16px] leading-relaxed text-slate-700">We do not publish an “as seen in” list until independent coverage exists. Journalists can review our <a className="text-teal-800 underline underline-offset-2" href="/resources/research">research and datasets</a>, including methodology, downloadable evidence, and correction history.</p>
          </section>

          <section className="rounded-2xl bg-slate-900 p-6 text-white sm:p-8" aria-labelledby="contact-heading">
            <h2 id="contact-heading" className="font-display text-2xl font-semibold">Media contact</h2>
            <p className="mt-3 leading-relaxed text-slate-200">Interview requests, fact checks, research questions, and accessible asset formats:</p>
            <p className="mt-3"><a className="font-semibold text-white underline underline-offset-2" href={`mailto:${SITE_LINKS.pressEmail}`}>{SITE_LINKS.pressEmail}</a></p>
          </section>
        </div>
      </main>
    </MarketingPageShell>
  )
}
