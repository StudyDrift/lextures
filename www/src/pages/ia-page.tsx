import { AudienceCta, AudienceHero, MarketingPageShell } from '../components/marketing-page-shell'
import { INTEGRATIONS } from '../lib/integrations'
import { COMPETITORS } from '../lib/competitors'
import { useSsrData } from '../lib/ssr-context'
import { GLOSSARY_TERMS, RESOURCE_CONTENT } from './ia/resource-content'
import { PLATFORM_CONTENT } from './ia/platform-content'
import type { ContentCard, IaPageContent } from './ia/ia-content-types'

function FeatureCards({ items }: { items: ContentCard[] }) {
  return <div className="mt-9 grid gap-5 md:grid-cols-2">
    {items.map(item => <article key={item.title} className="rounded-[18px] border p-6 shadow-[0_10px_26px_rgba(34,51,59,0.04)]" style={{ borderColor: 'var(--line-card)', background: 'var(--panel)' }}>
      <h3 className="font-display text-xl font-semibold" style={{ color: 'var(--ink-nav)' }}>{item.title}</h3>
      <p className="mt-3 leading-relaxed">{item.body}</p>
      {item.href && <a className="mt-4 inline-block font-semibold" href={item.href}>{item.linkLabel ?? 'Learn more'} <span aria-hidden>→</span></a>}
    </article>)}
  </div>
}

function Steps({ items }: { items: ContentCard[] }) {
  return <ol className="mt-9 grid gap-7 md:grid-cols-3">
    {items.map(item => <li key={item.title} className="border-t-2 pt-5" style={{ borderColor: 'var(--teal-solid)' }}>
      <h3 className="font-display text-xl font-semibold" style={{ color: 'var(--ink-nav)' }}>{item.title}</h3>
      <p className="mt-3 leading-relaxed">{item.body}</p>
    </li>)}
  </ol>
}

function Glossary() {
  const groups = GLOSSARY_TERMS.reduce<Record<string, typeof GLOSSARY_TERMS>>((result, item) => {
    const letter = item.term.charAt(0)
    ;(result[letter] ??= []).push(item)
    return result
  }, {})
  return <section id="terms" className="scroll-mt-24 py-16 md:py-20" aria-labelledby="glossary-heading">
    <div className="mx-auto max-w-[960px] px-5 md:px-10 xl:px-14">
      <p className="text-sm font-semibold uppercase tracking-[0.06em]" style={{ color: 'var(--coral)' }}>24 essential terms</p>
      <h2 id="glossary-heading" className="font-display mt-2 text-3xl font-semibold" style={{ color: 'var(--ink-nav)' }}>Glossary entries</h2>
      <nav className="mt-6 flex flex-wrap gap-2" aria-label="Glossary letters">
        {Object.keys(groups).map(letter => <a key={letter} href={`#letter-${letter}`} className="flex h-10 w-10 items-center justify-center rounded-full border font-semibold no-underline" style={{ borderColor: 'var(--line-card)', background: 'var(--panel)' }}>{letter}</a>)}
      </nav>
      <div className="mt-10 space-y-10">
        {Object.entries(groups).map(([letter, terms]) => <section id={`letter-${letter}`} className="scroll-mt-24" key={letter} aria-labelledby={`letter-${letter}-heading`}>
          <h3 id={`letter-${letter}-heading`} className="font-display border-b pb-3 text-3xl font-semibold" style={{ color: 'var(--teal-deep)', borderColor: 'var(--line-card)' }}>{letter}</h3>
          <dl className="divide-y" style={{ borderColor: 'var(--line-card)' }}>
            {terms.map(item => <div key={item.term} className="grid gap-2 py-6 md:grid-cols-[220px_1fr] md:gap-8">
              <dt className="font-display text-lg font-semibold" style={{ color: 'var(--ink-nav)' }}>{item.term}</dt>
              <dd><p className="leading-relaxed">{item.definition}</p><p className="mt-2 text-sm"><strong>Related:</strong> {item.related}</p></dd>
            </div>)}
          </dl>
        </section>)}
      </div>
    </div>
  </section>
}

function IntegrationDirectory() {
  return <section className="py-16 md:py-20" aria-labelledby="integration-directory">
    <div className="mx-auto max-w-[960px] px-5 md:px-10 xl:px-14">
      <h2 id="integration-directory" className="font-display text-3xl font-semibold" style={{ color: 'var(--ink-nav)' }}>Integration directory</h2>
      <p className="mt-3 max-w-[680px] leading-relaxed">Review supported capabilities, setup effort, limitations, and implementation guidance for each connection.</p>
      <div className="mt-8 grid gap-4 md:grid-cols-2">
        {INTEGRATIONS.map(item => <a key={item.slug} href={`/integrations/${item.slug}`} className="rounded-[16px] border p-5 no-underline transition-transform hover:-translate-y-0.5" style={{ borderColor: 'var(--line-card)', background: 'var(--panel)', color: 'var(--ink-nav)' }}>
          <strong className="font-display text-lg">{item.name}</strong><span className="float-right" aria-hidden>→</span>
        </a>)}
      </div>
    </div>
  </section>
}

function ComparisonDirectory({ alternatives }: { alternatives: boolean }) {
  return <section className="py-16 md:py-20" aria-labelledby="comparison-directory">
    <div className="mx-auto max-w-[960px] px-5 md:px-10 xl:px-14">
      <h2 id="comparison-directory" className="font-display text-3xl font-semibold" style={{ color: 'var(--ink-nav)' }}>{alternatives ? 'Alternative guides' : 'Platform comparisons'}</h2>
      <p className="mt-3 max-w-[680px] leading-relaxed">Review evidence-led criteria, limitations, hosting, accessibility, integrations, and cost before choosing a learning platform.</p>
      <div className="mt-8 grid gap-4 md:grid-cols-2">
        {COMPETITORS.map(item => <a key={item.slug} href={alternatives ? `/alternatives/${item.slug}` : `/compare/lextures-vs-${item.slug}`} className="rounded-[16px] border p-5 no-underline transition-transform hover:-translate-y-0.5" style={{ borderColor: 'var(--line-card)', background: 'var(--panel)', color: 'var(--ink-nav)' }}>
          <strong className="font-display text-lg">{alternatives ? `Alternatives to ${item.name}` : `Lextures vs ${item.name}`}</strong><span className="float-right" aria-hidden>→</span>
        </a>)}
      </div>
    </div>
  </section>
}

function RichIaPage({ content, path }: { content: IaPageContent; path: string }) {
  return <MarketingPageShell>
    <AudienceHero eyebrow={content.eyebrow} title={content.title} lead={content.lead} primaryHref={content.primaryHref} primaryLabel={content.primaryLabel} secondaryHref={content.secondaryHref} secondaryLabel={content.secondaryLabel} />

    <section className="py-14 md:py-16" aria-labelledby="direct-answer">
      <div className="mx-auto max-w-[760px] px-5 md:px-10">
        <p className="text-sm font-semibold uppercase tracking-[0.06em]" style={{ color: 'var(--coral)' }}>The short answer</p>
        <h2 id="direct-answer" className="font-display mt-2 text-3xl font-semibold leading-tight" style={{ color: 'var(--ink-nav)' }}>{content.answerTitle}</h2>
        <p className="mt-5 text-lg leading-relaxed">{content.answer}</p>
      </div>
    </section>

    {path === '/glossary' && <Glossary />}

    {path === '/integrations' && <IntegrationDirectory />}

    <section id={path === '/templates' ? 'template-library' : undefined} className="scroll-mt-24 py-16 md:py-20" style={{ background: 'var(--panel-sunken)' }} aria-labelledby="capabilities-heading">
      <div className="mx-auto max-w-[960px] px-5 md:px-10 xl:px-14">
        <h2 id="capabilities-heading" className="font-display text-3xl font-semibold" style={{ color: 'var(--ink-nav)' }}>{content.cardTitle}</h2>
        <p className="mt-3 max-w-[680px] leading-relaxed">{content.cardLead}</p>
        <FeatureCards items={content.cards} />
      </div>
    </section>

    <section className="py-16 md:py-20" aria-labelledby="workflow-heading">
      <div className="mx-auto max-w-[960px] px-5 md:px-10 xl:px-14">
        <h2 id="workflow-heading" className="font-display text-3xl font-semibold" style={{ color: 'var(--ink-nav)' }}>{content.workflowTitle}</h2>
        <p className="mt-3 max-w-[680px] leading-relaxed">{content.workflowLead}</p>
        <Steps items={content.steps} />
      </div>
    </section>

    {content.questions?.length && <section className="py-14" style={{ background: 'var(--deep)', color: '#fff' }} aria-labelledby="questions-heading">
      <div className="mx-auto grid max-w-[960px] gap-8 px-5 md:grid-cols-[0.8fr_1.2fr] md:px-10 xl:px-14">
        <h2 id="questions-heading" className="font-display text-3xl font-semibold leading-tight">{content.questionsTitle}</h2>
        <ul className="space-y-4">
          {content.questions.map(question => <li key={question} className="flex gap-3 leading-relaxed"><span aria-hidden style={{ color: 'var(--teal)' }}>●</span><span>{question}</span></li>)}
        </ul>
      </div>
    </section>}

    <section className="py-16 md:py-20" aria-labelledby="faq-heading">
      <div className="mx-auto max-w-[760px] px-5 md:px-10">
        <h2 id="faq-heading" className="font-display text-3xl font-semibold" style={{ color: 'var(--ink-nav)' }}>Frequently asked questions</h2>
        <div className="mt-7 divide-y" style={{ borderColor: 'var(--line-card)' }}>
          {content.faq.map(item => <details key={item.question} className="group py-5">
            <summary className="cursor-pointer list-none font-display text-lg font-semibold" style={{ color: 'var(--ink-nav)' }}>{item.question} <span className="float-right" aria-hidden>+</span></summary>
            <p className="mt-3 max-w-[680px] leading-relaxed">{item.answer}</p>
          </details>)}
        </div>
      </div>
    </section>

    <section className="pb-16" aria-labelledby="sources-heading">
      <div className="mx-auto max-w-[760px] rounded-[18px] border px-6 py-7 md:px-8" style={{ borderColor: 'var(--line-card)', background: 'var(--panel)' }}>
        <h2 id="sources-heading" className="font-display text-2xl font-semibold" style={{ color: 'var(--ink-nav)' }}>Standards and source material</h2>
        <ul className="mt-5 space-y-4">
          {content.sources.map(source => <li key={source.href}><a className="font-semibold" href={source.href}>{source.label}</a><p className="mt-1 text-sm leading-relaxed">{source.note}</p></li>)}
        </ul>
      </div>
    </section>

    <AudienceCta title={content.ctaTitle} body={content.ctaBody} primaryHref={content.primaryHref} primaryLabel={content.primaryLabel} secondaryHref={content.secondaryHref} secondaryLabel={content.secondaryLabel} />
  </MarketingPageShell>
}

function GenericIaPage({ path }: { path: string }) {
  const fallback = path === '/trust'
    ? { title: 'Trust center', description: 'Review Lextures security, privacy, accessibility, and legal commitments.' }
    : path === '/compare'
      ? { title: 'Compare Lextures', description: 'Compare Lextures with other approaches to learning management and course delivery.' }
      : { title: path.split('/').filter(Boolean).at(-1)?.replace(/-/g, ' ') ?? 'Lextures', description: 'Explore the Lextures learning platform.' }
  return <MarketingPageShell>
    <AudienceHero eyebrow="Lextures" title={fallback.title} lead={fallback.description} primaryHref="/get-started" primaryLabel="Get started" secondaryHref="/pricing" secondaryLabel="See pricing" />
    <section className="mx-auto max-w-[960px] px-5 py-16 md:px-10 xl:px-14">
      <h2 className="font-display text-3xl font-semibold" style={{ color: 'var(--ink-nav)' }}>Built for connected learning</h2>
      <p className="mt-4 max-w-2xl leading-relaxed">Connect this capability with <a href="/platform">the Lextures platform</a>, review <a href="/docs">implementation documentation</a>, or explore <a href="/resources">learning resources</a>.</p>
    </section>
    {(path === '/compare' || path === '/alternatives') && <ComparisonDirectory alternatives={path === '/alternatives'} />}
  </MarketingPageShell>
}

export function IaPage() {
  const data = useSsrData()
  const path = data.path ?? (typeof window !== 'undefined' ? window.location.pathname : '/')
  const content = PLATFORM_CONTENT[path] ?? RESOURCE_CONTENT[path]
  return content ? <RichIaPage content={content} path={path} /> : <GenericIaPage path={path} />
}
