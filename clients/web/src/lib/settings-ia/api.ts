import { authorizedFetch } from '../api'
import type { BlastRadius, EffectiveValue, SettingsIndexEntry } from './types'

export async function fetchSettingsIndexApi(): Promise<SettingsIndexEntry[]> {
  const res = await authorizedFetch('/api/v1/settings/index')
  if (!res.ok) {
    throw new Error(`settings index failed: ${res.status}`)
  }
  const data = (await res.json()) as { entries?: SettingsIndexEntry[] }
  return data.entries ?? []
}

export async function fetchSettingsBlastRadius(key: string): Promise<BlastRadius> {
  const res = await authorizedFetch(
    `/api/v1/settings/${encodeURIComponent(key)}/blast-radius`,
  )
  if (!res.ok) {
    throw new Error(`blast-radius failed: ${res.status}`)
  }
  return (await res.json()) as BlastRadius
}

export async function fetchSettingsEffective(key: string): Promise<EffectiveValue> {
  const res = await authorizedFetch(
    `/api/v1/settings/${encodeURIComponent(key)}/effective`,
  )
  if (!res.ok) {
    throw new Error(`effective value failed: ${res.status}`)
  }
  return (await res.json()) as EffectiveValue
}
