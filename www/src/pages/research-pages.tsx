import { MarketingPageShell } from '../components/marketing-page-shell'

const section = 'mx-auto max-w-[860px] px-5 py-12 md:px-10'

export function ResearchIndexPage() {
  return <MarketingPageShell>
    <main>
      <header className="mx-auto max-w-[960px] px-5 py-16 md:px-10 md:py-24">
        <p className="text-sm font-semibold uppercase tracking-[0.06em]" style={{ color: 'var(--coral)' }}>Original research</p>
        <h1 className="font-display mt-3 max-w-[760px] text-4xl font-semibold leading-tight md:text-6xl" style={{ color: 'var(--ink-nav)' }}>Evidence designed to be checked, reused, and cited</h1>
        <p className="mt-6 max-w-[720px] text-lg leading-relaxed">Lextures publishes aggregate education-technology research as permanent web pages with full methods, accessible charts, reusable datasets, and visible corrections. Reports are ungated and datasets use CC BY 4.0.</p>
      </header>
      <section className="py-14" style={{ background: 'var(--panel-sunken)' }} aria-labelledby="schedule-heading">
        <div className="mx-auto max-w-[960px] px-5 md:px-10">
          <h2 id="schedule-heading" className="font-display text-3xl font-semibold" style={{ color: 'var(--ink-nav)' }}>Publication schedule</h2>
          <div className="mt-7 grid gap-5 md:grid-cols-2">
            <article className="rounded-[18px] border p-6" style={{ background: 'var(--panel)', borderColor: 'var(--line-card)' }}><p className="text-sm font-semibold">December 2026 · preregistration in progress</p><h3 className="font-display mt-2 text-2xl font-semibold">The Adaptive Learning Outcomes Report</h3><p className="mt-3 leading-relaxed">Mastery, time-to-mastery, and retention, with assignment limits stated plainly. Analysis cannot begin until privacy and methodology approval.</p></article>
            <article className="rounded-[18px] border p-6" style={{ background: 'var(--panel)', borderColor: 'var(--line-card)' }}><p className="text-sm font-semibold">June 2027 · planned</p><h3 className="font-display mt-2 text-2xl font-semibold">How AI Is Actually Used in Assessment</h3><p className="mt-3 leading-relaxed">Descriptive, aggregate feature adoption and abandonment patterns by sufficiently large education segments.</p></article>
          </div>
        </div>
      </section>
      <section className={section} aria-labelledby="commitments-heading"><h2 id="commitments-heading" className="font-display text-3xl font-semibold">Our publication commitments</h2><ul className="mt-6 list-disc space-y-3 pl-6 leading-relaxed"><li>We preregister hypotheses and publish null or unfavorable findings.</li><li>No reported cell contains fewer than 50 learners or 10 institutions; complementary suppression prevents back-calculation.</li><li>Every claim maps to a stable figure in downloadable CSV and JSON data.</li><li>Prior versions and dated errata remain available at permanent URLs.</li></ul><p className="mt-7"><a className="font-semibold" href="/resources/research/methodology">Read the research methodology and privacy standard →</a></p></section>
      <section className={section} aria-labelledby="links-heading"><h2 id="links-heading" className="font-display text-3xl font-semibold">Use and scrutinize the work</h2><p className="mt-4 leading-relaxed">Datasets and chart images will be reusable with attribution. Review <a href="/trust">our trust commitments</a>, explore <a href="/resources/guides">evidence-led guides</a>, or <a href="/request-information">ask the research team a question</a>.</p></section>
    </main>
  </MarketingPageShell>
}

export function ResearchMethodologyPage() {
  return <MarketingPageShell><main>
    <header className={section}><p className="text-sm font-semibold uppercase tracking-[0.06em]" style={{ color: 'var(--coral)' }}>Research standard</p><h1 className="font-display mt-3 text-4xl font-semibold md:text-5xl" style={{ color: 'var(--ink-nav)' }}>Methodology, privacy, and corrections</h1><p className="mt-5 text-lg leading-relaxed">This standard applies to every Lextures original-research report. Report-specific pages add their exact population, dates, exclusions, methods, confidence intervals, limitations, and reviewers.</p></header>
    <section className={section} aria-labelledby="data-heading"><h2 id="data-heading" className="font-display text-3xl font-semibold">What data may be used</h2><p className="mt-4 leading-relaxed">Only records covered by an organization’s contract, jurisdictional decision, and explicit participation setting may enter a future extract. An unresolved setting is treated as excluded. Organization administrators can change the decision in Settings → Organizations; an opt-out excludes the organization from all future report extracts.</p><p className="mt-4 leading-relaxed">A reviewed pipeline removes direct identifiers before analysts receive data. Analysts work in a least-privilege, access-logged environment and do not receive names, email addresses, learner or instructor identifiers, course identifiers, school identifiers, or tenant identifiers.</p></section>
    <section className={section} aria-labelledby="privacy-heading"><h2 id="privacy-heading" className="font-display text-3xl font-semibold">Aggregation and suppression</h2><p className="mt-4 leading-relaxed">A public cell must include at least 50 learners and 10 institutions. Smaller cells are suppressed automatically. When a total could reveal one hidden cell by subtraction, another cell is also hidden. Segments remain coarse and no result may be attributable to one school, tenant, course, or instructor.</p></section>
    <section className={section} aria-labelledby="integrity-heading"><h2 id="integrity-heading" className="font-display text-3xl font-semibold">Analysis integrity</h2><p className="mt-4 leading-relaxed">Hypotheses and statistical methods are committed before outcomes are analyzed. A second person reviews analysis code. Null and unfavorable findings publish with the same prominence as favorable ones, and marketing wording cannot exceed what the dataset and methods support.</p></section>
    <section className={section} aria-labelledby="publication-heading"><h2 id="publication-heading" className="font-display text-3xl font-semibold">Publication and corrections</h2><p className="mt-4 leading-relaxed">Reports are ungated HTML with accessible chart tables. Aggregate data ships as CSV and JSON under <a href="https://creativecommons.org/licenses/by/4.0/">CC BY 4.0</a>. Each version has a citation block and history. Corrections create a dated erratum, preserve the prior version, and explain which figures changed; silent edits are prohibited.</p><p className="mt-6"><a className="font-semibold" href="/resources/research">Return to the research program →</a></p></section>
  </main></MarketingPageShell>
}

