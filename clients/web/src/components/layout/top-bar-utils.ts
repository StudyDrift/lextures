export type TopBarAccountProfile = {
  email: string
  displayName?: string | null
  firstName?: string | null
  lastName?: string | null
  avatarUrl?: string | null
}

function readOptionalString(value: unknown): string | null {
  return typeof value === 'string' ? value : null
}

/** Normalizes GET/PATCH /api/v1/settings/account JSON for the top bar. */
export function parseAccountProfile(raw: unknown): TopBarAccountProfile | null {
  if (!raw || typeof raw !== 'object') return null
  const o = raw as Record<string, unknown>
  const email = readOptionalString(o.email)?.trim() ?? ''
  if (!email) return null
  const avatarCandidate =
    readOptionalString(o.avatarUrl) ?? readOptionalString(o.avatar_url) ?? ''
  const avatarUrl = avatarCandidate.trim() || null
  return {
    email,
    displayName: readOptionalString(o.displayName) ?? readOptionalString(o.display_name),
    firstName: readOptionalString(o.firstName) ?? readOptionalString(o.first_name),
    lastName: readOptionalString(o.lastName) ?? readOptionalString(o.last_name),
    avatarUrl,
  }
}

export function nameFieldsFromProfile(profile: {
  firstName?: string | null
  lastName?: string | null
  displayName?: string | null
}): { firstName: string; lastName: string } {
  const first = profile.firstName?.trim() ?? ''
  const last = profile.lastName?.trim() ?? ''
  if (first || last) {
    return { firstName: first, lastName: last }
  }
  const display = profile.displayName?.trim() ?? ''
  if (!display) {
    return { firstName: '', lastName: '' }
  }
  const parts = display.split(/\s+/).filter(Boolean)
  if (parts.length === 0) {
    return { firstName: '', lastName: '' }
  }
  if (parts.length === 1) {
    return { firstName: parts[0], lastName: '' }
  }
  return { firstName: parts[0], lastName: parts.slice(1).join(' ') }
}

export function profileName(profile: TopBarAccountProfile | null): string {
  if (!profile) return 'Profile'
  const { firstName, lastName } = nameFieldsFromProfile(profile)
  const combined = [firstName, lastName].filter(Boolean).join(' ').trim()
  if (combined) return combined
  const display = profile.displayName?.trim() ?? ''
  if (display) return display
  return profile.email
}

export function initialsFromName(name: string | null | undefined): string {
  const parts = String(name ?? '')
    .split(/\s+/)
    .map((s) => s.trim())
    .filter(Boolean)
  if (parts.length === 0) return 'U'
  if (parts.length === 1) return parts[0].slice(0, 1).toUpperCase()
  return `${parts[0].slice(0, 1)}${parts[1].slice(0, 1)}`.toUpperCase()
}

/** Keyboard hint for opening the command palette (⌘K vs Ctrl+K). */
export function shortcutHint(): string {
  if (typeof navigator === 'undefined') return '⌘K'
  const p = navigator.platform ?? ''
  const ua = navigator.userAgent ?? ''
  const apple = /Mac|iPhone|iPad|iPod/.test(p) || /Mac OS/.test(ua)
  return apple ? '⌘K' : 'Ctrl+K'
}
