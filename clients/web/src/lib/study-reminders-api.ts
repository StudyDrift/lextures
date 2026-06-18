import { authorizedFetch } from './api'

export type ReminderConfig = {
  dailyGoalMinutes: number
  reminderTime: string
  reminderChannels: string[]
  weeklySummary: boolean
  enabled: boolean
  pausedUntil?: string
  minutesStudiedToday: number
  goalMetToday: boolean
  streakAtRiskBanner: boolean
}

export async function fetchReminderConfig(): Promise<ReminderConfig> {
  const res = await authorizedFetch('/api/v1/me/reminder-config')
  const raw: unknown = await res.json().catch(() => ({}))
  if (!res.ok) {
    throw new Error(
      typeof raw === 'object' && raw && 'message' in raw
        ? String((raw as { message: string }).message)
        : 'Could not load study reminder settings.',
    )
  }
  return raw as ReminderConfig
}

export async function patchReminderConfig(patch: Partial<ReminderConfig>): Promise<ReminderConfig> {
  const res = await authorizedFetch('/api/v1/me/reminder-config', {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(patch),
  })
  const raw: unknown = await res.json().catch(() => ({}))
  if (!res.ok) {
    throw new Error(
      typeof raw === 'object' && raw && 'message' in raw
        ? String((raw as { message: string }).message)
        : 'Could not save study reminder settings.',
    )
  }
  return raw as ReminderConfig
}

export async function pauseReminders(days: number): Promise<ReminderConfig> {
  const res = await authorizedFetch('/api/v1/me/reminder-config/pause', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ days }),
  })
  const raw: unknown = await res.json().catch(() => ({}))
  if (!res.ok) {
    throw new Error('Could not pause reminders.')
  }
  return raw as ReminderConfig
}

export function formatReminderTimeLabel(value: string): string {
  const [h, m] = value.split(':').map((x) => parseInt(x, 10))
  if (Number.isNaN(h) || Number.isNaN(m)) return value
  const d = new Date()
  d.setHours(h, m, 0, 0)
  return d.toLocaleTimeString(undefined, { hour: 'numeric', minute: '2-digit' })
}
