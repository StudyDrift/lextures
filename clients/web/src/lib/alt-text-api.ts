import { authorizedFetch } from './api'

export type CourseAccessibilityInfo = {
  altTextCoverage: {
    withAlt: number
    total: number
    percent: number
    uncoveredItems: Array<{
      itemId: string
      title: string
      kind: string
      withAlt: number
      total: number
      missing: number
    }>
  }
  hardBlockSave: boolean
}

export type AltTextSuggestion = {
  suggestion: string
  confidence: number
}

export async function fetchCourseAccessibility(
  courseCode: string,
): Promise<CourseAccessibilityInfo> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/accessibility`,
  )
  const raw = await res.json()
  if (!res.ok) {
    throw new Error(
      typeof raw === 'object' && raw && 'error' in raw
        ? String((raw as { error?: { message?: string } }).error?.message)
        : 'Failed to load accessibility coverage',
    )
  }
  const r = raw as Record<string, unknown>
  const cov = (r.altTextCoverage ?? {}) as Record<string, unknown>
  const items = Array.isArray(cov.uncoveredItems) ? cov.uncoveredItems : []
  return {
    altTextCoverage: {
      withAlt: Number(cov.withAlt ?? 0),
      total: Number(cov.total ?? 0),
      percent: Number(cov.percent ?? 100),
      uncoveredItems: items.map((it) => {
        const row = it as Record<string, unknown>
        return {
          itemId: String(row.itemId ?? ''),
          title: String(row.title ?? ''),
          kind: String(row.kind ?? ''),
          withAlt: Number(row.withAlt ?? 0),
          total: Number(row.total ?? 0),
          missing: Number(row.missing ?? 0),
        }
      }),
    },
    hardBlockSave: Boolean(r.hardBlockSave),
  }
}

export async function suggestAltText(
  courseCode: string,
  imageUrl: string,
  language?: string,
): Promise<AltTextSuggestion> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/alt-text/suggest`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ imageUrl, language: language ?? '' }),
    },
  )
  const raw = await res.json()
  if (!res.ok) {
    throw new Error(
      typeof raw === 'object' && raw && 'error' in raw
        ? String((raw as { error?: { message?: string } }).error?.message)
        : 'Alt-text suggestion failed',
    )
  }
  const r = raw as Record<string, unknown>
  return {
    suggestion: String(r.suggestion ?? ''),
    confidence: typeof r.confidence === 'number' ? r.confidence : 0,
  }
}
