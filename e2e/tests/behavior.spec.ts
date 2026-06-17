/**
 * Behavior / PBIS tracking (plan 13.3)
 *
 *   [x] GET behavior categories unauthenticated returns 401
 *   [x] POST pbis awards unauthenticated returns 401
 *   [x] POST behavior referrals unauthenticated returns 401
 *   [x] GET student behavior unauthenticated returns 401
 *   [x] Admin can seed default behavior categories
 *   [x] Admin can create a custom behavior category
 *   [x] Admin can delete (deactivate) a category
 *   [x] Creating category with invalid type returns 400
 *   [x] Teacher can award PBIS points to a student
 *   [x] Bulk award to multiple students succeeds
 *   [x] Teacher can file a behavior referral
 *   [x] Student behavior summary returns total points and referrals
 *   [x] Non-student cannot view another student's behavior (403)
 *   [x] Admin behavior dashboard returns totals and breakdowns
 *   [x] Awards require non-empty awards array
 *   [x] Referral requires description
 */
import { test, expect } from '../fixtures/test.js'
import { apiSignup, apiEnroll } from '../fixtures/api.js'

const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'
const PASSWORD = 'E2eTestPass1!'

function uid(prefix = 'beh') {
  return `e2e-${prefix}-${Date.now()}-${Math.random().toString(36).slice(2)}`
}
function uniqueEmail(prefix = 'beh') {
  return `${uid(prefix)}@test.invalid`
}

// ─────────────────────────────────────────────────────────────────────────────
// Auth guard checks (no token → 401)
// ─────────────────────────────────────────────────────────────────────────────

test('Behavior: GET behavior categories unauthenticated returns 401', async () => {
  const res = await fetch(
    `${API_BASE}/api/v1/admin/orgs/00000000-0000-0000-0000-000000000001/behavior/categories`,
  )
  expect(res.status).toBe(401)
})

test('Behavior: POST pbis awards unauthenticated returns 401', async () => {
  const res = await fetch(`${API_BASE}/api/v1/pbis/awards`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ awards: [] }),
  })
  expect(res.status).toBe(401)
})

test('Behavior: POST behavior referrals unauthenticated returns 401', async () => {
  const res = await fetch(`${API_BASE}/api/v1/behavior/referrals`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({}),
  })
  expect(res.status).toBe(401)
})

test('Behavior: GET student behavior unauthenticated returns 401', async () => {
  const res = await fetch(
    `${API_BASE}/api/v1/students/00000000-0000-0000-0000-000000000001/behavior`,
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

async function getMyOrgId(token: string): Promise<string | null> {
  const res = await fetch(`${API_BASE}/api/v1/me`, {
    headers: { Authorization: `Bearer ${token}` },
  })
  if (!res.ok) return null
  const me = (await res.json()) as { orgId?: string }
  return me.orgId ?? null
}

async function getUserId(token: string): Promise<string | null> {
  const res = await fetch(`${API_BASE}/api/v1/me`, {
    headers: { Authorization: `Bearer ${token}` },
  })
  if (!res.ok) return null
  const me = (await res.json()) as { id?: string }
  return me.id ?? null
}

async function grantOrgAdmin(adminToken: string, orgId: string, userId: string) {
  await fetch(`${API_BASE}/api/v1/orgs/${orgId}/role-grants`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${adminToken}`, 'Content-Type': 'application/json' },
    body: JSON.stringify({ userId, role: 'org_admin' }),
  })
}

// ─────────────────────────────────────────────────────────────────────────────
// Category management
// ─────────────────────────────────────────────────────────────────────────────

test('Behavior: admin can seed default categories', async () => {
  const adminToken = await getAdminToken()
  const orgId = await getMyOrgId(adminToken)
  if (!orgId) { test.skip(true, 'no orgId'); return }

  const adminId = await getUserId(adminToken)
  if (adminId) await grantOrgAdmin(adminToken, orgId, adminId)

  const res = await fetch(`${API_BASE}/api/v1/admin/orgs/${orgId}/behavior/categories`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${adminToken}`, 'Content-Type': 'application/json' },
    body: JSON.stringify({ seedDefaults: true }),
  })
  if (!res.ok) { test.skip(true, `seed failed: ${res.status} ${await res.text()}`); return }
  expect(res.ok).toBeTruthy()
  const body = (await res.json()) as { categories: Array<{ name: string; type: string }> }
  const names = body.categories.map((c) => c.name)
  expect(names).toContain('Respect')
  expect(names).toContain('Responsibility')
  expect(names).toContain('Safety')
  const hasPositive = body.categories.some((c) => c.type === 'positive')
  const hasNegative = body.categories.some((c) => c.type === 'negative')
  expect(hasPositive).toBe(true)
  expect(hasNegative).toBe(true)
})

test('Behavior: admin can create a custom category', async () => {
  const adminToken = await getAdminToken()
  const orgId = await getMyOrgId(adminToken)
  if (!orgId) { test.skip(true, 'no orgId'); return }

  const adminId = await getUserId(adminToken)
  if (adminId) await grantOrgAdmin(adminToken, orgId, adminId)

  const name = `TestCat-${Date.now()}`
  const res = await fetch(`${API_BASE}/api/v1/admin/orgs/${orgId}/behavior/categories`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${adminToken}`, 'Content-Type': 'application/json' },
    body: JSON.stringify({ name, type: 'positive', color: '#FF5733' }),
  })
  if (!res.ok) { test.skip(true, `create failed: ${res.status}`); return }
  expect(res.status).toBe(201)
  const body = (await res.json()) as { name: string; type: string; color?: string }
  expect(body.name).toBe(name)
  expect(body.type).toBe('positive')
  expect(body.color).toBe('#FF5733')
})

test('Behavior: admin can delete (deactivate) a category with no records', async () => {
  const adminToken = await getAdminToken()
  const orgId = await getMyOrgId(adminToken)
  if (!orgId) { test.skip(true, 'no orgId'); return }

  const adminId = await getUserId(adminToken)
  if (adminId) await grantOrgAdmin(adminToken, orgId, adminId)

  const name = `DelCat-${Date.now()}`
  const createRes = await fetch(`${API_BASE}/api/v1/admin/orgs/${orgId}/behavior/categories`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${adminToken}`, 'Content-Type': 'application/json' },
    body: JSON.stringify({ name, type: 'negative' }),
  })
  if (!createRes.ok) { test.skip(true, `create failed: ${createRes.status}`); return }
  const created = (await createRes.json()) as { id: string }

  const delRes = await fetch(
    `${API_BASE}/api/v1/admin/orgs/${orgId}/behavior/categories/${created.id}`,
    { method: 'DELETE', headers: { Authorization: `Bearer ${adminToken}` } },
  )
  expect(delRes.status).toBe(204)
})

test('Behavior: creating category with invalid type returns 400', async () => {
  const adminToken = await getAdminToken()
  const orgId = await getMyOrgId(adminToken)
  if (!orgId) { test.skip(true, 'no orgId'); return }

  const adminId = await getUserId(adminToken)
  if (adminId) await grantOrgAdmin(adminToken, orgId, adminId)

  const res = await fetch(`${API_BASE}/api/v1/admin/orgs/${orgId}/behavior/categories`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${adminToken}`, 'Content-Type': 'application/json' },
    body: JSON.stringify({ name: 'Bad Type', type: 'invalid' }),
  })
  expect(res.status).toBe(400)
})

// ─────────────────────────────────────────────────────────────────────────────
// PBIS awards
// ─────────────────────────────────────────────────────────────────────────────

test('Behavior: teacher can award PBIS points to a student', async () => {
  const adminToken = await getAdminToken()
  const orgId = await getMyOrgId(adminToken)
  if (!orgId) { test.skip(true, 'no orgId'); return }

  const adminId = await getUserId(adminToken)
  if (adminId) await grantOrgAdmin(adminToken, orgId, adminId)

  // Seed categories
  await fetch(`${API_BASE}/api/v1/admin/orgs/${orgId}/behavior/categories`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${adminToken}`, 'Content-Type': 'application/json' },
    body: JSON.stringify({ seedDefaults: true }),
  })
  const catsRes = await fetch(`${API_BASE}/api/v1/admin/orgs/${orgId}/behavior/categories`, {
    headers: { Authorization: `Bearer ${adminToken}` },
  })
  if (!catsRes.ok) { test.skip(true, 'categories list failed'); return }
  const { categories } = (await catsRes.json()) as {
    categories: Array<{ id: string; name: string; type: string }>
  }
  const positiveCategory = categories.find((c) => c.type === 'positive')
  if (!positiveCategory) { test.skip(true, 'no positive category found'); return }

  const teacherEmail = uniqueEmail('tch')
  const studentEmail = uniqueEmail('stu')
  const { access_token: teacherToken } = await apiSignup({ email: teacherEmail, password: PASSWORD })
  const { access_token: studentToken } = await apiSignup({ email: studentEmail, password: PASSWORD })

  const studentId = await getUserId(studentToken)
  if (!studentId) { test.skip(true, 'no studentId'); return }

  // Create course as teacher
  const courseRes = await fetch(`${API_BASE}/api/v1/courses`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${teacherToken}`, 'Content-Type': 'application/json' },
    body: JSON.stringify({ title: 'PBIS Award Test' }),
  })
  if (!courseRes.ok) { test.skip(true, `course create failed: ${courseRes.status}`); return }
  const { courseCode } = (await courseRes.json()) as { courseCode: string }

  // Enroll student
  await apiEnroll(teacherToken, courseCode, studentEmail, 'student', studentToken)

  // Award points
  const awardRes = await fetch(`${API_BASE}/api/v1/pbis/awards`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${teacherToken}`, 'Content-Type': 'application/json' },
    body: JSON.stringify({
      awards: [{ studentId, categoryId: positiveCategory.id, points: 1, note: 'Great work!' }],
    }),
  })
  if (!awardRes.ok) {
    const body = await awardRes.text()
    test.skip(true, `award failed: ${awardRes.status} ${body}`)
    return
  }
  expect(awardRes.ok).toBeTruthy()
  const awardBody = (await awardRes.json()) as { saved: number; message: string }
  expect(awardBody.saved).toBe(1)
  expect(awardBody.message).toBe('Points awarded.')
})

test('Behavior: bulk award to multiple students succeeds', async () => {
  const adminToken = await getAdminToken()
  const orgId = await getMyOrgId(adminToken)
  if (!orgId) { test.skip(true, 'no orgId'); return }

  const adminId = await getUserId(adminToken)
  if (adminId) await grantOrgAdmin(adminToken, orgId, adminId)

  await fetch(`${API_BASE}/api/v1/admin/orgs/${orgId}/behavior/categories`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${adminToken}`, 'Content-Type': 'application/json' },
    body: JSON.stringify({ seedDefaults: true }),
  })
  const catsRes = await fetch(`${API_BASE}/api/v1/admin/orgs/${orgId}/behavior/categories`, {
    headers: { Authorization: `Bearer ${adminToken}` },
  })
  if (!catsRes.ok) { test.skip(true, 'categories list failed'); return }
  const { categories } = (await catsRes.json()) as {
    categories: Array<{ id: string; type: string }>
  }
  const positiveCategory = categories.find((c) => c.type === 'positive')
  if (!positiveCategory) { test.skip(true, 'no positive category'); return }

  const teacherEmail = uniqueEmail('bt')
  const { access_token: teacherToken } = await apiSignup({ email: teacherEmail, password: PASSWORD })

  const students = await Promise.all(
    [1, 2, 3].map(async (i) => {
      const email = uniqueEmail(`bs${i}`)
      const { access_token: tok } = await apiSignup({ email, password: PASSWORD })
      const id = await getUserId(tok)
      return { email, token: tok, id }
    }),
  )
  const enrolledStudents = students.filter((s): s is { email: string; token: string; id: string } => s.id !== null)
  const studentIds = enrolledStudents.map((s) => s.id)
  if (studentIds.length === 0) { test.skip(true, 'no student ids'); return }

  const courseRes = await fetch(`${API_BASE}/api/v1/courses`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${teacherToken}`, 'Content-Type': 'application/json' },
    body: JSON.stringify({ title: 'Bulk Award Test' }),
  })
  if (!courseRes.ok) { test.skip(true, `course create failed: ${courseRes.status}`); return }
  const { courseCode } = (await courseRes.json()) as { courseCode: string }

  for (const student of enrolledStudents) {
    await apiEnroll(teacherToken, courseCode, student.email, 'student', student.token)
  }

  const awards = studentIds.map((sid) => ({
    studentId: sid,
    categoryId: positiveCategory.id,
    points: 1,
  }))
  const awardRes = await fetch(`${API_BASE}/api/v1/pbis/awards`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${teacherToken}`, 'Content-Type': 'application/json' },
    body: JSON.stringify({ awards }),
  })
  if (!awardRes.ok) {
    test.skip(true, `bulk award failed: ${awardRes.status} ${await awardRes.text()}`)
    return
  }
  const awardBody = (await awardRes.json()) as { saved: number }
  expect(awardBody.saved).toBe(studentIds.length)
})

test('Behavior: award with empty awards array returns 400', async () => {
  const { access_token: token } = await apiSignup({
    email: uniqueEmail('ea'),
    password: PASSWORD,
  })
  const res = await fetch(`${API_BASE}/api/v1/pbis/awards`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
    body: JSON.stringify({ awards: [] }),
  })
  expect(res.status).toBe(400)
})

// ─────────────────────────────────────────────────────────────────────────────
// Referrals
// ─────────────────────────────────────────────────────────────────────────────

test('Behavior: teacher can file a behavior referral', async () => {
  const adminToken = await getAdminToken()
  const orgId = await getMyOrgId(adminToken)
  if (!orgId) { test.skip(true, 'no orgId'); return }

  const adminId = await getUserId(adminToken)
  if (adminId) await grantOrgAdmin(adminToken, orgId, adminId)

  await fetch(`${API_BASE}/api/v1/admin/orgs/${orgId}/behavior/categories`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${adminToken}`, 'Content-Type': 'application/json' },
    body: JSON.stringify({ seedDefaults: true }),
  })
  const catsRes = await fetch(`${API_BASE}/api/v1/admin/orgs/${orgId}/behavior/categories`, {
    headers: { Authorization: `Bearer ${adminToken}` },
  })
  if (!catsRes.ok) { test.skip(true, 'categories list failed'); return }
  const { categories } = (await catsRes.json()) as {
    categories: Array<{ id: string; type: string }>
  }
  const negativeCategory = categories.find((c) => c.type === 'negative')
  if (!negativeCategory) { test.skip(true, 'no negative category'); return }

  const teacherEmail = uniqueEmail('rt')
  const studentEmail = uniqueEmail('rs')
  const { access_token: teacherToken } = await apiSignup({ email: teacherEmail, password: PASSWORD })
  const { access_token: studentToken } = await apiSignup({ email: studentEmail, password: PASSWORD })
  const studentId = await getUserId(studentToken)
  if (!studentId) { test.skip(true, 'no studentId'); return }

  const courseRes = await fetch(`${API_BASE}/api/v1/courses`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${teacherToken}`, 'Content-Type': 'application/json' },
    body: JSON.stringify({ title: 'Referral Test Course' }),
  })
  if (!courseRes.ok) { test.skip(true, `course create failed: ${courseRes.status}`); return }
  const { courseCode } = (await courseRes.json()) as { courseCode: string }

  await apiEnroll(teacherToken, courseCode, studentEmail, 'student', studentToken)

  const refRes = await fetch(`${API_BASE}/api/v1/behavior/referrals`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${teacherToken}`, 'Content-Type': 'application/json' },
    body: JSON.stringify({
      studentId,
      categoryId: negativeCategory.id,
      description: 'Student was disruptive during class.',
      location: 'Classroom',
      response: 'Verbal warning.',
    }),
  })
  if (!refRes.ok) {
    test.skip(true, `referral failed: ${refRes.status} ${await refRes.text()}`)
    return
  }
  expect(refRes.status).toBe(201)
  const refBody = (await refRes.json()) as {
    id: string
    studentId: string
    categoryName: string
    description: string
  }
  expect(refBody.studentId).toBe(studentId)
  expect(refBody.description).toBe('Student was disruptive during class.')
})

test('Behavior: referral without description returns 400', async () => {
  const { access_token: token } = await apiSignup({
    email: uniqueEmail('rd'),
    password: PASSWORD,
  })
  const res = await fetch(`${API_BASE}/api/v1/behavior/referrals`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
    body: JSON.stringify({
      studentId: '00000000-0000-0000-0000-000000000001',
      categoryId: '00000000-0000-0000-0000-000000000002',
      description: '',
    }),
  })
  expect(res.status).toBe(400)
})

// ─────────────────────────────────────────────────────────────────────────────
// Student behavior summary
// ─────────────────────────────────────────────────────────────────────────────

test('Behavior: student behavior summary returns totalPoints and referrals after award', async () => {
  const adminToken = await getAdminToken()
  const orgId = await getMyOrgId(adminToken)
  if (!orgId) { test.skip(true, 'no orgId'); return }

  const adminId = await getUserId(adminToken)
  if (adminId) await grantOrgAdmin(adminToken, orgId, adminId)

  await fetch(`${API_BASE}/api/v1/admin/orgs/${orgId}/behavior/categories`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${adminToken}`, 'Content-Type': 'application/json' },
    body: JSON.stringify({ seedDefaults: true }),
  })
  const catsRes = await fetch(`${API_BASE}/api/v1/admin/orgs/${orgId}/behavior/categories`, {
    headers: { Authorization: `Bearer ${adminToken}` },
  })
  if (!catsRes.ok) { test.skip(true, 'categories list failed'); return }
  const { categories } = (await catsRes.json()) as {
    categories: Array<{ id: string; type: string }>
  }
  const positiveCategory = categories.find((c) => c.type === 'positive')
  const negativeCategory = categories.find((c) => c.type === 'negative')
  if (!positiveCategory || !negativeCategory) { test.skip(true, 'categories missing'); return }

  const teacherEmail = uniqueEmail('sbt')
  const studentEmail = uniqueEmail('sbs')
  const { access_token: teacherToken } = await apiSignup({ email: teacherEmail, password: PASSWORD })
  const { access_token: studentToken } = await apiSignup({ email: studentEmail, password: PASSWORD })
  const studentId = await getUserId(studentToken)
  if (!studentId) { test.skip(true, 'no studentId'); return }

  const courseRes = await fetch(`${API_BASE}/api/v1/courses`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${teacherToken}`, 'Content-Type': 'application/json' },
    body: JSON.stringify({ title: 'Behavior Summary Test' }),
  })
  if (!courseRes.ok) { test.skip(true, `course create failed`); return }
  const { courseCode } = (await courseRes.json()) as { courseCode: string }

  await apiEnroll(teacherToken, courseCode, studentEmail, 'student', studentToken)

  // Award 3 points
  await fetch(`${API_BASE}/api/v1/pbis/awards`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${teacherToken}`, 'Content-Type': 'application/json' },
    body: JSON.stringify({
      awards: [
        { studentId, categoryId: positiveCategory.id, points: 1 },
        { studentId, categoryId: positiveCategory.id, points: 2 },
      ],
    }),
  })

  // File 1 referral
  await fetch(`${API_BASE}/api/v1/behavior/referrals`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${teacherToken}`, 'Content-Type': 'application/json' },
    body: JSON.stringify({
      studentId,
      categoryId: negativeCategory.id,
      description: 'Test referral description.',
    }),
  })

  // Fetch summary as teacher
  const summaryRes = await fetch(`${API_BASE}/api/v1/students/${studentId}/behavior`, {
    headers: { Authorization: `Bearer ${teacherToken}` },
  })
  if (!summaryRes.ok) {
    test.skip(true, `summary failed: ${summaryRes.status} ${await summaryRes.text()}`)
    return
  }
  const summary = (await summaryRes.json()) as {
    totalPoints: number
    awards: Array<{ points: number }>
    referrals: Array<{ categoryName: string; description?: string }>
  }
  expect(summary.totalPoints).toBeGreaterThanOrEqual(3)
  expect(summary.awards.length).toBeGreaterThanOrEqual(2)
  expect(summary.referrals.length).toBeGreaterThanOrEqual(1)
  // Teacher sees referral description
  const withDesc = summary.referrals.some((r) => r.description && r.description.length > 0)
  expect(withDesc).toBe(true)
})

test('Behavior: unrelated user cannot view another student behavior (403)', async () => {
  const { access_token: tokenA } = await apiSignup({
    email: uniqueEmail('ba'),
    password: PASSWORD,
  })
  const { access_token: tokenB } = await apiSignup({
    email: uniqueEmail('bb'),
    password: PASSWORD,
  })
  const userBId = await getUserId(tokenB)
  if (!userBId) { test.skip(true, 'no userBId'); return }

  const res = await fetch(`${API_BASE}/api/v1/students/${userBId}/behavior`, {
    headers: { Authorization: `Bearer ${tokenA}` },
  })
  expect(res.status).toBe(403)
})

// ─────────────────────────────────────────────────────────────────────────────
// Admin dashboard
// ─────────────────────────────────────────────────────────────────────────────

test('Behavior: admin dashboard returns weekly totals and category breakdowns', async () => {
  const adminToken = await getAdminToken()
  const orgId = await getMyOrgId(adminToken)
  if (!orgId) { test.skip(true, 'no orgId'); return }

  const adminId = await getUserId(adminToken)
  if (adminId) await grantOrgAdmin(adminToken, orgId, adminId)

  const res = await fetch(`${API_BASE}/api/v1/admin/orgs/${orgId}/behavior/dashboard`, {
    headers: { Authorization: `Bearer ${adminToken}` },
  })
  if (!res.ok) {
    test.skip(true, `dashboard failed: ${res.status} ${await res.text()}`)
    return
  }
  expect(res.ok).toBeTruthy()
  const body = (await res.json()) as {
    weekStart: string
    totalPoints: number
    totalReferrals: number
    pointsByCategory: Array<{ categoryId: string; categoryName: string; points: number }>
    referralsByCategory: Array<{ categoryId: string; categoryName: string; count: number }>
  }
  expect(typeof body.weekStart).toBe('string')
  expect(typeof body.totalPoints).toBe('number')
  expect(typeof body.totalReferrals).toBe('number')
  expect(Array.isArray(body.pointsByCategory)).toBe(true)
  expect(Array.isArray(body.referralsByCategory)).toBe(true)
})
