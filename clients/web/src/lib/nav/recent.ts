/**
 * Client-local recent destinations (FR-15). Not synced unless product opts in.
 * Privacy: behavioural; stored only in localStorage (S05 RoPA when sync lands).
 */

const PREFIX = 'lextures.nav.recent.'
const MAX = 5

function storageKey(scope: string): string {
  return `${PREFIX}${scope}`
}

export function readRecentDestinationIds(scope: string): string[] {
  try {
    if (typeof localStorage === 'undefined') return []
    const raw = localStorage.getItem(storageKey(scope))
    if (!raw) return []
    const parsed: unknown = JSON.parse(raw)
    if (!Array.isArray(parsed)) return []
    return parsed.filter((x): x is string => typeof x === 'string').slice(0, MAX)
  } catch {
    return []
  }
}

export function pushRecentDestination(scope: string, destinationId: string): string[] {
  const prev = readRecentDestinationIds(scope).filter((id) => id !== destinationId)
  const next = [destinationId, ...prev].slice(0, MAX)
  try {
    if (typeof localStorage !== 'undefined') {
      localStorage.setItem(storageKey(scope), JSON.stringify(next))
    }
  } catch {
    // quota / private mode
  }
  return next
}
