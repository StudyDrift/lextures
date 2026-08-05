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
