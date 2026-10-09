import { getBearerToken } from './auth'
import {
  applyAuthTokenResponse,
  clearSessionTokens,
  getRefreshToken,
} from './session-tokens'

const defaultApi = 'http://localhost:8080'

/** Resolves the API origin. Treats empty/whitespace VITE_API_URL as unset (same-origin when in the browser). */
export function apiBaseUrl(): string {
  const v = import.meta.env.VITE_API_URL
  if (v == null) {
    if (import.meta.env.MODE === 'test') {
      return defaultApi
    }
    return typeof window !== 'undefined' ? window.location.origin : defaultApi
  }
  const s = String(v).trim()
  if (s === '') {
    return typeof window !== 'undefined' ? window.location.origin : defaultApi
  }
  return s
}

const MAX_IDEMPOTENT_ATTEMPTS = 3

/** How long an API request may wait for response headers before it is aborted. */
export const DEFAULT_REQUEST_TIMEOUT_MS = 30_000

/** Thrown by {@link authorizedFetch} when the server does not answer within the request timeout. */
export class RequestTimeoutError extends Error {
  constructor(message = 'The request timed out. Check your connection and try again.') {
    super(message)
    this.name = 'RequestTimeoutError'
  }
}

export type AuthorizedFetchInit = RequestInit & {
  /**
   * Abort the request when response headers have not arrived after this many milliseconds.
   * GET/HEAD default to {@link DEFAULT_REQUEST_TIMEOUT_MS}; `0` disables the timeout. Writes
   * (POST/PUT/PATCH/DELETE) have no default because some legitimately run long (AI generation,
   * exports, uploads); pass `timeoutMs` for writes that should fail fast.
   */
  timeoutMs?: number
}

function resolveTimeoutMs(init: AuthorizedFetchInit | undefined): number {
  if (init?.timeoutMs !== undefined) return Math.max(0, init.timeoutMs)
  const method = (init?.method ?? 'GET').toUpperCase()
  return method === 'GET' || method === 'HEAD' ? DEFAULT_REQUEST_TIMEOUT_MS : 0
}

/**
 * `fetch` that rejects with {@link RequestTimeoutError} when no response arrives in time.
 * The timer only covers the wait for headers, so long-lived streaming bodies are not cut off.
 */
async function fetchWithTimeout(
  url: string,
  init: RequestInit,
  timeoutMs: number,
): Promise<Response> {
  if (timeoutMs <= 0) {
    return fetch(url, init)
  }
  const controller = new AbortController()
  const callerSignal = init.signal
  const onCallerAbort = () => controller.abort(callerSignal?.reason)
  if (callerSignal) {
    if (callerSignal.aborted) {
      controller.abort(callerSignal.reason)
    } else {
      callerSignal.addEventListener('abort', onCallerAbort, { once: true })
    }
  }
  let timedOut = false
  const timer = setTimeout(() => {
    timedOut = true
    controller.abort()
  }, timeoutMs)
  try {
    return await fetch(url, { ...init, signal: controller.signal })
  } catch (err) {
    if (timedOut) throw new RequestTimeoutError()
    throw err
  } finally {
    clearTimeout(timer)
    callerSignal?.removeEventListener('abort', onCallerAbort)
  }
}
const BASE_BACKOFF_MS = 250

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => {
    setTimeout(resolve, ms)
  })
}

/** Exponential backoff with jitter (50%–100% of base × 2^attempt). */
export function backoffWithJitterMs(attempt: number): number {
  const base = BASE_BACKOFF_MS * 2 ** attempt
  return base * (0.5 + Math.random() * 0.5)
}

/** Pure URL join — unit-tested without env; used by {@link apiUrl}. */
export function joinApiBase(base: string, path: string): string {
  const b = base.replace(/\/$/, '')
  const p = path.startsWith('/') ? path : `/${path}`
  return `${b}${p}`
}

export function apiUrl(path: string): string {
  return joinApiBase(apiBaseUrl(), path)
}

function shouldProxyWebSocketViaPageOrigin(): boolean {
  if (typeof window === 'undefined') {
    return false
  }
  const api = new URL(apiBaseUrl())
  if (api.host === window.location.host) {
    return false
  }
  // Local dev: Vite on :5173 proxying to Go API on :8080 (direct ws://localhost:8080
  // fails when port 8080 is SSH-tunneled without WebSocket upgrade support).
  return api.hostname === 'localhost' || api.hostname === '127.0.0.1'
}

/**
 * WebSocket URL for the same API host as {@link apiUrl}.
 * In local dev, when the SPA and API run on different ports, route through the page
 * origin so Vite's `/api` proxy handles the upgrade.
 */
export function wsUrl(path: string): string {
  const p = path.startsWith('/') ? path : `/${path}`
  if (shouldProxyWebSocketViaPageOrigin()) {
    const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    return `${proto}//${window.location.host}${p}`
  }
  return apiUrl(path).replace(/^http/, 'ws')
}

function dispatchAuthRequired(): void {
  if (typeof window !== 'undefined') {
    window.dispatchEvent(new Event('studydrift-auth-required'))
  }
}

function applyUnauthorizedHandling(res: Response): void {
  if (res.status === 401) {
    clearSessionTokens()
    dispatchAuthRequired()
  }
}

let refreshInFlight: Promise<boolean> | null = null

/** POST /api/v1/auth/refresh; returns true when a new access token was stored. */
export async function tryRefreshSession(): Promise<boolean> {
  const rt = getRefreshToken()
  if (!rt) {
    return false
  }
  if (refreshInFlight) {
    return refreshInFlight
  }
  refreshInFlight = (async () => {
    try {
      const res = await fetchWithTimeout(
        apiUrl('/api/v1/auth/refresh'),
        {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ refresh_token: rt }),
        },
        DEFAULT_REQUEST_TIMEOUT_MS,
      )
      if (!res.ok) {
        clearSessionTokens()
        dispatchAuthRequired()
        return false
      }
      const raw: unknown = await res.json().catch(() => ({}))
      const data = raw as { access_token?: string; refresh_token?: string }
      if (!data.access_token) {
        clearSessionTokens()
        dispatchAuthRequired()
        return false
      }
      applyAuthTokenResponse(data)
      return true
    } catch (err) {
      // A stalled refresh is not proof the session is invalid; keep the tokens for the next try.
      if (err instanceof RequestTimeoutError) return false
      clearSessionTokens()
      dispatchAuthRequired()
      return false
    } finally {
      refreshInFlight = null
    }
  })()
  return refreshInFlight
}

/**
 * `fetch` to the API with `Authorization: Bearer` when a token exists.
 * GET/HEAD requests retry transient 5xx responses and network failures (with jittered backoff).
 */
export async function authorizedFetch(path: string, init?: AuthorizedFetchInit): Promise<Response> {
  const fetchInit: RequestInit = { ...init }
  delete (fetchInit as AuthorizedFetchInit).timeoutMs
  const timeoutMs = resolveTimeoutMs(init)
  const method = (init?.method ?? 'GET').toUpperCase()
  const allowRetry = method === 'GET' || method === 'HEAD'
  const attempts = allowRetry ? MAX_IDEMPOTENT_ATTEMPTS : 1

  let lastNetworkError: unknown

  for (let attempt = 0; attempt < attempts; attempt++) {
    const headers = new Headers(init?.headers)
    const token = getBearerToken()
    if (token) {
      headers.set('Authorization', `Bearer ${token}`)
    }

    try {
      let res = await fetchWithTimeout(apiUrl(path), { ...fetchInit, headers }, timeoutMs)

      if (res.status === 401 && getRefreshToken() && path !== '/api/v1/auth/refresh') {
        const ok = await tryRefreshSession()
        if (ok) {
          const h2 = new Headers(init?.headers)
          const t2 = getBearerToken()
          if (t2) {
            h2.set('Authorization', `Bearer ${t2}`)
          }
          res = await fetchWithTimeout(apiUrl(path), { ...fetchInit, headers: h2 }, timeoutMs)
        }
      }

      applyUnauthorizedHandling(res)

      const transient =
        allowRetry &&
        attempt < attempts - 1 &&
        res.status >= 500 &&
        res.status < 600

      if (transient) {
        await sleep(backoffWithJitterMs(attempt))
        continue
      }

      return res
    } catch (err) {
      lastNetworkError = err
      // A stalled origin will likely stall again; surface the timeout so the page can offer Retry.
      if (err instanceof RequestTimeoutError) throw err
      if (init?.signal?.aborted) throw err
      if (allowRetry && attempt < attempts - 1) {
        await sleep(backoffWithJitterMs(attempt))
        continue
      }
      throw err
    }
  }

  throw lastNetworkError ?? new Error('authorizedFetch: exhausted retries without response')
}
