import { authorizedFetch } from './api'

export type QuestionType = 'rating' | 'multiple_choice' | 'open_text'

export type EvaluationQuestion = {
  type: QuestionType
  text: string
  options?: string[]
  required?: boolean
}

export type EvaluationTemplate = {
  id: string
  orgId: string
  name: string
  questions: EvaluationQuestion[]
  createdBy?: string
  createdAt: string
  updatedAt: string
}

export type EvaluationWindow = {
  id: string
  courseId: string
  templateId: string
  opensAt: string
  closesAt: string
  enrolledCount: number
  responseCount: number
  createdAt: string
}

export type EvaluationStatus = {
  windowOpen: boolean
  windowId?: string
  hasSubmitted: boolean
  opensAt?: string
  closesAt?: string
}

export type QuestionResult = {
  index: number
  type: QuestionType
  text: string
  average?: number
  distribution?: Record<string, number>
  openTexts?: string[]
}

export type EvaluationResults = {
  windowId: string
  opensAt: string
  closesAt: string
  responseCount: number
  enrolledCount: number
  completionPct: number
  meetsThreshold: boolean
  questions: QuestionResult[]
}

export type AdminReportRow = {
  courseId: string
  courseCode: string
  courseTitle: string
  windowId: string
  opensAt: string
  closesAt: string
  enrolledCount: number
  responseCount: number
  completionPct: number
  averageRating?: number
}

export async function fetchEvaluationStatus(courseCode: string): Promise<EvaluationStatus> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/evaluations/status`,
  )
  if (!res.ok) {
    throw new Error(`Failed to load evaluation status (${res.status})`)
  }
  return (await res.json()) as EvaluationStatus
}

export async function submitEvaluation(
  courseCode: string,
  windowId: string,
  answers: Record<string, unknown>,
): Promise<void> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/evaluations/${encodeURIComponent(windowId)}/submit`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ answers }),
    },
  )
  if (!res.ok) {
    const err = (await res.json().catch(() => ({}))) as { error?: { message?: string } }
    throw new Error(err.error?.message ?? `Failed to submit evaluation (${res.status})`)
  }
}

export async function fetchEvaluationResults(courseCode: string): Promise<EvaluationResults> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/evaluations/results`,
  )
  if (!res.ok) {
    throw new Error(`Failed to load evaluation results (${res.status})`)
  }
  return (await res.json()) as EvaluationResults
}

// Admin API

export async function listEvaluationTemplates(): Promise<EvaluationTemplate[]> {
  const res = await authorizedFetch('/api/v1/admin/evaluation-templates')
  if (!res.ok) {
    throw new Error(`Failed to load evaluation templates (${res.status})`)
  }
  const body = (await res.json()) as { templates: EvaluationTemplate[] }
  return body.templates ?? []
}

export async function createEvaluationTemplate(
  name: string,
  questions: EvaluationQuestion[],
): Promise<EvaluationTemplate> {
  const res = await authorizedFetch('/api/v1/admin/evaluation-templates', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name, questions }),
  })
  if (!res.ok) {
    const err = (await res.json().catch(() => ({}))) as { error?: { message?: string } }
    throw new Error(err.error?.message ?? `Failed to create template (${res.status})`)
  }
  return (await res.json()) as EvaluationTemplate
}

export async function updateEvaluationTemplate(
  templateId: string,
  name: string,
  questions: EvaluationQuestion[],
): Promise<EvaluationTemplate> {
  const res = await authorizedFetch(
    `/api/v1/admin/evaluation-templates/${encodeURIComponent(templateId)}`,
    {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name, questions }),
    },
  )
  if (!res.ok) {
    const err = (await res.json().catch(() => ({}))) as { error?: { message?: string } }
    throw new Error(err.error?.message ?? `Failed to update template (${res.status})`)
  }
  return (await res.json()) as EvaluationTemplate
}

export async function deleteEvaluationTemplate(templateId: string): Promise<void> {
  const res = await authorizedFetch(
    `/api/v1/admin/evaluation-templates/${encodeURIComponent(templateId)}`,
    { method: 'DELETE' },
  )
  if (!res.ok) {
    throw new Error(`Failed to delete template (${res.status})`)
  }
}

export async function fetchAdminEvaluationReport(closedOnly?: boolean): Promise<AdminReportRow[]> {
  const params = closedOnly ? '?closed_only=true' : ''
  const res = await authorizedFetch(`/api/v1/admin/evaluations/report${params}`)
  if (!res.ok) {
    throw new Error(`Failed to load evaluation report (${res.status})`)
  }
  const body = (await res.json()) as { rows: AdminReportRow[] }
  return body.rows ?? []
}
