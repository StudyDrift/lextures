import { authorizedFetch } from './api'
import { readApiErrorMessage } from './errors'

export type LearningActivityReport = {
  range: { from: string; to: string }
  summary: {
    totalEvents: number
    uniqueUsers: number
    uniqueCourses: number
  }
  byDay: {
    day: string
    courseVisit: number
    contentOpen: number
    contentLeave: number
  }[]
  byEventKind: { eventKind: string; count: number }[]
  topCourses: {
    courseId: string
    courseCode: string
    title: string
    eventCount: number
  }[]
}

export type ReportSchedule = {
  id: string
  reportType: string
  courseId?: string
  parameters: Record<string, string>
  recipients: string[]
  cadence: 'daily' | 'weekly' | 'monthly'
  cadenceDetail?: Record<string, unknown>
  enabled: boolean
  lastRunAt?: string
  nextRunAt: string
  createdAt: string
}

async function parseJson(res: Response): Promise<unknown> {
  return res.json().catch(() => ({}))
}

/** GET `/api/v1/reports/learning-activity` — requires `global:app:reports:view`. */
export async function fetchLearningActivityReport(params?: {
  from?: string
  to?: string
}): Promise<LearningActivityReport> {
  const search = new URLSearchParams()
  if (params?.from) search.set('from', params.from)
  if (params?.to) search.set('to', params.to)
  const qs = search.toString()
  const res = await authorizedFetch(`/api/v1/reports/learning-activity${qs ? `?${qs}` : ''}`)
  const raw = await parseJson(res)
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  return raw as LearningActivityReport
}

/** GET `/api/v1/reports/learning-activity/export.pdf` — downloads PDF. */
export async function downloadLearningActivityPDF(): Promise<void> {
  const res = await authorizedFetch('/api/v1/reports/learning-activity/export.pdf')
  if (!res.ok) {
    const raw = await parseJson(res)
    throw new Error(readApiErrorMessage(raw))
  }
  const blob = await res.blob()
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `learning-activity-${new Date().toISOString().slice(0, 10)}.pdf`
  a.click()
  URL.revokeObjectURL(url)
}

/** GET `/api/v1/courses/{code}/reports/{type}/export.pdf` — downloads PDF. */
export async function downloadCoursePDF(courseCode: string, reportType: string, params?: Record<string, string>): Promise<void> {
  const search = new URLSearchParams(params)
  const qs = search.toString()
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/reports/${encodeURIComponent(reportType)}/export.pdf${qs ? `?${qs}` : ''}`,
  )
  if (!res.ok) {
    const raw = await parseJson(res)
    throw new Error(readApiErrorMessage(raw))
  }
  const blob = await res.blob()
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `${reportType}-${courseCode}-${new Date().toISOString().slice(0, 10)}.pdf`
  a.click()
  URL.revokeObjectURL(url)
}

/** GET `/api/v1/reports/schedules` */
export async function listReportSchedules(): Promise<ReportSchedule[]> {
  const res = await authorizedFetch('/api/v1/reports/schedules')
  const raw = await parseJson(res)
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  return raw as ReportSchedule[]
}

/** POST `/api/v1/reports/schedules` */
export async function createReportSchedule(input: {
  reportType: string
  courseId?: string
  parameters?: Record<string, string>
  recipients: string[]
  cadence: string
  cadenceDetail?: Record<string, unknown>
}): Promise<ReportSchedule> {
  const res = await authorizedFetch('/api/v1/reports/schedules', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(input),
  })
  const raw = await parseJson(res)
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  return raw as ReportSchedule
}

/** PUT `/api/v1/reports/schedules/{id}` */
export async function updateReportSchedule(
  id: string,
  input: { recipients?: string[]; cadence?: string; enabled?: boolean },
): Promise<ReportSchedule> {
  const res = await authorizedFetch(`/api/v1/reports/schedules/${encodeURIComponent(id)}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(input),
  })
  const raw = await parseJson(res)
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  return raw as ReportSchedule
}

/** DELETE `/api/v1/reports/schedules/{id}` */
export async function deleteReportSchedule(id: string): Promise<void> {
  const res = await authorizedFetch(`/api/v1/reports/schedules/${encodeURIComponent(id)}`, {
    method: 'DELETE',
  })
  if (!res.ok && res.status !== 204) {
    const raw = await parseJson(res)
    throw new Error(readApiErrorMessage(raw))
  }
}
