import { authorizedFetch } from './api'
import { readApiErrorMessage } from './errors'

export type EngagementEvent = {
  eventType: 'heartbeat' | 'video_progress' | 'scroll_depth' | 'login'
  courseId?: string
  itemId?: string
  itemType?: 'content_page' | 'video' | 'quiz' | 'assignment'
  value?: number
  occurredAt?: string
}

export type EngagementSummary = {
  enrollmentId: string
  avgTimeOnTaskPerSession: number
  loginsLast7Days: number
  avgVideoWatchPct?: number | null
  avgScrollDepth?: number | null
  dataAsOf: string
}

export type VideoDropoffPoint = {
  second: number
  pctStillWatching: number
}

export type VideoDropoffReport = {
  objectId: string
  totalWatchers: number
  dropoff: VideoDropoffPoint[]
  medianStopPct?: number | null
}

export type EngagementOverviewRow = {
  enrollmentId: string
  userId: string
  displayName: string
  loginsLast7Days: number
  avgTimeOnTaskMin: number
  avgVideoWatchPct?: number | null
  avgScrollDepth?: number | null
  engagementScore: number
}

/** POST `/api/v1/analytics/events` — batch engagement events. */
export async function postEngagementEvents(events: EngagementEvent[]): Promise<number> {
  if (events.length === 0) return 0
  const res = await authorizedFetch('/api/v1/analytics/events', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(events),
  })
  const raw = await res.json().catch(() => ({}))
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  return (raw as { stored: number }).stored
}

/** GET `/api/v1/courses/:code/enrollments/:eid/engagement` */
export async function fetchEnrollmentEngagement(
  courseCode: string,
  enrollmentId: string,
): Promise<EngagementSummary> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/enrollments/${encodeURIComponent(enrollmentId)}/engagement`,
  )
  const raw = await res.json().catch(() => ({}))
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  return raw as EngagementSummary
}

/** GET `/api/v1/courses/:code/analytics/video-dropoff/:objectId` */
export async function fetchVideoDropoff(
  courseCode: string,
  objectId: string,
): Promise<VideoDropoffReport> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/analytics/video-dropoff/${encodeURIComponent(objectId)}`,
  )
  const raw = await res.json().catch(() => ({}))
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  return raw as VideoDropoffReport
}

/** GET `/api/v1/courses/:code/analytics/engagement-overview` */
export async function fetchEngagementOverview(
  courseCode: string,
): Promise<EngagementOverviewRow[]> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/analytics/engagement-overview`,
  )
  const raw = (await res.json().catch(() => ({}))) as { students: EngagementOverviewRow[] }
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  return raw.students ?? []
}
