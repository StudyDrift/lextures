import { authorizedFetch } from './api'
import { readApiErrorMessage } from './errors'

export type IntroCourseBackfillStatus = {
  startedAt: string | null
  completedAt: string | null
  enrolledCount?: number
  remaining: number
}

export type IntroCourseAdminStatus = {
  enabled: boolean
  coursePresent: boolean
  courseId?: string
  courseCode: string
  contentVersion: number
  moduleCount: number
  lastSyncedAt?: string | null
  lastSyncResult?: string | null
  lastValidatedAt?: string | null
  lastValidationResult?: string | null
  availableLocales: string[]
  localeCoverage: Record<string, number>
  backfill: IntroCourseBackfillStatus
}

export type IntroCourseModuleFunnel = {
  moduleSlug: string
  moduleTitle: string
  quizAttempted: number
  attemptRate: number
}

export type IntroCourseAdminAnalytics = {
  enrolled: number
  completed: number
  completionRate: number
  perModuleFunnel: IntroCourseModuleFunnel[]
  dropOffModuleSlug?: string
  avgTimeToCompleteHours?: number | null
}

export type IntroCourseResyncResult = {
  courseId: string
  status: string
  contentVersion: number
  contentSkipped: boolean
  modulesSynced: number
  pagesSynced: number
  quizzesSynced: number
  itemsArchived: number
}

async function parseJson<T>(res: Response): Promise<T> {
  const raw: unknown = await res.json().catch(() => ({}))
  if (!res.ok) {
    throw new Error(readApiErrorMessage(raw))
  }
  return raw as T
}

export async function fetchIntroCourseAdminStatus(): Promise<IntroCourseAdminStatus> {
  const res = await authorizedFetch('/api/v1/admin/intro-course')
  return parseJson(res)
}

export async function fetchIntroCourseAdminAnalytics(): Promise<IntroCourseAdminAnalytics> {
  const res = await authorizedFetch('/api/v1/admin/intro-course/analytics')
  return parseJson(res)
}

export async function resyncIntroCourse(): Promise<IntroCourseResyncResult> {
  const res = await authorizedFetch('/api/v1/admin/intro-course/resync', { method: 'POST' })
  return parseJson(res)
}

export async function startIntroCourseBackfill(): Promise<{ startedAt: string | null; remaining: number }> {
  const res = await authorizedFetch('/api/v1/admin/intro-course/backfill', { method: 'POST' })
  return parseJson(res)
}