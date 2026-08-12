/**
 * Content metadata only — no markdown-it (SEO.4 FR-5).
 * Used by route-manifest so interactive entry chunks stay free of the markdown runtime.
 */
import { requireAuthor } from '../lib/authors'

export type ContentMeta = {
  slug: string
  category?: string
  title: string
  date: string
  updated?: string
  description: string
  author: string
  reviewedBy?: string
  citations?: string[]
  faq?: Array<{ question: string; answer: string }>
  wordCount: number
}

function countWords(markdown: string): number {
  return markdown
    .replace(/```[\s\S]*?```/g, ' ')
    .replace(/[#>*_`[\]()!-]/g, ' ')
    .split(/\s+/)
    .filter(Boolean).length
}

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

const blogRaw = import.meta.glob('../blog/*.md', {
  query: '?raw',
  import: 'default',
  eager: true,
}) as Record<string, string>

const docsRaw = import.meta.glob('../docs/*/*.md', {
  query: '?raw',
  import: 'default',
  eager: true,
}) as Record<string, string>

function toMeta(
  path: string,
  raw: string,
  prefix: '../blog/' | '../docs/',
  kind: 'blog' | 'docs',
): ContentMeta {
  const relative = path.replace(prefix, '').replace(/\.md$/, '')
  const parts = relative.split('/')
  const slug = parts.at(-1) || relative
  const { meta, content } = parseFrontmatter(raw)
  const faqBlock = content.match(/^:::\s*faq\s*\r?\n([\s\S]*?)^:::\s*$/im)?.[1] || ''
  const faq = [...faqBlock.matchAll(/^###\s+(.+\?)\s*\r?\n([\s\S]*?)(?=^###\s+|(?![\s\S]))/gm)].map(m => ({
    question: m[1].trim(),
    answer: m[2].replace(/\[([^\]]+)\]\([^)]+\)/g, '$1').replace(/[*_`]/g, '').replace(/\s+/g, ' ').trim(),
  }))
  const author = meta.author || 'chase-willden'
  requireAuthor(author, `/${kind}/${slug}`)
  if (meta.reviewedBy) requireAuthor(meta.reviewedBy, `/${kind}/${slug} reviewedBy`)
  return {
    slug,
    category: kind === 'docs' ? (meta.category || (parts.length > 1 ? parts[0] : 'getting-started')) : undefined,
    title: meta.title ?? slug,
    date: meta.date ?? '',
    updated: meta.updated || undefined,
    description: meta.description ?? '',
    author,
    reviewedBy: meta.reviewedBy || undefined,
    citations: meta.citations
      ? meta.citations.split('|').map(s => s.trim()).filter(Boolean)
      : undefined,
    faq,
    wordCount: countWords(content),
  }
}

export const blogPostMeta: ContentMeta[] = Object.entries(blogRaw)
  .map(([path, raw]) => toMeta(path, raw, '../blog/', 'blog'))
  .sort((a, b) => b.date.localeCompare(a.date))

export const docArticleMeta: ContentMeta[] = Object.entries(docsRaw)
  .map(([path, raw]) => toMeta(path, raw, '../docs/', 'docs'))
  .sort((a, b) => b.date.localeCompare(a.date))
