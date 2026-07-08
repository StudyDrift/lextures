import { authorizedFetch } from './api'
import { readApiErrorMessage } from './errors'

/** Canonical intro course URL code (matches server introcourse.CourseCode). */
export const INTRO_COURSE_CODE = 'C-WLCOME'

export type IntroCourseModuleStatus = 'done' | 'current' | 'upcoming'

export type IntroCourseModuleProgress = {
  slug: string
  title: string
  status: IntroCourseModuleStatus
}

export type IntroCourseNextItem = {
  slug: string
  title: string
  route: string
}

export type IntroCourseProgress = {
  enrolled: boolean
  courseCode?: string
  modulesComplete: number
  modulesTotal: number
  percent: number
  runningGrade?: number | null
  completedAt?: string | null
  credentialId?: string | null
  nextItem?: IntroCourseNextItem | null
  modules?: IntroCourseModuleProgress[]
  welcomeBannerDismissed?: boolean
  celebrationSeen?: boolean
}

export type IntroCourseCardState = 'hidden' | 'loading' | 'error' | 'not-started' | 'in-progress' | 'completed'

export function introCourseCardState(progress: IntroCourseProgress | null, loading: boolean, error: boolean): IntroCourseCardState {
  if (loading) return 'loading'
  if (error || !progress) return 'error'
  if (!progress.enrolled) return 'hidden'
  if (progress.completedAt) return 'completed'
  if (progress.modulesComplete <= 0) return 'not-started'
  return 'in-progress'
}

export function shouldShowIntroWelcomeBanner(progress: IntroCourseProgress | null): boolean {
  if (!progress?.enrolled || progress.completedAt) return false
  if (progress.welcomeBannerDismissed) return false
  return progress.modulesComplete <= 0 && progress.percent <= 0
}

export function shouldShowIntroCelebration(progress: IntroCourseProgress | null): boolean {
  if (!progress?.enrolled || !progress.completedAt) return false
  return !progress.celebrationSeen
}

export async function fetchIntroCourseProgress(): Promise<IntroCourseProgress> {
  const res = await authorizedFetch('/api/v1/me/intro-course')
  const raw: unknown = await res.json().catch(() => ({}))
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  return raw as IntroCourseProgress
}

export async function dismissIntroWelcomeBanner(): Promise<void> {
  const res = await authorizedFetch('/api/v1/me/intro-course/welcome-banner-dismissed', {
    method: 'PUT',
  })
  if (!res.ok && res.status !== 204) {
    const raw: unknown = await res.json().catch(() => ({}))
    throw new Error(readApiErrorMessage(raw))
  }
}

export async function markIntroCelebrationSeen(): Promise<void> {
  const res = await authorizedFetch('/api/v1/me/intro-course/celebration-seen', {
    method: 'PUT',
  })
  if (!res.ok && res.status !== 204) {
    const raw: unknown = await res.json().catch(() => ({}))
    throw new Error(readApiErrorMessage(raw))
  }
}

export function introCourseFallbackHref(courseCode = INTRO_COURSE_CODE): string {
  return `/courses/${encodeURIComponent(courseCode)}`
}