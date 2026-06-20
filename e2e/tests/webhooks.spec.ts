import { test, expect } from '../fixtures/test'

const API_BASE = process.env.API_BASE ?? 'http://localhost:8080'

test('Webhooks: GET subscriptions unauthenticated returns 401', async () => {
  const res = await fetch(`${API_BASE}/api/v1/webhooks`)
  expect(res.status).toBe(401)
})

test('Webhooks: POST subscription unauthenticated returns 401', async () => {
  const res = await fetch(`${API_BASE}/api/v1/webhooks`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      label: 'Test',
      endpointUrl: 'https://hooks.example.edu/lextures',
      eventTypes: ['grade.posted'],
    }),
  })
  expect(res.status).toBe(401)
})

test('Webhooks: GET admin subscriptions unauthenticated returns 401', async () => {
  const res = await fetch(
    `${API_BASE}/api/v1/admin/orgs/00000000-0000-0000-0000-000000000001/webhooks`,
  )
  expect(res.status).toBe(401)
})

test('Webhooks: GET event-types returns catalog when feature enabled', async ({ request }) => {
  const res = await request.get(`${API_BASE}/api/v1/webhooks/event-types`)
  if (res.status() === 501) {
    test.skip()
    return
  }
  expect(res.status()).toBe(200)
  const data = (await res.json()) as { eventTypes: string[] }
  expect(data.eventTypes).toContain('grade.posted')
  expect(data.eventTypes).toContain('enrollment.created')
})
