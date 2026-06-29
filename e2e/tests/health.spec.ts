/**
 * Health probes — liveness and readiness (plan 17.8).
 */
import { test, expect } from '../fixtures/test.js'

const apiBase = process.env.E2E_API_URL ?? 'http://localhost:8080'

test.describe('Health probes', () => {
  test('GET /health/live returns ok without dependency checks (AC-3)', async () => {
    const res = await fetch(`${apiBase}/health/live`)
    expect(res.status).toBe(200)
    const body = await res.json()
    expect(body).toEqual({ status: 'ok' })
  })

  test('GET /health/ready returns structured checks when dependencies are up (AC-1)', async () => {
    const res = await fetch(`${apiBase}/health/ready`)
    expect(res.status).toBe(200)
    const body = await res.json()
    expect(body.status).toBe('ready')
    expect(body.checks.postgres).toBe('ok')
    expect(body.checks.redis).toBeDefined()
    const text = JSON.stringify(body)
    expect(text).not.toMatch(/postgres:\/\//)
    expect(text).not.toMatch(/password/i)
  })

  test('GET /health/detailed requires authentication (AC-4)', async ({ request }) => {
    const res = await request.get(`${apiBase}/health/detailed`)
    expect(res.status()).toBe(401)
  })

  test('legacy GET /health remains a liveness alias', async () => {
    const res = await fetch(`${apiBase}/health`)
    expect(res.status).toBe(200)
    const body = await res.json()
    expect(body).toEqual({ status: 'ok' })
  })
})
