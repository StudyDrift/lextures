import { apiBaseUrl } from './api'
import { getAccessToken } from './auth'

/** WebSocket URL for in-app bell notification updates (auth is sent in first WS message). */
export function notificationsWebSocketUrl(): string | null {
  if (!getAccessToken()) return null
  const base = apiBaseUrl()
  const u = new URL(base)
  u.protocol = u.protocol === 'https:' ? 'wss:' : 'ws:'
  return `${u.origin}/api/v1/ws/notifications`
}

export type NotificationsWsMessage = {
  type?: string
}

export function parseNotificationsWsMessage(raw: string): NotificationsWsMessage | null {
  try {
    return JSON.parse(raw) as NotificationsWsMessage
  } catch {
    return null
  }
}