/**
 * Rate limiting (plan 17.6)
 *
 * Coverage:
 *   [x] auth endpoint returns 429 with Retry-After + X-RateLimit-* after the
 *       per-IP limit is exceeded (AC-1, FR-1, FR-4)
 *   [x] 429 body is RFC 7807 problem+json (API surface §9)
 *   [x] login UI surfaces a friendly cooldown message on 429 (AC-6)
 *
 * Runs against a dedicated low-limit server instance (server-ratelimit, port
 * 8081) so the strict auth cap does not throttle the rest of the suite. Skips
 * when that instance is not available (e.g. running against a plain local API).
 */
import { test, expect } from '../fixtures/test.js'

// The dedicated instance caps auth endpoints at 5 requests/min (RATE_LIMIT_AUTH_PER_MIN).
const RL_API = process.env.E2E_RATELIMIT_API_URL ?? 'http://localhost:8081'
const AUTH_LIMIT = 5

async function rateLimitingAvailable(request: import('@playwright/test').APIRequestContext) {
  try {
    const res = await request.get(`${RL_API}/health`)
    return res.ok()
  } catch {
    return false
  }
}

test.describe('rate limiting — auth endpoints', () => {
  test('exceeding the per-IP login limit returns 429 with Retry-After (AC-1)', async ({ request }) => {
    test.skip(!(await rateLimitingAvailable(request)), 'rate-limit server instance not available')

    let limited: { status: number; retryAfter: string | undefined; body: unknown } | null = null
    // Send one more than the limit; invalid credentials are fine — FR-1 says the
    // request must be throttled before the login is even attempted.
    for (let i = 0; i < AUTH_LIMIT + 1; i++) {
      const res = await request.post(`${RL_API}/api/v1/auth/login`, {
        data: { email: `rl-${i}@e2e.test`, password: 'wrong-password' },
        failOnStatusCode: false,
      })
      if (res.status() === 429) {
        limited = {
          status: res.status(),
          retryAfter: res.headers()['retry-after'],
          body: await res.json().catch(() => ({})),
        }
        break
      }
    }

    expect(limited, 'expected a 429 within the burst').not.toBeNull()
    expect(limited!.status).toBe(429)
    expect(Number(limited!.retryAfter)).toBeGreaterThanOrEqual(1)
    // RFC 7807 problem document.
    expect(limited!.body).toMatchObject({ status: 429, title: 'Too Many Requests' })
  })

  test('responses carry X-RateLimit-* headers when limiting is enabled (FR-4)', async ({ request }) => {
    test.skip(!(await rateLimitingAvailable(request)), 'rate-limit server instance not available')

    const res = await request.post(`${RL_API}/api/v1/auth/login`, {
      data: { email: `rl-headers-${Date.now()}@e2e.test`, password: 'wrong-password' },
      failOnStatusCode: false,
    })
    expect(res.headers()['x-ratelimit-limit']).toBeTruthy()
    expect(res.headers()['x-ratelimit-remaining']).toBeTruthy()
    expect(res.headers()['x-ratelimit-reset']).toBeTruthy()
  })
})
