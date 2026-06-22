import { afterEach, describe, expect, it, vi } from 'vitest'
import { clearAccessToken, setAccessToken } from '../auth'
import {
  notificationsWebSocketUrl,
  parseNotificationsWsMessage,
} from '../notifications-realtime'

describe('parseNotificationsWsMessage', () => {
  it('parses notification_updated', () => {
    expect(parseNotificationsWsMessage('{"type":"notification_updated"}')).toEqual({
      type: 'notification_updated',
    })
  })

  it('returns null for invalid JSON', () => {
    expect(parseNotificationsWsMessage('not json')).toBeNull()
  })
})

describe('notificationsWebSocketUrl', () => {
  afterEach(() => {
    vi.unstubAllEnvs()
    clearAccessToken()
  })

  it('returns null without auth token', () => {
    clearAccessToken()
    expect(notificationsWebSocketUrl()).toBeNull()
  })

  it('builds wss URL from API base', () => {
    vi.stubEnv('VITE_API_URL', 'https://api.example.com')
    setAccessToken('tok-abc')
    const u = notificationsWebSocketUrl()
    expect(u).toBe('wss://api.example.com/api/v1/ws/notifications')
  })
})