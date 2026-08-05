import { authorizedFetch } from './api'
import {
  checklistItemSchema,
  checklistResponseSchema,
  checklistSummarySchema,
  type ChecklistItem,
  type ChecklistResponse,
  type ChecklistSummary,
  type DismissReason,
} from './course-checklist-api-schemas'
import { parseApiResponse } from './courses-api-schemas'
import { readApiErrorMessage } from './errors'

function checklistBase(courseCode: string): string {
  return `/api/v1/courses/${encodeURIComponent(courseCode)}/checklist`
}

export class CourseChecklistApiError extends Error {
  readonly status: number

  constructor(status: number, message: string) {
    super(message)
    this.name = 'CourseChecklistApiError'
    this.status = status
  }
}

async function throwApiError(res: Response, fallback: string): Promise<never> {
  const raw = (await res.json().catch(() => ({}))) as Record<string, unknown>
  throw new CourseChecklistApiError(res.status, readApiErrorMessage(raw) || fallback)
}

export async function fetchCourseChecklist(courseCode: string): Promise<ChecklistResponse> {
  const res = await authorizedFetch(checklistBase(courseCode))
  if (!res.ok) {
    await throwApiError(res, 'Could not load course checklist.')
  }
  return parseApiResponse('fetchCourseChecklist', checklistResponseSchema, await res.json())
}

export async function fetchCourseChecklistSummary(courseCode: string): Promise<ChecklistSummary> {
  const res = await authorizedFetch(`${checklistBase(courseCode)}/summary`)
  if (!res.ok) {
    await throwApiError(res, 'Could not load checklist summary.')
  }
  return parseApiResponse('fetchCourseChecklistSummary', checklistSummarySchema, await res.json())
}

export async function refreshCourseChecklist(courseCode: string): Promise<ChecklistResponse> {
  const res = await authorizedFetch(`${checklistBase(courseCode)}/refresh`, { method: 'POST' })
  if (!res.ok) {
    await throwApiError(res, 'Could not re-check the course checklist.')
  }
  return parseApiResponse('refreshCourseChecklist', checklistResponseSchema, await res.json())
}

/** CC.10 FR-6: read-only outcome-mapping proposals (no writes). */
export type OutcomeMappingProposal = {
  structureItemId: string
  itemTitle?: string
  itemKind?: string
  outcomeId: string
  outcomeTitle?: string
  measurementLevel: string
  intensityLevel: string
  confidence: number
  rationale: string
}

export async function suggestOutcomeLinks(
  courseCode: string,
): Promise<OutcomeMappingProposal[]> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/outcomes/suggest-links`,
    { method: 'POST' },
  )
  if (!res.ok) {
    await throwApiError(res, 'Could not suggest outcome mappings.')
  }
  const body = (await res.json()) as { proposals?: OutcomeMappingProposal[] }
  return body.proposals ?? []
}

/** Apply one accepted proposal via the existing link-create endpoint. */
export async function createOutcomeLink(
  courseCode: string,
  outcomeId: string,
  body: {
    structureItemId: string
    measurementLevel?: string
    intensityLevel?: string
    targetKind?: string
  },
): Promise<void> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/outcomes/${encodeURIComponent(outcomeId)}/links`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        structureItemId: body.structureItemId,
        measurementLevel: body.measurementLevel ?? 'summative',
        intensityLevel: body.intensityLevel ?? 'medium',
        targetKind: body.targetKind === 'quiz' ? 'quiz' : 'assignment',
      }),
    },
  )
  if (!res.ok) {
    await throwApiError(res, 'Could not create outcome link.')
  }
}

export type WelcomeDraft = { subject: string; body: string }

export async function draftWelcomeAnnouncement(courseCode: string): Promise<WelcomeDraft> {
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/feed/draft-welcome`,
    { method: 'POST' },
  )
  if (!res.ok) {
    await throwApiError(res, 'Could not draft a welcome announcement.')
  }
  return (await res.json()) as WelcomeDraft
}

export async function dismissChecklistItem(
  courseCode: string,
  itemId: string,
  body: { reason: DismissReason; note?: string },
): Promise<ChecklistItem> {
  const res = await authorizedFetch(
    `${checklistBase(courseCode)}/items/${encodeURIComponent(itemId)}/dismiss`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        reason: body.reason,
        note: body.note ?? '',
      }),
    },
  )
  if (!res.ok) {
    await throwApiError(res, 'Could not dismiss checklist item.')
  }
  return parseApiResponse('dismissChecklistItem', checklistItemSchema, await res.json())
}

export async function restoreChecklistItem(
  courseCode: string,
  itemId: string,
): Promise<ChecklistItem> {
  const res = await authorizedFetch(
    `${checklistBase(courseCode)}/items/${encodeURIComponent(itemId)}/restore`,
    { method: 'POST' },
  )
  if (!res.ok) {
    await throwApiError(res, 'Could not restore checklist item.')
  }
  return parseApiResponse('restoreChecklistItem', checklistItemSchema, await res.json())
}

export async function recheckChecklistItem(
  courseCode: string,
  itemId: string,
): Promise<ChecklistItem> {
  const res = await authorizedFetch(
    `${checklistBase(courseCode)}/items/${encodeURIComponent(itemId)}/recheck`,
    { method: 'POST' },
  )
  if (!res.ok) {
    await throwApiError(res, 'Could not re-check checklist item.')
  }
  return parseApiResponse('recheckChecklistItem', checklistItemSchema, await res.json())
}
