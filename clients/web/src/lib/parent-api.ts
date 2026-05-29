import { authorizedFetch } from './api'

export type ParentChildSummary = {
  linkId: string
  studentUserId: string
  displayName: string | null
  email: string
  relationship: string
  status: string
  linkedAt: string
}

export type ParentChildrenResponse = {
  children: ParentChildSummary[]
}

export type ParentCourseGradesRow = {
  courseCode: string
  title: string
  grades: Record<string, string>
}

export type ParentGradesResponse = {
  courses: ParentCourseGradesRow[]
}

export type ParentAssignmentRow = {
  courseCode: string
  courseTitle: string
  itemId: string
  kind: string
  title: string
  dueAt?: string | null
}

export type ParentAssignmentsResponse = {
  assignments: ParentAssignmentRow[]
}

export async function fetchParentChildren(): Promise<ParentChildrenResponse> {
  const res = await authorizedFetch('/api/v1/parent/children')
  if (!res.ok) {
    throw new Error(`Failed to load children (${res.status})`)
  }
  return (await res.json()) as ParentChildrenResponse
}

export async function fetchParentStudentGrades(studentId: string): Promise<ParentGradesResponse> {
  const res = await authorizedFetch(`/api/v1/parent/students/${encodeURIComponent(studentId)}/grades`)
  if (!res.ok) {
    throw new Error(`Failed to load grades (${res.status})`)
  }
  return (await res.json()) as ParentGradesResponse
}

export async function fetchParentStudentAssignments(
  studentId: string,
): Promise<ParentAssignmentsResponse> {
  const res = await authorizedFetch(`/api/v1/parent/students/${encodeURIComponent(studentId)}/assignments`)
  if (!res.ok) {
    throw new Error(`Failed to load assignments (${res.status})`)
  }
  return (await res.json()) as ParentAssignmentsResponse
}

export type ParentWeeklySummaryItem = {
  childName: string
  courseCode: string
  courseTitle: string
  itemId: string
  kind: string
  title: string
  dueAt?: string | null
}

export type ParentWeeklySummaryResponse = {
  items: ParentWeeklySummaryItem[]
  weekStart: string
  weekEnd: string
}

export async function fetchParentWeeklySummary(): Promise<ParentWeeklySummaryResponse> {
  const res = await authorizedFetch('/api/v1/parent/weekly-summary')
  if (!res.ok) {
    throw new Error(`Failed to load weekly summary (${res.status})`)
  }
  return (await res.json()) as ParentWeeklySummaryResponse
}

export type ParentNotificationPrefs = {
  gradePosted: boolean
  missingAssignment: boolean
  lowGradeThreshold: number | null
  attendanceEvent: boolean
}

export async function fetchParentNotificationPrefs(): Promise<ParentNotificationPrefs> {
  const res = await authorizedFetch('/api/v1/parent/notification-prefs')
  if (!res.ok) {
    throw new Error(`Failed to load notification prefs (${res.status})`)
  }
  return (await res.json()) as ParentNotificationPrefs
}

export async function updateParentNotificationPrefs(
  patch: Partial<ParentNotificationPrefs> & { clearThreshold?: boolean },
): Promise<ParentNotificationPrefs> {
  const res = await authorizedFetch('/api/v1/parent/notification-prefs', {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(patch),
  })
  if (!res.ok) {
    throw new Error(`Failed to update notification prefs (${res.status})`)
  }
  return (await res.json()) as ParentNotificationPrefs
}
