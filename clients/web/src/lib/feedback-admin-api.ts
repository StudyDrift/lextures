import { authorizedFetch } from './api'
import { readApiErrorMessage } from './errors'
import type { FeedbackCategory } from './feedback-api'

export type FeedbackStatus =
  | 'new'
  | 'triaged'
  | 'in_progress'
  | 'resolved'
  | 'wont_fix'
  | 'archived'

export type FeedbackSource = 'web' | 'ios' | 'android'

export type FeedbackPerson = {
  name: string
  email: string
}

export type FeedbackContext = {
  route?: string
  locale?: string
  viewport?: string
  userAgent?: string
}

export type FeedbackListItem = {
  id: string
  message_preview: string
  category: FeedbackCategory
  source: FeedbackSource
  status: FeedbackStatus
  submitter: FeedbackPerson
  created_at: string
}

export type FeedbackDetail = {
  id: string
  user_id?: string
  org_id?: string
  message: string
  category: FeedbackCategory
  source: FeedbackSource
  app_version?: string
  context: FeedbackContext
  status: FeedbackStatus
  admin_note?: string
  resolved_by?: FeedbackPerson
  resolved_at?: string
  submitter: FeedbackPerson
  created_at: string
  updated_at: string
}

export type FeedbackListParams = {
  status?: FeedbackStatus | ''
  category?: FeedbackCategory | ''
  source?: FeedbackSource | ''
  q?: string
  from?: string
  to?: string
  limit?: number
  cursor?: string
}

export type FeedbackListResponse = {
  items: FeedbackListItem[]
  next_cursor?: string
  total?: number
}

export const FEEDBACK_STATUSES: FeedbackStatus[] = [
  'new',
  'triaged',
  'in_progress',
  'resolved',
  'wont_fix',
  'archived',
]

export const FEEDBACK_SOURCES: FeedbackSource[] = ['web', 'ios', 'android']

export const FEEDBACK_CATEGORIES: FeedbackCategory[] = [
  'bug',
  'idea',
  'question',
  'praise',
  'other',
]

export const FEEDBACK_LIST_PAGE_SIZE = 25

export function buildFeedbackListQuery(params: FeedbackListParams): string {
  const sp = new URLSearchParams()
  if (params.status) sp.set('status', params.status)
  if (params.category) sp.set('category', params.category)
  if (params.source) sp.set('source', params.source)
  if (params.q?.trim()) sp.set('q', params.q.trim())
  if (params.from) sp.set('from', params.from)
  if (params.to) sp.set('to', params.to)
  if (params.limit != null && params.limit > 0) sp.set('limit', String(params.limit))
  if (params.cursor) sp.set('cursor', params.cursor)
  return sp.toString()
}

/** Local date input (YYYY-MM-DD) → RFC3339 start of day. */
export function dateInputToFromIso(dateStr: string): string {
  if (!dateStr.trim()) return ''
  return new Date(`${dateStr}T00:00:00`).toISOString()
}

/** Local date input (YYYY-MM-DD) → RFC3339 end of day. */
export function dateInputToToIso(dateStr: string): string {
  if (!dateStr.trim()) return ''
  return new Date(`${dateStr}T23:59:59.999`).toISOString()
}

export function feedbackPersonLabel(person: FeedbackPerson): string {
  const name = person.name.trim()
  if (name) return name
  return person.email.trim() || '—'
}

export async function fetchFeedbackList(
  params: FeedbackListParams,
): Promise<FeedbackListResponse> {
  const qs = buildFeedbackListQuery({
    limit: FEEDBACK_LIST_PAGE_SIZE,
    ...params,
  })
  const res = await authorizedFetch(`/api/v1/admin/feedback${qs ? `?${qs}` : ''}`)
  const raw: unknown = await res.json().catch(() => ({}))
  if (!res.ok) {
    throw new Error(readApiErrorMessage(raw))
  }
  const data = raw as FeedbackListResponse
  return {
    items: data.items ?? [],
    next_cursor: data.next_cursor,
    total: data.total,
  }
}

export async function fetchFeedbackDetail(id: string): Promise<FeedbackDetail> {
  const res = await authorizedFetch(`/api/v1/admin/feedback/${encodeURIComponent(id)}`)
  const raw: unknown = await res.json().catch(() => ({}))
  if (!res.ok) {
    throw new Error(readApiErrorMessage(raw))
  }
  return raw as FeedbackDetail
}

export type PatchFeedbackInput = {
  status?: FeedbackStatus
  admin_note?: string
}

export async function patchFeedback(
  id: string,
  body: PatchFeedbackInput,
): Promise<FeedbackDetail> {
  const res = await authorizedFetch(`/api/v1/admin/feedback/${encodeURIComponent(id)}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  const raw: unknown = await res.json().catch(() => ({}))
  if (!res.ok) {
    throw new Error(readApiErrorMessage(raw))
  }
  return raw as FeedbackDetail
}
