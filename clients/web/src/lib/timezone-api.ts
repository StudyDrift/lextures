import { apiUrl, authorizedFetch } from './api'
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

export async function fetchUserTimezone(): Promise<string | null> {
  const res = await authorizedFetch('/api/v1/settings/timezone')
  const raw: unknown = await res.json().catch(() => ({}))
  if (!res.ok) {
    throw new Error(readApiErrorMessage(raw))
  }
  const data = raw as { timezone?: string | null }
  return data.timezone ?? null
}

export async function updateUserTimezone(timezone: string | null): Promise<string | null> {
  const res = await authorizedFetch('/api/v1/settings/timezone', {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ timezone }),
  })
  const raw: unknown = await res.json().catch(() => ({}))
  if (!res.ok) {
    throw new Error(readApiErrorMessage(raw))
  }
  const data = raw as { timezone?: string | null }
  return data.timezone ?? null
}
