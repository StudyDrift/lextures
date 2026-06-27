import { authorizedFetch } from './api'

export type ReportCard = {
  id: string
  studentId: string
  courseId: string
  gradingPeriod: string
  finalGradePct?: number | null
  letterGrade?: string | null
  comment?: string | null
  status: 'draft' | 'submitted' | 'approved' | 'released'
  pdfUrl?: string | null
  generatedAt?: string | null
  releasedAt?: string | null
  createdAt: string
  updatedAt: string
}

export type CommentBankEntry = {
  id: string
  orgId: string
  category: string
  text: string
  active: boolean
}

export type ListReportCardsResponse = {
  reportCards: ReportCard[]
  period: string
}

export type CommentBankResponse = {
  entries: CommentBankEntry[]
}

export type AICommentResponse = {
  suggestion: string
}

export async function fetchCourseReportCards(
  courseCode: string,
  period: string,
): Promise<ListReportCardsResponse> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/report-cards/${encodeURIComponent(period)}`,
  )
  if (!res.ok) throw new Error(`Failed to load report cards: ${res.status}`)
  return (await res.json()) as ListReportCardsResponse
}

export async function patchReportCard(
  cardId: string,
  patch: { comment?: string; status?: string },
): Promise<ReportCard> {
  const res = await authorizedFetch(`/api/v1/report-cards/${encodeURIComponent(cardId)}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(patch),
  })
  if (!res.ok) throw new Error(`Failed to update report card: ${res.status}`)
  return (await res.json()) as ReportCard
}

export async function releaseReportCards(
  courseCode: string,
  period: string,
): Promise<{ released: number; message: string }> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/report-cards/${encodeURIComponent(period)}/release`,
    { method: 'POST' },
  )
  if (!res.ok) throw new Error(`Failed to release report cards: ${res.status}`)
  return (await res.json()) as { released: number; message: string }
}

export async function fetchCommentBank(
  orgId: string,
  category?: string,
): Promise<CommentBankResponse> {
  const params = category ? `?category=${encodeURIComponent(category)}` : ''
  const res = await authorizedFetch(
    `/api/v1/admin/orgs/${encodeURIComponent(orgId)}/report-cards/comment-bank${params}`,
  )
  if (!res.ok) throw new Error(`Failed to load comment bank: ${res.status}`)
  return (await res.json()) as CommentBankResponse
}

export async function createCommentBankEntry(
  orgId: string,
  category: string,
  text: string,
): Promise<CommentBankEntry> {
  const res = await authorizedFetch(
    `/api/v1/admin/orgs/${encodeURIComponent(orgId)}/report-cards/comment-bank`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ category, text }),
    },
  )
  if (!res.ok) throw new Error(`Failed to create comment bank entry: ${res.status}`)
  return (await res.json()) as CommentBankEntry
}

export async function deleteCommentBankEntry(orgId: string, entryId: string): Promise<void> {
  const res = await authorizedFetch(
    `/api/v1/admin/orgs/${encodeURIComponent(orgId)}/report-cards/comment-bank/${encodeURIComponent(entryId)}`,
    { method: 'DELETE' },
  )
  if (!res.ok && res.status !== 204) throw new Error(`Failed to delete entry: ${res.status}`)
}

export async function fetchParentReportCards(studentId: string): Promise<ListReportCardsResponse> {
  const res = await authorizedFetch(
    `/api/v1/parent/students/${encodeURIComponent(studentId)}/report-cards`,
  )
  if (!res.ok) throw new Error(`Failed to load report cards: ${res.status}`)
  return (await res.json()) as ListReportCardsResponse
}

export async function fetchAICommentSuggestion(
  courseName: string,
  gradePct: number,
  absences: number,
): Promise<string> {
  const res = await authorizedFetch('/api/v1/ai/report-card-comment', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ courseName, gradePct, absences }),
  })
  if (!res.ok) throw new Error(`AI comment failed: ${res.status}`)
  const data = (await res.json()) as AICommentResponse
  return data.suggestion
}
