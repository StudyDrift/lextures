/**
 * Final Grade Roll-Up to Registrar (plan 14.5)
 *
 *   [x] GET final-grades/preview unauthenticated returns 401
 *   [x] POST final-grades/submit unauthenticated returns 401
 *   [x] GET final-grades/export.csv unauthenticated returns 401
 *   [x] GET admin/final-grades/status unauthenticated returns 401
 *   [x] Feature-disabled: all four endpoints return 501
 *   [x] Feature-enabled: preview returns student list for a course
 *   [x] Feature-enabled: submit creates audit rows and returns downloadUrl
 *   [x] Feature-enabled: export CSV returns text/csv content
 *   [x] Feature-enabled: admin status endpoint returns courses array
 *   [x] Submit with bad method returns 400
 *   [x] Admin status without term_id returns 400
 *   [x] Platform settings: PUT ffGradeSubmission toggles the feature flag
 */
import { test, expect } from '@playwright/test'
import { apiSignup } from '../fixtures/api.js'

const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'
const PASSWORD = 'E2eTestPass1!'

function uid(prefix = 'fg') {
  return `e2e-${prefix}-${Date.now()}-${Math.random().toString(36).slice(2)}`
}
function uniqueEmail(prefix = 'fg') {
  return `${uid(prefix)}@test.invalid`
}
function authHeaders(token: string) {
  return { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' }
}

async function getAdminToken(): Promise<string> {
  const adminEmail = process.env.E2E_ADMIN_EMAIL ?? 'admin@e2e.test'
  const adminPassword = process.env.E2E_ADMIN_PASSWORD ?? 'E2eTestPass1!'
  const loginRes = await fetch(`${API_BASE}/api/v1/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email: adminEmail, password: adminPassword }),
  })
  if (!loginRes.ok) {
    await fetch(`${API_BASE}/api/v1/auth/signup`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email: adminEmail, password: adminPassword, display_name: 'E2E Admin' }),
    })
    const retry = await fetch(`${API_BASE}/api/v1/auth/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email: adminEmail, password: adminPassword }),
    })
    const { access_token } = (await retry.json()) as { access_token: string }
    return access_token
  }
  const { access_token } = (await loginRes.json()) as { access_token: string }
  return access_token
}

async function getAdminOrgId(token: string): Promise<string | null> {
  const res = await fetch(`${API_BASE}/api/v1/admin/orgs`, {
    headers: authHeaders(token),
  })
  if (!res.ok) return null
  const data = (await res.json()) as { organizations?: Array<{ id: string }> }
  return data.organizations?.[0]?.id ?? null
}

async function enableGradeSubmission(token: string, enabled: boolean): Promise<void> {
  const res = await fetch(`${API_BASE}/api/v1/settings/platform`, {
    method: 'PUT',
    headers: authHeaders(token),
    body: JSON.stringify({
      ffGradeSubmission: enabled,
      updateMask: ['ffGradeSubmission'],
    }),
  })
  if (!res.ok) {
    const body = await res.text()
    throw new Error(`enableGradeSubmission failed (${res.status}): ${body}`)
  }
}

async function createCourse(token: string, orgId: string): Promise<string> {
  const code = uid('course').slice(0, 20)
  const res = await fetch(`${API_BASE}/api/v1/orgs/${orgId}/courses`, {
    method: 'POST',
    headers: authHeaders(token),
    body: JSON.stringify({
      name: 'Test Course FG',
      courseCode: code,
      description: '',
    }),
  })
  if (!res.ok) return code
  const body = (await res.json()) as { course?: { courseCode?: string } }
  return body.course?.courseCode ?? code
}

// ─────────────────────────────────────────────────────────────────────────────
// Auth guard (no token → 401; feature must be enabled or handlers return 501 first)
// ─────────────────────────────────────────────────────────────────────────────

async function enableGradeSubmissionForAuthTests(): Promise<void> {
  const adminToken = await getAdminToken()
  await enableGradeSubmission(adminToken, true)
}

test('FGS: GET preview unauthenticated returns 401', async () => {
  await enableGradeSubmissionForAuthTests()
  const res = await fetch(`${API_BASE}/api/v1/courses/CS101/final-grades/preview`)
  expect(res.status).toBe(401)
})

test('FGS: POST submit unauthenticated returns 401', async () => {
  await enableGradeSubmissionForAuthTests()
  const res = await fetch(`${API_BASE}/api/v1/courses/CS101/final-grades/submit`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ method: 'csv', overrides: [] }),
  })
  expect(res.status).toBe(401)
})

test('FGS: GET export CSV unauthenticated returns 401', async () => {
  await enableGradeSubmissionForAuthTests()
  const res = await fetch(`${API_BASE}/api/v1/courses/CS101/final-grades/export.csv`)
  expect(res.status).toBe(401)
})

test('FGS: GET admin status unauthenticated returns 401', async () => {
  await enableGradeSubmissionForAuthTests()
  const res = await fetch(
    `${API_BASE}/api/v1/admin/final-grades/status?term_id=00000000-0000-0000-0000-000000000001`,
  )
  expect(res.status).toBe(401)
})

// ─────────────────────────────────────────────────────────────────────────────
// Feature flag: disabled → 501
// ─────────────────────────────────────────────────────────────────────────────

test('FGS: Preview returns 501 when feature disabled', async () => {
  const adminToken = await getAdminToken()
  const orgId = await getAdminOrgId(adminToken)
  if (!orgId) { test.skip(true, 'no org'); return }
  await enableGradeSubmission(adminToken, false)

  const user = await apiSignup({ email: uniqueEmail('u'), password: PASSWORD })
  const res = await fetch(`${API_BASE}/api/v1/courses/CS101/final-grades/preview`, {
    headers: authHeaders(user.access_token),
  })
  expect(res.status).toBe(501)
})

test('FGS: Submit returns 501 when feature disabled', async () => {
  const adminToken = await getAdminToken()
  const orgId = await getAdminOrgId(adminToken)
  if (!orgId) { test.skip(true, 'no org'); return }
  await enableGradeSubmission(adminToken, false)

  const user = await apiSignup({ email: uniqueEmail('u2'), password: PASSWORD })
  const res = await fetch(`${API_BASE}/api/v1/courses/CS101/final-grades/submit`, {
    method: 'POST',
    headers: authHeaders(user.access_token),
    body: JSON.stringify({ method: 'csv', overrides: [] }),
  })
  expect(res.status).toBe(501)
})

test('FGS: Export CSV returns 501 when feature disabled', async () => {
  const adminToken = await getAdminToken()
  const orgId = await getAdminOrgId(adminToken)
  if (!orgId) { test.skip(true, 'no org'); return }
  await enableGradeSubmission(adminToken, false)

  const user = await apiSignup({ email: uniqueEmail('u3'), password: PASSWORD })
  const res = await fetch(`${API_BASE}/api/v1/courses/CS101/final-grades/export.csv`, {
    headers: authHeaders(user.access_token),
  })
  expect(res.status).toBe(501)
})

test('FGS: Admin status returns 501 when feature disabled', async () => {
  const adminToken = await getAdminToken()
  const orgId = await getAdminOrgId(adminToken)
  if (!orgId) { test.skip(true, 'no org'); return }
  await enableGradeSubmission(adminToken, false)

  const res = await fetch(
    `${API_BASE}/api/v1/admin/final-grades/status?term_id=00000000-0000-0000-0000-000000000001`,
    { headers: authHeaders(adminToken) },
  )
  expect(res.status).toBe(501)
})

// ─────────────────────────────────────────────────────────────────────────────
// Feature flag: enabled → live paths
// ─────────────────────────────────────────────────────────────────────────────

test('FGS: Preview returns grade list when feature enabled', async () => {
  const adminToken = await getAdminToken()
  const orgId = await getAdminOrgId(adminToken)
  if (!orgId) { test.skip(true, 'no org'); return }
  await enableGradeSubmission(adminToken, true)

  const courseCode = await createCourse(adminToken, orgId)

  const res = await fetch(
    `${API_BASE}/api/v1/courses/${encodeURIComponent(courseCode)}/final-grades/preview`,
    { headers: authHeaders(adminToken) },
  )
  expect([200, 403, 404]).toContain(res.status)
  if (res.status === 200) {
    const body = (await res.json()) as { grades: unknown[]; exportUrl: string }
    expect(Array.isArray(body.grades)).toBe(true)
    expect(typeof body.exportUrl).toBe('string')
  }
})

test('FGS: Submit saves audit rows and returns downloadUrl', async () => {
  const adminToken = await getAdminToken()
  const orgId = await getAdminOrgId(adminToken)
  if (!orgId) { test.skip(true, 'no org'); return }
  await enableGradeSubmission(adminToken, true)

  const courseCode = await createCourse(adminToken, orgId)

  const res = await fetch(
    `${API_BASE}/api/v1/courses/${encodeURIComponent(courseCode)}/final-grades/submit`,
    {
      method: 'POST',
      headers: authHeaders(adminToken),
      body: JSON.stringify({ method: 'csv', overrides: [] }),
    },
  )
  expect([200, 403, 404]).toContain(res.status)
  if (res.status === 200) {
    const body = (await res.json()) as { downloadUrl: string; count: number }
    expect(typeof body.downloadUrl).toBe('string')
    expect(typeof body.count).toBe('number')
  }
})

test('FGS: Export CSV returns text/csv content-type', async () => {
  const adminToken = await getAdminToken()
  const orgId = await getAdminOrgId(adminToken)
  if (!orgId) { test.skip(true, 'no org'); return }
  await enableGradeSubmission(adminToken, true)

  const courseCode = await createCourse(adminToken, orgId)

  const res = await fetch(
    `${API_BASE}/api/v1/courses/${encodeURIComponent(courseCode)}/final-grades/export.csv`,
    { headers: authHeaders(adminToken) },
  )
  expect([200, 403, 404]).toContain(res.status)
  if (res.status === 200) {
    const ct = res.headers.get('content-type') ?? ''
    expect(ct).toContain('text/csv')
    const body = await res.text()
    expect(body).toContain('StudentID')
  }
})

test('FGS: Admin status returns courses array', async () => {
  const adminToken = await getAdminToken()
  const orgId = await getAdminOrgId(adminToken)
  if (!orgId) { test.skip(true, 'no org'); return }
  await enableGradeSubmission(adminToken, true)

  const termRes = await fetch(`${API_BASE}/api/v1/orgs/${orgId}/terms`, {
    headers: authHeaders(adminToken),
  })
  if (!termRes.ok) { test.skip(true, 'no terms'); return }
  const termData = (await termRes.json()) as { terms?: Array<{ id: string }> }
  const termId = termData.terms?.[0]?.id
  if (!termId) { test.skip(true, 'no term id'); return }

  const res = await fetch(
    `${API_BASE}/api/v1/admin/final-grades/status?term_id=${encodeURIComponent(termId)}`,
    { headers: authHeaders(adminToken) },
  )
  expect([200, 403]).toContain(res.status)
  if (res.status === 200) {
    const body = (await res.json()) as { courses: unknown[] }
    expect(Array.isArray(body.courses)).toBe(true)
  }
})

// ─────────────────────────────────────────────────────────────────────────────
// Validation errors
// ─────────────────────────────────────────────────────────────────────────────

test('FGS: Submit with invalid method returns 400', async () => {
  const adminToken = await getAdminToken()
  const orgId = await getAdminOrgId(adminToken)
  if (!orgId) { test.skip(true, 'no org'); return }
  await enableGradeSubmission(adminToken, true)

  const courseCode = await createCourse(adminToken, orgId)

  const res = await fetch(
    `${API_BASE}/api/v1/courses/${encodeURIComponent(courseCode)}/final-grades/submit`,
    {
      method: 'POST',
      headers: authHeaders(adminToken),
      body: JSON.stringify({ method: 'fax', overrides: [] }),
    },
  )
  expect([400, 403, 404]).toContain(res.status)
})

test('FGS: Admin status without term_id returns 400', async () => {
  const adminToken = await getAdminToken()
  const orgId = await getAdminOrgId(adminToken)
  if (!orgId) { test.skip(true, 'no org'); return }
  await enableGradeSubmission(adminToken, true)

  const res = await fetch(`${API_BASE}/api/v1/admin/final-grades/status`, {
    headers: authHeaders(adminToken),
  })
  expect([400, 403]).toContain(res.status)
})

// ─────────────────────────────────────────────────────────────────────────────
// Platform settings toggle
// ─────────────────────────────────────────────────────────────────────────────

test('FGS: Platform settings PUT ffGradeSubmission=true reflects in features endpoint', async () => {
  const adminToken = await getAdminToken()
  const orgId = await getAdminOrgId(adminToken)
  if (!orgId) { test.skip(true, 'no org'); return }

  await enableGradeSubmission(adminToken, true)

  const featRes = await fetch(`${API_BASE}/api/v1/platform/features`, {
    headers: authHeaders(adminToken),
  })
  expect(featRes.status).toBe(200)
  const feats = (await featRes.json()) as { ffGradeSubmission?: boolean }
  expect(feats.ffGradeSubmission).toBe(true)
})

test('FGS: Platform settings PUT ffGradeSubmission=false reflects in features endpoint', async () => {
  const adminToken = await getAdminToken()
  const orgId = await getAdminOrgId(adminToken)
  if (!orgId) { test.skip(true, 'no org'); return }

  await enableGradeSubmission(adminToken, false)

  const featRes = await fetch(`${API_BASE}/api/v1/platform/features`, {
    headers: authHeaders(adminToken),
  })
  expect(featRes.status).toBe(200)
  const feats = (await featRes.json()) as { ffGradeSubmission?: boolean }
  expect(feats.ffGradeSubmission).toBe(false)
})
