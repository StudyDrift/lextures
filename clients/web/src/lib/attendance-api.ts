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

export async function fetchSectionAttendance(
  sectionId: string,
  date: string,
): Promise<SectionAttendanceResponse> {
  const res = await authorizedFetch(
    `/api/v1/sections/${encodeURIComponent(sectionId)}/attendance/${encodeURIComponent(date)}`,
  )
  if (!res.ok) {
    throw new Error(`Failed to load attendance (${res.status})`)
  }
  return (await res.json()) as SectionAttendanceResponse
}

export async function saveSectionAttendance(
  sectionId: string,
  date: string,
  records: UpsertRecord[],
): Promise<BatchSaveResponse> {
  const res = await authorizedFetch(
    `/api/v1/sections/${encodeURIComponent(sectionId)}/attendance/${encodeURIComponent(date)}`,
    {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ records }),
    },
  )
  if (!res.ok) {
    const body = (await res.json().catch(() => ({ error: {} }))) as {
      error?: { message?: string }
    }
    throw new Error(body.error?.message ?? `Failed to save attendance (${res.status})`)
  }
  return (await res.json()) as BatchSaveResponse
}

export async function fetchStudentAttendance(studentId: string): Promise<{ records: AttendanceRecord[] }> {
  const res = await authorizedFetch(`/api/v1/students/${encodeURIComponent(studentId)}/attendance`)
  if (!res.ok) {
    throw new Error(`Failed to load student attendance (${res.status})`)
  }
  return (await res.json()) as { records: AttendanceRecord[] }
}

export async function fetchParentStudentAttendance(
  studentId: string,
): Promise<{ records: AttendanceRecord[] }> {
  const res = await authorizedFetch(
    `/api/v1/parent/students/${encodeURIComponent(studentId)}/attendance`,
  )
  if (!res.ok) {
    throw new Error(`Failed to load attendance (${res.status})`)
  }
  return (await res.json()) as { records: AttendanceRecord[] }
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

export async function fetchAttendanceCodes(orgId: string): Promise<AttendanceCodesResponse> {
  const res = await authorizedFetch(
    `/api/v1/admin/orgs/${encodeURIComponent(orgId)}/attendance/codes`,
  )
  if (!res.ok) {
    throw new Error(`Failed to load attendance codes (${res.status})`)
  }
  return (await res.json()) as AttendanceCodesResponse
}

export async function createAttendanceCode(
  orgId: string,
  payload: { code: string; label: string; stateCode?: string; category: string },
): Promise<AttendanceCode> {
  const res = await authorizedFetch(
    `/api/v1/admin/orgs/${encodeURIComponent(orgId)}/attendance/codes`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    },
  )
  if (!res.ok) {
    throw new Error(`Failed to create code (${res.status})`)
  }
  return (await res.json()) as AttendanceCode
}

export async function seedDefaultAttendanceCodes(orgId: string): Promise<AttendanceCodesResponse> {
  const res = await authorizedFetch(
    `/api/v1/admin/orgs/${encodeURIComponent(orgId)}/attendance/codes`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ seedDefaults: true }),
    },
  )
  if (!res.ok) {
    throw new Error(`Failed to seed default codes (${res.status})`)
  }
  return (await res.json()) as AttendanceCodesResponse
}

export async function deleteAttendanceCode(orgId: string, codeId: string): Promise<void> {
  const res = await authorizedFetch(
    `/api/v1/admin/orgs/${encodeURIComponent(orgId)}/attendance/codes/${encodeURIComponent(codeId)}`,
    { method: 'DELETE' },
  )
  if (!res.ok && res.status !== 204) {
    throw new Error(`Failed to delete code (${res.status})`)
  }
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
