import { tryRefreshSession } from './api'
import { getAccessToken } from './auth'
import { closeWebSocket } from './close-websocket'
import { getRefreshToken } from './session-tokens'
import { wsReconnectDelayMs } from './ws-reconnect'

/** Close code the API sends when it rejects the auth frame (server `ws_common.go`). */
const WS_STATUS_AUTH_FAILED = 1008

/**
 * A socket that stayed up this long counts as healthy, so the next drop restarts
 * backoff at zero. Resetting on `open` instead defeats backoff entirely: these
 * endpoints upgrade before authenticating, so a rejected token still fires
 * `open`, and the client would reconnect roughly once a second forever.
 */
const HEALTHY_AFTER_MS = 30_000

/**
 * Consecutive open-then-immediately-closed cycles before we refresh the access
 * token anyway. Covers proxies that drop the 1008 close code (the client then
 * sees a bare 1006) without refreshing on every transient blip.
 */
const SHORT_CLOSES_BEFORE_REFRESH = 3

export type AppWebSocketOptions = {
  /** Resolves the socket URL, or null when there is no session to connect with. */
  url: () => string | null
  onMessage: (data: string) => void
}

export type AppWebSocketHandle = {
  /** Tears down the socket and stops all reconnect work. */
  close: () => void
}

/**
 * Long-lived authenticated app socket (bell notifications, mailbox) with
 * backoff, token refresh, and reconnect-on-login.
 *
 * Stays connected across navigations — reconnects only on unexpected close or
 * auth token change.
 */
export function openAppWebSocket({ url, onMessage }: AppWebSocketOptions): AppWebSocketHandle {
  let cancelled = false
  let socket: WebSocket | null = null
  let socketToken: string | null = null
  let reconnectTimer: ReturnType<typeof setTimeout> | null = null
  let visibilityWait: (() => void) | null = null
  let attempt = 0
  let shortCloses = 0
  let refreshBeforeConnect = false

  const clearReconnectTimer = () => {
    if (reconnectTimer) {
      clearTimeout(reconnectTimer)
      reconnectTimer = null
    }
    if (visibilityWait) {
      document.removeEventListener('visibilitychange', visibilityWait)
      visibilityWait = null
    }
  }

  const disconnect = () => {
    clearReconnectTimer()
    closeWebSocket(socket)
    socket = null
    socketToken = null
  }

  const scheduleReconnect = () => {
    if (cancelled || reconnectTimer || visibilityWait) return
    // Pause aggressive reconnect while the tab is backgrounded (origin 504 storms).
    if (typeof document !== 'undefined' && document.visibilityState === 'hidden') {
      const onVis = () => {
        if (document.visibilityState !== 'visible') return
        document.removeEventListener('visibilitychange', onVis)
        visibilityWait = null
        if (!cancelled) scheduleReconnect()
      }
      visibilityWait = onVis
      document.addEventListener('visibilitychange', onVis)
      return
    }
    const delay = wsReconnectDelayMs(attempt)
    attempt += 1
    reconnectTimer = setTimeout(() => {
      reconnectTimer = null
      if (!cancelled) void connect()
    }, delay)
  }

  const connect = async () => {
    if (cancelled) return
    if (!getAccessToken()) {
      disconnect()
      return
    }

    // These endpoints upgrade first and authenticate on the first frame, so an
    // expired access token opens cleanly and is then closed by the server. Renew
    // it up front rather than reconnecting with the same dead token.
    if (refreshBeforeConnect) {
      refreshBeforeConnect = false
      if (getRefreshToken()) {
        const ok = await tryRefreshSession()
        if (cancelled) return
        if (!ok) {
          // Session is gone; `tryRefreshSession` already signalled auth-required.
          disconnect()
          return
        }
        shortCloses = 0
      }
    }

    const authToken = getAccessToken()
    if (!authToken) {
      disconnect()
      return
    }

    const target = url()
    if (!target) return

    if (socket && socketToken === authToken) return

    closeWebSocket(socket)
    socketToken = authToken

    let ws: WebSocket
    try {
      ws = new WebSocket(target)
    } catch {
      scheduleReconnect()
      return
    }
    socket = ws
    let openedAt = 0

    ws.onopen = () => {
      openedAt = Date.now()
      const currentToken = getAccessToken()
      if (!currentToken) {
        closeWebSocket(ws)
        return
      }
      try {
        ws.send(JSON.stringify({ authToken: currentToken }))
      } catch {
        /* socket may already be closing */
      }
    }

    ws.onmessage = (ev) => {
      onMessage(String(ev.data))
    }

    ws.onclose = (ev) => {
      if (socket === ws) {
        socket = null
        socketToken = null
      }
      if (openedAt > 0 && Date.now() - openedAt >= HEALTHY_AFTER_MS) {
        // Connection did real work before dropping — treat the next retry as fresh.
        attempt = 0
        shortCloses = 0
      } else if (openedAt > 0) {
        shortCloses += 1
      }
      if (ev.code === WS_STATUS_AUTH_FAILED || shortCloses >= SHORT_CLOSES_BEFORE_REFRESH) {
        refreshBeforeConnect = true
      }
      // Browser already closed the socket; do not call close() again.
      scheduleReconnect()
    }

    // Errors are followed by onclose — avoid close() here (double-close noise).
    ws.onerror = null
  }

  const onAuthToken = () => {
    if (cancelled) return
    if (!getAccessToken()) {
      disconnect()
      return
    }
    // A fresh token invalidates any pending backoff wait.
    clearReconnectTimer()
    attempt = 0
    shortCloses = 0
    void connect()
  }

  void connect()
  window.addEventListener('studydrift-auth-token', onAuthToken)

  return {
    close: () => {
      cancelled = true
      window.removeEventListener('studydrift-auth-token', onAuthToken)
      disconnect()
    },
  }
}
