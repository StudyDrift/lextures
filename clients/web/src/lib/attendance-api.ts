import { authorizedFetch } from './api'

export type AttendanceCode = {
  id: string
  orgId: string
  code: string
  label: string
  stateCode?: string | null
  category: 'present' | 'absent' | 'tardy' | 'other'
}

export type AttendanceRecord = {
  id: string
  studentId: string
  sectionId: string
  date: string
  period?: string | null
  codeId: string
  code: string
  codeLabel: string
  category: string
  note?: string | null
  recordedBy?: string | null
  recordedAt: string
  updatedAt: string
}

export type RosterEntry = {
  userId: string
  email: string
  displayName?: string | null
}

export type SectionAttendanceResponse = {
  records: AttendanceRecord[]
  roster: RosterEntry[]
  codes: AttendanceCode[]
}

export type UpsertRecord = {
  studentId: string
  codeId: string
  period?: string | null
  note?: string | null
  schoolId?: string | null
}

export type BatchSaveResponse = {
  saved: number
  message: string
}

export type DashboardEntry = {
  sectionId: string
  sectionCode: string
  courseName: string
  date: string
  totalStudents: number
  presentCount: number
  absentCount: number
  tardyCount: number
  notTaken: boolean
}

export type AttendanceDashboardResponse = {
  entries: DashboardEntry[]
  date: string
}

export type AttendanceCodesResponse = {
  codes: AttendanceCode[]
}

export async function fetchAttendanceDashboard(
  unitId: string,
  date?: string,
): Promise<AttendanceDashboardResponse> {
  const params = date ? `?date=${encodeURIComponent(date)}` : ''
  const res = await authorizedFetch(
    `/api/v1/org-units/${encodeURIComponent(unitId)}/attendance/dashboard${params}`,
  )
  if (!res.ok) {
    throw new Error(`Failed to load dashboard (${res.status})`)
  }
  return (await res.json()) as AttendanceDashboardResponse
}

export async function exportAttendance(
  orgId: string,
  startDate: string,
  endDate: string,
  format: 'csv' | 'calpads' = 'csv',
): Promise<Blob> {
  const res = await authorizedFetch(
    `/api/v1/admin/orgs/${encodeURIComponent(orgId)}/attendance/export`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ startDate, endDate, format }),
    },
  )
  if (!res.ok) {
    throw new Error(`Export failed (${res.status})`)
  }
  return res.blob()
}
