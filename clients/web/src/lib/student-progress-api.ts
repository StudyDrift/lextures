import { authorizedFetch } from './api'
import { readApiErrorMessage } from './errors'

export type StudentProgressSummary = {
  enrollmentId: string
  courseId: string
  studentUserId: string
  studentDisplayName: string
  assignmentsSubmittedPct: number
  modulesViewedPct: number
  avgQuizScore?: number | null
  avgGradePercent?: number | null
  lastActiveAt?: string | null
  missingCount: number
  dataAsOf: string
  staleMinutes: number
  canManageNotes: boolean
}

export type StudentProgressMissingItem = {
  itemId: string
  title: string
  kind: string
  dueAt?: string | null
  daysOverdue: number
  gradeStatus: string
}

export type StudentProgressAssignmentRow = {
  itemId: string
  title: string
  dueAt?: string | null
  submittedAt?: string | null
  grade: string
  status: string
}

export type StudentProgressQuizRow = {
  attemptId: string
  itemId: string
  title: string
  submittedAt: string
  scorePercent?: number | null
}

export type StudentProgressNote = {
  id: string
  authorId: string
  noteText: string
  createdAt: string
  updatedAt: string
}

export type StudentProgressResponse = {
  summary: StudentProgressSummary
  missing: StudentProgressMissingItem[]
  assignments: StudentProgressAssignmentRow[]
  quizzes: StudentProgressQuizRow[]
  notes?: StudentProgressNote[]
}

export type StudentProgressActivityEvent = {
  occurredAt: string
  kind: string
  label: string
  detail?: string
}

export async function fetchStudentProgress(
  courseCode: string,
  enrollmentId: string,
): Promise<StudentProgressResponse> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/enrollments/${encodeURIComponent(enrollmentId)}/progress`,
  )
  const raw = (await res.json()) as StudentProgressResponse
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  return raw
}

export async function fetchStudentProgressActivity(
  courseCode: string,
  enrollmentId: string,
  cursor?: string,
): Promise<{ events: StudentProgressActivityEvent[]; nextCursor?: string | null }> {
  const q = cursor ? `?cursor=${encodeURIComponent(cursor)}` : ''
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/enrollments/${encodeURIComponent(enrollmentId)}/progress/activity${q}`,
  )
  const raw = (await res.json()) as { events: StudentProgressActivityEvent[]; nextCursor?: string | null }
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  return raw
}

export async function createStudentProgressNote(
  courseCode: string,
  enrollmentId: string,
  noteText: string,
): Promise<StudentProgressNote> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/enrollments/${encodeURIComponent(enrollmentId)}/notes`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ noteText }),
    },
  )
  const raw = (await res.json()) as StudentProgressNote
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  return raw
}

export async function updateStudentProgressNote(
  courseCode: string,
  enrollmentId: string,
  noteId: string,
  noteText: string,
): Promise<StudentProgressNote> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/enrollments/${encodeURIComponent(enrollmentId)}/notes/${encodeURIComponent(noteId)}`,
    {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ noteText }),
    },
  )
  const raw = (await res.json()) as StudentProgressNote
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  return raw
}

export async function deleteStudentProgressNote(
  courseCode: string,
  enrollmentId: string,
  noteId: string,
): Promise<void> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/enrollments/${encodeURIComponent(enrollmentId)}/notes/${encodeURIComponent(noteId)}`,
    { method: 'DELETE' },
  )
  if (!res.ok) {
    const raw = await res.json().catch(() => null)
    throw new Error(readApiErrorMessage(raw))
  }
}
