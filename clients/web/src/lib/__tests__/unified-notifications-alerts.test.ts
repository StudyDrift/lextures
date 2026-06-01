import { describe, expect, it } from 'vitest'
import { inboxAlertsToUnified, notificationActionHref } from '../unified-notifications'

describe('notificationActionHref', () => {
  it('keeps app-relative paths', () => {
    expect(notificationActionHref('/courses/C-ABC123')).toBe('/courses/C-ABC123')
  })

  it('extracts pathname from absolute URLs', () => {
    expect(notificationActionHref('https://app.example.com/courses/C-ABC123?tab=1')).toBe(
      '/courses/C-ABC123?tab=1',
    )
  })

  it('falls back to home for empty input', () => {
    expect(notificationActionHref('')).toBe('/')
  })
})

describe('inboxAlertsToUnified', () => {
  it('maps persisted alerts into the unified list shape', () => {
    const rows = inboxAlertsToUnified([
      {
        id: 'n1',
        title: 'Course imported from Canvas',
        body: 'Algebra is ready in Lextures.',
        actionUrl: '/courses/C-TEST01',
        isRead: false,
        createdAt: '2026-06-01T12:00:00.000Z',
      },
    ])
    expect(rows).toHaveLength(1)
    expect(rows[0]?.kind).toBe('alert')
    expect(rows[0]?.alertId).toBe('n1')
    expect(rows[0]?.href).toBe('/courses/C-TEST01')
  })
})
