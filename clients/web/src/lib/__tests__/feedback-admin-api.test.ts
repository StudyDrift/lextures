import { describe, expect, it } from 'vitest'
import {
  buildFeedbackListQuery,
  dateInputToFromIso,
  dateInputToToIso,
  feedbackPersonLabel,
} from '../feedback-admin-api'

describe('buildFeedbackListQuery', () => {
  it('maps filters to query params', () => {
    const qs = buildFeedbackListQuery({
      status: 'new',
      category: 'bug',
      source: 'web',
      q: 'broken login',
      from: '2026-01-01T00:00:00.000Z',
      to: '2026-01-31T23:59:59.999Z',
      limit: 25,
      cursor: 'abc',
    })
    const params = new URLSearchParams(qs)
    expect(params.get('status')).toBe('new')
    expect(params.get('category')).toBe('bug')
    expect(params.get('source')).toBe('web')
    expect(params.get('q')).toBe('broken login')
    expect(params.get('from')).toBe('2026-01-01T00:00:00.000Z')
    expect(params.get('to')).toBe('2026-01-31T23:59:59.999Z')
    expect(params.get('limit')).toBe('25')
    expect(params.get('cursor')).toBe('abc')
  })

  it('omits empty filters', () => {
    expect(buildFeedbackListQuery({})).toBe('')
  })
})

describe('dateInputToFromIso', () => {
  it('converts local date to ISO start of day', () => {
    const iso = dateInputToFromIso('2026-07-10')
    expect(iso).toMatch(/^2026-07-10T\d{2}:00:00.000Z$/)
  })
})

describe('dateInputToToIso', () => {
  it('converts local date to ISO end of day', () => {
    const iso = dateInputToToIso('2026-07-10')
    expect(iso).toMatch(/^2026-07-1[01]T\d{2}:59:59.999Z$/)
  })
})

describe('feedbackPersonLabel', () => {
  it('prefers display name', () => {
    expect(feedbackPersonLabel({ name: 'Ada Lovelace', email: 'ada@example.com' })).toBe('Ada Lovelace')
  })

  it('falls back to email', () => {
    expect(feedbackPersonLabel({ name: '', email: 'ada@example.com' })).toBe('ada@example.com')
  })
})
