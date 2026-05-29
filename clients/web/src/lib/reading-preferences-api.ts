import { authorizedFetch } from './api'

export type ReadingPreferences = {
  ttsEnabled: boolean
  ttsSpeed: number
  ttsVoiceName: string | null
  updatedAt?: string | null
}

const defaults: ReadingPreferences = {
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
    ttsEnabled: body.ttsEnabled ?? false,
    ttsSpeed: body.ttsSpeed ?? 1,
    ttsVoiceName: body.ttsVoiceName ?? null,
    updatedAt: body.updatedAt ?? null,
  }
}

export async function patchReadingPreferences(
  patch: Partial<Pick<ReadingPreferences, 'ttsEnabled' | 'ttsSpeed' | 'ttsVoiceName'>>,
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

export type MyAccommodationSummary = {
  accommodations: Array<{ ttsEnabled?: boolean }>
}

export async function fetchMyAccommodationSummary(): Promise<MyAccommodationSummary> {
  const res = await authorizedFetch('/api/v1/me/accommodations')
  if (!res.ok) {
    return { accommodations: [] }
  }
  return (await res.json()) as MyAccommodationSummary
}
