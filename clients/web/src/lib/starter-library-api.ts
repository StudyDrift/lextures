import { authorizedFetch } from './api'
import { messageFromApiErrorBody } from './errors'

export type StarterSubject = {
  id: string
  label: string
}

export type StarterModuleSummary = {
  title: string
  pageCount: number
}

export type StarterSummary = {
  slug: string
  title: string
  summary: string
  gradeBand: string
  subject: string
  subjectLabel: string
  estimatedMinutes: number
  standards: string[]
  modules: StarterModuleSummary[]
}

export type StarterPage = {
  title: string
  markdown: string
}

export type StarterModule = {
  title: string
  pages: StarterPage[]
}

export type StarterDetail = Omit<StarterSummary, 'modules'> & {
  modules: StarterModule[]
}

export type StarterLibraryList = {
  gradeBands: string[]
  subjects: StarterSubject[]
  courses: StarterSummary[]
}

export type StarterCopyResult = {
  courseCode: string
  title: string
  source?: string
  enrolled: boolean
  learnerUserId?: string
}

async function readError(res: Response, fallback: string): Promise<Error> {
  const raw = await res.text()
  return new Error(messageFromApiErrorBody(raw, fallback))
}

export async function listStarterCourses(filters: {
  gradeBand?: string
  subject?: string
}): Promise<StarterLibraryList> {
  const params = new URLSearchParams()
  if (filters.gradeBand) params.set('gradeBand', filters.gradeBand)
  if (filters.subject) params.set('subject', filters.subject)
  const qs = params.toString()
  const res = await authorizedFetch(`/api/v1/starter-library${qs ? `?${qs}` : ''}`)
  if (!res.ok) throw await readError(res, 'Could not load the starter library.')
  return (await res.json()) as StarterLibraryList
}

export async function getStarterCourse(slug: string): Promise<StarterDetail> {
  const res = await authorizedFetch(`/api/v1/starter-library/${encodeURIComponent(slug)}`)
  if (!res.ok) throw await readError(res, 'Could not load that starter course.')
  return (await res.json()) as StarterDetail
}

export async function copyStarterCourse(
  slug: string,
  learnerUserId?: string,
): Promise<StarterCopyResult> {
  const res = await authorizedFetch(`/api/v1/starter-library/${encodeURIComponent(slug)}/use`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ learnerUserId: learnerUserId ?? '' }),
  })
  if (!res.ok) throw await readError(res, 'Could not copy this course.')
  return (await res.json()) as StarterCopyResult
}

export async function generateStarterCourse(input: {
  gradeBand: string
  subject: string
  focus?: string
  learnerUserId?: string
}): Promise<StarterCopyResult> {
  const res = await authorizedFetch('/api/v1/starter-library/generate', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(input),
  })
  if (!res.ok) throw await readError(res, 'Could not generate a starter course.')
  return (await res.json()) as StarterCopyResult
}

export function formatStarterLength(minutes: number): string {
  if (!Number.isFinite(minutes) || minutes <= 0) return ''
  if (minutes < 60) return `About ${minutes} minutes`
  const hours = Math.floor(minutes / 60)
  const rest = minutes % 60
  if (rest === 0) return `About ${hours} hour${hours === 1 ? '' : 's'}`
  return `About ${hours} hour${hours === 1 ? '' : 's'} ${rest} minutes`
}
