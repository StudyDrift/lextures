import { renderMarkdown } from './markdown'

export type ContentAuthor = { slug: string; name: string; jobTitle?: string; bio?: string; knowsAbout?: string[]; status?: 'active' | 'retired'; links?: { sameAs?: string[]; website?: string } | string[] }
export type ContentCategory = { slug: string; locale?: string; title: string; description: string; sortOrder?: number }
export type ContentRedirect = { from: string; to: string; statusCode: number }
export type ContentArticle = {
  path: string; kind: 'blog' | 'doc'; slug: string; locale: string; category?: string
  title: string; description: string; date: string; updated?: string; author: string; reviewedBy?: string
  reviewedAt?: string; roles: string[]; segments: string[]; verifiedAgainst: string; relatedTo: string[]
  citations: string[]; pillar: string; briefRef: string; reviewDue: string; content: string; html: string
  faq?: Array<{ question: string; answer: string }>
  noindex?: boolean; canonicalOverride?: string; contentHash?: string
  publishedAt?: string; contentUpdatedAt?: string; heroMediaId?: string; media?: Array<{ id?: string; usage?: string; checksum?: string; renditions?: Array<{ name: string; ext: string; url: string; width?: number; height?: number }> }>
  availableLocales?: Array<{ locale: string; path: string }>
}
export type ContentSnapshot = {
  source: 'api'; generatedAt?: string; articles: ContentArticle[]
  categories: ContentCategory[]; authors: ContentAuthor[]; redirects: ContentRedirect[]
  fetched?: number; cacheHits?: number; fallbackUsed?: boolean
}
export interface ContentSource {
  listArticles(locale?: string): ContentArticle[]
  listAllArticles(): ContentArticle[]
  getArticle(articlePath: string, locale?: string): ContentArticle | undefined
  listCategories(locale?: string): ContentCategory[]
  listAuthors(locale?: string): ContentAuthor[]
  listRedirects(locale?: string): ContentRedirect[]
}

export type ApiArticle = Record<string, unknown> & { bodyMd?: string }
declare global { var __LEXTURES_BUILD_CONTENT__: ({ source: 'api'; articles: ApiArticle[]; categories?: ContentCategory[]; authors?: ContentAuthor[]; redirects?: ContentRedirect[]; generatedAt?: string; fetched?: number; cacheHits?: number; fallbackUsed?: boolean }) | undefined }

export function apiArticle(raw: ApiArticle): ContentArticle {
  const value = raw as Record<string, any>; const body = String(value.bodyMd || '')
  const date = String(value.publishedAt || '').slice(0, 10); const updated = String(value.contentUpdatedAt || value.updatedAt || '').slice(0, 10) || undefined
  return {
    path: value.path, kind: value.kind === 'blog' ? 'blog' : 'doc', slug: value.slug, locale: value.locale || 'en', category: value.categorySlug || undefined,
    title: value.title, description: value.description, date, updated, author: value.author?.slug || 'chase-willden',
    reviewedBy: value.reviewer?.slug, reviewedAt: value.reviewedAt, roles: value.roles || [], segments: value.segments || [],
    verifiedAgainst: value.verifiedAgainst || '', relatedTo: value.relatedTo || [], citations: value.citations || [], pillar: value.pillar || '',
    briefRef: value.briefRef || '', reviewDue: value.reviewDueOn || '', content: body, html: renderMarkdown(body), noindex: Boolean(value.noindex),
    canonicalOverride: value.canonicalOverride || undefined, contentHash: value.contentHash,
    publishedAt: value.publishedAt, contentUpdatedAt: value.contentUpdatedAt, heroMediaId: value.heroMediaId, media: value.media || [],
    availableLocales: Array.isArray(value.availableLocales) ? value.availableLocales : undefined,
  }
}

function selectedSnapshot(): ContentSnapshot {
  const injected = typeof window !== 'undefined'
    ? (window.__LEXTURES_SSR__?.articleIndex as unknown as ContentSnapshot | undefined)
    : globalThis.__LEXTURES_BUILD_CONTENT__
  if (injected?.source === 'api') return { ...injected, articles: injected.articles.map(apiArticle), categories: injected.categories || [], authors: injected.authors || [], redirects: injected.redirects || [] }
  return { source: 'api', articles: [], categories: [], authors: [], redirects: [], fallbackUsed: true }
}

export const contentSnapshot = selectedSnapshot()
export const contentSource: ContentSource = {
  listArticles: (locale = 'en') => contentSnapshot.articles.filter(a => a.locale === locale),
  listAllArticles: () => contentSnapshot.articles,
  getArticle: (articlePath) => contentSnapshot.articles.find(a => a.path === articlePath),
  listCategories: (locale = 'en') => contentSnapshot.categories.filter(c => !c.locale || c.locale === locale),
  listAuthors: () => contentSnapshot.authors,
  listRedirects: () => contentSnapshot.redirects,
}
