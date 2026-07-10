import { authorizedFetch } from './api'

export const FEEDBACK_MAX_MESSAGE_LEN = 5000

export type FeedbackCategory = 'bug' | 'idea' | 'question' | 'praise' | 'other'
export type FeedbackCategoryOption = FeedbackCategory | ''

export type FeedbackContext = {
  route: string
  locale?: string
  viewport?: string
}

export type SubmitFeedbackInput = {
  message: string
  category: FeedbackCategoryOption
  route: string
  locale?: string
  viewport?: string
}

export function buildFeedbackPayload(input: SubmitFeedbackInput): {
  message: string
  source: 'web'
  app_version: string
  context: FeedbackContext
  category?: FeedbackCategory
} {
  const context: FeedbackContext = { route: input.route }
  if (input.locale) context.locale = input.locale
  if (input.viewport) context.viewport = input.viewport

  const body: {
    message: string
    source: 'web'
    app_version: string
    context: FeedbackContext
    category?: FeedbackCategory
  } = {
    message: input.message.trim(),
    source: 'web',
    app_version: typeof __APP_RELEASE_VERSION__ !== 'undefined' ? __APP_RELEASE_VERSION__ : '0.0.0',
    context,
  }
  const category = input.category.trim() as FeedbackCategoryOption
  if (category) {
    body.category = category
  }
  return body
}

export type SubmitFeedbackResult =
  | { ok: true }
  | { ok: false; kind: 'offline' | 'rate_limited' | 'error' }

export async function submitFeedback(input: SubmitFeedbackInput): Promise<SubmitFeedbackResult> {
  if (typeof navigator !== 'undefined' && !navigator.onLine) {
    return { ok: false, kind: 'offline' }
  }
  try {
    const res = await authorizedFetch('/api/v1/feedback', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(buildFeedbackPayload(input)),
    })
    if (res.status === 201) return { ok: true }
    if (res.status === 429) return { ok: false, kind: 'rate_limited' }
    return { ok: false, kind: 'error' }
  } catch {
    return { ok: false, kind: 'offline' }
  }
}

export function feedbackMessageValid(message: string): boolean {
  return message.trim().length > 0
}
