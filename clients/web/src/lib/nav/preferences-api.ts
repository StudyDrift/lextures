import { authorizedFetch } from '../api'
import { readApiErrorMessage } from '../errors'
import type { NavPreferences } from './types'
import { emptyPreferences } from './resolve'

export type { NavPreferences }

function parsePreferences(raw: unknown, fallbackScope: string): NavPreferences {
  if (!raw || typeof raw !== 'object') return emptyPreferences(fallbackScope)
  const o = raw as Record<string, unknown>
  const scope = typeof o.scope === 'string' && o.scope ? o.scope : fallbackScope
  const pinned = Array.isArray(o.pinned)
    ? o.pinned.filter((x): x is string => typeof x === 'string')
    : []
  const hidden = Array.isArray(o.hidden)
    ? o.hidden.filter((x): x is string => typeof x === 'string')
    : []
  const collapsed = Array.isArray(o.collapsed)
    ? o.collapsed.filter((x): x is string => typeof x === 'string')
    : []
  return { scope, pinned, hidden, collapsed }
}

/** GET /api/v1/nav/preferences?scope=… */
export async function fetchNavPreferences(scope: string): Promise<NavPreferences> {
  const q = encodeURIComponent(scope)
  const res = await authorizedFetch(`/api/v1/nav/preferences?scope=${q}`)
  if (res.status === 404) {
    // Endpoint not deployed yet or flag off — defaults.
    return emptyPreferences(scope)
  }
  const raw: unknown = await res.json().catch(() => ({}))
  if (!res.ok) {
    throw new Error(readApiErrorMessage(raw) || 'Could not load navigation preferences.')
  }
  return parsePreferences(raw, scope)
}

/** PUT /api/v1/nav/preferences */
export async function putNavPreferences(prefs: NavPreferences): Promise<NavPreferences> {
  const res = await authorizedFetch('/api/v1/nav/preferences', {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      scope: prefs.scope,
      pinned: prefs.pinned,
      hidden: prefs.hidden,
      collapsed: prefs.collapsed,
    }),
  })
  const raw: unknown = await res.json().catch(() => ({}))
  if (!res.ok) {
    throw new Error(readApiErrorMessage(raw) || 'Could not save navigation preferences.')
  }
  return parsePreferences(raw, prefs.scope)
}

/** DELETE /api/v1/nav/preferences?scope=… — reset to defaults. */
export async function resetNavPreferences(scope: string): Promise<NavPreferences> {
  const q = encodeURIComponent(scope)
  const res = await authorizedFetch(`/api/v1/nav/preferences?scope=${q}`, {
    method: 'DELETE',
  })
  if (res.status === 204 || res.status === 404) {
    return emptyPreferences(scope)
  }
  const raw: unknown = await res.json().catch(() => ({}))
  if (!res.ok) {
    throw new Error(readApiErrorMessage(raw) || 'Could not reset navigation preferences.')
  }
  return parsePreferences(raw, scope)
}
