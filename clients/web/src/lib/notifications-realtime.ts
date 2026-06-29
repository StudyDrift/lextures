import { wsUrl } from './api'
import { getAccessToken } from './auth'

/** WebSocket URL for in-app bell notification updates (auth is sent in first WS message). */
export function notificationsWebSocketUrl(): string | null {
  if (!getAccessToken()) return null
  return wsUrl('/api/v1/ws/notifications')
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