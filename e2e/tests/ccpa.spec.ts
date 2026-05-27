/**
 * CCPA / CPRA compliance (plan 10.4)
 *
 *   [x] GET opt-out without auth returns 401
 *   [x] POST opt-out without auth returns 401
 *   [x] POST requests without auth returns 401
 *   [x] GET requests without auth returns 401
 *   [x] GET pi-categories returns 200 without auth (public endpoint)
 *   [x] All CCPA endpoints return 404 when feature is disabled
 *   [x] User can read initial opt-out state (feature on)
 *   [x] User can toggle Do Not Sell opt-out (feature on)
 *   [x] Sec-GPC header triggers automatic opt-out (feature on)
 *   [x] User can submit a rights request (feature on)
 *   [x] Duplicate request returns 409 (feature on)
 *   [x] Invalid requestType returns 400 (feature on)
 *   [x] PI categories endpoint returns categories (feature on)
 */
import { test, expect } from '../fixtures/test.js'
import { apiSignup } from '../fixtures/api.js'

const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'
const PASSWORD = 'E2eTestPass1!'
const CCPA_ENABLED =
  process.env.FEATURE_CCPA_MODULE === 'true' || process.env.CCPA_MODULE_ENABLED === 'true'

function uniqueEmail(prefix = 'ccpa') {
  return `e2e-${prefix}-${Date.now()}-${Math.random().toString(36).slice(2)}@test.invalid`
}

// ──────────────────────────────────────────────────────────
// Unauthenticated 401 checks (no feature flag dependency)
// ──────────────────────────────────────────────────────────

test('CCPA: GET opt-out unauthenticated returns 401', async () => {
  test.skip(!CCPA_ENABLED, 'requires FEATURE_CCPA_MODULE=true')
  const res = await fetch(`${API_BASE}/api/v1/compliance/ccpa/opt-out`)
  expect(res.status).toBe(401)
})

test('CCPA: POST opt-out unauthenticated returns 401', async () => {
  test.skip(!CCPA_ENABLED, 'requires FEATURE_CCPA_MODULE=true')
  const res = await fetch(`${API_BASE}/api/v1/compliance/ccpa/opt-out`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ doNotSell: true }),
  })
  expect(res.status).toBe(401)
})

test('CCPA: POST requests unauthenticated returns 401', async () => {
  test.skip(!CCPA_ENABLED, 'requires FEATURE_CCPA_MODULE=true')
  const res = await fetch(`${API_BASE}/api/v1/compliance/ccpa/requests`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ requestType: 'know_categories' }),
  })
  expect(res.status).toBe(401)
})

test('CCPA: GET requests unauthenticated returns 401', async () => {
  test.skip(!CCPA_ENABLED, 'requires FEATURE_CCPA_MODULE=true')
  const res = await fetch(`${API_BASE}/api/v1/compliance/ccpa/requests`)
  expect(res.status).toBe(401)
})

test('CCPA: GET pi-categories returns 200 without auth (public)', async () => {
  test.skip(!CCPA_ENABLED, 'requires FEATURE_CCPA_MODULE=true')
  const res = await fetch(`${API_BASE}/api/v1/compliance/ccpa/pi-categories`)
  expect(res.status).toBe(200)
  const body = (await res.json()) as { categories?: unknown[] }
  expect(Array.isArray(body.categories)).toBe(true)
  expect((body.categories ?? []).length).toBeGreaterThan(0)
})

// ──────────────────────────────────────────────────────────
// Feature-disabled 404 checks
// ──────────────────────────────────────────────────────────

test('CCPA: All endpoints return 404 when feature disabled', async () => {
  test.skip(CCPA_ENABLED, 'skipped when FEATURE_CCPA_MODULE=true')
  const { access_token } = await apiSignup({ email: uniqueEmail('dis'), password: PASSWORD })
  const headers = { Authorization: `Bearer ${access_token}`, 'Content-Type': 'application/json' }
  const checks = [
    fetch(`${API_BASE}/api/v1/compliance/ccpa/opt-out`, { headers }),
    fetch(`${API_BASE}/api/v1/compliance/ccpa/requests`, { headers }),
    fetch(`${API_BASE}/api/v1/compliance/ccpa/pi-categories`),
  ]
  const results = await Promise.all(checks)
  for (const res of results) {
    expect(res.status).toBe(404)
  }
})

// ──────────────────────────────────────────────────────────
// Functional tests (require FEATURE_CCPA_MODULE=true)
// ──────────────────────────────────────────────────────────

test('CCPA: User can read and toggle Do Not Sell opt-out', async () => {
  test.skip(!CCPA_ENABLED, 'requires FEATURE_CCPA_MODULE=true')

  const { access_token } = await apiSignup({ email: uniqueEmail('optout'), password: PASSWORD })
  const headers = { Authorization: `Bearer ${access_token}`, 'Content-Type': 'application/json' }

  // Initial state: opted in (do_not_sell = false).
  const getRes = await fetch(`${API_BASE}/api/v1/compliance/ccpa/opt-out`, { headers })
  expect(getRes.ok).toBeTruthy()
  const initial = (await getRes.json()) as { doNotSell: boolean }
  expect(initial.doNotSell).toBe(false)

  // Opt out.
  const postRes = await fetch(`${API_BASE}/api/v1/compliance/ccpa/opt-out`, {
    method: 'POST',
    headers,
    body: JSON.stringify({ doNotSell: true }),
  })
  expect(postRes.ok).toBeTruthy()
  const afterPost = (await postRes.json()) as { doNotSell: boolean; gpcHonoured: boolean }
  expect(afterPost.doNotSell).toBe(true)
  expect(afterPost.gpcHonoured).toBe(false)

  // Confirm persisted.
  const getRes2 = await fetch(`${API_BASE}/api/v1/compliance/ccpa/opt-out`, { headers })
  const persisted = (await getRes2.json()) as { doNotSell: boolean }
  expect(persisted.doNotSell).toBe(true)

  // Opt back in.
  const postRes2 = await fetch(`${API_BASE}/api/v1/compliance/ccpa/opt-out`, {
    method: 'POST',
    headers,
    body: JSON.stringify({ doNotSell: false }),
  })
  expect(postRes2.ok).toBeTruthy()
  const backin = (await postRes2.json()) as { doNotSell: boolean }
  expect(backin.doNotSell).toBe(false)
})

test('CCPA: Sec-GPC header triggers automatic opt-out (AC-1)', async () => {
  test.skip(!CCPA_ENABLED, 'requires FEATURE_CCPA_MODULE=true')

  const { access_token } = await apiSignup({ email: uniqueEmail('gpc'), password: PASSWORD })
  const headers = {
    Authorization: `Bearer ${access_token}`,
    'Content-Type': 'application/json',
    'Sec-GPC': '1',
  }

  const res = await fetch(`${API_BASE}/api/v1/compliance/ccpa/opt-out`, {
    method: 'POST',
    headers,
    body: JSON.stringify({}),
  })
  expect(res.ok).toBeTruthy()
  const body = (await res.json()) as { doNotSell: boolean; gpcHonoured: boolean }
  expect(body.doNotSell).toBe(true)
  expect(body.gpcHonoured).toBe(true)
})

test('CCPA: User can submit a rights request and it appears in their list', async () => {
  test.skip(!CCPA_ENABLED, 'requires FEATURE_CCPA_MODULE=true')

  const { access_token } = await apiSignup({ email: uniqueEmail('req'), password: PASSWORD })
  const headers = { Authorization: `Bearer ${access_token}`, 'Content-Type': 'application/json' }

  // Submit a know_categories request.
  const postRes = await fetch(`${API_BASE}/api/v1/compliance/ccpa/requests`, {
    method: 'POST',
    headers,
    body: JSON.stringify({ requestType: 'know_categories' }),
  })
  expect(postRes.status).toBe(201)
  const createBody = (await postRes.json()) as { id?: string }
  expect(typeof createBody.id).toBe('string')
  expect(createBody.id).toMatch(/^[0-9a-f-]{36}$/)

  // List requests — should contain the new one.
  const listRes = await fetch(`${API_BASE}/api/v1/compliance/ccpa/requests`, { headers })
  expect(listRes.ok).toBeTruthy()
  const listBody = (await listRes.json()) as {
    requests: Array<{ id: string; requestType: string; status: string; dueAt: string }>
  }
  const found = listBody.requests.find((r) => r.id === createBody.id)
  expect(found).toBeDefined()
  expect(found?.requestType).toBe('know_categories')
  expect(found?.status).toBe('pending')

  // Verify due_at is ~45 days from now.
  if (found?.dueAt) {
    const due = new Date(found.dueAt)
    const diffDays = (due.getTime() - Date.now()) / (1000 * 60 * 60 * 24)
    expect(diffDays).toBeGreaterThan(44)
    expect(diffDays).toBeLessThan(46)
  }

  // Get individual request.
  const getRes = await fetch(`${API_BASE}/api/v1/compliance/ccpa/requests/${createBody.id}`, {
    headers,
  })
  expect(getRes.ok).toBeTruthy()
  const getBody = (await getRes.json()) as { id: string; status: string }
  expect(getBody.id).toBe(createBody.id)
  expect(getBody.status).toBe('pending')
})

test('CCPA: Duplicate request returns 409', async () => {
  test.skip(!CCPA_ENABLED, 'requires FEATURE_CCPA_MODULE=true')

  const { access_token } = await apiSignup({ email: uniqueEmail('dup'), password: PASSWORD })
  const headers = { Authorization: `Bearer ${access_token}`, 'Content-Type': 'application/json' }

  // First submission.
  const first = await fetch(`${API_BASE}/api/v1/compliance/ccpa/requests`, {
    method: 'POST',
    headers,
    body: JSON.stringify({ requestType: 'delete' }),
  })
  expect(first.status).toBe(201)

  // Second submission of same type.
  const second = await fetch(`${API_BASE}/api/v1/compliance/ccpa/requests`, {
    method: 'POST',
    headers,
    body: JSON.stringify({ requestType: 'delete' }),
  })
  expect(second.status).toBe(409)
})

test('CCPA: Invalid requestType returns 400', async () => {
  test.skip(!CCPA_ENABLED, 'requires FEATURE_CCPA_MODULE=true')

  const { access_token } = await apiSignup({ email: uniqueEmail('invalid'), password: PASSWORD })

  const res = await fetch(`${API_BASE}/api/v1/compliance/ccpa/requests`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${access_token}`, 'Content-Type': 'application/json' },
    body: JSON.stringify({ requestType: 'sell_everything' }),
  })
  expect(res.status).toBe(400)
})

test('CCPA: PI categories returns expected structure', async () => {
  test.skip(!CCPA_ENABLED, 'requires FEATURE_CCPA_MODULE=true')

  const res = await fetch(`${API_BASE}/api/v1/compliance/ccpa/pi-categories`)
  expect(res.ok).toBeTruthy()
  const body = (await res.json()) as {
    categories: Array<{ category: string; purpose: string; examples: string[] }>
  }
  expect(body.categories.length).toBeGreaterThan(0)
  for (const cat of body.categories) {
    expect(cat.category).toBeTruthy()
    expect(cat.purpose).toBeTruthy()
    expect(Array.isArray(cat.examples)).toBe(true)
  }
})

test('CCPA: User can toggle Limit Sensitive PI', async () => {
  test.skip(!CCPA_ENABLED, 'requires FEATURE_CCPA_MODULE=true')

  const { access_token } = await apiSignup({ email: uniqueEmail('senspi'), password: PASSWORD })
  const headers = { Authorization: `Bearer ${access_token}`, 'Content-Type': 'application/json' }

  const res = await fetch(`${API_BASE}/api/v1/compliance/ccpa/opt-out`, {
    method: 'POST',
    headers,
    body: JSON.stringify({ limitSensitivePI: true }),
  })
  expect(res.ok).toBeTruthy()
  const body = (await res.json()) as { limitSensitivePI: boolean }
  expect(body.limitSensitivePI).toBe(true)
})
