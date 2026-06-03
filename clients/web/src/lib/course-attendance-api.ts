import { authorizedFetch } from './api'

export type CollectionMethod = 'roll_call' | 'self_report'

export type AttendanceStatus =
  | 'present'
  | 'absent'
  | 'tardy'
  | 'excused'
  | 'not_recorded'

export type AttendanceSession = {
  id: string
  title: string
  collectionMethod: CollectionMethod
  sessionDate: string
  status: 'open' | 'closed'
  gradebookEnabled: boolean
  sectionId?: string
  structureItemId?: string
  opensAt?: string
  closesAt?: string
  pointsPossible?: number
  closedAt?: string
}

export type AttendanceRecord = {
  studentUserId: string
  displayName?: string
  status: AttendanceStatus
  source?: string
  recordedAt?: string
  recordedBy?: string
}

export type AttendanceSessionDetail = AttendanceSession & {
  records?: AttendanceRecord[]
  myRecord?: AttendanceRecord
  canSelfReport?: boolean
}

export type CreateAttendanceSessionInput = {
  collectionMethod: CollectionMethod
  title?: string
  sessionDate?: string
  sectionId?: string | null
  gradebookEnabled?: boolean
  pointsPossible?: number
  opensAt?: string
  closesAt?: string
}

export async function listAttendanceSessions(courseCode: string): Promise<AttendanceSession[]> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/attendance/sessions`,
  )
  if (!res.ok) {
    throw new Error(`Failed to load attendance sessions (${res.status})`)
  }
  const body = (await res.json()) as { sessions: AttendanceSession[] }
  return body.sessions ?? []
}

export async function createAttendanceSession(
  courseCode: string,
  input: CreateAttendanceSessionInput,
): Promise<AttendanceSession> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/attendance/sessions`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(input),
    },
  )
  if (!res.ok) {
    const err = (await res.json().catch(() => ({}))) as { error?: { message?: string } }
    throw new Error(err.error?.message ?? `Failed to create session (${res.status})`)
  }
  return (await res.json()) as AttendanceSession
}

export async function getAttendanceSession(
  courseCode: string,
  sessionId: string,
): Promise<AttendanceSessionDetail> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/attendance/sessions/${encodeURIComponent(sessionId)}`,
  )
  if (!res.ok) {
    throw new Error(`Failed to load session (${res.status})`)
  }
  return (await res.json()) as AttendanceSessionDetail
}

export async function saveAttendanceRecords(
  courseCode: string,
  sessionId: string,
  records: Array<{ studentUserId: string; status: AttendanceStatus; source?: string }>,
): Promise<{ saved: number; message: string }> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/attendance/sessions/${encodeURIComponent(sessionId)}/records`,
    {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ records }),
    },
  )
  if (!res.ok) {
    const err = (await res.json().catch(() => ({}))) as { error?: { message?: string } }
    throw new Error(err.error?.message ?? `Failed to save records (${res.status})`)
  }
  return (await res.json()) as { saved: number; message: string }
}

export async function selfReportAttendance(
  courseCode: string,
  sessionId: string,
  status: 'present' | 'tardy' = 'present',
): Promise<void> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/attendance/sessions/${encodeURIComponent(sessionId)}/self-report`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ status }),
    },
  )
  if (!res.ok) {
    const err = (await res.json().catch(() => ({}))) as { error?: { message?: string } }
    throw new Error(err.error?.message ?? `Failed to check in (${res.status})`)
  }
}

export async function closeAttendanceSession(
  courseCode: string,
  sessionId: string,
  finalizeMissingAsAbsent = true,
): Promise<AttendanceSession> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/attendance/sessions/${encodeURIComponent(sessionId)}/close`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ finalizeMissingAsAbsent }),
    },
  )
  if (!res.ok) {
    const err = (await res.json().catch(() => ({}))) as { error?: { message?: string } }
    throw new Error(err.error?.message ?? `Failed to close session (${res.status})`)
  }
  return (await res.json()) as AttendanceSession
}

export const ATTENDANCE_STATUS_OPTIONS: { value: AttendanceStatus; label: string }[] = [
  { value: 'present', label: 'Present' },
  { value: 'absent', label: 'Absent' },
  { value: 'tardy', label: 'Tardy' },
  { value: 'excused', label: 'Excused' },
  { value: 'not_recorded', label: 'Not recorded' },
]
