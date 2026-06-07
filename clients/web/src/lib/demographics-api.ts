import { authorizedFetch } from './api'
import { readApiErrorMessage } from './errors'

async function apiJson<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await authorizedFetch(path, init)
  if (!res.ok) {
    throw new Error(await readApiErrorMessage(res))
  }
  if (res.status === 204) {
    return undefined as T
  }
  return res.json() as Promise<T>
}

export type StudentDemographics = {
  studentId: string
  freeLunch?: boolean
  reducedLunch?: boolean
  ellStatus?: boolean
  disabilityStatus?: boolean
  raceEthnicityCode?: string
  homelessIndicator?: boolean
  migrantIndicator?: boolean
  dataSource?: string
  lastVerifiedAt?: string
  updatedAt?: string
}

export type Title1Report = {
  schoolId: string
  totalStudents: number
  freeLunchCount: number
  reducedLunchCount: number
  economicDisadvantaged: number
  economicDisadvantagePct: number
  ellCount: number
  disabilityCount: number
  homelessCount: number
  migrantCount: number
  raceBreakdown: Record<string, number>
}

export type SubgroupPerformance = {
  label: string
  count: number
  suppressed: boolean
  passRate: number | null
}

export type DisaggregatedReport = {
  dimension: string
  subgroups: SubgroupPerformance[]
}

const RACE_LABELS: Record<string, string> = {
  '1': 'Hispanic/Latino',
  '2': 'American Indian/Alaska Native',
  '3': 'Asian',
  '4': 'Black/African American',
  '5': 'Native Hawaiian/Pacific Islander',
  '6': 'White',
  '7': 'Two or more races',
  unknown: 'Not reported',
}

export function raceEthnicityLabel(code: string): string {
  return RACE_LABELS[code] ?? code
}

export async function fetchStudentDemographics(studentId: string): Promise<StudentDemographics> {
  return apiJson<StudentDemographics>(`/api/v1/admin/students/${studentId}/demographics`)
}

export async function patchStudentDemographics(
  studentId: string,
  body: Partial<Omit<StudentDemographics, 'studentId' | 'dataSource' | 'updatedAt'>>,
): Promise<StudentDemographics> {
  return apiJson<StudentDemographics>(`/api/v1/admin/students/${studentId}/demographics`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
}

export async function fetchTitle1Report(orgUnitId: string): Promise<Title1Report> {
  return apiJson<Title1Report>(`/api/v1/admin/org-units/${orgUnitId}/demographics/report`)
}

export async function fetchDisaggregatedPerformance(
  orgUnitId: string,
  dimension = 'ell',
): Promise<DisaggregatedReport> {
  const q = new URLSearchParams({ dimension })
  return apiJson<DisaggregatedReport>(
    `/api/v1/admin/org-units/${orgUnitId}/demographics/disaggregated-performance?${q}`,
  )
}
