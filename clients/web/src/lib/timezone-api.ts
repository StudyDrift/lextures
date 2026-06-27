import { apiUrl } from './api'
import { readApiErrorMessage } from './errors'

export type TimezoneEntry = {
  id: string
  offsetMinutes: number
}

export async function fetchTimezones(): Promise<TimezoneEntry[]> {
  const res = await fetch(apiUrl('/api/v1/timezones'))
  const raw: unknown = await res.json().catch(() => ({}))
  if (!res.ok) {
    throw new Error(readApiErrorMessage(raw))
  }
  const data = raw as { timezones?: TimezoneEntry[] }
  return data.timezones ?? []
}
