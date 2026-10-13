import { authorizedFetch } from './api'

export type ParentRecordsSettings = {
  schoolName: string
  schoolYearStartMonth: number
  schoolYearStartDay: number
  currentSchoolYear: string
  currentFrom: string
  currentTo: string
}

export type ParentHoursLogEntry = {
  id?: string
  date: string
  subject: string
  courseId?: string | null
  hours: number
  notes?: string
  source: string
  editable: boolean
}

export type ParentHoursSubjectTotal = {
  subject: string
  hours: number
}

export type ParentHoursLog = {
  schoolYear: string
  from: string
  to: string
  entries: ParentHoursLogEntry[]
  totalsBySubject: ParentHoursSubjectTotal[]
  totalHours: number
}

export type ParentRecordSubmission = {
  submissionId: string
  courseId: string
  courseCode: string
  courseTitle: string
  itemId: string
  itemTitle: string
  submittedAt: string
  bodyPreview: string
  instructorComment: string
  pointsEarned?: number
}

function recordsQuery(schoolYear?: string, from?: string, to?: string): string {
  const q = new URLSearchParams()
  if (schoolYear) q.set('schoolYear', schoolYear)
  if (from) q.set('from', from)
  if (to) q.set('to', to)
  const s = q.toString()
  return s ? `?${s}` : ''
}

export async function fetchParentRecordsSettings(): Promise<ParentRecordsSettings> {
  const res = await authorizedFetch('/api/v1/parent/records/settings')
  if (!res.ok) throw new Error(`Failed to load records settings (${res.status})`)
  return (await res.json()) as ParentRecordsSettings
}

export async function updateParentRecordsSettings(body: {
  schoolName?: string
  schoolYearStartMonth?: number
  schoolYearStartDay?: number
}): Promise<ParentRecordsSettings> {
  const res = await authorizedFetch('/api/v1/parent/records/settings', {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  if (!res.ok) throw new Error(`Failed to save records settings (${res.status})`)
  return (await res.json()) as ParentRecordsSettings
}

export async function fetchParentHoursLog(
  studentId: string,
  opts?: { schoolYear?: string; from?: string; to?: string },
): Promise<ParentHoursLog> {
  const qs = recordsQuery(opts?.schoolYear, opts?.from, opts?.to)
  const res = await authorizedFetch(`/api/v1/parent/students/${encodeURIComponent(studentId)}/records/hours${qs}`)
  if (!res.ok) throw new Error(`Failed to load hours log (${res.status})`)
  const data = (await res.json()) as ParentHoursLog
  return {
    ...data,
    entries: data.entries ?? [],
    totalsBySubject: data.totalsBySubject ?? [],
    totalHours: data.totalHours ?? 0,
  }
}

export async function createParentHoursEntry(
  studentId: string,
  body: { date: string; subject: string; hours: number; notes?: string; source?: string; courseId?: string },
): Promise<ParentHoursLogEntry> {
  const res = await authorizedFetch(`/api/v1/parent/students/${encodeURIComponent(studentId)}/records/hours`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  if (!res.ok) throw new Error(`Failed to add hours (${res.status})`)
  return (await res.json()) as ParentHoursLogEntry
}

export async function updateParentHoursEntry(
  studentId: string,
  entryId: string,
  body: { date?: string; subject?: string; hours?: number; notes?: string; courseId?: string },
): Promise<ParentHoursLogEntry> {
  const res = await authorizedFetch(
    `/api/v1/parent/students/${encodeURIComponent(studentId)}/records/hours/${encodeURIComponent(entryId)}`,
    { method: 'PATCH', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) },
  )
  if (!res.ok) throw new Error(`Failed to update hours (${res.status})`)
  return (await res.json()) as ParentHoursLogEntry
}

export async function deleteParentHoursEntry(studentId: string, entryId: string): Promise<void> {
  const res = await authorizedFetch(
    `/api/v1/parent/students/${encodeURIComponent(studentId)}/records/hours/${encodeURIComponent(entryId)}`,
    { method: 'DELETE' },
  )
  if (!res.ok) throw new Error(`Failed to delete hours (${res.status})`)
}

export async function fetchParentRecordSubmissions(
  studentId: string,
  opts?: { schoolYear?: string },
): Promise<{ schoolYear: string; submissions: ParentRecordSubmission[] }> {
  const qs = recordsQuery(opts?.schoolYear)
  const res = await authorizedFetch(
    `/api/v1/parent/students/${encodeURIComponent(studentId)}/records/submissions${qs}`,
  )
  if (!res.ok) throw new Error(`Failed to load submissions (${res.status})`)
  const data = (await res.json()) as { schoolYear: string; submissions: ParentRecordSubmission[] }
  return { schoolYear: data.schoolYear, submissions: data.submissions ?? [] }
}

function triggerBlobDownload(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  a.rel = 'noopener'
  document.body.appendChild(a)
  a.click()
  a.remove()
  URL.revokeObjectURL(url)
}

async function downloadAuthorized(path: string, filename: string): Promise<void> {
  const res = await authorizedFetch(path)
  if (!res.ok) throw new Error(`Download failed (${res.status})`)
  triggerBlobDownload(await res.blob(), filename)
}

export async function downloadHoursLogCSV(studentId: string, schoolYear?: string): Promise<void> {
  const qs = recordsQuery(schoolYear)
  await downloadAuthorized(
    `/api/v1/parent/students/${encodeURIComponent(studentId)}/records/hours/export.csv${qs}`,
    'hours-log.csv',
  )
}

export async function downloadHoursLogPDF(studentId: string, schoolYear?: string): Promise<void> {
  const qs = recordsQuery(schoolYear)
  await downloadAuthorized(
    `/api/v1/parent/students/${encodeURIComponent(studentId)}/records/hours/export.pdf${qs}`,
    'hours-log.pdf',
  )
}

export async function downloadTranscriptPDF(studentId: string, schoolYear?: string): Promise<void> {
  const qs = recordsQuery(schoolYear)
  await downloadAuthorized(
    `/api/v1/parent/students/${encodeURIComponent(studentId)}/records/transcript.pdf${qs}`,
    `transcript-${schoolYear ?? 'current'}.pdf`,
  )
}

export async function downloadPortfolio(
  studentId: string,
  body: { schoolYear?: string; submissionIds: string[]; format: 'pdf' | 'zip' },
): Promise<void> {
  const res = await authorizedFetch(`/api/v1/parent/students/${encodeURIComponent(studentId)}/records/portfolio`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  if (!res.ok) throw new Error(`Portfolio export failed (${res.status})`)
  const ext = body.format === 'zip' ? 'zip' : 'pdf'
  triggerBlobDownload(await res.blob(), `portfolio.${ext}`)
}
