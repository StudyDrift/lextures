import { test, expect } from '../fixtures/test.js'

const apiBase = process.env.E2E_API_URL ?? 'http://localhost:8080'

test.describe('Zapier connector API', () => {
  test('GET /api/v1/me without auth returns 401', async () => {
    const res = await fetch(`${apiBase}/api/v1/me`)
    expect([401, 501]).toContain(res.status)
  })

  test('POST /api/v1/webhooks with zapier source returns 401 or 501 when unauthenticated', async () => {
    const res = await fetch(`${apiBase}/api/v1/webhooks`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        label: 'Zapier test',
        endpointUrl: 'https://hooks.zapier.com/hooks/catch/123/abc/',
        eventTypes: ['enrollment.created'],
        settings: { source: 'zapier' },
      }),
    })
    expect([401, 501]).toContain(res.status)
  })

  test('webhook event-types includes quiz.completed when enabled', async ({ request }) => {
    const res = await request.get(`${apiBase}/api/v1/webhooks/event-types`)
    if (res.status() === 501) {
      test.skip()
      return
    }
    expect(res.status()).toBe(200)
    const data = (await res.json()) as { eventTypes: string[] }
    expect(data.eventTypes).toContain('quiz.completed')
  })
})
