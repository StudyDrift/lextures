import { afterEach, describe, expect, it, vi } from 'vitest'
import { BridgeRateLimiter, measureMessageBytes } from '@lextures/tool-sdk'
import { createHostBridge, opaqueParticipantId } from '../bridge'

describe('content tools sandbox bridge (CT.5)', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('drops malformed messages and wrong origins', () => {
    const onInvalid = vi.fn()
    const metrics: string[] = []
    const fakeSource = {} as Window
    const bridge = createHostBridge({
      toolId: 'sandbox_probe',
      expectedOrigin: 'null',
      source: fakeSource,
      handlers: {
        onReady: vi.fn(),
        onSave: vi.fn(),
        onRunAction: vi.fn(),
        onResize: vi.fn(),
        onAnnounce: vi.fn(),
        onInvalid,
      },
      onMetric: (t, o) => metrics.push(`${t}:${o}`),
    })

    window.dispatchEvent(
      new MessageEvent('message', {
        data: { t: 'save', v: 1, state: {}, revision: 0 },
        origin: 'https://evil.example',
        source: fakeSource as MessageEventSource,
      }),
    )
    expect(onInvalid).toHaveBeenCalledWith('origin')

    window.dispatchEvent(
      new MessageEvent('message', {
        data: { hello: 'world' },
        origin: 'null',
        source: fakeSource as MessageEventSource,
      }),
    )
    expect(onInvalid).toHaveBeenCalledWith('schema')
    expect(metrics).toContain('unknown:malformed')
    bridge.dispose()
  })

  it('accepts valid ready from opaque origin', () => {
    const onReady = vi.fn()
    const fakeSource = {} as Window
    const bridge = createHostBridge({
      toolId: 'sandbox_probe',
      expectedOrigin: 'null',
      source: fakeSource,
      handlers: {
        onReady,
        onSave: vi.fn(),
        onRunAction: vi.fn(),
        onResize: vi.fn(),
        onAnnounce: vi.fn(),
        onInvalid: vi.fn(),
      },
    })
    window.dispatchEvent(
      new MessageEvent('message', {
        data: { t: 'ready', v: 1, contract: '1' },
        origin: 'null',
        source: fakeSource as MessageEventSource,
      }),
    )
    expect(onReady).toHaveBeenCalledWith('1')
    bridge.dispose()
  })

  it('rate-limits burst traffic', () => {
    const limiter = new BridgeRateLimiter(2)
    expect(limiter.allow(1000)).toBe(true)
    expect(limiter.allow(1001)).toBe(true)
    expect(limiter.allow(1002)).toBe(false)
  })

  it('measures message bytes and builds opaque participant ids', () => {
    expect(measureMessageBytes({ t: 'ready', v: 1, contract: '1' })).toBeGreaterThan(10)
    expect(opaqueParticipantId('inst', 'enr')).toMatch(/^p_[0-9a-f]+$/)
  })
})
