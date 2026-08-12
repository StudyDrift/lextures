import { mkdir, readFile, writeFile } from 'node:fs/promises'
import path from 'node:path'

const sleep = ms => new Promise(resolve => setTimeout(resolve, ms))

export async function fetchWithRetry(url, { userAgent, attempts = 3 } = {}) {
  let lastError
  for (let attempt = 1; attempt <= attempts; attempt++) {
    try {
      const response = await fetch(url, { headers: { Accept: 'application/json', 'User-Agent': userAgent } })
      if (response.ok) return response
      const body = await response.text().catch(() => '')
      lastError = new Error(`GET ${url} → ${response.status} ${body.slice(0, 200)}`)
      if (response.status !== 429 && response.status < 500) break
    } catch (error) {
      lastError = error
    }
    if (attempt < attempts) await sleep(250 * 2 ** (attempt - 1))
  }
  throw lastError
}

async function mapPool(items, concurrency, fn) {
  let cursor = 0
  const output = new Array(items.length)
  async function worker() {
    while (cursor < items.length) {
      const index = cursor++
      output[index] = await fn(items[index], index)
    }
  }
  await Promise.all(Array.from({ length: Math.min(concurrency, Math.max(1, items.length)) }, worker))
  return output
}

export async function loadApiContent({ apiBase, cacheDir, concurrency = 8, userAgent }) {
  await mkdir(cacheDir, { recursive: true })
  const indexFile = path.join(cacheDir, 'index.json')
  let index
  let fallbackUsed = false
  try {
    const response = await fetchWithRetry(`${apiBase}/api/v1/public/content/index`, { userAgent })
    index = await response.json()
    await writeFile(indexFile, JSON.stringify(index), 'utf8')
  } catch (error) {
    fallbackUsed = true
    try {
      index = JSON.parse(await readFile(indexFile, 'utf8'))
      console.warn(`[generate-site] WARN: content API unreachable (${error.message || error}); using cached content index`)
    } catch {
      console.warn(`[generate-site] WARN: content API unreachable (${error.message || error}); no content cache is available`)
      index = { generatedAt: new Date(0).toISOString(), articles: [], categories: [], authors: [], redirects: [] }
    }
  }

  let fetched = 0
  let cacheHits = 0
  const articles = await mapPool(index.articles || [], concurrency, async summary => {
    const hash = String(summary.contentHash || '').replace(/[^a-zA-Z0-9_-]/g, '')
    const cacheFile = path.join(cacheDir, `${hash || encodeURIComponent(summary.path)}.json`)
    if (hash) {
      try {
        const cached = JSON.parse(await readFile(cacheFile, 'utf8'))
        cacheHits++
        return cached
      } catch { /* fetch below */ }
    }
    if (fallbackUsed) return { ...summary, bodyMd: '' }
    const detailPath = summary.kind === 'blog'
      ? `/api/v1/public/content/articles/blog/${encodeURIComponent(summary.slug)}`
      : `/api/v1/public/content/articles/docs/${encodeURIComponent(summary.categorySlug || '')}/${encodeURIComponent(summary.slug)}`
    const locale = summary.locale && summary.locale !== 'en' ? `?locale=${encodeURIComponent(summary.locale)}` : ''
    try {
      const response = await fetchWithRetry(`${apiBase}${detailPath}${locale}`, { userAgent })
      const detail = await response.json()
      const article = detail.article || detail
      fetched++
      if (hash) await writeFile(cacheFile, JSON.stringify(article), 'utf8')
      return article
    } catch (error) {
      fallbackUsed = true
      console.warn(`[generate-site] WARN: content body unavailable for ${summary.path}: ${error.message || error}`)
      return { ...summary, bodyMd: '' }
    }
  })
  return {
    source: 'api', generatedAt: index.generatedAt, articles,
    categories: index.categories || [], authors: index.authors || [], redirects: index.redirects || [],
    fetched, cacheHits, fallbackUsed,
  }
}

export function mergeRedirects(staticRules, apiRules) {
  const occupied = new Set(staticRules.map(rule => rule.from))
  return [...staticRules, ...apiRules.filter(rule => !occupied.has(rule.from)).map(rule => ({
    from: rule.from, to: rule.to, status: rule.statusCode || rule.status || 301,
  }))]
}

/** Reciprocal hreflang cluster for a translated content article. Empty when only one locale exists. */
export function contentHreflangAlternates(article, origin) {
  const alts = article?.availableLocales || []
  if (alts.length < 2) return []
  const english = alts.find(item => item.locale === 'en') || alts[0]
  return [
    ...alts.map(item => ({ hreflang: item.locale, href: `${origin}${item.path}` })),
    { hreflang: 'x-default', href: `${origin}${english.path}` },
  ]
}
