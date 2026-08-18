import { authorDisplayName } from './authors'
import { applyBlogIndexFilters, blogPostSearchText } from './blog-index-filters'
import { editorialPillar } from './editorial-pillars'
import { fetchPublishedArticleSummaries } from './public-content'
import { formatDate } from '../utils/blog'

function cardSearchText(post: { title: string; description: string; slug: string; author: string; pillar?: string }): string {
  return blogPostSearchText([
    post.title,
    post.description,
    post.slug,
    post.author,
    authorDisplayName(post.author),
    editorialPillar(post.pillar || '')?.title,
    post.pillar ? `pillar:${post.pillar}` : '',
    post.author ? `author:${post.author}` : '',
  ])
}

function buildCard(post: {
  slug: string
  title: string
  description: string
  date: string
  author: string
  pillar?: string
}): HTMLElement {
  const article = document.createElement('article')
  article.dataset.blogPost = ''
  article.dataset.search = cardSearchText(post)
  article.className = 'group py-10 first:pt-0'
  const pillarTitle = editorialPillar(post.pillar || '')?.title || ''
  article.innerHTML = `
    <div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between sm:gap-8">
      <div class="flex-1">
        <time class="text-xs font-medium uppercase tracking-widest text-slate-400"></time>
        <h2 class="mt-2 text-xl font-semibold leading-snug text-slate-900 sm:text-2xl">
          <a class="no-underline transition-colors hover:text-accent"></a>
        </h2>
        <p class="mt-3 max-w-2xl text-base leading-relaxed text-slate-600"></p>
        <p class="mt-2 text-sm text-slate-400"></p>
      </div>
      <a class="btn-primary shrink-0 gap-2 self-start"></a>
    </div>`
  const href = `/blog/${post.slug}`
  const time = article.querySelector('time')
  if (time) {
    time.dateTime = post.date
    time.textContent = formatDate(post.date)
  }
  const titleLink = article.querySelector('h2 a')
  if (titleLink) {
    titleLink.setAttribute('href', href)
    titleLink.textContent = post.title
  }
  const description = article.querySelector('p.mt-3')
  if (description) description.textContent = post.description
  const byline = article.querySelector('p.mt-2')
  if (byline) byline.textContent = `By ${authorDisplayName(post.author)}${pillarTitle ? ` · ${pillarTitle}` : ''}`
  const read = article.querySelector('a.btn-primary')
  if (read) {
    read.setAttribute('href', href)
    read.setAttribute('aria-label', `Read ${post.title}`)
    read.textContent = 'Read'
  }
  return article
}

function ensureAuthorOption(select: HTMLSelectElement | null, slug: string): void {
  if (!select || !slug) return
  if ([...select.options].some(option => option.value === slug)) return
  const option = document.createElement('option')
  option.value = slug
  option.textContent = authorDisplayName(slug)
  select.append(option)
}

export async function initLiveBlogIndex(): Promise<void> {
  if (typeof document === 'undefined') return
  const root = document.querySelector<HTMLElement>('[data-blog-index]')
  if (!root) return
  const list = root.querySelector<HTMLElement>('[data-blog-list]')
  if (!list) return
  let live
  try {
    live = await fetchPublishedArticleSummaries('blog')
  } catch {
    return
  }
  const existing = new Set([...root.querySelectorAll<HTMLElement>('[data-blog-post]')].map(node => {
    const href = node.querySelector('a[href^="/blog/"]')?.getAttribute('href') || ''
    return href.replace(/^\/blog\//, '')
  }))
  const authorSelect = root.querySelector<HTMLSelectElement>('[data-blog-author]')
  let added = 0
  for (const post of live) {
    ensureAuthorOption(authorSelect, post.author)
    if (existing.has(post.slug)) continue
    list.append(buildCard(post))
    added += 1
  }
  if (added === 0) return
  const cards = [...list.querySelectorAll<HTMLElement>('[data-blog-post]')]
  cards.sort((a, b) => (b.querySelector('time')?.dateTime || '').localeCompare(a.querySelector('time')?.dateTime || ''))
  for (const card of cards) list.append(card)
  applyBlogIndexFilters(root, 1)
}
