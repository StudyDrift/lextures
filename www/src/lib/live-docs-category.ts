import { fetchPublishedArticleSummaries } from './public-content'

export async function initLiveDocsCategory(): Promise<void> {
  if (typeof document === 'undefined') return
  const root = document.querySelector<HTMLElement>('[data-docs-category]')
  if (!root) return
  const list = root.querySelector<HTMLElement>('[data-docs-article-list]')
  const category = root.dataset.docsCategory
  if (!list || !category) return
  let live
  try {
    live = await fetchPublishedArticleSummaries('doc', { category })
  } catch {
    return
  }
  const existing = new Set(
    [...list.querySelectorAll('a[href^="/docs/"]')].map(link => link.getAttribute('href') || ''),
  )
  for (const article of live) {
    const href = `/docs/${article.category || category}/${article.slug}`
    if (existing.has(href)) continue
    const card = document.createElement('article')
    card.className = 'rounded-xl border border-slate-200 p-5'
    const heading = document.createElement('h3')
    heading.className = 'text-lg font-semibold'
    const link = document.createElement('a')
    link.href = href
    link.textContent = article.title
    heading.append(link)
    const description = document.createElement('p')
    description.className = 'mt-2 text-sm leading-6 text-slate-600'
    description.textContent = article.description
    card.append(heading, description)
    list.append(card)
  }
}
