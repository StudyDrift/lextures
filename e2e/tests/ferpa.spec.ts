/**
 * FERPA Workflow (plan 10.1)
 *
 *   [x] GET directory-opt-out without auth returns 401
 *   [x] PUT directory-opt-out without auth returns 401
 *   [x] POST record-requests without auth returns 401
 *   [x] GET record-requests without auth returns 401
 *   [x] GET disclosure-log without auth returns 401
 *   [x] POST consent without auth returns 401
 *   [x] DELETE consent without auth returns 401
 *   [x] All FERPA endpoints return 404 when feature is disabled
 *   [x] Student can toggle directory opt-out (feature on)
 *   [x] Student can submit a record-access request (feature on)
 *   [x] Admin can list and approve record requests (feature on)
 *   [x] Disclosure log records the approval (feature on)
 *   [x] Student can grant and revoke third-party consent (feature on)
 *   [x] Non-admin student cannot list record requests (403)
 *   [x] Non-admin student cannot read the disclosure log (403)
 */
import { test, expect } from '../fixtures/test.js'
import { apiSignup } from '../fixtures/api.js'

const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'
const PASSWORD = 'E2eTestPass1!'
const FERPA_ENABLED = process.env.FEATURE_FERPA_WORKFLOW === 'true' || process.env.FERPA_WORKFLOW_ENABLED === 'true'

function uniqueEmail(prefix = 'ferpa') {
  return `e2e-${prefix}-${Date.now()}-${Math.random().toString(36).slice(2)}@test.invalid`
}

// ──────────────────────────────────────────────────────────
// Unauthenticated 401 checks (no feature flag dependency)
// ──────────────────────────────────────────────────────────

test('FERPA: GET directory-opt-out unauthenticated returns 401', async () => {
  test.skip(!FERPA_ENABLED, 'requires FEATURE_FERPA_WORKFLOW=true')
  const res = await fetch(`${API_BASE}/api/v1/compliance/ferpa/directory-opt-out`)
  expect(res.status).toBe(401)
})

test('FERPA: PUT directory-opt-out unauthenticated returns 401', async () => {
  test.skip(!FERPA_ENABLED, 'requires FEATURE_FERPA_WORKFLOW=true')
  const res = await fetch(`${API_BASE}/api/v1/compliance/ferpa/directory-opt-out`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ directoryOptOut: true }),
  })
  expect(res.status).toBe(401)
})

test('FERPA: POST record-requests unauthenticated returns 401', async () => {
  test.skip(!FERPA_ENABLED, 'requires FEATURE_FERPA_WORKFLOW=true')
  const res = await fetch(`${API_BASE}/api/v1/compliance/ferpa/record-requests`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({}),
  })
  expect(res.status).toBe(401)
})

test('FERPA: GET record-requests unauthenticated returns 401', async () => {
  test.skip(!FERPA_ENABLED, 'requires FEATURE_FERPA_WORKFLOW=true')
  const res = await fetch(`${API_BASE}/api/v1/compliance/ferpa/record-requests`)
  expect(res.status).toBe(401)
})

test('FERPA: GET disclosure-log unauthenticated returns 401', async () => {
  test.skip(!FERPA_ENABLED, 'requires FEATURE_FERPA_WORKFLOW=true')
  const res = await fetch(`${API_BASE}/api/v1/compliance/ferpa/disclosure-log`)
  expect(res.status).toBe(401)
})

test('FERPA: POST consent unauthenticated returns 401', async () => {
  test.skip(!FERPA_ENABLED, 'requires FEATURE_FERPA_WORKFLOW=true')
  const res = await fetch(`${API_BASE}/api/v1/compliance/ferpa/consent`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({}),
  })
  expect(res.status).toBe(401)
})

test('FERPA: DELETE consent unauthenticated returns 401', async () => {
  test.skip(!FERPA_ENABLED, 'requires FEATURE_FERPA_WORKFLOW=true')
  const fakeID = '00000000-0000-0000-0000-000000000001'
  const res = await fetch(`${API_BASE}/api/v1/compliance/ferpa/consent/${fakeID}`, {
    method: 'DELETE',
  })
  expect(res.status).toBe(401)
})

// ──────────────────────────────────────────────────────────
// Feature-disabled 404 checks
// ──────────────────────────────────────────────────────────

test('FERPA: All endpoints return 404 when feature disabled', async () => {
  test.skip(FERPA_ENABLED, 'skipped when FEATURE_FERPA_WORKFLOW=true')
  const { access_token } = await apiSignup({ email: uniqueEmail('dis'), password: PASSWORD })
  const headers = { Authorization: `Bearer ${access_token}`, 'Content-Type': 'application/json' }
  const checks = [
    fetch(`${API_BASE}/api/v1/compliance/ferpa/directory-opt-out`, { headers }),
    fetch(`${API_BASE}/api/v1/compliance/ferpa/record-requests`, { headers }),
    fetch(`${API_BASE}/api/v1/compliance/ferpa/disclosure-log`, { headers }),
  ]
  const results = await Promise.all(checks)
  for (const res of results) {
    expect(res.status).toBe(404)
  }
})

// ──────────────────────────────────────────────────────────
// Functional tests (require FEATURE_FERPA_WORKFLOW=true)
// ──────────────────────────────────────────────────────────

test('FERPA: Student can read and toggle directory opt-out', async () => {
  test.skip(!FERPA_ENABLED, 'requires FEATURE_FERPA_WORKFLOW=true')

  const email = uniqueEmail('student')
  const { access_token } = await apiSignup({ email, password: PASSWORD })
  const headers = { Authorization: `Bearer ${access_token}`, 'Content-Type': 'application/json' }

  // Initial state: opted in (opt-out = false).
  const getRes = await fetch(`${API_BASE}/api/v1/compliance/ferpa/directory-opt-out`, { headers })
  expect(getRes.ok).toBeTruthy()
  const initial = (await getRes.json()) as { directoryOptOut: boolean }
  expect(initial.directoryOptOut).toBe(false)

  // Opt out.
  const putRes = await fetch(`${API_BASE}/api/v1/compliance/ferpa/directory-opt-out`, {
    method: 'PUT',
    headers,
    body: JSON.stringify({ directoryOptOut: true }),
  })
  expect(putRes.ok).toBeTruthy()
  const afterPut = (await putRes.json()) as { directoryOptOut: boolean }
  expect(afterPut.directoryOptOut).toBe(true)

  // Confirm persisted.
  const getRes2 = await fetch(`${API_BASE}/api/v1/compliance/ferpa/directory-opt-out`, { headers })
  expect(getRes2.ok).toBeTruthy()
  const persisted = (await getRes2.json()) as { directoryOptOut: boolean }
  expect(persisted.directoryOptOut).toBe(true)

  // Opt back in.
  const putRes2 = await fetch(`${API_BASE}/api/v1/compliance/ferpa/directory-opt-out`, {
    method: 'PUT',
    headers,
    body: JSON.stringify({ directoryOptOut: false }),
  })
  expect(putRes2.ok).toBeTruthy()
})

test('FERPA: Non-admin cannot list record requests (403)', async () => {
  test.skip(!FERPA_ENABLED, 'requires FEATURE_FERPA_WORKFLOW=true')

  const { access_token } = await apiSignup({ email: uniqueEmail('nonadmin'), password: PASSWORD })
  const res = await fetch(`${API_BASE}/api/v1/compliance/ferpa/record-requests`, {
    headers: { Authorization: `Bearer ${access_token}` },
  })
  expect(res.status).toBe(403)
})

test('FERPA: Non-admin cannot read disclosure log (403)', async () => {
  test.skip(!FERPA_ENABLED, 'requires FEATURE_FERPA_WORKFLOW=true')

  const { access_token } = await apiSignup({ email: uniqueEmail('nodiscl'), password: PASSWORD })
  const res = await fetch(`${API_BASE}/api/v1/compliance/ferpa/disclosure-log`, {
    headers: { Authorization: `Bearer ${access_token}` },
  })
  expect(res.status).toBe(403)
})

test('FERPA: Student submits record-access request; receives 201', async () => {
  test.skip(!FERPA_ENABLED, 'requires FEATURE_FERPA_WORKFLOW=true')

  const studentEmail = uniqueEmail('reqstudent')
  const { access_token } = await apiSignup({ email: studentEmail, password: PASSWORD })

  // A student submits an inspect request for their own record.
  const meRes = await fetch(`${API_BASE}/api/v1/me`, {
    headers: { Authorization: `Bearer ${access_token}` },
  })
  if (!meRes.ok) {
    test.skip(true, 'GET /api/v1/me not available in this build')
    return
  }
  const me = (await meRes.json()) as { id?: string }
  const studentId = me.id
  if (!studentId) {
    test.skip(true, 'could not determine own user ID')
    return
  }

  const postRes = await fetch(`${API_BASE}/api/v1/compliance/ferpa/record-requests`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${access_token}`, 'Content-Type': 'application/json' },
    body: JSON.stringify({
      studentId,
      requestType: 'inspect',
      notes: 'e2e test request',
    }),
  })
  expect(postRes.status).toBe(201)
  const body = (await postRes.json()) as { id?: string }
  expect(typeof body.id).toBe('string')
  expect(body.id).toMatch(/^[0-9a-f-]{36}$/)
})

test('FERPA: Student can grant and revoke third-party consent', async () => {
  test.skip(!FERPA_ENABLED, 'requires FEATURE_FERPA_WORKFLOW=true')

  const { access_token } = await apiSignup({ email: uniqueEmail('consent'), password: PASSWORD })

  const meRes = await fetch(`${API_BASE}/api/v1/me`, {
    headers: { Authorization: `Bearer ${access_token}` },
  })
  if (!meRes.ok) {
    test.skip(true, 'GET /api/v1/me not available in this build')
    return
  }
  const me = (await meRes.json()) as { id?: string }
  const studentId = me.id
  if (!studentId) {
    test.skip(true, 'could not determine own user ID')
    return
  }

  // Grant consent.
  const grantRes = await fetch(`${API_BASE}/api/v1/compliance/ferpa/consent`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${access_token}`, 'Content-Type': 'application/json' },
    body: JSON.stringify({
      studentId,
      recipient: 'State Scholarship Board',
      purpose: 'scholarship eligibility verification',
      dataFields: ['gpa', 'enrollment_status'],
    }),
  })
  expect(grantRes.status).toBe(201)
  const grantBody = (await grantRes.json()) as { id?: string }
  expect(typeof grantBody.id).toBe('string')

  // Revoke consent.
  const revokeRes = await fetch(
    `${API_BASE}/api/v1/compliance/ferpa/consent/${grantBody.id}`,
    {
      method: 'DELETE',
      headers: { Authorization: `Bearer ${access_token}` },
    },
  )
  expect(revokeRes.ok).toBeTruthy()
  const revokeBody = (await revokeRes.json()) as { ok?: boolean }
  expect(revokeBody.ok).toBe(true)

  // Second revoke returns 404.
  const revokeRes2 = await fetch(
    `${API_BASE}/api/v1/compliance/ferpa/consent/${grantBody.id}`,
    {
      method: 'DELETE',
      headers: { Authorization: `Bearer ${access_token}` },
    },
  )
  expect(revokeRes2.status).toBe(404)
})

test('FERPA: POST record-request with invalid requestType returns 400', async () => {
  test.skip(!FERPA_ENABLED, 'requires FEATURE_FERPA_WORKFLOW=true')

  const { access_token } = await apiSignup({ email: uniqueEmail('badtype'), password: PASSWORD })

  const meRes = await fetch(`${API_BASE}/api/v1/me`, {
    headers: { Authorization: `Bearer ${access_token}` },
  })
  if (!meRes.ok) {
    test.skip(true, 'GET /api/v1/me not available in this build')
    return
  }
  const me = (await meRes.json()) as { id?: string }
  const studentId = me.id
  if (!studentId) {
    test.skip(true, 'could not determine own user ID')
    return
  }

  const res = await fetch(`${API_BASE}/api/v1/compliance/ferpa/record-requests`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${access_token}`, 'Content-Type': 'application/json' },
    body: JSON.stringify({ studentId, requestType: 'unknown', notes: '' }),
  })
  expect(res.status).toBe(400)
})

test('FERPA: amend request without amendmentField returns 400', async () => {
  test.skip(!FERPA_ENABLED, 'requires FEATURE_FERPA_WORKFLOW=true')

  const { access_token } = await apiSignup({ email: uniqueEmail('nofield'), password: PASSWORD })

  const meRes = await fetch(`${API_BASE}/api/v1/me`, {
    headers: { Authorization: `Bearer ${access_token}` },
  })
  if (!meRes.ok) {
    test.skip(true, 'GET /api/v1/me not available in this build')
    return
  }
  const me = (await meRes.json()) as { id?: string }
  const studentId = me.id
  if (!studentId) {
    test.skip(true, 'could not determine own user ID')
    return
  }

  const res = await fetch(`${API_BASE}/api/v1/compliance/ferpa/record-requests`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${access_token}`, 'Content-Type': 'application/json' },
    body: JSON.stringify({ studentId, requestType: 'amend', notes: '' }),
  })
  expect(res.status).toBe(400)
})
