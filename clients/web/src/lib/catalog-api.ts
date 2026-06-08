import { authorizedFetch } from './api'

export type MeetingPattern = {
  days?: string
  startTime?: string
  endTime?: string
  instructor?: string
}

export type CatalogPrerequisite = {
  code: string
  title?: string
}

export type PrereqStatus = {
  code: string
  status: 'met' | 'not_met' | 'waived'
}

export type CatalogSection = {
  id: string
  orgId: string
  termId: string
  sisCourseId: string
  sisSectionId: string
  crn?: string
  subject: string
  courseNumber: string
  sectionNumber?: string
  title: string
  credits?: number
  meetingPattern?: MeetingPattern
  room?: string
  department?: string
  prerequisites?: CatalogPrerequisite[]
  instructorName?: string
  status: string
  lmsCourseId?: string
  syncedAt?: string
  prerequisiteStatus?: PrereqStatus[]
}

export type CatalogListFilter = {
  termId?: string
  department?: string
  days?: string
  minCredits?: number
  maxCredits?: number
  q?: string
  cursor?: string
  limit?: number
}

export type ScheduleEntry = {
  section: CatalogSection
  registrationStatus: string
  courseCode?: string
  courseTitle?: string
  prerequisiteStatus?: PrereqStatus[]
}

export async function listCatalogSections(
  filter: CatalogListFilter = {},
): Promise<{ sections: CatalogSection[]; nextCursor?: string; lastSyncedAt?: string }> {
  const params = new URLSearchParams()
  if (filter.termId) params.set('term_id', filter.termId)
  if (filter.department) params.set('department', filter.department)
  if (filter.days) params.set('days', filter.days)
  if (filter.minCredits != null) params.set('min_credits', String(filter.minCredits))
  if (filter.maxCredits != null) params.set('max_credits', String(filter.maxCredits))
  if (filter.q) params.set('q', filter.q)
  if (filter.cursor) params.set('cursor', filter.cursor)
  if (filter.limit) params.set('limit', String(filter.limit))
  const qs = params.toString()
  const res = await authorizedFetch(`/api/v1/catalog/sections${qs ? `?${qs}` : ''}`)
  const data = (await res.json()) as {
    sections?: CatalogSection[]
    nextCursor?: string
    lastSyncedAt?: string
  }
  return {
    sections: data.sections ?? [],
    nextCursor: data.nextCursor,
    lastSyncedAt: data.lastSyncedAt,
  }
}

export async function getCatalogSection(id: string): Promise<CatalogSection> {
  const res = await authorizedFetch(`/api/v1/catalog/sections/${encodeURIComponent(id)}`)
  const data = (await res.json()) as { section: CatalogSection }
  return data.section
}

export async function fetchCatalogSchedule(): Promise<ScheduleEntry[]> {
  const res = await authorizedFetch('/api/v1/catalog/schedule')
  const data = (await res.json()) as { schedule?: ScheduleEntry[] }
  return data.schedule ?? []
}

export async function fetchCourseCatalogInfo(courseCode: string): Promise<CatalogSection | null> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/catalog-info`,
  )
  const data = (await res.json()) as { catalogInfo: CatalogSection | null }
  return data.catalogInfo
}
