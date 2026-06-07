import { describe, expect, it } from 'vitest'
import type { InboxNotification } from '../../context/inbox-notifications-context'
import {
  newCanvasImportNotifications,
  pickCanvasImportNotificationsToToast,
} from '../canvas-import-toast'

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

describe('newCanvasImportNotifications', () => {
  it('returns only new canvas_course_imported rows', () => {
    const prev = [notif('a', 'canvas_course_imported')]
    const incoming = [
      notif('a', 'canvas_course_imported'),
      notif('b', 'canvas_course_imported'),
      notif('c', 'grade_posted'),
    ]
    expect(newCanvasImportNotifications(prev, incoming).map((n) => n.id)).toEqual(['b'])
  })
})

describe('pickCanvasImportNotificationsToToast', () => {
  it('returns nothing before the inbox has hydrated', () => {
    const incoming = [notif('a', 'canvas_course_imported')]
    expect(pickCanvasImportNotificationsToToast([], incoming, new Set(), false)).toEqual([])
  })

  it('skips ids that already triggered a toast', () => {
    const incoming = [notif('a', 'canvas_course_imported')]
    expect(pickCanvasImportNotificationsToToast([], incoming, new Set(['a']), true)).toEqual([])
  })

  it('toasts genuinely new imports after hydration', () => {
    const prev = [notif('a', 'canvas_course_imported')]
    const incoming = [notif('a', 'canvas_course_imported'), notif('b', 'canvas_course_imported')]
    expect(pickCanvasImportNotificationsToToast(prev, incoming, new Set(), true).map((n) => n.id)).toEqual([
      'b',
    ])
  })
})
