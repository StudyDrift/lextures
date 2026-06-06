import type { SearchGroup, SearchListItem } from './build-search-items'

const STORAGE_KEY = 'lextures:search-recents:v1'
const MAX_RECENTS = 10

export type SearchRecentEntry = {
  id: string
  group: SearchGroup
  title: string
  subtitle: string
  path: string
  visitedAt: string
}

function readStore(): SearchRecentEntry[] {
  if (typeof localStorage === 'undefined') return []
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return []
    const parsed = JSON.parse(raw) as unknown
    if (!Array.isArray(parsed)) return []
    return parsed.filter(
      (row): row is SearchRecentEntry =>
        !!row &&
        typeof row === 'object' &&
        typeof (row as SearchRecentEntry).id === 'string' &&
        typeof (row as SearchRecentEntry).path === 'string',
    )
  } catch {
    return []
  }
}

function writeStore(entries: SearchRecentEntry[]) {
  if (typeof localStorage === 'undefined') return
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(entries))
  } catch {
    /* quota / private mode */
  }
}

export function listSearchRecents(): SearchRecentEntry[] {
  return readStore().sort((a, b) => Date.parse(b.visitedAt) - Date.parse(a.visitedAt))
}

export function recordSearchRecent(item: SearchListItem): void {
  const entry: SearchRecentEntry = {
    id: item.id,
    group: item.group,
    title: item.title,
    subtitle: item.subtitle,
    path: item.path,
    visitedAt: new Date().toISOString(),
  }
  const prev = readStore().filter((r) => r.id !== entry.id)
  writeStore([entry, ...prev].slice(0, MAX_RECENTS))
}

export function recentsToSearchItems(recents: SearchRecentEntry[]): SearchListItem[] {
  return recents.map((r) => ({
    id: `recent:${r.id}`,
    group: 'recent' as const,
    title: r.title,
    subtitle: r.subtitle,
    path: r.path,
    haystack: `${r.title} ${r.subtitle} recent`.toLowerCase(),
  }))
}
