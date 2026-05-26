import { authorizedFetch } from './api'
import { readApiErrorMessage } from './errors'

export type StudyStats = {
  optedIn: boolean
  loginStreakDays: number
  timeOnTaskSecondsThisWeek: number
  weeklyGoalHours?: number
  goalProgressHours: number
  goalRemainingHours?: number
  studyEfficiency?: number
  lowStudyEfficiency: boolean
  timeAllocation: { moduleId: string; moduleTitle: string; minutes: number }[]
  weekStart: string
  weekEnd: string
}

export type JournalEntry = {
  id: string
  courseId?: string
  entryText: string
  createdAt: string
}

export type CoachingTip = {
  id: string
  tipText: string
  weekOf: string
  rating?: number
  createdAt: string
}

async function parseError(res: Response, fallback: string): Promise<never> {
  const raw: unknown = await res.json().catch(() => null)
  throw new Error(readApiErrorMessage(raw) || fallback)
}

export async function fetchStudyStats(): Promise<StudyStats> {
  const res = await authorizedFetch('/api/v1/me/study-stats')
  if (!res.ok) await parseError(res, 'Could not load study stats.')
  const data = (await res.json()) as StudyStats
  return {
    ...data,
    timeAllocation: data.timeAllocation ?? [],
  }
}

export async function fetchStudyGoal(): Promise<{ weeklyHours: number; optedIn: boolean }> {
  const res = await authorizedFetch('/api/v1/me/study-goal')
  if (!res.ok) await parseError(res, 'Could not load study goal.')
  return res.json() as Promise<{ weeklyHours: number; optedIn: boolean }>
}

export async function putStudyGoal(body: {
  weeklyHours?: number
  optedIn?: boolean
}): Promise<{ weeklyHours: number; optedIn: boolean }> {
  const res = await authorizedFetch('/api/v1/me/study-goal', {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  if (!res.ok) await parseError(res, 'Could not save study goal.')
  return res.json() as Promise<{ weeklyHours: number; optedIn: boolean }>
}

export async function fetchReflectionJournal(): Promise<JournalEntry[]> {
  const res = await authorizedFetch('/api/v1/me/reflection-journal')
  if (!res.ok) await parseError(res, 'Could not load journal.')
  const data = (await res.json()) as { entries: JournalEntry[] }
  return data.entries ?? []
}

export async function createReflectionJournalEntry(body: {
  entryText: string
  courseId?: string
}): Promise<string> {
  const res = await authorizedFetch('/api/v1/me/reflection-journal', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  if (!res.ok) await parseError(res, 'Could not save journal entry.')
  const data = (await res.json()) as { id: string }
  return data.id
}

export async function deleteReflectionJournalEntry(id: string): Promise<void> {
  const res = await authorizedFetch(`/api/v1/me/reflection-journal/${encodeURIComponent(id)}`, {
    method: 'DELETE',
  })
  if (!res.ok) await parseError(res, 'Could not delete entry.')
}

export async function fetchCoachingTips(): Promise<{
  latest: CoachingTip | null
  history: CoachingTip[]
}> {
  const res = await authorizedFetch('/api/v1/me/coaching-tips')
  if (!res.ok) await parseError(res, 'Could not load coaching tips.')
  return res.json() as Promise<{ latest: CoachingTip | null; history: CoachingTip[] }>
}

export async function rateCoachingTip(id: string, rating: -1 | 1): Promise<void> {
  const res = await authorizedFetch(`/api/v1/me/coaching-tips/${encodeURIComponent(id)}/rating`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ rating }),
  })
  if (!res.ok) await parseError(res, 'Could not save rating.')
}
