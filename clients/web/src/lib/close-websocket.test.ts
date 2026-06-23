import { describe, expect, it, vi } from 'vitest'
import { closeWebSocket } from './close-websocket'

describe('closeWebSocket', () => {
  it('defers close while connecting', () => {
    const ws = {
      readyState: WebSocket.CONNECTING,
      onopen: null as (() => void) | null,
      onclose: null as (() => void) | null,
      onerror: null as (() => void) | null,
      onmessage: null as (() => void) | null,
      close: vi.fn(),
    } as unknown as WebSocket

    closeWebSocket(ws)
    expect(ws.close).not.toHaveBeenCalled()
    expect(ws.onopen).toBeTypeOf('function')

    ws.onopen?.(new Event('open'))
    expect(ws.close).toHaveBeenCalledOnce()
  })

  it('closes immediately when open', () => {
    const ws = {
      readyState: WebSocket.OPEN,
      onopen: null,
      onclose: null,
      onerror: null,
      onmessage: null,
      close: vi.fn(),
    } as unknown as WebSocket

    closeWebSocket(ws)
    expect(ws.close).toHaveBeenCalledOnce()
  })
})