import { authorizedFetch } from './api'
import { readingLevelFeatureEnabled } from './platform-features'

export type ReadingLevelInfo = {
  fkgl?: number
  fre?: number
  sufficient: boolean
  wordCount?: number
  simplifiedMarkdown?: string
  simplifiedNotice?: boolean
  targetFkgl?: number
  aboveThreshold?: boolean
}

export type SimplifyResult = {
  original: string
  simplified: string
  targetFkgl: number
  computedFkgl?: number
  cached: boolean
}

export function isReadingLevelEnabled(): boolean {
  return readingLevelFeatureEnabled()
}

export async function fetchItemReadingLevel(
  courseCode: string,
  itemId: string,
): Promise<ReadingLevelInfo> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/items/${encodeURIComponent(itemId)}/reading-level`,
  )
  const raw = await res.json()
  if (!res.ok) {
    throw new Error(
      typeof raw === 'object' && raw && 'error' in raw
        ? String((raw as { error?: { message?: string } }).error?.message)
        : 'Failed to load reading level',
    )
  }
  const r = raw as Record<string, unknown>
  return {
    fkgl: typeof r.fkgl === 'number' ? r.fkgl : undefined,
    fre: typeof r.fre === 'number' ? r.fre : undefined,
    sufficient: Boolean(r.sufficient),
    wordCount: typeof r.wordCount === 'number' ? r.wordCount : undefined,
    simplifiedMarkdown:
      typeof r.simplifiedMarkdown === 'string' ? r.simplifiedMarkdown : undefined,
    simplifiedNotice: Boolean(r.simplifiedNotice),
    targetFkgl: typeof r.targetFkgl === 'number' ? r.targetFkgl : undefined,
    aboveThreshold: Boolean(r.aboveThreshold),
  }
}

export async function simplifyItemContent(
  courseCode: string,
  itemId: string,
  targetFkgl: number,
  text?: string,
): Promise<SimplifyResult> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/items/${encodeURIComponent(itemId)}/simplify`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ targetFkgl, text: text ?? '' }),
    },
  )
  const raw = await res.json()
  if (!res.ok) {
    throw new Error(
      typeof raw === 'object' && raw && 'error' in raw
        ? String((raw as { error?: { message?: string } }).error?.message)
        : 'Simplification failed',
    )
  }
  const r = raw as Record<string, unknown>
  return {
    original: String(r.original ?? ''),
    simplified: String(r.simplified ?? ''),
    targetFkgl: Number(r.targetFkgl),
    computedFkgl: typeof r.computedFkgl === 'number' ? r.computedFkgl : undefined,
    cached: Boolean(r.cached),
  }
}

export function formatFkglLabel(fkgl: number | undefined): string {
  if (fkgl == null || Number.isNaN(fkgl)) return '—'
  return `Grade ${fkgl.toFixed(1)}`
}
