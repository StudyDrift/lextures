const SLUG_PATTERN = /^[a-z][a-z0-9]*(?:-[a-z0-9]+)*$/

const RESERVED_SLUGS = new Set([
  'default',
  'www',
  'app',
  'api',
  'admin',
  'login',
  'signup',
  'mfa',
  'magic-link',
])

export function normalizeOrgSlug(raw: string): string {
  return raw.trim().toLowerCase()
}

export function suggestOrgSlugFromName(name: string): string {
  const trimmed = name.trim().toLowerCase()
  if (!trimmed) return ''
  const parts = trimmed
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/-+/g, '-')
    .replace(/^-|-$/g, '')
  return parts.length > 32 ? parts.slice(0, 32).replace(/-$/, '') : parts
}

export function validateOrgSlug(slug: string): string | null {
  const s = normalizeOrgSlug(slug)
  if (!s) return 'Short name is required.'
  if (s.length < 2) return 'Short name must be at least 2 characters.'
  if (s.length > 32) return 'Short name must be at most 32 characters.'
  if (!SLUG_PATTERN.test(s)) {
    return 'Use lowercase letters, numbers, and hyphens only.'
  }
  if (RESERVED_SLUGS.has(s)) return 'That short name is reserved.'
  return null
}

export function orgLoginPath(slug: string): string {
  const s = normalizeOrgSlug(slug)
  return s ? `/login/${encodeURIComponent(s)}` : '/login'
}