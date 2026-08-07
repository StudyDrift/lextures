import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { openAppWebSocket } from '../app-websocket'
import { clearAccessToken, setAccessToken } from '../auth'
import { clearRefreshToken, setRefreshToken } from '../session-tokens'

const WS_STATUS_AUTH_FAILED = 1008

/** Minimal scriptable WebSocket stand-in; jsdom does not implement one. */
class FakeWebSocket {
  static instances: FakeWebSocket[] = []
  static readonly CONNECTING = 0
  static readonly OPEN = 1
  static readonly CLOSING = 2
  static readonly CLOSED = 3

  readyState = FakeWebSocket.CONNECTING
  sent: string[] = []
  url: string
  onopen: (() => void) | null = null
  onmessage: ((ev: { data: string }) => void) | null = null
  onclose: ((ev: { code: number }) => void) | null = null
  onerror: (() => void) | null = null

  constructor(url: string) {
    this.url = url
    FakeWebSocket.instances.push(this)
  }

  send(data: string) {
    this.sent.push(data)
  }

  close() {
    this.readyState = FakeWebSocket.CLOSED
  }

  /** Simulates the server accepting the upgrade. */
  open() {
    this.readyState = FakeWebSocket.OPEN
    this.onopen?.()
  }

  /** Simulates the server (or network) closing the socket. */
  serverClose(code = 1006) {
    this.readyState = FakeWebSocket.CLOSED
    this.onclose?.({ code })
  }
}

beforeEach(() => {
  vi.useFakeTimers()
  FakeWebSocket.instances = []
  vi.stubGlobal('WebSocket', FakeWebSocket)
  setAccessToken('tok-initial')
})

afterEach(() => {
  vi.useRealTimers()
  vi.unstubAllGlobals()
  clearAccessToken()
  clearRefreshToken()
  vi.restoreAllMocks()
})

/** Runs pending timers and lets queued promise callbacks settle. */
async function advance(ms: number) {
  await vi.advanceTimersByTimeAsync(ms)
}

describe('openAppWebSocket', () => {
  it('connects and sends the auth frame on open', async () => {
    const handle = openAppWebSocket({ url: () => 'wss://x/ws', onMessage: () => {} })
    await advance(0)

    const ws = FakeWebSocket.instances[0]
    expect(ws.url).toBe('wss://x/ws')
    ws.open()
    expect(ws.sent).toEqual([JSON.stringify({ authToken: 'tok-initial' })])

    handle.close()
  })

  it('delivers message payloads to onMessage', async () => {
    const seen: string[] = []
    const handle = openAppWebSocket({ url: () => 'wss://x/ws', onMessage: (d) => seen.push(d) })
    await advance(0)

    const ws = FakeWebSocket.instances[0]
    ws.open()
    ws.onmessage?.({ data: '{"type":"mailbox_updated"}' })
    expect(seen).toEqual(['{"type":"mailbox_updated"}'])

    handle.close()
  })

  // Regression: backoff used to reset on `open`. These endpoints upgrade before
  // authenticating, so a rejected token still fires `open` and the client
  // reconnected roughly once a second forever.
  it('backs off when the server accepts then immediately closes', async () => {
    const handle = openAppWebSocket({ url: () => 'wss://x/ws', onMessage: () => {} })
    await advance(0)

    // Six accept-then-close cycles, draining whatever delay each one scheduled.
    for (let i = 0; i < 6; i++) {
      const ws = FakeWebSocket.instances[i]
      ws.open()
      ws.serverClose()
      await advance(60_000)
    }

    const beforeStalling = FakeWebSocket.instances.length
    const stalled = FakeWebSocket.instances[beforeStalling - 1]
    stalled.open()
    stalled.serverClose()

    // With working backoff the next attempt is now tens of seconds out, so a
    // short wait must not produce a new socket.
    await advance(5_000)
    expect(FakeWebSocket.instances.length).toBe(beforeStalling)

    handle.close()
  })

  it('restarts backoff after a connection that stayed up', async () => {
    const handle = openAppWebSocket({ url: () => 'wss://x/ws', onMessage: () => {} })
    await advance(0)

    for (let i = 0; i < 4; i++) {
      const ws = FakeWebSocket.instances[i]
      ws.open()
      ws.serverClose()
      await advance(60_000)
    }

    // A socket that lives past the healthy threshold clears the penalty.
    const healthy = FakeWebSocket.instances[FakeWebSocket.instances.length - 1]
    healthy.open()
    await advance(31_000)
    healthy.serverClose()

    const before = FakeWebSocket.instances.length
    await advance(1_000)
    expect(FakeWebSocket.instances.length).toBe(before + 1)

    handle.close()
  })

  it('refreshes the session before reconnecting after an auth rejection', async () => {
    setRefreshToken('refresh-abc')
    const fetchMock = vi.fn(async () => {
      setAccessToken('tok-refreshed')
      return new Response(JSON.stringify({ access_token: 'tok-refreshed' }), { status: 200 })
    })
    vi.stubGlobal('fetch', fetchMock)

    const handle = openAppWebSocket({ url: () => 'wss://x/ws', onMessage: () => {} })
    await advance(0)

    const first = FakeWebSocket.instances[0]
    first.open()
    first.serverClose(WS_STATUS_AUTH_FAILED)

    await advance(60_000)

    expect(fetchMock).toHaveBeenCalled()
    const refreshCall = fetchMock.mock.calls.find((c) =>
      String((c as unknown[])[0] ?? '').includes('/auth/refresh'),
    )
    expect(refreshCall).toBeDefined()

    const reconnected = FakeWebSocket.instances[FakeWebSocket.instances.length - 1]
    reconnected.open()
    expect(reconnected.sent).toEqual([JSON.stringify({ authToken: 'tok-refreshed' })])

    handle.close()
  })

  it('stops reconnecting once the session is gone', async () => {
    const handle = openAppWebSocket({ url: () => 'wss://x/ws', onMessage: () => {} })
    await advance(0)

    const ws = FakeWebSocket.instances[0]
    ws.open()
    clearAccessToken()
    ws.serverClose()

    await advance(120_000)
    expect(FakeWebSocket.instances.length).toBe(1)

    handle.close()
  })

  it('close() stops all reconnect work', async () => {
    const handle = openAppWebSocket({ url: () => 'wss://x/ws', onMessage: () => {} })
    await advance(0)

    const ws = FakeWebSocket.instances[0]
    ws.open()
    ws.serverClose()
    handle.close()

    await advance(120_000)
    expect(FakeWebSocket.instances.length).toBe(1)
  })
})
