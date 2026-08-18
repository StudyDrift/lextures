import { DEFAULT_LOCALE, localeFromPath, sourcePathFor } from './locales.ts'

export type ContentPath = {
  kind: 'blog' | 'doc'
  slug: string
  category?: string
  locale: string
  path: string
}

const SLUG = '[a-z0-9]+(?:-[a-z0-9]+)*'

export function parseContentPath(pathname: string): ContentPath | null {
  const normalized = pathname.replace(/\/+$/, '') || '/'
  const locale = localeFromPath(normalized).code
  const path = sourcePathFor(normalized)
  const blog = path.match(new RegExp(`^/blog/(${SLUG})$`))
  if (blog) {
    return { kind: 'blog', slug: blog[1], locale, path: normalized }
  }
  const doc = path.match(new RegExp(`^/docs/(${SLUG})/(${SLUG})$`))
  if (doc) {
    return { kind: 'doc', slug: doc[2], category: doc[1], locale, path: normalized }
  }
  return null
}

export function previewTokenFromSearch(search: string): string | undefined {
  const token = new URLSearchParams(search.startsWith('?') ? search.slice(1) : search).get('preview_token')
  return token?.trim() || undefined
}

export function publicArticleUrl(ref: ContentPath, previewToken?: string, apiBase = ''): string {
  const suffix =
    ref.kind === 'blog'
      ? `/api/v1/public/content/articles/blog/${encodeURIComponent(ref.slug)}`
      : `/api/v1/public/content/articles/docs/${encodeURIComponent(ref.category || '')}/${encodeURIComponent(ref.slug)}`
  const params = new URLSearchParams()
  if (ref.locale && ref.locale !== DEFAULT_LOCALE) params.set('locale', ref.locale)
  if (previewToken) params.set('preview_token', previewToken)
  const query = params.toString()
  return `${apiBase.replace(/\/$/, '')}${suffix}${query ? `?${query}` : ''}`
}

export function publicArticlesUrl(
  kind: 'blog' | 'doc',
  opts: { locale?: string; category?: string; author?: string; cursor?: string; limit?: number } = {},
  apiBase = '',
): string {
  const params = new URLSearchParams({ kind, limit: String(opts.limit ?? 200) })
  if (opts.locale && opts.locale !== DEFAULT_LOCALE) params.set('locale', opts.locale)
  if (opts.category) params.set('category', opts.category)
  if (opts.author) params.set('author', opts.author)
  if (opts.cursor) params.set('cursor', opts.cursor)
  return `${apiBase.replace(/\/$/, '')}/api/v1/public/content/articles?${params}`
}

export function articleDate(value: string | undefined): string {
  return String(value || '').slice(0, 10)
}

export function mergePublishedArticles<T extends { slug: string; date: string }>(existing: T[], live: T[]): T[] {
  const bySlug = new Map<string, T>()
  for (const item of existing) bySlug.set(item.slug, item)
  for (const item of live) {
    const prior = bySlug.get(item.slug)
    bySlug.set(item.slug, prior ? { ...prior, ...item } : item)
  }
  return [...bySlug.values()].sort((a, b) => b.date.localeCompare(a.date))
}
