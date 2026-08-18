import { API_BASE } from './api-base'
import { apiArticle, type ApiArticle, type ContentArticle } from './content-source'
import {
  publicArticlesUrl as articlesUrl,
  publicArticleUrl as articleUrl,
  type ContentPath,
} from './public-content-core'

export {
  articleDate,
  mergePublishedArticles,
  parseContentPath,
  previewTokenFromSearch,
  type ContentPath,
} from './public-content-core'

export function publicArticleUrl(ref: ContentPath, previewToken?: string): string {
  return articleUrl(ref, previewToken, API_BASE)
}

export function publicArticlesUrl(
  kind: 'blog' | 'doc',
  opts: { locale?: string; category?: string; author?: string; cursor?: string; limit?: number } = {},
): string {
  return articlesUrl(kind, opts, API_BASE)
}

export async function fetchPublishedArticle(ref: ContentPath, previewToken?: string): Promise<ContentArticle | null> {
  try {
    const response = await fetch(publicArticleUrl(ref, previewToken))
    if (!response.ok) return null
    const raw = (await response.json()) as ApiArticle
    if (!raw || raw.kind !== ref.kind || raw.slug !== ref.slug) return null
    return apiArticle(raw)
  } catch {
    return null
  }
}

export async function fetchPublishedArticleSummaries(
  kind: 'blog' | 'doc',
  opts: { locale?: string; category?: string; author?: string } = {},
): Promise<ContentArticle[]> {
  const out: ContentArticle[] = []
  let cursor: string | undefined
  for (let page = 0; page < 10; page += 1) {
    const response = await fetch(publicArticlesUrl(kind, { ...opts, cursor }))
    if (!response.ok) break
    const body = (await response.json()) as { articles?: ApiArticle[]; nextCursor?: string }
    for (const raw of body.articles || []) out.push(apiArticle(raw))
    cursor = body.nextCursor || undefined
    if (!cursor) break
  }
  return out
}
