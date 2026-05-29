import { authorizedFetch } from './api'

export type ReadingPreferences = {
  sttEnabled: boolean
  sttLanguage: string
}

export async function fetchReadingPreferences(): Promise<ReadingPreferences> {
  const res = await authorizedFetch('/api/v1/me/reading-preferences')
  if (!res.ok) {
    throw new Error('Failed to load reading preferences')
  }
  const data = (await res.json()) as { sttEnabled?: boolean; sttLanguage?: string }
  return {
    sttEnabled: data.sttEnabled === true,
    sttLanguage: data.sttLanguage ?? 'en-US',
  }
}

export async function patchReadingPreferences(
  patch: Partial<ReadingPreferences>,
): Promise<ReadingPreferences> {
  const res = await authorizedFetch('/api/v1/me/reading-preferences', {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      sttEnabled: patch.sttEnabled,
      sttLanguage: patch.sttLanguage,
    }),
  })
  if (!res.ok) {
    throw new Error('Failed to save reading preferences')
  }
  const data = (await res.json()) as { sttEnabled?: boolean; sttLanguage?: string }
  return {
    sttEnabled: data.sttEnabled === true,
    sttLanguage: data.sttLanguage ?? 'en-US',
  }
}

export type MyAccommodationEntry = {
  courseCode?: string
  speechToTextEnabled?: boolean
}

export async function fetchMyAccommodations(): Promise<MyAccommodationEntry[]> {
  const res = await authorizedFetch('/api/v1/me/accommodations')
  if (!res.ok) return []
  const data = (await res.json()) as { accommodations?: MyAccommodationEntry[] }
  return data.accommodations ?? []
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
