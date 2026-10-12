import { MarketingPageShell, AudienceHero } from '../components/marketing-page-shell'
import { EDITORIAL_PILLARS } from '../lib/editorial-pillars'
import { allPostMeta } from '../utils/blog'
import { publishedGuides } from '../lib/guides-data'

export function GuidesIndexPage() {
  return <MarketingPageShell>
    <AudienceHero eyebrow="Resources" title="Editorial guides" lead="Six concentrated learning-design topics, built from reviewed articles and primary evidence. A guide launches only after its cluster has enough useful coverage." primaryHref="/blog" primaryLabel="Read the articles" secondaryHref="/resources" secondaryLabel="All resources" />
    {publishedGuides().length > 0 && <section className="mx-auto max-w-[960px] px-5 pt-16 md:px-10 xl:px-14" aria-labelledby="published-guides">
      <h2 id="published-guides" className="font-display text-3xl font-semibold" style={{ color: 'var(--ink-nav)' }}>Published guides</h2>
      <ul className="mt-6 grid gap-3">
        {publishedGuides().map(guide => <li key={guide.slug}><a className="font-semibold" href={guide.path}>{guide.headline}</a></li>)}
      </ul>
    </section>}
    <section className="mx-auto max-w-[960px] px-5 py-16 md:px-10 xl:px-14" aria-labelledby="guide-clusters">
      <h2 id="guide-clusters" className="font-display text-3xl font-semibold" style={{ color: 'var(--ink-nav)' }}>The six editorial pillars</h2>
      <ul className="mt-8 grid gap-5 md:grid-cols-2">
        {EDITORIAL_PILLARS.map(pillar => {
          const count = allPostMeta.filter(post => post.pillar === pillar.id).length
          return <li id={pillar.id} key={pillar.id} className="scroll-mt-24 rounded-[18px] border p-6" style={{ borderColor: 'var(--line-card)', background: 'var(--panel)' }}>
            <p className="text-sm font-semibold uppercase tracking-wide">{pillar.id} · {pillar.audience}</p>
            <h3 className="font-display mt-2 text-xl font-semibold">{pillar.title}</h3>
            <p className="mt-3 leading-relaxed">{count} reviewed {count === 1 ? 'article' : 'articles'} published toward a {pillar.targetArticles}-article cluster.</p>
            <a className="mt-4 inline-block font-semibold" href={`/blog?pillar=${pillar.id}`}>Browse this cluster</a>
          </li>
        })}
      </ul>
    </section>
  </MarketingPageShell>
}
