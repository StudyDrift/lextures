import { describe, expect, it, vi } from 'vitest'
import { closeWebSocket } from './close-websocket'

describe('closeWebSocket', () => {
  it('defers close while connecting', () => {
    const ws = {
      readyState: WebSocket.CONNECTING,
      onopen: null as ((ev?: Event) => void) | null,
      onclose: null as (() => void) | null,
      onerror: null as (() => void) | null,
      onmessage: null as (() => void) | null,
      close: vi.fn(),
    } as unknown as WebSocket

    closeWebSocket(ws)
    expect(ws.close).not.toHaveBeenCalled()
    expect(ws.onerror).toBeNull()
    expect(ws.onclose).toBeNull()
    expect(ws.onopen).toBeTypeOf('function')

    ws.onopen?.(new Event('open'))
    expect(ws.close).toHaveBeenCalledOnce()
  })

  it('closes immediately when open', () => {
    const ws = {
      readyState: WebSocket.OPEN,
      onopen: null as (() => void) | null,
      onclose: (() => undefined) as (() => void) | null,
      onerror: (() => undefined) as (() => void) | null,
      onmessage: (() => undefined) as (() => void) | null,
      close: vi.fn(),
    } as unknown as WebSocket

    closeWebSocket(ws)
    expect(ws.close).toHaveBeenCalledOnce()
    expect(ws.onclose).toBeNull()
    expect(ws.onerror).toBeNull()
    expect(ws.onmessage).toBeNull()
  })

  it('does not call close when already closing or closed', () => {
    for (const readyState of [WebSocket.CLOSING, WebSocket.CLOSED]) {
      const ws = {
        readyState,
        onopen: null,
        onclose: (() => undefined) as (() => void) | null,
        onerror: null,
        onmessage: null,
        close: vi.fn(),
      } as unknown as WebSocket

      closeWebSocket(ws)
      expect(ws.close).not.toHaveBeenCalled()
      expect(ws.onclose).toBeNull()
    }
  })

  it('is a no-op for null/undefined', () => {
    expect(() => closeWebSocket(null)).not.toThrow()
    expect(() => closeWebSocket(undefined)).not.toThrow()
  })
})
