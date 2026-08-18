import { authorizedFetch } from './api'
import { readApiErrorMessage } from './errors'
import type { MarketingContentKind, MarketingFinding } from './marketing-content-api'

export type MarketingArticleAIDraft = {
  title: string
  slug: string
  description: string
  socialTitle: string
  socialDescription: string
  bodyMd: string
  primaryQuestion: string
  cluster: string
  pillar: string
  keywords: string[]
}

function draftFromBody(body: Partial<MarketingArticleAIDraft>): MarketingArticleAIDraft {
  return {
    title: body.title ?? '',
    slug: body.slug ?? '',
    description: body.description ?? '',
    socialTitle: body.socialTitle ?? '',
    socialDescription: body.socialDescription ?? '',
    bodyMd: body.bodyMd ?? '',
    primaryQuestion: body.primaryQuestion ?? '',
    cluster: body.cluster ?? '',
    pillar: body.pillar ?? '',
    keywords: Array.isArray(body.keywords) ? body.keywords.filter(Boolean) : [],
  }
}

function generateErrorMessage(status: number, body: unknown): string {
  const message = readApiErrorMessage(body)
  if (message !== 'Request failed') return message
  if (status === 503 || status === 502) return 'AI is temporarily unavailable. Try Solve with AI again.'
  if (status === 429) return 'Too many AI requests. Wait a moment and try again.'
  return `Request failed (${status}). Try Solve with AI again.`
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
  const attempts = input.mode === 'repair' ? 2 : 1
  let lastError: Error | null = null
  for (let attempt = 0; attempt < attempts; attempt += 1) {
    const res = await authorizedFetch('/api/v1/admin/marketing/articles/generate', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(input),
    })
    const body = await res.json().catch(() => ({}))
    if (res.ok) return draftFromBody(body as Partial<MarketingArticleAIDraft>)
    lastError = new Error(generateErrorMessage(res.status, body))
    if (res.status !== 502 && res.status !== 503) break
  }
  throw lastError ?? new Error('Request failed')
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
