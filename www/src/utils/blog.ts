import { requireAuthor } from '../lib/authors'
import { renderMarkdown } from '../lib/markdown'

export interface BlogPost {
  slug: string
  title: string
  date: string
  /** Optional content-driven lastmod (SEO.2). */
  updated?: string
  description: string
  /** Author registry slug (SEO.3 FR-20). */
  author: string
  reviewedBy?: string
  pillar: string
  briefRef: string
  reviewDue: string
  /** Primary sources for Article.citation (SEO.3 FR-11). */
  citations: string[]
  content: string
  /** Build-time HTML (SEO.4 FR-5 — no react-markdown in the browser). */
  html: string
}

/** Lightweight meta for route-manifest / indexes (no markdown-it). */
export type BlogPostMeta = Omit<BlogPost, 'content' | 'html'>

const rawModules = import.meta.glob('../blog/*.md', {
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
    let value = line.slice(colon + 1).trim().replace(/^["']|["']$/g, '')
    if (value.startsWith('[') && value.endsWith(']')) {
      value = value
        .slice(1, -1)
        .split(',')
        .map(s => s.trim().replace(/^["']|["']$/g, ''))
        .filter(Boolean)
        .join('|')
    }
    meta[key] = value
  }

  return { meta, content: match[2].trim() }
}

function wordCount(markdown: string): number {
  return markdown
    .replace(/```[\s\S]*?```/g, ' ')
    .replace(/[#>*_`[\]()!-]/g, ' ')
    .split(/\s+/)
    .filter(Boolean).length
}

const parsed = Object.entries(rawModules).map(([path, raw]) => {
  const slug = path.replace('../blog/', '').replace(/\.md$/, '')
  const { meta, content } = parseFrontmatter(raw)
  const author = meta.author || 'chase-willden'
  requireAuthor(author, `/blog/${slug}`)
  if (meta.reviewedBy) requireAuthor(meta.reviewedBy, `/blog/${slug} reviewedBy`)
  return {
    slug,
    title: meta.title ?? slug,
    date: meta.date ?? '',
    updated: meta.updated || undefined,
    description: meta.description ?? '',
    author,
    reviewedBy: meta.reviewedBy || undefined,
    pillar: meta.pillar || '',
    briefRef: meta.briefRef || '',
    reviewDue: meta.reviewDue || '',
    citations: meta.citations
      ? meta.citations.split('|').map(s => s.trim()).filter(Boolean)
      : [],
    content,
  }
})

/** Meta-only list — safe for route-manifest (no markdown-it). */
export const allPostMeta: BlogPostMeta[] = parsed
  .map(({ content: _c, ...meta }) => meta)
  .sort((a, b) => b.date.localeCompare(a.date))

/** Full posts with HTML. Prefer getPost(); avoids pulling markdown-it into indexes. */
export const allPosts: BlogPost[] = parsed
  .map(p => ({
    ...p,
    html: renderMarkdown(p.content),
  }))
  .sort((a, b) => b.date.localeCompare(a.date))

export function getPost(slug: string): BlogPost | undefined {
  return allPosts.find(p => p.slug === slug)
}

export function formatDate(iso: string): string {
  if (!iso) return ''
  const d = new Date(iso + 'T00:00:00')
  return d.toLocaleDateString('en-US', { year: 'numeric', month: 'long', day: 'numeric' })
}

export function postWordCount(post: BlogPost): number {
  return wordCount(post.content)
}

export function postsByAuthor(authorSlug: string): BlogPost[] {
  return allPosts.filter(p => p.author === authorSlug)
}

export function postsMetaByAuthor(authorSlug: string): BlogPostMeta[] {
  return allPostMeta.filter(p => p.author === authorSlug)
}
