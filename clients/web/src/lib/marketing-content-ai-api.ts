import { authorizedFetch } from './api'
import { readApiErrorMessage } from './errors'
import type { MarketingContentKind, MarketingFinding } from './marketing-content-api'

export type MarketingArticleAIDraft = {
  title: string
  slug: string
  description: string
  bodyMd: string
  primaryQuestion: string
  cluster: string
  pillar: string
  keywords: string[]
}

/** POST `/api/v1/admin/marketing/articles/generate` — draft fields only (not persisted). */
export async function generateMarketingArticle(input: {
  prompt?: string
  kind: MarketingContentKind
  existingTitle?: string
  existingBodyMd?: string
  mode?: 'article' | 'metadata' | 'repair'
  description?: string
  primaryQuestion?: string
  cluster?: string
  pillar?: string
  keywords?: string[]
  knownPaths?: string[]
  findings?: Array<Pick<MarketingFinding, 'rule' | 'severity' | 'message' | 'line' | 'path'>>
}): Promise<MarketingArticleAIDraft> {
  const res = await authorizedFetch('/api/v1/admin/marketing/articles/generate', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(input),
  })
  const body = await res.json().catch(() => ({}))
  if (!res.ok) throw new Error(readApiErrorMessage(body))
  const value = body as Partial<MarketingArticleAIDraft>
  return {
    title: value.title ?? '',
    slug: value.slug ?? '',
    description: value.description ?? '',
    bodyMd: value.bodyMd ?? '',
    primaryQuestion: value.primaryQuestion ?? '',
    cluster: value.cluster ?? '',
    pillar: value.pillar ?? '',
    keywords: Array.isArray(value.keywords) ? value.keywords.filter(Boolean) : [],
  }
}

/** Fill slug, description, and search metadata from the current title/body. */
export async function generateMarketingArticleMetadata(input: {
  kind: MarketingContentKind
  existingTitle?: string
  existingBodyMd?: string
}): Promise<MarketingArticleAIDraft> {
  return generateMarketingArticle({ ...input, mode: 'metadata' })
}

/** Revise the current article so every listed finding, including warnings, is resolved. */
export async function repairMarketingArticle(input: {
  kind: MarketingContentKind
  existingTitle?: string
  existingBodyMd?: string
  description?: string
  primaryQuestion?: string
  cluster?: string
  pillar?: string
  keywords?: string[]
  knownPaths?: string[]
  findings: Array<Pick<MarketingFinding, 'rule' | 'severity' | 'message' | 'line' | 'path'>>
}): Promise<MarketingArticleAIDraft> {
  return generateMarketingArticle({ ...input, mode: 'repair' })
}
