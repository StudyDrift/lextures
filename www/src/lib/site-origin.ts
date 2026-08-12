/** Public marketing site origin (canonicals, OG, sitemap). */
export const SITE_ORIGIN = (
  (typeof import.meta !== 'undefined' &&
    (import.meta as ImportMeta & { env?: { VITE_SITE_ORIGIN?: string } }).env?.VITE_SITE_ORIGIN) ||
  'https://lextures.com'
).replace(/\/$/, '')

/** Absolute canonical URL for a path (`/` stays `/`; others lose trailing slash). */
export function canonicalUrl(path: string, origin = SITE_ORIGIN): string {
  if (!path || path === '/') return `${origin}/`
  const clean = path.startsWith('/') ? path : `/${path}`
  return `${origin}${clean.replace(/\/+$/, '')}`
}
