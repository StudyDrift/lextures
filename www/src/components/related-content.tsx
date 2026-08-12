import { relatedRoutes } from '../lib/information-architecture'

export function RelatedContent({ path }: { path: string }) {
  const related = relatedRoutes(path)
  return <aside data-related-content aria-labelledby="related-heading" className="mt-12 border-t pt-8" style={{ borderColor: 'var(--line)' }}>
    <h2 id="related-heading" className="font-display text-2xl font-semibold" style={{ color: 'var(--ink-nav)' }}>Related resources</h2>
    <ul className="mt-4 grid gap-3 sm:grid-cols-2">
      {related.map(item => <li key={item.path}><a href={item.path}>{item.label}</a></li>)}
    </ul>
  </aside>
}
