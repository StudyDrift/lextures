import type { NavDestination } from './types'
import { COURSE_NAV_CORE } from './registry-course-core'
import { COURSE_NAV_SECONDARY } from './registry-course-secondary'

/**
 * In-course destinations. Routes use `:courseCode` placeholder.
 *
 * Legacy sections match current IA. V2 uses task-based Teach / Engage /
 * Assess & analyse / Manage (instructor) and Participate / My work (student).
 * Gradebook priority is intentionally first in grades-insights (AC-1).
 */
export const COURSE_NAV: NavDestination[] = [
  ...COURSE_NAV_CORE,
  ...COURSE_NAV_SECONDARY,
]
