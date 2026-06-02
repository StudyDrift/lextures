import { authorizedFetch } from './api'

export type StandardDomain = {
  id: string
  orgId: string
  code: string
  name: string
  gradeLevel?: string | null
  createdAt: string
}

export type Standard = {
  id: string
  domainId: string
  code: string
  description: string
  createdAt: string
}

export type MasteryScaleEntry = {
  id: string
  orgId: string
  label: string
  value: number
  color?: string | null
  createdAt: string
}

export type MasteryScore = {
  id: string
  studentId: string
  standardId: string
  courseId: string
  gradingPeriod: string
  scoreValue: number
  assessedBy?: string | null
  source: 'assignment' | 'quiz' | 'observation'
  sourceId?: string | null
  assessedAt: string
}

export type HeatmapCell = {
  studentId: string
  standardId: string
  scoreValue: number
}

export type CSVImportResult = {
  domainsCreated: number
  standardsImported: number
  errors: string[]
}

// ─── Standard Domains ────────────────────────────────────────────────────────

export async function listStandardDomains(orgId: string): Promise<StandardDomain[]> {
  const res = await authorizedFetch(
    `/api/v1/admin/orgs/${encodeURIComponent(orgId)}/sbg/standard-domains`,
  )
  if (!res.ok) throw new Error(`Failed to list standard domains: ${res.status}`)
  const body = (await res.json()) as { domains: StandardDomain[] }
  return body.domains
}

export async function createStandardDomain(
  orgId: string,
  code: string,
  name: string,
  gradeLevel?: string,
): Promise<StandardDomain> {
  const res = await authorizedFetch(
    `/api/v1/admin/orgs/${encodeURIComponent(orgId)}/sbg/standard-domains`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ code, name, gradeLevel: gradeLevel ?? null }),
    },
  )
  if (!res.ok) throw new Error(`Failed to create domain: ${res.status}`)
  return (await res.json()) as StandardDomain
}

// ─── Mastery Scale ────────────────────────────────────────────────────────────

export async function getMasteryScale(orgId: string): Promise<MasteryScaleEntry[]> {
  const res = await authorizedFetch(
    `/api/v1/admin/orgs/${encodeURIComponent(orgId)}/sbg/mastery-scale`,
  )
  if (!res.ok) throw new Error(`Failed to load mastery scale: ${res.status}`)
  const body = (await res.json()) as { scale: MasteryScaleEntry[] }
  return body.scale
}

export async function putMasteryScale(
  orgId: string,
  scale: Array<{ label: string; value: number; color?: string }>,
): Promise<MasteryScaleEntry[]> {
  const res = await authorizedFetch(
    `/api/v1/admin/orgs/${encodeURIComponent(orgId)}/sbg/mastery-scale`,
    {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ scale }),
    },
  )
  if (!res.ok) throw new Error(`Failed to save mastery scale: ${res.status}`)
  const body = (await res.json()) as { scale: MasteryScaleEntry[] }
  return body.scale
}

// ─── CSV Import ───────────────────────────────────────────────────────────────

export async function importStandardsCSV(
  orgId: string,
  file: File,
): Promise<CSVImportResult> {
  const form = new FormData()
  form.append('file', file)
  const res = await authorizedFetch(
    `/api/v1/admin/orgs/${encodeURIComponent(orgId)}/sbg/standards/import`,
    { method: 'POST', body: form },
  )
  if (!res.ok) throw new Error(`Import failed: ${res.status}`)
  return (await res.json()) as CSVImportResult
}

// ─── Course Standards ─────────────────────────────────────────────────────────

export async function listCourseStandards(courseCode: string): Promise<Standard[]> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/sbg/standards`,
  )
  if (!res.ok) throw new Error(`Failed to list standards: ${res.status}`)
  const body = (await res.json()) as { standards: Standard[] }
  return body.standards
}

// ─── Mastery Scores ───────────────────────────────────────────────────────────

export async function recordMasteryScore(params: {
  studentId: string
  standardId: string
  courseCode: string
  gradingPeriod: string
  scoreValue: number
  source?: 'assignment' | 'quiz' | 'observation'
  sourceId?: string
}): Promise<MasteryScore> {
  const res = await authorizedFetch('/api/v1/sbg/mastery-scores', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      studentId: params.studentId,
      standardId: params.standardId,
      courseCode: params.courseCode,
      gradingPeriod: params.gradingPeriod,
      scoreValue: params.scoreValue,
      source: params.source ?? 'observation',
      sourceId: params.sourceId ?? null,
    }),
  })
  if (!res.ok) throw new Error(`Failed to record mastery score: ${res.status}`)
  return (await res.json()) as MasteryScore
}

// ─── Heatmap ──────────────────────────────────────────────────────────────────

export async function getSBGHeatmap(
  courseCode: string,
  period: string,
): Promise<HeatmapCell[]> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/sbg/heatmap/${encodeURIComponent(period)}`,
  )
  if (!res.ok) throw new Error(`Failed to load heatmap: ${res.status}`)
  const body = (await res.json()) as { cells: HeatmapCell[] }
  return body.cells
}

// ─── Student SBG Report ───────────────────────────────────────────────────────

export type StudentSBGScore = {
  standardId: string
  scoreValue: number
}

export async function getStudentSBGReport(
  studentId: string,
  period: string,
  method?: string,
): Promise<{ studentId: string; period: string; method: string; scores: StudentSBGScore[] }> {
  const params = method ? `?method=${encodeURIComponent(method)}` : ''
  const res = await authorizedFetch(
    `/api/v1/students/${encodeURIComponent(studentId)}/sbg/${encodeURIComponent(period)}${params}`,
  )
  if (!res.ok) throw new Error(`Failed to load SBG report: ${res.status}`)
  return (await res.json()) as {
    studentId: string
    period: string
    method: string
    scores: StudentSBGScore[]
  }
}
