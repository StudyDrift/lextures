import { authorizedFetch } from './api'
import type { CoursePublic } from './courses-api'
import { courseSchema, parseApiResponse } from './courses-api-schemas'
import { readApiErrorMessage } from './errors'

export type ArchivedCourseRow = {
  id: string
  courseCode: string
  title: string
  archivedAt: string
  archivedByUserId?: string | null
  archivedByName?: string | null
  archivedByEmail?: string | null
}

export async function fetchArchivedCourses(): Promise<ArchivedCourseRow[]> {
  const res = await authorizedFetch('/api/v1/settings/archived-courses')
  const raw: unknown = await res.json().catch(() => ({}))
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  const data = raw as { courses?: ArchivedCourseRow[] }
  return data.courses ?? []
}

export async function restoreArchivedCourse(courseCode: string): Promise<CoursePublic> {
  const res = await authorizedFetch(
    `/api/v1/settings/archived-courses/${encodeURIComponent(courseCode)}/restore`,
    { method: 'POST' },
  )
  const raw = await res.json().catch(() => ({}))
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  return parseApiResponse('restoreArchivedCourse', courseSchema, raw)
}

export async function deleteArchivedCoursePermanently(courseCode: string): Promise<void> {
  const res = await authorizedFetch(
    `/api/v1/settings/archived-courses/${encodeURIComponent(courseCode)}`,
    { method: 'DELETE' },
  )
  if (res.ok) return
  const raw: unknown = await res.json().catch(() => ({}))
  throw new Error(readApiErrorMessage(raw))
}