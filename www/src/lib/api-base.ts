import { APP_ORIGIN } from './site-links'

/** API origin for www pages that call the Lextures backend (defaults to the self-learner app). */
export const API_BASE = (import.meta.env.VITE_API_BASE_URL ?? APP_ORIGIN).replace(/\/$/, '')

/** Turn API-relative asset paths into absolute URLs on the self-learner origin. */
export function resolveApiAssetUrl(url: string | null | undefined): string | null {
  if (!url) return null
  const trimmed = url.trim()
  if (!trimmed) return null
  if (trimmed.startsWith('http://') || trimmed.startsWith('https://')) return trimmed
  if (trimmed.startsWith('/')) return `${API_BASE}${trimmed}`
  return trimmed
}
