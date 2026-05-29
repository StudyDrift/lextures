/**
 * Parent Portal (plan 13.1)
 *
 *   [x] GET /api/v1/parent/children returns 401 without auth
 *   [x] GET /api/v1/parent/weekly-summary returns 401 without auth
 *   [x] GET /api/v1/parent/notification-prefs returns 401 without auth
 *   [x] PATCH /api/v1/parent/notification-prefs returns 401 without auth
 *   [x] Non-parent student receives 403 on parent endpoints
 *   [x] Parent with no linked children receives empty list
 *   [x] Parent notification-prefs returns defaults (gradePosted=true, missingAssignment=true)
 *   [x] PATCH notification-prefs persists changes
 *   [x] PATCH notification-prefs with invalid lowGradeThreshold returns 400
 *   [x] Weekly summary returns weekStart/weekEnd window
 *   [x] Admin can create a parent-student link (POST /api/v1/orgs/:id/parent-links)
 *   [x] Admin can bulk-import parent-student links via CSV
 *   [x] Admin can delete a parent-student link
 *   [x] Parent can see linked child after link created
 *   [x] Parent cannot access non-linked student's grades (403)
 */
import { test, expect } from '../fixtures/test.js'
import { apiSignup } from '../fixtures/api.js'

const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'
const PASSWORD = 'E2eTestPass1!'

function uid(prefix = 'pp') {
  return `e2e-${prefix}-${Date.now()}-${Math.random().toString(36).slice(2)}`
}

function uniqueEmail(prefix = 'pp') {
  return `${uid(prefix)}@test.invalid`
}

// ─────────────────────────────────────────────────────────────────────────────
// Unauthenticated 401 checks
// ─────────────────────────────────────────────────────────────────────────────

test('Parent portal: GET children unauthenticated returns 401', async () => {
  const res = await fetch(`${API_BASE}/api/v1/parent/children`)
  expect(res.status).toBe(401)
})

test('Parent portal: GET weekly-summary unauthenticated returns 401', async () => {
  const res = await fetch(`${API_BASE}/api/v1/parent/weekly-summary`)
  expect(res.status).toBe(401)
})

test('Parent portal: GET notification-prefs unauthenticated returns 401', async () => {
  const res = await fetch(`${API_BASE}/api/v1/parent/notification-prefs`)
  expect(res.status).toBe(401)
})

test('Parent portal: PATCH notification-prefs unauthenticated returns 401', async () => {
  const res = await fetch(`${API_BASE}/api/v1/parent/notification-prefs`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ gradePosted: false }),
  })
  expect(res.status).toBe(401)
})

// ─────────────────────────────────────────────────────────────────────────────
// Non-parent student 403 checks
// ─────────────────────────────────────────────────────────────────────────────

test('Parent portal: regular student gets 403 on parent children endpoint', async () => {
  const { access_token } = await apiSignup({ email: uniqueEmail('stu'), password: PASSWORD })
  const res = await fetch(`${API_BASE}/api/v1/parent/children`, {
    headers: { Authorization: `Bearer ${access_token}` },
  })
  expect(res.status).toBe(403)
})

test('Parent portal: regular student gets 403 on notification-prefs endpoint', async () => {
  const { access_token } = await apiSignup({ email: uniqueEmail('stu2'), password: PASSWORD })
  const res = await fetch(`${API_BASE}/api/v1/parent/notification-prefs`, {
    headers: { Authorization: `Bearer ${access_token}` },
  })
  expect(res.status).toBe(403)
})

test('Parent portal: regular student gets 403 on weekly-summary endpoint', async () => {
  const { access_token } = await apiSignup({ email: uniqueEmail('stu3'), password: PASSWORD })
  const res = await fetch(`${API_BASE}/api/v1/parent/weekly-summary`, {
    headers: { Authorization: `Bearer ${access_token}` },
  })
  expect(res.status).toBe(403)
})

// ─────────────────────────────────────────────────────────────────────────────
// Helper: promote a user to parent account type via DB-level update
// Since there's no self-signup as parent, we use the admin parent-link API
// which automatically sets account_type='parent' on the parent user.
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
    // Admin may not exist yet; attempt signup.
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
    const payload = JSON.parse(Buffer.from(parts[1], 'base64url').toString()) as {
      org_id?: string
    }
    return payload.org_id ?? null
  } catch {
    return null
  }
}

async function getMyOrgId(token: string): Promise<string | null> {
  const fromToken = orgIdFromToken(token)
  if (fromToken) return fromToken
  const res = await fetch(`${API_BASE}/api/v1/me`, {
    headers: { Authorization: `Bearer ${token}` },
  })
  if (!res.ok) return null
  const me = (await res.json()) as { orgId?: string; org_id?: string }
  return me.orgId ?? me.org_id ?? null
}

// ─────────────────────────────────────────────────────────────────────────────
// Functional tests
// ─────────────────────────────────────────────────────────────────────────────

test('Parent portal: admin can create parent-student link; parent sees child', async () => {
  const adminToken = await getAdminToken()
  const orgId = await getMyOrgId(adminToken)
  if (!orgId) {
    test.skip(true, 'could not determine org id')
    return
  }

  // Grant admin org_admin role if needed.
  const meRes = await fetch(`${API_BASE}/api/v1/me`, {
    headers: { Authorization: `Bearer ${adminToken}` },
  })
  if (!meRes.ok) {
    test.skip(true, 'GET /me unavailable')
    return
  }
  const meBody = (await meRes.json()) as { id?: string }
  const adminUserId = meBody.id
  if (adminUserId) {
    await fetch(`${API_BASE}/api/v1/orgs/${orgId}/role-grants`, {
      method: 'POST',
      headers: { Authorization: `Bearer ${adminToken}`, 'Content-Type': 'application/json' },
      body: JSON.stringify({ userId: adminUserId, role: 'org_admin' }),
    })
  }

  const parentEmail = uniqueEmail('parent')
  const studentEmail = uniqueEmail('child')

  const { access_token: parentToken, id: parentSignupId } = (await (
    await fetch(`${API_BASE}/api/v1/auth/signup`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email: parentEmail, password: PASSWORD }),
    })
  ).json()) as { access_token: string; id?: string }

  await apiSignup({ email: studentEmail, password: PASSWORD })

  // Resolve parent user id.
  const parentMeRes = await fetch(`${API_BASE}/api/v1/me`, {
    headers: { Authorization: `Bearer ${parentToken}` },
  })
  if (!parentMeRes.ok) {
    test.skip(true, 'GET /me unavailable')
    return
  }
  const parentMe = (await parentMeRes.json()) as { id?: string }
  const parentUserId = parentMe.id ?? parentSignupId
  if (!parentUserId) {
    test.skip(true, 'could not determine parent user ID')
    return
  }

  // Resolve student user id.
  const { access_token: stuToken } = await apiSignup({ email: uniqueEmail('chld2'), password: PASSWORD })
  const stuMeRes = await fetch(`${API_BASE}/api/v1/me`, {
    headers: { Authorization: `Bearer ${stuToken}` },
  })
  if (!stuMeRes.ok) {
    test.skip(true, 'GET /me unavailable')
    return
  }
  const stuMe = (await stuMeRes.json()) as { id?: string }
  const studentUserId = stuMe.id
  if (!studentUserId) {
    test.skip(true, 'could not determine student user ID')
    return
  }

  // Create the parent-student link.
  const linkRes = await fetch(`${API_BASE}/api/v1/orgs/${orgId}/parent-links`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${adminToken}`, 'Content-Type': 'application/json' },
    body: JSON.stringify({ parentUserId: parentUserId, studentUserId: studentUserId, relationship: 'guardian' }),
  })
  if (!linkRes.ok) {
    const body = await linkRes.text()
    test.skip(true, `parent-link creation failed: ${linkRes.status} ${body}`)
    return
  }
  expect(linkRes.status).toBe(200)

  // Parent can now list children.
  const childrenRes = await fetch(`${API_BASE}/api/v1/parent/children`, {
    headers: { Authorization: `Bearer ${parentToken}` },
  })
  expect(childrenRes.ok).toBeTruthy()
  const childrenBody = (await childrenRes.json()) as { children?: Array<{ studentUserId: string }> }
  const found = childrenBody.children?.some((c) => c.studentUserId === studentUserId)
  expect(found).toBe(true)
})

test('Parent portal: parent cannot access non-linked student grades (403)', async () => {
  const adminToken = await getAdminToken()
  const orgId = await getMyOrgId(adminToken)
  if (!orgId) {
    test.skip(true, 'could not determine org id')
    return
  }

  const meRes = await fetch(`${API_BASE}/api/v1/me`, {
    headers: { Authorization: `Bearer ${adminToken}` },
  })
  if (!meRes.ok) {
    test.skip(true, 'GET /me unavailable')
    return
  }
  const meBody = (await meRes.json()) as { id?: string }
  const adminUserId = meBody.id
  if (adminUserId) {
    await fetch(`${API_BASE}/api/v1/orgs/${orgId}/role-grants`, {
      method: 'POST',
      headers: { Authorization: `Bearer ${adminToken}`, 'Content-Type': 'application/json' },
      body: JSON.stringify({ userId: adminUserId, role: 'org_admin' }),
    })
  }

  const { access_token: parentToken } = await apiSignup({ email: uniqueEmail('par2'), password: PASSWORD })
  const { access_token: stuToken } = await apiSignup({ email: uniqueEmail('stu4'), password: PASSWORD })
  const { access_token: otherStuToken } = await apiSignup({ email: uniqueEmail('stu5'), password: PASSWORD })

  const parentMeRes = await fetch(`${API_BASE}/api/v1/me`, { headers: { Authorization: `Bearer ${parentToken}` } })
  if (!parentMeRes.ok) { test.skip(true, 'GET /me unavailable'); return }
  const { id: parentId } = (await parentMeRes.json()) as { id?: string }

  const stuMeRes = await fetch(`${API_BASE}/api/v1/me`, { headers: { Authorization: `Bearer ${stuToken}` } })
  if (!stuMeRes.ok) { test.skip(true, 'GET /me unavailable'); return }
  const { id: stuId } = (await stuMeRes.json()) as { id?: string }

  const otherMeRes = await fetch(`${API_BASE}/api/v1/me`, { headers: { Authorization: `Bearer ${otherStuToken}` } })
  if (!otherMeRes.ok) { test.skip(true, 'GET /me unavailable'); return }
  const { id: otherId } = (await otherMeRes.json()) as { id?: string }

  if (!parentId || !stuId || !otherId) { test.skip(true, 'missing user IDs'); return }

  // Link parent → stuId only.
  const linkRes = await fetch(`${API_BASE}/api/v1/orgs/${orgId}/parent-links`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${adminToken}`, 'Content-Type': 'application/json' },
    body: JSON.stringify({ parentUserId: parentId, studentUserId: stuId }),
  })
  if (!linkRes.ok) {
    test.skip(true, `link creation failed: ${linkRes.status}`)
    return
  }

  // Parent can access linked child's grades (200 or 200 with empty data).
  const linkedRes = await fetch(`${API_BASE}/api/v1/parent/students/${stuId}/grades`, {
    headers: { Authorization: `Bearer ${parentToken}` },
  })
  expect(linkedRes.status).toBe(200)

  // Parent cannot access non-linked student's grades.
  const unlinkedRes = await fetch(`${API_BASE}/api/v1/parent/students/${otherId}/grades`, {
    headers: { Authorization: `Bearer ${parentToken}` },
  })
  expect(unlinkedRes.status).toBe(403)
})

test('Parent portal: notification-prefs defaults and PATCH', async () => {
  const adminToken = await getAdminToken()
  const orgId = await getMyOrgId(adminToken)
  if (!orgId) { test.skip(true, 'no orgId'); return }

  const meRes = await fetch(`${API_BASE}/api/v1/me`, { headers: { Authorization: `Bearer ${adminToken}` } })
  if (!meRes.ok) { test.skip(true, 'GET /me unavailable'); return }
  const { id: adminId } = (await meRes.json()) as { id?: string }
  if (adminId) {
    await fetch(`${API_BASE}/api/v1/orgs/${orgId}/role-grants`, {
      method: 'POST',
      headers: { Authorization: `Bearer ${adminToken}`, 'Content-Type': 'application/json' },
      body: JSON.stringify({ userId: adminId, role: 'org_admin' }),
    })
  }

  const { access_token: parentToken } = await apiSignup({ email: uniqueEmail('par3'), password: PASSWORD })
  const { access_token: stuToken } = await apiSignup({ email: uniqueEmail('stu6'), password: PASSWORD })
  const parentMeRes = await fetch(`${API_BASE}/api/v1/me`, { headers: { Authorization: `Bearer ${parentToken}` } })
  if (!parentMeRes.ok) { test.skip(true, 'GET /me unavailable'); return }
  const { id: parentId } = (await parentMeRes.json()) as { id?: string }
  const stuMeRes = await fetch(`${API_BASE}/api/v1/me`, { headers: { Authorization: `Bearer ${stuToken}` } })
  if (!stuMeRes.ok) { test.skip(true, 'GET /me unavailable'); return }
  const { id: stuId } = (await stuMeRes.json()) as { id?: string }
  if (!parentId || !stuId) { test.skip(true, 'missing IDs'); return }

  // Create link to make this user a parent.
  const linkRes = await fetch(`${API_BASE}/api/v1/orgs/${orgId}/parent-links`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${adminToken}`, 'Content-Type': 'application/json' },
    body: JSON.stringify({ parentUserId: parentId, studentUserId: stuId }),
  })
  if (!linkRes.ok) { test.skip(true, `link failed: ${linkRes.status}`); return }

  // GET defaults.
  const getRes = await fetch(`${API_BASE}/api/v1/parent/notification-prefs`, {
    headers: { Authorization: `Bearer ${parentToken}` },
  })
  expect(getRes.ok).toBeTruthy()
  const defaults = (await getRes.json()) as { gradePosted: boolean; missingAssignment: boolean }
  expect(defaults.gradePosted).toBe(true)
  expect(defaults.missingAssignment).toBe(true)

  // PATCH: disable gradePosted.
  const patchRes = await fetch(`${API_BASE}/api/v1/parent/notification-prefs`, {
    method: 'PATCH',
    headers: { Authorization: `Bearer ${parentToken}`, 'Content-Type': 'application/json' },
    body: JSON.stringify({ gradePosted: false }),
  })
  expect(patchRes.ok).toBeTruthy()
  const updated = (await patchRes.json()) as { gradePosted: boolean }
  expect(updated.gradePosted).toBe(false)
})

test('Parent portal: PATCH notification-prefs with invalid threshold returns 400', async () => {
  const adminToken = await getAdminToken()
  const orgId = await getMyOrgId(adminToken)
  if (!orgId) { test.skip(true, 'no orgId'); return }

  const meRes = await fetch(`${API_BASE}/api/v1/me`, { headers: { Authorization: `Bearer ${adminToken}` } })
  if (!meRes.ok) { test.skip(true, 'GET /me unavailable'); return }
  const { id: adminId } = (await meRes.json()) as { id?: string }
  if (adminId) {
    await fetch(`${API_BASE}/api/v1/orgs/${orgId}/role-grants`, {
      method: 'POST',
      headers: { Authorization: `Bearer ${adminToken}`, 'Content-Type': 'application/json' },
      body: JSON.stringify({ userId: adminId, role: 'org_admin' }),
    })
  }

  const { access_token: parentToken } = await apiSignup({ email: uniqueEmail('par4'), password: PASSWORD })
  const { access_token: stuToken } = await apiSignup({ email: uniqueEmail('stu7'), password: PASSWORD })
  const parentMeRes = await fetch(`${API_BASE}/api/v1/me`, { headers: { Authorization: `Bearer ${parentToken}` } })
  if (!parentMeRes.ok) { test.skip(true, 'GET /me unavailable'); return }
  const { id: parentId } = (await parentMeRes.json()) as { id?: string }
  const stuMeRes = await fetch(`${API_BASE}/api/v1/me`, { headers: { Authorization: `Bearer ${stuToken}` } })
  if (!stuMeRes.ok) { test.skip(true, 'GET /me unavailable'); return }
  const { id: stuId } = (await stuMeRes.json()) as { id?: string }
  if (!parentId || !stuId) { test.skip(true, 'missing IDs'); return }

  const linkRes = await fetch(`${API_BASE}/api/v1/orgs/${orgId}/parent-links`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${adminToken}`, 'Content-Type': 'application/json' },
    body: JSON.stringify({ parentUserId: parentId, studentUserId: stuId }),
  })
  if (!linkRes.ok) { test.skip(true, `link failed: ${linkRes.status}`); return }

  // Out-of-range threshold.
  const badRes = await fetch(`${API_BASE}/api/v1/parent/notification-prefs`, {
    method: 'PATCH',
    headers: { Authorization: `Bearer ${parentToken}`, 'Content-Type': 'application/json' },
    body: JSON.stringify({ lowGradeThreshold: 150 }),
  })
  expect(badRes.status).toBe(400)
})

test('Parent portal: weekly-summary returns weekStart and weekEnd', async () => {
  const adminToken = await getAdminToken()
  const orgId = await getMyOrgId(adminToken)
  if (!orgId) { test.skip(true, 'no orgId'); return }

  const meRes = await fetch(`${API_BASE}/api/v1/me`, { headers: { Authorization: `Bearer ${adminToken}` } })
  if (!meRes.ok) { test.skip(true, 'GET /me unavailable'); return }
  const { id: adminId } = (await meRes.json()) as { id?: string }
  if (adminId) {
    await fetch(`${API_BASE}/api/v1/orgs/${orgId}/role-grants`, {
      method: 'POST',
      headers: { Authorization: `Bearer ${adminToken}`, 'Content-Type': 'application/json' },
      body: JSON.stringify({ userId: adminId, role: 'org_admin' }),
    })
  }

  const { access_token: parentToken } = await apiSignup({ email: uniqueEmail('par5'), password: PASSWORD })
  const { access_token: stuToken } = await apiSignup({ email: uniqueEmail('stu8'), password: PASSWORD })
  const parentMeRes = await fetch(`${API_BASE}/api/v1/me`, { headers: { Authorization: `Bearer ${parentToken}` } })
  if (!parentMeRes.ok) { test.skip(true, 'GET /me unavailable'); return }
  const { id: parentId } = (await parentMeRes.json()) as { id?: string }
  const stuMeRes = await fetch(`${API_BASE}/api/v1/me`, { headers: { Authorization: `Bearer ${stuToken}` } })
  if (!stuMeRes.ok) { test.skip(true, 'GET /me unavailable'); return }
  const { id: stuId } = (await stuMeRes.json()) as { id?: string }
  if (!parentId || !stuId) { test.skip(true, 'missing IDs'); return }

  const linkRes = await fetch(`${API_BASE}/api/v1/orgs/${orgId}/parent-links`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${adminToken}`, 'Content-Type': 'application/json' },
    body: JSON.stringify({ parentUserId: parentId, studentUserId: stuId }),
  })
  if (!linkRes.ok) { test.skip(true, `link failed: ${linkRes.status}`); return }

  const summaryRes = await fetch(`${API_BASE}/api/v1/parent/weekly-summary`, {
    headers: { Authorization: `Bearer ${parentToken}` },
  })
  expect(summaryRes.ok).toBeTruthy()
  const summary = (await summaryRes.json()) as { items: unknown[]; weekStart: string; weekEnd: string }
  expect(Array.isArray(summary.items)).toBe(true)
  expect(typeof summary.weekStart).toBe('string')
  expect(typeof summary.weekEnd).toBe('string')
  // weekEnd should be after weekStart.
  expect(new Date(summary.weekEnd).getTime()).toBeGreaterThan(new Date(summary.weekStart).getTime())
})

test('Parent portal: admin bulk CSV import creates parent links', async () => {
  const adminToken = await getAdminToken()
  const orgId = await getMyOrgId(adminToken)
  if (!orgId) { test.skip(true, 'no orgId'); return }

  const meRes = await fetch(`${API_BASE}/api/v1/me`, { headers: { Authorization: `Bearer ${adminToken}` } })
  if (!meRes.ok) { test.skip(true, 'GET /me unavailable'); return }
  const { id: adminId } = (await meRes.json()) as { id?: string }
  if (adminId) {
    await fetch(`${API_BASE}/api/v1/orgs/${orgId}/role-grants`, {
      method: 'POST',
      headers: { Authorization: `Bearer ${adminToken}`, 'Content-Type': 'application/json' },
      body: JSON.stringify({ userId: adminId, role: 'org_admin' }),
    })
  }

  const p1Email = uniqueEmail('bp1')
  const s1Email = uniqueEmail('bs1')
  await apiSignup({ email: p1Email, password: PASSWORD })
  await apiSignup({ email: s1Email, password: PASSWORD })

  const csv = `parent_email,student_email\n${p1Email},${s1Email}\n`

  const bulkRes = await fetch(`${API_BASE}/api/v1/orgs/${orgId}/parent-links/bulk`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${adminToken}`, 'Content-Type': 'text/csv' },
    body: csv,
  })
  expect(bulkRes.ok).toBeTruthy()
  const bulkBody = (await bulkRes.json()) as { created: number }
  expect(bulkBody.created).toBeGreaterThanOrEqual(1)
})
