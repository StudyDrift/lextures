import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { authorizedFetch, RequestTimeoutError } from '../api'

/** A fetch that never answers but honours AbortSignal, like a real stalled request. */
function stalledFetch(): typeof fetch {
  return vi.fn((_url: RequestInfo | URL, init?: RequestInit) => {
    return new Promise<Response>((_resolve, reject) => {
      init?.signal?.addEventListener('abort', () => {
        reject(new DOMException('The operation was aborted.', 'AbortError'))
      })
    })
  }) as unknown as typeof fetch
}

describe('authorizedFetch timeout', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })
  afterEach(() => {
    vi.useRealTimers()
    vi.unstubAllGlobals()
  })

  it('rejects with RequestTimeoutError when the server never answers', async () => {
    const f = stalledFetch()
    vi.stubGlobal('fetch', f)
    const p = authorizedFetch('/api/v1/courses/C-1', { timeoutMs: 1000 })
    const assertion = expect(p).rejects.toBeInstanceOf(RequestTimeoutError)
    await vi.advanceTimersByTimeAsync(1000)
    await assertion
    // Timeouts are not retried: the page should show Retry instead of waiting 3x longer.
    expect(f).toHaveBeenCalledTimes(1)
  })

  it('times out a stalled POST so a submit button can recover', async () => {
    vi.stubGlobal('fetch', stalledFetch())
    const p = authorizedFetch('/api/v1/courses/C-1/enrollments', {
      method: 'POST',
      body: '{}',
      timeoutMs: 500,
    })
    const assertion = expect(p).rejects.toThrow(/timed out/i)
    await vi.advanceTimersByTimeAsync(500)
    await assertion
  })

  it('returns the response and clears the timer when the server answers in time', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('{}', { status: 200 })))
    const res = await authorizedFetch('/api/v1/ok', { timeoutMs: 1000 })
    expect(res.status).toBe(200)
    expect(vi.getTimerCount()).toBe(0)
  })

  it('applies the default timeout to GET but not to writes', async () => {
    const f = vi.fn().mockResolvedValue(new Response('{}', { status: 200 }))
    vi.stubGlobal('fetch', f)
    await authorizedFetch('/api/v1/x')
    await authorizedFetch('/api/v1/upload', { method: 'POST', body: new FormData() })
    const initOf = (call: number) => f.mock.calls[call][1] as RequestInit
    expect(initOf(0).signal).toBeDefined()
    expect(initOf(1).signal).toBeUndefined()
  })

  it('still aborts when the caller signal aborts', async () => {
    vi.stubGlobal('fetch', stalledFetch())
    const ctl = new AbortController()
    const p = authorizedFetch('/api/v1/x', { signal: ctl.signal })
    const assertion = expect(p).rejects.not.toBeInstanceOf(RequestTimeoutError)
    ctl.abort()
    await assertion
  })
})
