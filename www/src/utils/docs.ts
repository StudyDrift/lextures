import { requireAuthor } from '../lib/authors'
import { renderMarkdown } from '../lib/markdown'

export interface DocArticle {
  slug: string
  category: string
  title: string
  date: string
  updated?: string
  description: string
  /** Author registry slug (SEO.3 FR-20). */
  author: string
  reviewedBy?: string
  reviewedAt?: string
  roles: string[]
  segments: string[]
  verifiedAgainst: string
  relatedTo: string[]
  content: string
  /** Build-time HTML (SEO.4 FR-5). */
  html: string
}

export type DocArticleMeta = Omit<DocArticle, 'content' | 'html'>

const rawModules = import.meta.glob('../docs/*/*.md', {
  query: '?raw',
  import: 'default',
  eager: true,
}) as Record<string, string>

function parseFrontmatter(raw: string): { meta: Record<string, string>; content: string } {
  const match = raw.match(/^---\r?\n([\s\S]*?)\r?\n---\r?\n([\s\S]*)$/)
  if (!match) return { meta: {}, content: raw }

  const meta: Record<string, string> = {}
  for (const line of match[1].split('\n')) {
    const colon = line.indexOf(':')
    if (colon === -1) continue
    const key = line.slice(0, colon).trim()
    const value = line.slice(colon + 1).trim().replace(/^["']|["']$/g, '')
    meta[key] = value
  }

  return { meta, content: match[2].trim() }
}

const parsed = Object.entries(rawModules).map(([path, raw]) => {
  const relative = path.replace('../docs/', '').replace(/\.md$/, '')
  const parts = relative.split('/')
  const slug = parts.at(-1) || relative
  const { meta, content } = parseFrontmatter(raw)
  const author = meta.author || 'chase-willden'
  const category = meta.category || (parts.length > 1 ? parts[0] : 'getting-started')
  requireAuthor(author, `/docs/${category}/${slug}`)
  if (meta.reviewedBy) requireAuthor(meta.reviewedBy, `/docs/${category}/${slug} reviewedBy`)
  const list = (value = '') => value.replace(/^\[|\]$/g, '').split(',').map(item => item.trim()).filter(Boolean)
  return {
    slug, category,
    title: meta.title ?? slug,
    date: meta.date ?? '',
    updated: meta.updated || undefined,
    description: meta.description ?? '',
    author,
    reviewedBy: meta.reviewedBy || undefined,
    reviewedAt: meta.reviewedAt || undefined,
    roles: list(meta.roles),
    segments: list(meta.segments),
    verifiedAgainst: meta.verifiedAgainst || '',
    relatedTo: list(meta.relatedTo),
    content,
  }
})

export const allArticleMeta: DocArticleMeta[] = parsed
  .map(({ content: _c, ...meta }) => meta)
  .sort((a, b) => b.date.localeCompare(a.date))

export const allArticles: DocArticle[] = parsed
  .map(p => ({
    ...p,
    html: renderMarkdown(p.content),
  }))
  .sort((a, b) => b.date.localeCompare(a.date))

export function getArticle(slug: string): DocArticle | undefined {
  return allArticles.find(p => p.slug === slug)
}

export function getCategorizedArticle(category: string, slug: string): DocArticle | undefined {
  return allArticles.find(article => article.category === category && article.slug === slug)
}

export function articlePath(article: Pick<DocArticle, 'category' | 'slug'>): string {
  return `/docs/${article.category}/${article.slug}`
}

export function formatDate(iso: string): string {
  if (!iso) return ''
  const d = new Date(iso + 'T00:00:00')
  return d.toLocaleDateString('en-US', { year: 'numeric', month: 'long', day: 'numeric' })
}

export function articlesByAuthor(authorSlug: string): DocArticle[] {
  return allArticles.filter(a => a.author === authorSlug)
}

export function articlesMetaByAuthor(authorSlug: string): DocArticleMeta[] {
  return allArticleMeta.filter(a => a.author === authorSlug)
}
