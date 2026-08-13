import { authorizedFetch } from './api'
import { readApiErrorMessage } from './errors'
import type { MarketingContentKind } from './marketing-content-api'

export type MarketingArticleAIDraft = {
  title: string
  description: string
  bodyMd: string
  primaryQuestion: string
  cluster: string
  pillar: string
  keywords: string[]
}

/** POST `/api/v1/admin/marketing/articles/generate` — draft fields only (not persisted). */
export async function generateMarketingArticle(input: {
  prompt: string
  kind: MarketingContentKind
  existingTitle?: string
  existingBodyMd?: string
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
    description: value.description ?? '',
    bodyMd: value.bodyMd ?? '',
    primaryQuestion: value.primaryQuestion ?? '',
    cluster: value.cluster ?? '',
    pillar: value.pillar ?? '',
    keywords: Array.isArray(value.keywords) ? value.keywords.filter(Boolean) : [],
  }
}
