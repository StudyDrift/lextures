import { act, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { StatusBanner } from '../StatusBanner'
import * as bannerApi from '../../lib/banner-api'

vi.mock('../../context/platform-features-context', () => ({
  usePlatformFeatures: () => ({ maintenanceBannerEnabled: true, loading: false }),
}))

vi.mock('../../lib/banner-api', async (importOriginal) => {
  const actual = await importOriginal<typeof bannerApi>()
  return {
    ...actual,
    fetchActiveBanner: vi.fn(),
    BANNER_POLL_INTERVAL_MS: 60_000,
  }
})

vi.mock('../../lib/api', () => ({
  wsUrl: (path: string) => `ws://localhost${path}`,
  apiUrl: (path: string) => `http://localhost${path}`,
}))

vi.mock('../../lib/auth', () => ({
  getAccessToken: () => null,
}))

type MockWS = {
  readyState: number
  onopen: ((ev?: Event) => void) | null
  onmessage: ((ev: MessageEvent) => void) | null
  onclose: ((ev?: CloseEvent) => void) | null
  onerror: ((ev?: Event) => void) | null
  send: ReturnType<typeof vi.fn>
  close: ReturnType<typeof vi.fn>
}

let lastWs: MockWS | null = null

function registerMockWs(ws: MockWS) {
  lastWs = ws
}

function installMockWebSocket() {
  lastWs = null
  class FakeWebSocket {
    static CONNECTING = 0
    static OPEN = 1
    static CLOSING = 2
    static CLOSED = 3
    readyState = FakeWebSocket.CONNECTING
    onopen: MockWS['onopen'] = null
    onmessage: MockWS['onmessage'] = null
    onclose: MockWS['onclose'] = null
    onerror: MockWS['onerror'] = null
    send = vi.fn()
    close = vi.fn(() => {
      this.readyState = FakeWebSocket.CLOSED
    })
    constructor(_url: string) {
      registerMockWs(this)
      queueMicrotask(() => {
        this.readyState = FakeWebSocket.OPEN
        this.onopen?.(new Event('open'))
      })
    }
  }
  vi.stubGlobal('WebSocket', FakeWebSocket)
}

function renderBanner() {
  return render(<StatusBanner />)
}

function mockLocalStorage(): Storage {
  const store = new Map<string, string>()
  return {
    get length() {
      return store.size
    },
    clear() {
      store.clear()
    },
    getItem(key: string) {
      return store.get(key) ?? null
    },
    setItem(key: string, value: string) {
      store.set(key, value)
    },
    removeItem(key: string) {
      store.delete(key)
    },
    key(index: number) {
      return [...store.keys()][index] ?? null
    },
  }
}

describe('StatusBanner', () => {
  beforeEach(() => {
    Object.defineProperty(globalThis, 'localStorage', {
      value: mockLocalStorage(),
      configurable: true,
    })
    vi.mocked(bannerApi.fetchActiveBanner).mockReset()
    installMockWebSocket()
  })

  it('renders an active warning banner', async () => {
    vi.mocked(bannerApi.fetchActiveBanner).mockResolvedValue({
      id: 'b-1',
      scope: 'global',
      message: 'Maintenance at midnight',
      severity: 'warning',
      isActive: true,
      updatedAt: '2026-06-30T12:00:00.000Z',
    })
    renderBanner()
    expect(await screen.findByRole('status')).toHaveTextContent('Maintenance at midnight')
  })

  it('dismisses the banner and persists in localStorage', async () => {
    const user = userEvent.setup()
    vi.mocked(bannerApi.fetchActiveBanner).mockResolvedValue({
      id: 'b-2',
      scope: 'global',
      message: 'Scheduled outage',
      severity: 'error',
      isActive: true,
      updatedAt: '2026-06-30T12:00:00.000Z',
    })
    renderBanner()
    await screen.findByRole('status')
    await user.click(screen.getByRole('button', { name: /dismiss maintenance notice/i }))
    expect(screen.queryByRole('status')).not.toBeInTheDocument()
    expect(localStorage.getItem('lextures.maintenanceBanner.dismissed')).toContain('b-2')
  })

  it('clears the banner immediately when WebSocket reports cleared', async () => {
    vi.mocked(bannerApi.fetchActiveBanner).mockResolvedValue({
      id: 'b-3',
      scope: 'global',
      message: 'Will be cleared',
      severity: 'warning',
      isActive: true,
      updatedAt: '2026-06-30T12:00:00.000Z',
    })
    renderBanner()
    expect(await screen.findByRole('status')).toHaveTextContent('Will be cleared')

    vi.mocked(bannerApi.fetchActiveBanner).mockResolvedValue(null)
    await act(async () => {
      lastWs?.onmessage?.(
        new MessageEvent('message', {
          data: JSON.stringify({ type: 'banner_changed', action: 'cleared', id: 'b-3', scope: 'global' }),
        }),
      )
    })

    expect(screen.queryByRole('status')).not.toBeInTheDocument()
  })
})
