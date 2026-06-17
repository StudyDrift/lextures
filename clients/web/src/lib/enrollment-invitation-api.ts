import { authorizedFetch } from './api'
import { readApiErrorMessage } from './errors'

export async function approveEnrollmentInvitation(
  courseCode: string,
  enrollmentId: string,
): Promise<void> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/enrollments/${encodeURIComponent(enrollmentId)}/invitation/approve`,
    { method: 'POST' },
  )
  if (!res.ok) {
    const body: unknown = await res.json().catch(() => ({}))
    throw new Error(readApiErrorMessage(body))
  }
}

export async function declineEnrollmentInvitation(
  courseCode: string,
  enrollmentId: string,
): Promise<void> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/enrollments/${encodeURIComponent(enrollmentId)}/invitation/decline`,
    { method: 'POST' },
  )
  if (!res.ok) {
    const body: unknown = await res.json().catch(() => ({}))
    throw new Error(readApiErrorMessage(body))
  }
}