export const MARKETING_SITE_ORIGIN = (
  (typeof import.meta !== 'undefined' && import.meta.env?.VITE_MARKETING_SITE_ORIGIN) ||
  'https://lextures.com'
).replace(/\/$/, '')

export const MARKETING_SITE_URLS = {
  privacy: `${MARKETING_SITE_ORIGIN}/privacy`,
  privacyHistory: `${MARKETING_SITE_ORIGIN}/privacy/history`,
  terms: `${MARKETING_SITE_ORIGIN}/terms`,
  termsHistory: `${MARKETING_SITE_ORIGIN}/terms/history`,
  security: `${MARKETING_SITE_ORIGIN}/security`,
  accessibility: `${MARKETING_SITE_ORIGIN}/accessibility`,
  accessibilityVpat: `${MARKETING_SITE_ORIGIN}/accessibility/vpat`,
  californiaPrivacyRights: `${MARKETING_SITE_ORIGIN}/privacy-rights/california`,
} as const

/** @deprecated Prefer MARKETING_SITE_URLS */
export const MARKETING_LEGAL_URLS = MARKETING_SITE_URLS

/** Absolute public URL for a blog or help-center article path. */
export function publicMarketingArticleUrl(
  path: string,
  query: Record<string, string | undefined> = {},
): string {
  const clean = path.startsWith('/') ? path : `/${path}`
  const url = new URL(clean.replace(/\/+$/, '') || '/', `${MARKETING_SITE_ORIGIN}/`)
  for (const [key, value] of Object.entries(query)) {
    if (value) url.searchParams.set(key, value)
  }
  return url.href
}

/**
 * Preview links from the API should open on the marketing site, never the SPA.
 * Legacy mint responses used `/preview/{id}?token=` — rewrite those to the article path.
 */
export function resolveMarketingPreviewUrl(apiUrl: string | undefined, articlePath: string): string {
  if (!apiUrl) return publicMarketingArticleUrl(articlePath)
  try {
    const parsed = new URL(apiUrl, `${MARKETING_SITE_ORIGIN}/`)
    if (parsed.pathname.startsWith('/preview/')) {
      const token = parsed.searchParams.get('preview_token') ?? parsed.searchParams.get('token')
      return publicMarketingArticleUrl(articlePath, { preview_token: token ?? undefined })
    }
    return parsed.href
  } catch {
    return publicMarketingArticleUrl(articlePath)
  }
}
