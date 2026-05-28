import { authorizedFetch } from './api'
import { readApiErrorMessage } from './errors'
import { translationMemoryFeatureEnabled } from './platform-features'

export function isTranslationMemoryEnabled(): boolean {
  return translationMemoryFeatureEnabled()
}

export type TranslationCoverage = {
  targetLocale: string
  totalItems: number
  translatedItems: number
  percent: number
  untranslated?: Array<{
    itemId: string
    itemType: string
    title: string
    body: string
  }>
}

export type TranslationListItem = {
  itemId: string
  itemType: string
  title: string
  body: string
  hasPublished?: boolean
  hasDraft?: boolean
  targetLocale?: string
  translatedTitle?: string | null
  translatedBody?: string | null
  isDraft?: boolean
  machineTranslationDraft?: boolean
  publishedAt?: string | null
  version?: number
  glossaryMatches?: Array<{
    sourceTerm: string
    targetTerm: string
    start: number
    end: number
  }>
}

export type TMMatch = {
  translatedText: string
  similarity: number
  exact: boolean
}

export async function fetchCourseTranslations(
  courseCode: string,
  targetLocale: string,
): Promise<{ items: TranslationListItem[]; coverage: TranslationCoverage }> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/translations?target_locale=${encodeURIComponent(targetLocale)}`,
  )
  const raw = await res.json()
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  return raw as { items: TranslationListItem[]; coverage: TranslationCoverage }
}

export async function saveCourseTranslation(
  courseCode: string,
  itemId: string,
  body: {
    targetLocale: string
    sourceLocale?: string
    translatedTitle?: string | null
    translatedBody?: string | null
    isDraft?: boolean
    machineTranslationDraft?: boolean
    version?: number
  },
): Promise<Record<string, unknown>> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/translations/${encodeURIComponent(itemId)}`,
    {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    },
  )
  const raw = await res.json()
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  return raw as Record<string, unknown>
}

export async function publishCourseTranslation(
  courseCode: string,
  itemId: string,
  targetLocale: string,
): Promise<void> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/translations/${encodeURIComponent(itemId)}/publish`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ targetLocale }),
    },
  )
  if (!res.ok) {
    const raw = await res.json().catch(() => ({}))
    throw new Error(readApiErrorMessage(raw))
  }
}

export async function requestAIDraftTranslation(
  courseCode: string,
  itemId: string,
  targetLocale: string,
  sourceLocale = 'en',
): Promise<Record<string, unknown>> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/translations/${encodeURIComponent(itemId)}/ai-draft`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ targetLocale, sourceLocale }),
    },
  )
  const raw = await res.json()
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  return raw as Record<string, unknown>
}

export async function queryTranslationMemory(
  courseCode: string,
  sourceLocale: string,
  targetLocale: string,
  text: string,
): Promise<TMMatch[]> {
  const q = new URLSearchParams({
    course_code: courseCode,
    source_locale: sourceLocale,
    target_locale: targetLocale,
    text,
  })
  const res = await authorizedFetch(`/api/v1/translation-memory?${q}`)
  const raw = await res.json()
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  return (raw as { matches?: TMMatch[] }).matches ?? []
}

export async function fetchTranslationCoverage(
  courseCode: string,
  targetLocale?: string,
): Promise<TranslationCoverage | { locales: TranslationCoverage[] }> {
  const path = targetLocale
    ? `/api/v1/courses/${encodeURIComponent(courseCode)}/translation-coverage?target_locale=${encodeURIComponent(targetLocale)}`
    : `/api/v1/courses/${encodeURIComponent(courseCode)}/translation-coverage`
  const res = await authorizedFetch(path)
  const raw = await res.json()
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  return raw as TranslationCoverage | { locales: TranslationCoverage[] }
}

export async function patchMyContentLocale(
  courseCode: string,
  contentLocale: string | null,
): Promise<void> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/me/content-locale`,
    {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ contentLocale }),
    },
  )
  if (!res.ok) {
    const raw = await res.json().catch(() => ({}))
    throw new Error(readApiErrorMessage(raw))
  }
}

export async function fetchCourseGlossary(
  courseCode: string,
  targetLocale: string,
  sourceLocale = 'en',
): Promise<Array<{ id: string; sourceTerm: string; targetTerm: string }>> {
  const q = new URLSearchParams({ target_locale: targetLocale, source_locale: sourceLocale })
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/glossary?${q}`,
  )
  const raw = await res.json()
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  return (raw as { entries?: Array<{ id: string; sourceTerm: string; targetTerm: string }> }).entries ?? []
}

export async function addGlossaryEntry(
  courseCode: string,
  sourceTerm: string,
  targetTerm: string,
  targetLocale: string,
  sourceLocale = 'en',
): Promise<void> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/glossary`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ sourceTerm, targetTerm, targetLocale, sourceLocale }),
    },
  )
  if (!res.ok) {
    const raw = await res.json().catch(() => ({}))
    throw new Error(readApiErrorMessage(raw))
  }
}
