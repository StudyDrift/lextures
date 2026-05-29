import { authorizedFetch } from './api'

export type ReadingPreferences = {
  sttEnabled: boolean
  sttLanguage: string
  ttsEnabled: boolean
  ttsSpeed: number
  ttsVoiceName: string | null
  updatedAt?: string | null
}

const defaults: ReadingPreferences = {
  sttEnabled: false,
  sttLanguage: 'en-US',
  ttsEnabled: false,
  ttsSpeed: 1,
  ttsVoiceName: null,
}

export async function fetchReadingPreferences(): Promise<ReadingPreferences> {
  const res = await authorizedFetch('/api/v1/me/reading-preferences')
  if (!res.ok) {
    return defaults
  }
  const body = (await res.json()) as Partial<ReadingPreferences>
  return {
    sttEnabled: body.sttEnabled === true,
    sttLanguage: body.sttLanguage ?? 'en-US',
    ttsEnabled: body.ttsEnabled ?? false,
    ttsSpeed: body.ttsSpeed ?? 1,
    ttsVoiceName: body.ttsVoiceName ?? null,
    updatedAt: body.updatedAt ?? null,
  }
}

export async function patchReadingPreferences(
  patch: Partial<
    Pick<
      ReadingPreferences,
      'sttEnabled' | 'sttLanguage' | 'ttsEnabled' | 'ttsSpeed' | 'ttsVoiceName'
    >
  >,
): Promise<ReadingPreferences> {
  const res = await authorizedFetch('/api/v1/me/reading-preferences', {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(patch),
  })
  if (!res.ok) {
    throw new Error('Could not save reading preferences.')
  }
  const body = (await res.json()) as ReadingPreferences
  return body
}

export type MyAccommodationEntry = {
  courseCode?: string
  speechToTextEnabled?: boolean
  ttsEnabled?: boolean
}

export type MyAccommodationSummary = {
  accommodations: MyAccommodationEntry[]
}

export async function fetchMyAccommodations(): Promise<MyAccommodationEntry[]> {
  const res = await authorizedFetch('/api/v1/me/accommodations')
  if (!res.ok) return []
  const data = (await res.json()) as { accommodations?: MyAccommodationEntry[] }
  return data.accommodations ?? []
}

export async function fetchMyAccommodationSummary(): Promise<MyAccommodationSummary> {
  const res = await authorizedFetch('/api/v1/me/accommodations')
  if (!res.ok) {
    return { accommodations: [] }
  }
  return (await res.json()) as MyAccommodationSummary
}

export function accommodationSpeechToTextEnabled(
  entries: MyAccommodationEntry[],
  courseCode?: string,
): boolean {
  if (entries.some((e) => e.speechToTextEnabled && !e.courseCode)) return true
  if (courseCode) {
    return entries.some((e) => e.speechToTextEnabled && e.courseCode === courseCode)
  }
  return entries.some((e) => e.speechToTextEnabled)
}
