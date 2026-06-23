/** Closes a WebSocket without the browser warning when still connecting. */
export function closeWebSocket(ws: WebSocket | null | undefined): void {
  if (!ws) return

  ws.onmessage = null

  if (ws.readyState === WebSocket.CONNECTING) {
    ws.onerror = null
    ws.onclose = null
    ws.onopen = () => {
      ws.onopen = null
      ws.close()
    }
    return
  }

  if (ws.readyState === WebSocket.OPEN) {
    ws.onerror = null
    ws.onclose = null
    ws.close()
  }
}