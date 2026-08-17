/**
 * Blog listing filters for the static-island (no React on /blog).
 * Search/select markup is emitted by BlogIndex; this module drives it.
 */

export const BLOG_PAGE_SIZE = 10

export type BlogListingFilter = {
  query: string
  pillar: string
  author: string
}

export function normalizeBlogQuery(value: string): string {
  return value.toLowerCase().trim()
}

export function blogPostSearchText(parts: Array<string | undefined>): string {
  return parts
    .flatMap(part => {
      const value = String(part || '').toLowerCase().trim()
      if (!value) return []
      return value.includes('-') ? [value, value.replaceAll('-', ' ')] : [value]
    })
    .join(' ')
    .replace(/\s+/g, ' ')
    .trim()
}

export function blogPostMatches(haystack: string, filter: BlogListingFilter): boolean {
  if (filter.pillar && !haystack.includes(`pillar:${filter.pillar}`)) return false
  if (filter.author && !haystack.includes(`author:${filter.author}`)) return false
  const query = normalizeBlogQuery(filter.query)
  if (!query) return true
  const searchable = haystack
    .replace(/\bpillar:[^\s]+/g, ' ')
    .replace(/\bauthor:[^\s]+/g, ' ')
  return searchable.includes(query)
}

export function paginateBlogPosts<T>(items: T[], page: number, pageSize = BLOG_PAGE_SIZE): T[] {
  const safePage = Math.max(1, page)
  const start = (safePage - 1) * pageSize
  return items.slice(start, start + pageSize)
}

function readFilter(root: HTMLElement): BlogListingFilter {
  const search = root.querySelector<HTMLInputElement>('[data-blog-search]')
  const pillar = root.querySelector<HTMLSelectElement>('[data-blog-pillar]')
  const author = root.querySelector<HTMLSelectElement>('[data-blog-author]')
  return {
    query: search?.value || '',
    pillar: pillar?.value || '',
    author: author?.value || '',
  }
}

function syncUrl(filter: BlogListingFilter): void {
  const params = new URLSearchParams(window.location.search)
  const setOrDelete = (key: string, value: string) => {
    if (value) params.set(key, value)
    else params.delete(key)
  }
  setOrDelete('q', normalizeBlogQuery(filter.query))
  setOrDelete('pillar', filter.pillar)
  setOrDelete('author', filter.author)
  const next = params.toString()
  const url = `${window.location.pathname}${next ? `?${next}` : ''}${window.location.hash}`
  window.history.replaceState(null, '', url)
}

export function applyBlogIndexFilters(root: HTMLElement, page = 1): number {
  const filter = readFilter(root)
  const pageSize = Number(root.dataset.pageSize || BLOG_PAGE_SIZE) || BLOG_PAGE_SIZE
  const posts = [...root.querySelectorAll<HTMLElement>('[data-blog-post]')]
  const matched = posts.filter(post => blogPostMatches(post.dataset.search || '', filter))
  const visible = paginateBlogPosts(matched, page, pageSize)
  const visibleSet = new Set(visible)
  for (const post of posts) post.hidden = !visibleSet.has(post)

  const empty = root.querySelector<HTMLElement>('[data-blog-empty]')
  const list = root.querySelector<HTMLElement>('[data-blog-list]')
  if (empty) empty.hidden = matched.length > 0
  if (list) list.hidden = matched.length === 0

  const status = root.querySelector<HTMLElement>('[data-blog-status]')
  if (status) {
    status.textContent = filter.query || filter.pillar || filter.author
      ? `${matched.length} ${matched.length === 1 ? 'article' : 'articles'} found`
      : ''
  }

  const nav = root.querySelector<HTMLElement>('[data-blog-pagination]')
  const totalPages = Math.max(1, Math.ceil(matched.length / pageSize))
  const currentPage = Math.min(Math.max(1, page), totalPages)
  if (nav) {
    nav.hidden = matched.length <= pageSize
    nav.dataset.page = String(currentPage)
    const label = nav.querySelector('[data-blog-page-label]')
    if (label) label.textContent = `Page ${currentPage} of ${totalPages}`
    const prev = nav.querySelector<HTMLButtonElement>('[data-blog-prev]')
    const next = nav.querySelector<HTMLButtonElement>('[data-blog-next]')
    if (prev) prev.disabled = currentPage <= 1
    if (next) next.disabled = currentPage >= totalPages
  }
  return matched.length
}

export function initBlogIndexFilters(): void {
  if (typeof document === 'undefined') return
  const root = document.querySelector<HTMLElement>('[data-blog-index]')
  if (!root) return

  const params = new URLSearchParams(window.location.search)
  const search = root.querySelector<HTMLInputElement>('[data-blog-search]')
  const pillar = root.querySelector<HTMLSelectElement>('[data-blog-pillar]')
  const author = root.querySelector<HTMLSelectElement>('[data-blog-author]')
  if (search && params.get('q')) search.value = params.get('q') || ''
  if (pillar && params.get('pillar')) pillar.value = params.get('pillar') || ''
  if (author && params.get('author')) author.value = params.get('author') || ''

  let page = 1
  const apply = () => {
    applyBlogIndexFilters(root, page)
    syncUrl(readFilter(root))
  }

  search?.addEventListener('input', () => { page = 1; apply() })
  pillar?.addEventListener('change', () => { page = 1; apply() })
  author?.addEventListener('change', () => { page = 1; apply() })
  root.querySelector('[data-blog-clear]')?.addEventListener('click', () => {
    if (search) search.value = ''
    if (pillar) pillar.value = ''
    if (author) author.value = ''
    page = 1
    apply()
  })
  root.querySelector('form[data-blog-filters]')?.addEventListener('submit', event => {
    event.preventDefault()
    page = 1
    apply()
  })
  root.querySelector('[data-blog-prev]')?.addEventListener('click', () => { page -= 1; apply() })
  root.querySelector('[data-blog-next]')?.addEventListener('click', () => { page += 1; apply() })

  apply()
}
