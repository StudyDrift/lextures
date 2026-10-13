import { authorizedFetch } from './api'
import { readApiErrorMessage } from './errors'

/** Simplified Settings for a managed learner (#775): no email, password, privacy, or AI fields. */
export type LearnerSettings = {
  displayName: string
  avatarUrl: string | null
  uiTheme: 'light' | 'dark'
}

function normalize(raw: Partial<LearnerSettings>): LearnerSettings {
  return {
    displayName: raw.displayName ?? '',
    avatarUrl: raw.avatarUrl ?? null,
    uiTheme: raw.uiTheme === 'dark' ? 'dark' : 'light',
  }
}

export async function fetchLearnerSettings(): Promise<LearnerSettings> {
  const res = await authorizedFetch('/api/v1/me/learner-settings')
  if (!res.ok) {
    const raw = await res.json().catch(() => ({}))
    throw new Error(readApiErrorMessage(raw) || `Failed to load settings (${res.status})`)
  }
  return normalize((await res.json()) as Partial<LearnerSettings>)
}

/** Patch any subset. `avatarUrl: ''` removes the picture. */
export async function updateLearnerSettings(body: {
  displayName?: string
  avatarUrl?: string
  uiTheme?: 'light' | 'dark'
}): Promise<LearnerSettings> {
  const res = await authorizedFetch('/api/v1/me/learner-settings', {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  if (!res.ok) {
    const raw = await res.json().catch(() => ({}))
    throw new Error(readApiErrorMessage(raw) || `Failed to save settings (${res.status})`)
  }
  return normalize((await res.json()) as Partial<LearnerSettings>)
}
