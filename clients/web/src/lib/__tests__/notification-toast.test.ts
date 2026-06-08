import { describe, expect, it } from 'vitest'
import type { InboxNotification } from '../../context/inbox-notifications-context'
import { applyInboxRefreshForToasts } from '../notification-toast'

function notif(id: string, eventType: string): InboxNotification {
  return {
    id,
    userId: 'u1',
    eventType,
    title: 'Course imported from Canvas',
    body: 'Demo is ready.',
    actionUrl: '/courses/demo',
    isRead: false,
    createdAt: '2026-06-07T00:00:00.000Z',
  }
}

describe('applyInboxRefreshForToasts', () => {
  it('does not toast existing notifications on first hydration', () => {
    const incoming = [notif('a', 'canvas_course_imported')]
    const toasted = new Set<string>()
    const result = applyInboxRefreshForToasts([], incoming, toasted, false)
    expect(result.toToast).toEqual([])
    expect(result.nowHydrated).toBe(true)
    expect(toasted.has('a')).toBe(true)
  })

  it('toasts only notifications that arrive after hydration', () => {
    const prev = [notif('a', 'canvas_course_imported')]
    const incoming = [notif('a', 'canvas_course_imported'), notif('b', 'canvas_course_imported')]
    const toasted = new Set(['a'])
    const result = applyInboxRefreshForToasts(prev, incoming, toasted, true)
    expect(result.toToast.map((n) => n.id)).toEqual(['b'])
    expect(toasted.has('b')).toBe(true)
  })
})
