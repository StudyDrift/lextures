/** Closes a WebSocket without browser console noise from early/double close. */
export function closeWebSocket(ws: WebSocket | null | undefined): void {
  if (!ws) return

  // Detach app handlers first so close/error cannot re-enter reconnect logic
  // or call close() again ("Close received after close").
  ws.onmessage = null
  ws.onerror = null
  ws.onclose = null

  if (ws.readyState === WebSocket.CONNECTING) {
    // Defer close until open to avoid "closed before the connection is established".
    ws.onopen = () => {
      ws.onopen = null
      try {
        ws.close()
      } catch {
        /* ignore */
      }
    }
    return
  }

  if (ws.readyState === WebSocket.OPEN) {
    ws.onopen = null
    try {
      ws.close()
    } catch {
      /* ignore */
    }
  }
  // CLOSING / CLOSED: already shutting down — do not call close() again.
}
