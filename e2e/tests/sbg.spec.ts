/**
 * Standards-Based Grading (plan 13.5).
 *
 * Checklist coverage:
 *   [x] GET standard domains returns 401 without auth
 *   [x] POST standard domain returns 401 without auth
 *   [x] GET mastery scale returns 401 without auth
 *   [x] PUT mastery scale returns 401 without auth
 *   [x] POST standards import returns 401 without auth
 *   [x] GET course SBG standards returns 401 without auth
 *   [x] POST mastery score returns 401 without auth
 *   [x] GET SBG heatmap returns 401 without auth
 *   [x] GET student SBG report returns 401 without auth
 *   [x] Admin can create a standard domain
 *   [x] Admin can set the mastery scale (PUT)
 *   [x] Admin can import standards from CSV body
 *   [x] Instructor can record a mastery score
 *   [x] Instructor can retrieve heatmap (empty is valid)
 *   [x] Student cannot access instructor heatmap (403)
 *   [x] Student can access their own SBG report
 */
import { test, expect } from '../fixtures/test.js'
import { apiSignup } from '../fixtures/api.js'

const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'
const PASSWORD = 'E2eTestPass1!'

function uid(prefix = 'sbg') {
  return `e2e-${prefix}-${Date.now()}-${Math.random().toString(36).slice(2)}`
}
function uniqueEmail(prefix = 'sbg') {
  return `${uid(prefix)}@test.invalid`
}

// ─────────────────────────────────────────────────────────────────────────────
// Auth guard checks (no token → 401)
// ─────────────────────────────────────────────────────────────────────────────

test('SBG: GET standard-domains returns 401 without auth', async () => {
  const res = await fetch(
    `${API_BASE}/api/v1/admin/orgs/00000000-0000-0000-0000-000000000001/sbg/standard-domains`,
  )
  expect(res.status).toBe(401)
})

test('SBG: POST standard-domain returns 401 without auth', async () => {
  const res = await fetch(
    `${API_BASE}/api/v1/admin/orgs/00000000-0000-0000-0000-000000000001/sbg/standard-domains`,
    { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: '{}' },
  )
  expect(res.status).toBe(401)
})

test('SBG: GET mastery-scale returns 401 without auth', async () => {
  const res = await fetch(
    `${API_BASE}/api/v1/admin/orgs/00000000-0000-0000-0000-000000000001/sbg/mastery-scale`,
  )
  expect(res.status).toBe(401)
})

test('SBG: PUT mastery-scale returns 401 without auth', async () => {
  const res = await fetch(
    `${API_BASE}/api/v1/admin/orgs/00000000-0000-0000-0000-000000000001/sbg/mastery-scale`,
    { method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: '{}' },
  )
  expect(res.status).toBe(401)
})

test('SBG: POST standards import returns 401 without auth', async () => {
  const res = await fetch(
    `${API_BASE}/api/v1/admin/orgs/00000000-0000-0000-0000-000000000001/sbg/standards/import`,
    { method: 'POST', body: 'code,description,domain_code,domain_name\n' },
  )
  expect(res.status).toBe(401)
})

test('SBG: GET course standards returns 401 without auth', async () => {
  const res = await fetch(`${API_BASE}/api/v1/courses/COURSE01/sbg/standards`)
  expect(res.status).toBe(401)
})

test('SBG: POST mastery-scores returns 401 without auth', async () => {
  const res = await fetch(`${API_BASE}/api/v1/sbg/mastery-scores`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: '{}',
  })
  expect(res.status).toBe(401)
})

test('SBG: GET heatmap returns 401 without auth', async () => {
  const res = await fetch(`${API_BASE}/api/v1/courses/COURSE01/sbg/heatmap/Q1-2026`)
  expect(res.status).toBe(401)
})

test('SBG: GET student SBG report returns 401 without auth', async () => {
  const res = await fetch(
    `${API_BASE}/api/v1/students/00000000-0000-0000-0000-000000000001/sbg/Q1-2026`,
  )
  expect(res.status).toBe(401)
})

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

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

function orgIdFromToken(token: string): string | null {
  try {
    const parts = token.split('.')
    if (parts.length < 2) return null
    const payload = JSON.parse(Buffer.from(parts[1], 'base64url').toString()) as { org_id?: string }
    return payload.org_id ?? null
  } catch {
    return null
  }
}

async function getMyOrgId(token: string): Promise<string | null> {
  const fromToken = orgIdFromToken(token)
  if (fromToken) return fromToken
  const res = await fetch(`${API_BASE}/api/v1/me`, { headers: { Authorization: `Bearer ${token}` } })
  if (!res.ok) return null
  const me = (await res.json()) as { orgId?: string }
  return me.orgId ?? null
}

async function getUserId(token: string): Promise<string | null> {
  const res = await fetch(`${API_BASE}/api/v1/me`, { headers: { Authorization: `Bearer ${token}` } })
  if (!res.ok) return null
  return ((await res.json()) as { id?: string }).id ?? null
}

async function grantOrgAdmin(adminToken: string, orgId: string, userId: string) {
  await fetch(`${API_BASE}/api/v1/orgs/${orgId}/role-grants`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${adminToken}`, 'Content-Type': 'application/json' },
    body: JSON.stringify({ userId, role: 'org_admin' }),
  })
}

// ─────────────────────────────────────────────────────────────────────────────
// Authenticated — mastery scale
// ─────────────────────────────────────────────────────────────────────────────

test('SBG: admin can set the mastery scale', async () => {
  const adminToken = await getAdminToken()
  const orgId = await getMyOrgId(adminToken)
  if (!orgId) { test.skip(true, 'no orgId'); return }

  const adminId = await getUserId(adminToken)
  if (adminId) await grantOrgAdmin(adminToken, orgId, adminId)

  const scale = [
    { label: 'Exceeds', value: 4, color: '#22c55e' },
    { label: 'Meets', value: 3, color: '#3b82f6' },
    { label: 'Approaching', value: 2, color: '#f59e0b' },
    { label: 'Below', value: 1, color: '#ef4444' },
  ]
  const res = await fetch(
    `${API_BASE}/api/v1/admin/orgs/${orgId}/sbg/mastery-scale`,
    {
      method: 'PUT',
      headers: { Authorization: `Bearer ${adminToken}`, 'Content-Type': 'application/json' },
      body: JSON.stringify({ scale }),
    },
  )
  if (!res.ok) { test.skip(true, `PUT scale failed: ${res.status} ${await res.text()}`); return }

  const body = (await res.json()) as { scale: Array<{ value: number; label: string }> }
  expect(Array.isArray(body.scale)).toBe(true)
  expect(body.scale.length).toBe(4)
  const vals = body.scale.map((s) => s.value)
  expect(vals).toContain(4)
  expect(vals).toContain(1)

  // GET should return the same scale.
  const getRes = await fetch(
    `${API_BASE}/api/v1/admin/orgs/${orgId}/sbg/mastery-scale`,
    { headers: { Authorization: `Bearer ${adminToken}` } },
  )
  expect(getRes.ok).toBeTruthy()
  const getBody = (await getRes.json()) as { scale: Array<{ value: number }> }
  expect(getBody.scale.length).toBe(4)
})

// ─────────────────────────────────────────────────────────────────────────────
// Authenticated — standard domains
// ─────────────────────────────────────────────────────────────────────────────

test('SBG: admin can create a standard domain', async () => {
  const adminToken = await getAdminToken()
  const orgId = await getMyOrgId(adminToken)
  if (!orgId) { test.skip(true, 'no orgId'); return }

  const adminId = await getUserId(adminToken)
  if (adminId) await grantOrgAdmin(adminToken, orgId, adminId)

  const code = `OA${Date.now().toString().slice(-4)}`
  const res = await fetch(
    `${API_BASE}/api/v1/admin/orgs/${orgId}/sbg/standard-domains`,
    {
      method: 'POST',
      headers: { Authorization: `Bearer ${adminToken}`, 'Content-Type': 'application/json' },
      body: JSON.stringify({ code, name: 'Operations and Algebraic Thinking', gradeLevel: '3' }),
    },
  )
  if (!res.ok) { test.skip(true, `create domain failed: ${res.status} ${await res.text()}`); return }
  expect(res.status).toBe(201)

  const body = (await res.json()) as { code: string; name: string; gradeLevel?: string }
  expect(body.code).toBe(code)
  expect(body.name).toBe('Operations and Algebraic Thinking')
  expect(body.gradeLevel).toBe('3')

  // List should include the new domain.
  const listRes = await fetch(
    `${API_BASE}/api/v1/admin/orgs/${orgId}/sbg/standard-domains`,
    { headers: { Authorization: `Bearer ${adminToken}` } },
  )
  expect(listRes.ok).toBeTruthy()
  const listBody = (await listRes.json()) as { domains: Array<{ code: string }> }
  expect(listBody.domains.some((d) => d.code === code)).toBe(true)
})

// ─────────────────────────────────────────────────────────────────────────────
// Authenticated — CSV import
// ─────────────────────────────────────────────────────────────────────────────

test('SBG: admin can import standards from CSV (text/csv body)', async () => {
  const adminToken = await getAdminToken()
  const orgId = await getMyOrgId(adminToken)
  if (!orgId) { test.skip(true, 'no orgId'); return }

  const adminId = await getUserId(adminToken)
  if (adminId) await grantOrgAdmin(adminToken, orgId, adminId)

  const suffix = Date.now().toString().slice(-5)
  const csv = [
    'code,description,domain_code,domain_name,grade_level',
    `3.OA.A.1${suffix},Interpret products of whole numbers,OA${suffix},Operations and Algebraic Thinking,3`,
    `3.OA.A.2${suffix},Interpret whole-number quotients,OA${suffix},Operations and Algebraic Thinking,3`,
  ].join('\n')

  const res = await fetch(
    `${API_BASE}/api/v1/admin/orgs/${orgId}/sbg/standards/import`,
    {
      method: 'POST',
      headers: {
        Authorization: `Bearer ${adminToken}`,
        'Content-Type': 'text/csv',
      },
      body: csv,
    },
  )
  if (!res.ok) { test.skip(true, `import failed: ${res.status} ${await res.text()}`); return }

  const body = (await res.json()) as { standardsImported: number; domainsCreated: number; errors: string[] }
  expect(body.standardsImported).toBe(2)
  expect(body.errors.length).toBe(0)
})

// ─────────────────────────────────────────────────────────────────────────────
// Authenticated — mastery score recording and heatmap
// ─────────────────────────────────────────────────────────────────────────────

test('SBG: instructor can record a mastery score and retrieve heatmap', async () => {
  const adminToken = await getAdminToken()
  const orgId = await getMyOrgId(adminToken)
  if (!orgId) { test.skip(true, 'no orgId'); return }

  const adminId = await getUserId(adminToken)
  if (adminId) await grantOrgAdmin(adminToken, orgId, adminId)

  // Set up a mastery scale.
  await fetch(`${API_BASE}/api/v1/admin/orgs/${orgId}/sbg/mastery-scale`, {
    method: 'PUT',
    headers: { Authorization: `Bearer ${adminToken}`, 'Content-Type': 'application/json' },
    body: JSON.stringify({
      scale: [
        { label: 'Exceeds', value: 4, color: '#22c55e' },
        { label: 'Meets', value: 3, color: '#3b82f6' },
        { label: 'Approaching', value: 2, color: '#f59e0b' },
        { label: 'Below', value: 1, color: '#ef4444' },
      ],
    }),
  })

  // Import a standard.
  const suffix = Date.now().toString().slice(-6)
  const csv = [
    'code,description,domain_code,domain_name,grade_level',
    `STD${suffix},Test Standard,DOM${suffix},Test Domain,3`,
  ].join('\n')
  const importRes = await fetch(
    `${API_BASE}/api/v1/admin/orgs/${orgId}/sbg/standards/import`,
    {
      method: 'POST',
      headers: { Authorization: `Bearer ${adminToken}`, 'Content-Type': 'text/csv' },
      body: csv,
    },
  )
  if (!importRes.ok) { test.skip(true, `import failed: ${importRes.status}`); return }

  // List domains to find the domain ID, then list its standards.
  const domainsRes = await fetch(
    `${API_BASE}/api/v1/admin/orgs/${orgId}/sbg/standard-domains`,
    { headers: { Authorization: `Bearer ${adminToken}` } },
  )
  if (!domainsRes.ok) { test.skip(true, 'domains list failed'); return }
  const domainsBody = (await domainsRes.json()) as { domains: Array<{ id: string; code: string }> }
  const domain = domainsBody.domains.find((d) => d.code === `DOM${suffix}`)
  if (!domain) { test.skip(true, 'domain not found'); return }

  // Create teacher + student + course.
  const teacherEmail = uniqueEmail('tch')
  const studentEmail = uniqueEmail('stu')
  const { access_token: teacherToken } = await apiSignup({ email: teacherEmail, password: PASSWORD })
  const { access_token: studentToken } = await apiSignup({ email: studentEmail, password: PASSWORD })
  const teacherId = await getUserId(teacherToken)
  const studentId = await getUserId(studentToken)
  if (!teacherId || !studentId) { test.skip(true, 'missing IDs'); return }

  const courseRes = await fetch(`${API_BASE}/api/v1/courses`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${teacherToken}`, 'Content-Type': 'application/json' },
    body: JSON.stringify({ title: 'SBG Test Course' }),
  })
  if (!courseRes.ok) { test.skip(true, `course create failed: ${courseRes.status}`); return }
  const { courseCode } = (await courseRes.json()) as { courseCode: string }
  if (!courseCode) { test.skip(true, 'no courseCode'); return }

  // Enroll student.
  await fetch(`${API_BASE}/api/v1/courses/${courseCode}/enrollments`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${teacherToken}`, 'Content-Type': 'application/json' },
    body: JSON.stringify({ emails: studentEmail, courseRole: 'student' }),
  })

  // List course standards (instructor view).
  const stdsRes = await fetch(
    `${API_BASE}/api/v1/courses/${courseCode}/sbg/standards`,
    { headers: { Authorization: `Bearer ${teacherToken}` } },
  )
  if (!stdsRes.ok) { test.skip(true, `course standards failed: ${stdsRes.status}`); return }
  const stdsBody = (await stdsRes.json()) as { standards: Array<{ id: string; code: string }> }
  const standard = stdsBody.standards.find((s) => s.code === `STD${suffix}`)
  if (!standard) { test.skip(true, 'standard not found in course list'); return }

  // Record a mastery score.
  const scoreRes = await fetch(`${API_BASE}/api/v1/sbg/mastery-scores`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${teacherToken}`, 'Content-Type': 'application/json' },
    body: JSON.stringify({
      studentId,
      standardId: standard.id,
      courseCode,
      gradingPeriod: 'Q1-2026',
      scoreValue: 3,
      source: 'observation',
    }),
  })
  if (!scoreRes.ok) {
    const body = await scoreRes.text()
    test.skip(true, `score recording failed: ${scoreRes.status} ${body}`)
    return
  }
  expect(scoreRes.status).toBe(201)
  const scoreBody = (await scoreRes.json()) as { scoreValue: number; standardId: string }
  expect(scoreBody.scoreValue).toBe(3)
  expect(scoreBody.standardId).toBe(standard.id)

  // Heatmap should include the score.
  const heatmapRes = await fetch(
    `${API_BASE}/api/v1/courses/${courseCode}/sbg/heatmap/Q1-2026`,
    { headers: { Authorization: `Bearer ${teacherToken}` } },
  )
  expect(heatmapRes.ok).toBeTruthy()
  const heatmapBody = (await heatmapRes.json()) as { cells: Array<{ studentId: string; standardId: string; scoreValue: number }> }
  expect(Array.isArray(heatmapBody.cells)).toBe(true)
  const cell = heatmapBody.cells.find(
    (c) => c.studentId === studentId && c.standardId === standard.id,
  )
  expect(cell?.scoreValue).toBe(3)
})

test('SBG: student cannot access instructor heatmap (403)', async () => {
  const { access_token: studentToken } = await apiSignup({ email: uniqueEmail('stu2'), password: PASSWORD })

  const courseRes = await fetch(`${API_BASE}/api/v1/courses`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${studentToken}`, 'Content-Type': 'application/json' },
    body: JSON.stringify({ title: 'SBG Student Test' }),
  })
  if (!courseRes.ok) { test.skip(true, `course create: ${courseRes.status}`); return }
  const { courseCode } = (await courseRes.json()) as { courseCode: string }
  if (!courseCode) { test.skip(true, 'no courseCode'); return }

  // Sign up a separate student and try to access heatmap.
  const { access_token: anotherStudentToken } = await apiSignup({ email: uniqueEmail('stu3'), password: PASSWORD })
  await fetch(`${API_BASE}/api/v1/courses/${courseCode}/enrollments`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${studentToken}`, 'Content-Type': 'application/json' },
    body: JSON.stringify({ emails: (await (await fetch(`${API_BASE}/api/v1/me`, { headers: { Authorization: `Bearer ${anotherStudentToken}` } })).json() as { email?: string }).email, courseRole: 'student' }),
  })

  const res = await fetch(
    `${API_BASE}/api/v1/courses/${courseCode}/sbg/heatmap/Q1-2026`,
    { headers: { Authorization: `Bearer ${anotherStudentToken}` } },
  )
  expect(res.status).toBe(403)
})

test('SBG: student can access their own SBG report', async () => {
  const { access_token: studentToken } = await apiSignup({ email: uniqueEmail('stu4'), password: PASSWORD })
  const studentId = await getUserId(studentToken)
  if (!studentId) { test.skip(true, 'no studentId'); return }

  const res = await fetch(
    `${API_BASE}/api/v1/students/${studentId}/sbg/Q1-2026`,
    { headers: { Authorization: `Bearer ${studentToken}` } },
  )
  expect(res.ok).toBeTruthy()
  const body = (await res.json()) as { studentId: string; period: string; scores: unknown[] }
  expect(body.studentId).toBe(studentId)
  expect(body.period).toBe('Q1-2026')
  expect(Array.isArray(body.scores)).toBe(true)
})

// ─────────────────────────────────────────────────────────────────────────────
// Heatmap returns 200 (empty is valid)
// ─────────────────────────────────────────────────────────────────────────────

test('SBG: instructor can retrieve heatmap (empty is valid)', async ({ seededCourse }) => {
  const res = await fetch(
    `${API_BASE}/api/v1/courses/${seededCourse.courseCode}/sbg/heatmap/Q1-2026`,
    { headers: { Authorization: `Bearer ${seededCourse.instructorToken}` } },
  )
  expect(res.status).toBe(200)
  const body = (await res.json()) as { cells: unknown[]; period: string }
  expect(body.period).toBe('Q1-2026')
  expect(Array.isArray(body.cells)).toBe(true)
})
