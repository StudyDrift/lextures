import { describe, expect, it, vi, beforeEach } from 'vitest'
import {
  FEEDBACK_MAX_MESSAGE_LEN,
  buildFeedbackPayload,
  feedbackMessageValid,
  submitFeedback,
} from '../feedback-api'
import * as api from '../api'

describe('buildFeedbackPayload', () => {
  it('includes source web, app version, and route context', () => {
    const payload = buildFeedbackPayload({
      message: '  Great app!  ',
      category: 'praise',
      route: '/courses/demo',
      locale: 'en',
      viewport: '1280x720',
    })
    expect(payload.message).toBe('Great app!')
    expect(payload.source).toBe('web')
    expect(payload.app_version).toBeTruthy()
    expect(payload.context).toEqual({
      route: '/courses/demo',
      locale: 'en',
      viewport: '1280x720',
    })
    expect(payload.category).toBe('praise')
  })

  it('omits category when unset', () => {
    const payload = buildFeedbackPayload({
      message: 'Hello',
      category: '',
      route: '/dashboard',
    })
    expect(payload.category).toBeUndefined()
    expect(payload.context.route).toBe('/dashboard')
  })
})

describe('feedbackMessageValid', () => {
  it('rejects empty or whitespace-only messages', () => {
    expect(feedbackMessageValid('')).toBe(false)
    expect(feedbackMessageValid('   ')).toBe(false)
    expect(feedbackMessageValid('note')).toBe(true)
  })
})

describe('submitFeedback', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
  })

  it('returns offline when navigator is offline', async () => {
    const online = Object.getOwnPropertyDescriptor(navigator, 'onLine')
    Object.defineProperty(navigator, 'onLine', { configurable: true, value: false })
    const result = await submitFeedback({
      message: 'Hi',
      category: '',
      route: '/',
    })
    Object.defineProperty(navigator, 'onLine', online ?? { configurable: true, value: true })
    expect(result).toEqual({ ok: false, kind: 'offline' })
  })

  it('maps HTTP 201 to success', async () => {
    vi.spyOn(api, 'authorizedFetch').mockResolvedValue(new Response(null, { status: 201 }))
    const result = await submitFeedback({
      message: 'Hi',
      category: 'bug',
      route: '/settings',
    })
    expect(result).toEqual({ ok: true })
    expect(api.authorizedFetch).toHaveBeenCalledWith(
      '/api/v1/feedback',
      expect.objectContaining({
        method: 'POST',
        body: expect.stringContaining('"source":"web"'),
      }),
    )
  })

  it('maps HTTP 429 to rate_limited', async () => {
    vi.spyOn(api, 'authorizedFetch').mockResolvedValue(new Response(null, { status: 429 }))
    const result = await submitFeedback({
      message: 'Hi',
      category: '',
      route: '/',
    })
    expect(result).toEqual({ ok: false, kind: 'rate_limited' })
  })
})

describe('FEEDBACK_MAX_MESSAGE_LEN', () => {
  it('matches server cap', () => {
    expect(FEEDBACK_MAX_MESSAGE_LEN).toBe(5000)
  })
})
