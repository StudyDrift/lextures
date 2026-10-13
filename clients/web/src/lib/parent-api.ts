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

export type ParentGradeItem = {
  itemId: string
  title: string
  category?: string | null
  score: string
  percentage?: number | null
  status: string
  dueAt?: string | null
  postedAt?: string | null
}

export type ParentCourseGradesRow = {
  courseCode: string
  title: string
  teacherEmail?: string | null
  teacherName?: string | null
  grades: Record<string, string>
  items: ParentGradeItem[]
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

export type ParentAttendanceDay = {
  date: string
  code: string
  codeLabel: string
  category: string
  period?: string | null
}

export type ParentAttendanceSummary = {
  termStart: string
  present: number
  absent: number
  tardy: number
  recentDays: ParentAttendanceDay[]
}

export type ParentBehaviorAward = {
  id: string
  categoryName?: string | null
  points?: number | null
  awardedAt?: string | null
}

export type ParentBehaviorReferral = {
  id: string
  categoryName?: string | null
  incidentAt?: string | null
  createdAt?: string | null
}

export type ParentBehaviorResponse = {
  studentId?: string
  totalPoints?: number
  awards?: ParentBehaviorAward[]
  referrals?: ParentBehaviorReferral[]
}

export type ParentReportCard = {
  id: string
  gradingPeriod: string
  pdfUrl?: string | null
  letterGrade?: string | null
  finalGradePct?: number | null
  releasedAt?: string | null
}

export type ParentReportCardsResponse = {
  reportCards: ParentReportCard[]
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
  const data = (await res.json()) as ParentGradesResponse
  return {
    courses: (data.courses ?? []).map((course) => ({
      ...course,
      items: course.items ?? [],
      grades: course.grades ?? {},
    })),
  }
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

export async function fetchParentStudentAttendanceSummary(
  studentId: string,
): Promise<ParentAttendanceSummary> {
  const res = await authorizedFetch(
    `/api/v1/parent/students/${encodeURIComponent(studentId)}/attendance-summary`,
  )
  if (!res.ok) {
    throw new Error(`Failed to load attendance (${res.status})`)
  }
  return (await res.json()) as ParentAttendanceSummary
}

export async function fetchParentStudentBehavior(studentId: string): Promise<ParentBehaviorResponse> {
  const res = await authorizedFetch(`/api/v1/parent/students/${encodeURIComponent(studentId)}/behavior`)
  if (!res.ok) {
    throw new Error(`Failed to load behavior (${res.status})`)
  }
  return (await res.json()) as ParentBehaviorResponse
}

export async function fetchParentStudentReportCards(
  studentId: string,
): Promise<ParentReportCardsResponse> {
  const res = await authorizedFetch(
    `/api/v1/parent/students/${encodeURIComponent(studentId)}/report-cards`,
  )
  if (!res.ok) {
    throw new Error(`Failed to load report cards (${res.status})`)
  }
  const data = (await res.json()) as ParentReportCardsResponse
  return { reportCards: data.reportCards ?? [] }
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
export type ParentNotificationPrefs = {
  gradePosted: boolean
  missingAssignment: boolean
  lowGradeThreshold: number | null
  attendanceEvent: boolean
}

export type ParentHomeWorkItem = {
  courseCode: string
  courseTitle: string
  itemId: string
  kind: string
  title: string
  dueAt?: string | null
  completedAt?: string | null
  overdue: boolean
  /** Due-this-week item the learner already turned in (#787). */
  completed?: boolean
}

export type ParentHomeCourse = {
  courseCode: string
  title: string
  enrollmentId?: string | null
  gradePercent?: number | null
}

export type ParentHomeChild = {
  studentUserId: string
  displayName: string
  courses: ParentHomeCourse[]
  dueThisWeek: ParentHomeWorkItem[]
  completedThisWeek: ParentHomeWorkItem[]
  timeSpentSeconds: number
}

export type ParentHomeResponse = {
  children: ParentHomeChild[]
  weekStart: string
  weekEnd: string
  timezone: string
}

/** Parent home (#772): per-child due/completed this week, grades, and time spent. */
export async function fetchParentHome(): Promise<ParentHomeResponse> {
  let tz = ''
  try {
    tz = Intl.DateTimeFormat().resolvedOptions().timeZone ?? ''
  } catch {
    tz = ''
  }
  const qs = tz ? `?tz=${encodeURIComponent(tz)}` : ''
  const res = await authorizedFetch(`/api/v1/parent/home${qs}`)
  if (!res.ok) {
    throw new Error(`Failed to load parent home (${res.status})`)
  }
  const data = (await res.json()) as ParentHomeResponse
  return {
    ...data,
    children: (data.children ?? []).map((c) => ({
      ...c,
      courses: c.courses ?? [],
      dueThisWeek: c.dueThisWeek ?? [],
      completedThisWeek: c.completedThisWeek ?? [],
      timeSpentSeconds: c.timeSpentSeconds ?? 0,
    })),
  }
}

/** Resolves true when the signed-in account has linked students or managed learners. */
export async function viewerHasParentChildren(): Promise<boolean> {
  try {
    const res = await fetchParentChildren()
    return res.children.length > 0
  } catch {
    return false
  }
}
