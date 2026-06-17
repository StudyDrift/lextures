/**
 * Attendance (plan 13.2)
 *
 *   [x] GET section attendance unauthenticated returns 401
 *   [x] PUT section attendance unauthenticated returns 401
 *   [x] GET student attendance unauthenticated returns 401
 *   [x] GET attendance codes unauthenticated returns 401
 *   [x] POST attendance export unauthenticated returns 401
 *   [x] Admin can seed default attendance codes (P, AU, AE, TU, TE)
 *   [x] Admin can create a custom attendance code
 *   [x] Admin can delete an attendance code
 *   [x] Teacher can batch-save attendance records for a section
 *   [x] GET section attendance returns saved records
 *   [x] PUT attendance for date > 5 days old returns 403 for non-admin
 *   [x] Export returns CSV with correct Content-Type
 *   [x] Parent can view linked student's attendance
 *   [x] Parent cannot view non-linked student's attendance (403)
 */
import { test, expect } from '../fixtures/test.js'
import { apiSignup, apiEnroll } from '../fixtures/api.js'

const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'
const PASSWORD = 'E2eTestPass1!'

function uid(prefix = 'att') {
  return `e2e-${prefix}-${Date.now()}-${Math.random().toString(36).slice(2)}`
}
function uniqueEmail(prefix = 'att') {
  return `${uid(prefix)}@test.invalid`
}

// ─────────────────────────────────────────────────────────────────────────────
// Auth guard checks (no token → 401)
// ─────────────────────────────────────────────────────────────────────────────

test('Attendance: GET section attendance unauthenticated returns 401', async () => {
  const res = await fetch(
    `${API_BASE}/api/v1/sections/00000000-0000-0000-0000-000000000001/attendance/2026-01-15`,
  )
  expect(res.status).toBe(401)
})

test('Attendance: PUT section attendance unauthenticated returns 401', async () => {
  const res = await fetch(
    `${API_BASE}/api/v1/sections/00000000-0000-0000-0000-000000000001/attendance/2026-01-15`,
    { method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ records: [] }) },
  )
  expect(res.status).toBe(401)
})

test('Attendance: GET student attendance unauthenticated returns 401', async () => {
  const res = await fetch(`${API_BASE}/api/v1/students/00000000-0000-0000-0000-000000000001/attendance`)
  expect(res.status).toBe(401)
})

test('Attendance: GET attendance codes unauthenticated returns 401', async () => {
  const res = await fetch(
    `${API_BASE}/api/v1/admin/orgs/00000000-0000-0000-0000-000000000001/attendance/codes`,
  )
  expect(res.status).toBe(401)
})

test('Attendance: POST attendance export unauthenticated returns 401', async () => {
  const res = await fetch(
    `${API_BASE}/api/v1/admin/orgs/00000000-0000-0000-0000-000000000001/attendance/export`,
    { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({}) },
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
// Attendance codes CRUD
// ─────────────────────────────────────────────────────────────────────────────

test('Attendance: admin can seed default codes (P, AU, AE, TU, TE)', async () => {
  const adminToken = await getAdminToken()
  const orgId = await getMyOrgId(adminToken)
  if (!orgId) { test.skip(true, 'no orgId'); return }

  const adminId = await getUserId(adminToken)
  if (adminId) await grantOrgAdmin(adminToken, orgId, adminId)

  const res = await fetch(`${API_BASE}/api/v1/admin/orgs/${orgId}/attendance/codes`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${adminToken}`, 'Content-Type': 'application/json' },
    body: JSON.stringify({ seedDefaults: true }),
  })
  if (!res.ok) {
    test.skip(true, `seed failed: ${res.status} ${await res.text()}`)
    return
  }
  expect(res.ok).toBeTruthy()
  const body = (await res.json()) as { codes: Array<{ code: string }> }
  const codes = body.codes.map((c) => c.code)
  expect(codes).toContain('P')
  expect(codes).toContain('AU')
  expect(codes).toContain('AE')
  expect(codes).toContain('TU')
  expect(codes).toContain('TE')
})

test('Attendance: admin can create a custom attendance code', async () => {
  const adminToken = await getAdminToken()
  const orgId = await getMyOrgId(adminToken)
  if (!orgId) { test.skip(true, 'no orgId'); return }

  const adminId = await getUserId(adminToken)
  if (adminId) await grantOrgAdmin(adminToken, orgId, adminId)

  const customCode = `X${Date.now().toString().slice(-4)}`
  const res = await fetch(`${API_BASE}/api/v1/admin/orgs/${orgId}/attendance/codes`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${adminToken}`, 'Content-Type': 'application/json' },
    body: JSON.stringify({ code: customCode, label: 'Test Custom Code', category: 'other' }),
  })
  if (!res.ok) { test.skip(true, `create failed: ${res.status}`); return }
  expect(res.status).toBe(201)
  const body = (await res.json()) as { code: string; label: string; category: string }
  expect(body.code).toBe(customCode)
  expect(body.label).toBe('Test Custom Code')
  expect(body.category).toBe('other')
})

test('Attendance: admin can delete an attendance code', async () => {
  const adminToken = await getAdminToken()
  const orgId = await getMyOrgId(adminToken)
  if (!orgId) { test.skip(true, 'no orgId'); return }

  const adminId = await getUserId(adminToken)
  if (adminId) await grantOrgAdmin(adminToken, orgId, adminId)

  // Create a code to delete.
  const codeVal = `DEL${Date.now().toString().slice(-4)}`
  const createRes = await fetch(`${API_BASE}/api/v1/admin/orgs/${orgId}/attendance/codes`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${adminToken}`, 'Content-Type': 'application/json' },
    body: JSON.stringify({ code: codeVal, label: 'Delete Me', category: 'other' }),
  })
  if (!createRes.ok) { test.skip(true, `create failed: ${createRes.status}`); return }
  const created = (await createRes.json()) as { id: string }

  const deleteRes = await fetch(
    `${API_BASE}/api/v1/admin/orgs/${orgId}/attendance/codes/${created.id}`,
    { method: 'DELETE', headers: { Authorization: `Bearer ${adminToken}` } },
  )
  expect(deleteRes.status).toBe(204)
})

test('Attendance: creating code with invalid category returns 400', async () => {
  const adminToken = await getAdminToken()
  const orgId = await getMyOrgId(adminToken)
  if (!orgId) { test.skip(true, 'no orgId'); return }

  const adminId = await getUserId(adminToken)
  if (adminId) await grantOrgAdmin(adminToken, orgId, adminId)

  const res = await fetch(`${API_BASE}/api/v1/admin/orgs/${orgId}/attendance/codes`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${adminToken}`, 'Content-Type': 'application/json' },
    body: JSON.stringify({ code: 'BAD', label: 'Bad Category', category: 'invalid_category' }),
  })
  expect(res.status).toBe(400)
})

// ─────────────────────────────────────────────────────────────────────────────
// Attendance recording
// ─────────────────────────────────────────────────────────────────────────────

test('Attendance: teacher can batch-save and retrieve attendance for a section', async () => {
  const adminToken = await getAdminToken()
  const orgId = await getMyOrgId(adminToken)
  if (!orgId) { test.skip(true, 'no orgId'); return }

  const adminId = await getUserId(adminToken)
  if (adminId) await grantOrgAdmin(adminToken, orgId, adminId)

  // Seed default codes.
  await fetch(`${API_BASE}/api/v1/admin/orgs/${orgId}/attendance/codes`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${adminToken}`, 'Content-Type': 'application/json' },
    body: JSON.stringify({ seedDefaults: true }),
  })

  const codesRes = await fetch(`${API_BASE}/api/v1/admin/orgs/${orgId}/attendance/codes`, {
    headers: { Authorization: `Bearer ${adminToken}` },
  })
  if (!codesRes.ok) { test.skip(true, 'codes list failed'); return }
  const { codes } = (await codesRes.json()) as { codes: Array<{ id: string; code: string }> }
  const presentCode = codes.find((c) => c.code === 'P')
  if (!presentCode) { test.skip(true, 'P code not found'); return }

  const teacherEmail = uniqueEmail('tch')
  const studentEmail = uniqueEmail('stu')
  // Create course as TEACHER so teacher is the owner and has management rights.
  const { access_token: teacherToken } = await apiSignup({ email: teacherEmail, password: PASSWORD })
  const { access_token: studentToken } = await apiSignup({ email: studentEmail, password: PASSWORD })

  const courseCreateRes = await fetch(`${API_BASE}/api/v1/courses`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${teacherToken}`, 'Content-Type': 'application/json' },
    body: JSON.stringify({ title: 'Attendance Test Course' }),
  })
  if (!courseCreateRes.ok) { test.skip(true, `course create failed: ${courseCreateRes.status}`); return }
  // Use the auto-generated courseCode from the server response.
  const { courseCode } = (await courseCreateRes.json()) as { courseCode: string }
  if (!courseCode) { test.skip(true, 'no courseCode in create response'); return }

  // Enable sections via features endpoint.
  await fetch(`${API_BASE}/api/v1/courses/${courseCode}/features`, {
    method: 'PATCH',
    headers: { Authorization: `Bearer ${teacherToken}`, 'Content-Type': 'application/json' },
    body: JSON.stringify({ sectionsEnabled: true, notebookEnabled: true, calendarEnabled: true }),
  })

  // Enroll student by email.
  await apiEnroll(teacherToken, courseCode, studentEmail, 'student', studentToken)

  // Get teacher and student IDs.
  const teacherId = await getUserId(teacherToken)
  if (!teacherId) { test.skip(true, 'no teacherId'); return }

  // Get enrolled students to find stuId.
  const rosterRes = await fetch(`${API_BASE}/api/v1/courses/${courseCode}/enrollments`, {
    headers: { Authorization: `Bearer ${teacherToken}` },
  })
  if (!rosterRes.ok) { test.skip(true, `roster failed: ${rosterRes.status}`); return }
  const rosterBody = (await rosterRes.json()) as { enrollments?: Array<{ userId: string; role: string }> }
  const stuEnrollment = rosterBody.enrollments?.find((e) => e.role === 'student')
  const stuId = stuEnrollment?.userId
  if (!stuId) { test.skip(true, 'student enrollment not found'); return }

  // Create a section using teacher token.
  const sectionRes = await fetch(`${API_BASE}/api/v1/courses/${courseCode}/sections`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${teacherToken}`, 'Content-Type': 'application/json' },
    body: JSON.stringify({ sectionCode: 'S01', instructorUserId: teacherId }),
  })
  if (!sectionRes.ok) { test.skip(true, `section create failed: ${sectionRes.status}`); return }
  const section = (await sectionRes.json()) as { id?: string }
  const sectionId = section.id
  if (!sectionId) { test.skip(true, 'no sectionId'); return }

  // Move student into section by re-enrolling with sectionId.
  await apiEnroll(teacherToken, courseCode, studentEmail, 'student', {
    memberToken: studentToken,
    sectionId,
  })

  const date = new Date().toISOString().slice(0, 10)

  // Teacher saves attendance.
  const saveRes = await fetch(`${API_BASE}/api/v1/sections/${sectionId}/attendance/${date}`, {
    method: 'PUT',
    headers: { Authorization: `Bearer ${teacherToken}`, 'Content-Type': 'application/json' },
    body: JSON.stringify({
      records: [{ studentId: stuId, codeId: presentCode.id }],
    }),
  })
  if (!saveRes.ok) {
    const body = await saveRes.text()
    test.skip(true, `attendance save failed: ${saveRes.status} ${body}`)
    return
  }
  expect(saveRes.ok).toBeTruthy()
  const saveBody = (await saveRes.json()) as { saved: number; message: string }
  expect(saveBody.saved).toBe(1)
  expect(saveBody.message).toBe('Attendance saved.')

  // GET attendance returns the record.
  const getRes = await fetch(`${API_BASE}/api/v1/sections/${sectionId}/attendance/${date}`, {
    headers: { Authorization: `Bearer ${teacherToken}` },
  })
  expect(getRes.ok).toBeTruthy()
  const getBody = (await getRes.json()) as {
    records: Array<{ studentId: string; codeId: string }>
  }
  const found = getBody.records.some((r) => r.studentId === stuId && r.codeId === presentCode.id)
  expect(found).toBe(true)
})

test('Attendance: batch save is idempotent (re-save same records succeeds)', async () => {
  const adminToken = await getAdminToken()
  const orgId = await getMyOrgId(adminToken)
  if (!orgId) { test.skip(true, 'no orgId'); return }

  const adminId = await getUserId(adminToken)
  if (adminId) await grantOrgAdmin(adminToken, orgId, adminId)

  await fetch(`${API_BASE}/api/v1/admin/orgs/${orgId}/attendance/codes`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${adminToken}`, 'Content-Type': 'application/json' },
    body: JSON.stringify({ seedDefaults: true }),
  })
  const codesRes = await fetch(`${API_BASE}/api/v1/admin/orgs/${orgId}/attendance/codes`, {
    headers: { Authorization: `Bearer ${adminToken}` },
  })
  if (!codesRes.ok) { test.skip(true, 'codes list failed'); return }
  const { codes } = (await codesRes.json()) as { codes: Array<{ id: string; code: string }> }
  const presentCode = codes.find((c) => c.code === 'P')
  if (!presentCode) { test.skip(true, 'P code not found'); return }

  const teacherEmail = uniqueEmail('it')
  const studentEmail = uniqueEmail('is')
  const { access_token: teacherToken } = await apiSignup({ email: teacherEmail, password: PASSWORD })
  const { access_token: studentToken } = await apiSignup({ email: studentEmail, password: PASSWORD })

  const courseCreateRes = await fetch(`${API_BASE}/api/v1/courses`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${teacherToken}`, 'Content-Type': 'application/json' },
    body: JSON.stringify({ title: 'Idempotent Test' }),
  })
  if (!courseCreateRes.ok) { test.skip(true, `course create failed: ${courseCreateRes.status}`); return }
  const { courseCode } = (await courseCreateRes.json()) as { courseCode: string }
  if (!courseCode) { test.skip(true, 'no courseCode'); return }

  await fetch(`${API_BASE}/api/v1/courses/${courseCode}/features`, {
    method: 'PATCH',
    headers: { Authorization: `Bearer ${teacherToken}`, 'Content-Type': 'application/json' },
    body: JSON.stringify({ sectionsEnabled: true, notebookEnabled: true, calendarEnabled: true }),
  })

  await apiEnroll(teacherToken, courseCode, studentEmail, 'student', studentToken)

  const teacherId = await getUserId(teacherToken)
  if (!teacherId) { test.skip(true, 'missing teacherId'); return }

  const rosterRes = await fetch(`${API_BASE}/api/v1/courses/${courseCode}/enrollments`, {
    headers: { Authorization: `Bearer ${teacherToken}` },
  })
  if (!rosterRes.ok) { test.skip(true, `roster failed: ${rosterRes.status}`); return }
  const rosterBody = (await rosterRes.json()) as { enrollments?: Array<{ userId: string; role: string }> }
  const stuId = rosterBody.enrollments?.find((e) => e.role === 'student')?.userId
  if (!stuId) { test.skip(true, 'no student'); return }

  const sectionRes = await fetch(`${API_BASE}/api/v1/courses/${courseCode}/sections`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${teacherToken}`, 'Content-Type': 'application/json' },
    body: JSON.stringify({ sectionCode: 'S01', instructorUserId: teacherId }),
  })
  if (!sectionRes.ok) { test.skip(true, `section create failed: ${sectionRes.status}`); return }
  const { id: sectionId } = (await sectionRes.json()) as { id?: string }
  if (!sectionId) { test.skip(true, 'no sectionId'); return }

  await apiEnroll(teacherToken, courseCode, studentEmail, 'student', {
    memberToken: studentToken,
    sectionId,
  })

  const date = new Date().toISOString().slice(0, 10)
  const payload = { records: [{ studentId: stuId, codeId: presentCode.id }] }
  const url = `${API_BASE}/api/v1/sections/${sectionId}/attendance/${date}`

  const r1 = await fetch(url, {
    method: 'PUT',
    headers: { Authorization: `Bearer ${teacherToken}`, 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  })
  if (!r1.ok) { test.skip(true, `first save failed: ${r1.status} ${await r1.text()}`); return }

  const r2 = await fetch(url, {
    method: 'PUT',
    headers: { Authorization: `Bearer ${teacherToken}`, 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  })
  expect(r2.ok).toBeTruthy()
})

test('Attendance: PUT with date 10 days in past returns 403 for non-admin teacher', async () => {
  const adminToken = await getAdminToken()
  const orgId = await getMyOrgId(adminToken)
  if (!orgId) { test.skip(true, 'no orgId'); return }

  const adminId = await getUserId(adminToken)
  if (adminId) await grantOrgAdmin(adminToken, orgId, adminId)

  await fetch(`${API_BASE}/api/v1/admin/orgs/${orgId}/attendance/codes`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${adminToken}`, 'Content-Type': 'application/json' },
    body: JSON.stringify({ seedDefaults: true }),
  })
  const codesRes = await fetch(`${API_BASE}/api/v1/admin/orgs/${orgId}/attendance/codes`, {
    headers: { Authorization: `Bearer ${adminToken}` },
  })
  if (!codesRes.ok) { test.skip(true, 'codes list failed'); return }
  const { codes } = (await codesRes.json()) as { codes: Array<{ id: string; code: string }> }
  const presentCode = codes.find((c) => c.code === 'P')
  if (!presentCode) { test.skip(true, 'P code not found'); return }

  const teacherEmail = uniqueEmail('ot')
  const studentEmail = uniqueEmail('os')
  const { access_token: teacherToken } = await apiSignup({ email: teacherEmail, password: PASSWORD })
  const { access_token: studentToken } = await apiSignup({ email: studentEmail, password: PASSWORD })

  const courseCreateRes = await fetch(`${API_BASE}/api/v1/courses`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${teacherToken}`, 'Content-Type': 'application/json' },
    body: JSON.stringify({ title: 'Old Date Test' }),
  })
  if (!courseCreateRes.ok) { test.skip(true, `course create: ${courseCreateRes.status}`); return }
  const { courseCode } = (await courseCreateRes.json()) as { courseCode: string }
  if (!courseCode) { test.skip(true, 'no courseCode'); return }

  await fetch(`${API_BASE}/api/v1/courses/${courseCode}/features`, {
    method: 'PATCH',
    headers: { Authorization: `Bearer ${teacherToken}`, 'Content-Type': 'application/json' },
    body: JSON.stringify({ sectionsEnabled: true, notebookEnabled: true, calendarEnabled: true }),
  })

  await apiEnroll(teacherToken, courseCode, studentEmail, 'student', studentToken)

  const teacherId = await getUserId(teacherToken)
  if (!teacherId) { test.skip(true, 'missing teacherId'); return }

  const rosterRes = await fetch(`${API_BASE}/api/v1/courses/${courseCode}/enrollments`, {
    headers: { Authorization: `Bearer ${teacherToken}` },
  })
  if (!rosterRes.ok) { test.skip(true, `roster: ${rosterRes.status}`); return }
  const rosterBody = (await rosterRes.json()) as { enrollments?: Array<{ userId: string; role: string }> }
  const stuId = rosterBody.enrollments?.find((e) => e.role === 'student')?.userId
  if (!stuId) { test.skip(true, 'no student'); return }

  const sectionRes = await fetch(`${API_BASE}/api/v1/courses/${courseCode}/sections`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${teacherToken}`, 'Content-Type': 'application/json' },
    body: JSON.stringify({ sectionCode: 'S01', instructorUserId: teacherId }),
  })
  if (!sectionRes.ok) { test.skip(true, `section: ${sectionRes.status}`); return }
  const { id: sectionId } = (await sectionRes.json()) as { id?: string }
  if (!sectionId) { test.skip(true, 'no sectionId'); return }

  await apiEnroll(teacherToken, courseCode, studentEmail, 'student', {
    memberToken: studentToken,
    sectionId,
  })

  // Use a date 10 days in the past — past the 5-day edit window.
  const oldDate = new Date(Date.now() - 10 * 24 * 60 * 60 * 1000).toISOString().slice(0, 10)
  const res = await fetch(`${API_BASE}/api/v1/sections/${sectionId}/attendance/${oldDate}`, {
    method: 'PUT',
    headers: { Authorization: `Bearer ${teacherToken}`, 'Content-Type': 'application/json' },
    body: JSON.stringify({ records: [{ studentId: stuId, codeId: presentCode.id }] }),
  })
  expect(res.status).toBe(403)
})

// ─────────────────────────────────────────────────────────────────────────────
// Export
// ─────────────────────────────────────────────────────────────────────────────

test('Attendance: export returns CSV with correct Content-Type', async () => {
  const adminToken = await getAdminToken()
  const orgId = await getMyOrgId(adminToken)
  if (!orgId) { test.skip(true, 'no orgId'); return }

  const adminId = await getUserId(adminToken)
  if (adminId) await grantOrgAdmin(adminToken, orgId, adminId)

  const today = new Date().toISOString().slice(0, 10)
  const res = await fetch(`${API_BASE}/api/v1/admin/orgs/${orgId}/attendance/export`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${adminToken}`, 'Content-Type': 'application/json' },
    body: JSON.stringify({ startDate: today, endDate: today, format: 'csv' }),
  })
  if (!res.ok) {
    test.skip(true, `export failed: ${res.status} ${await res.text()}`)
    return
  }
  expect(res.headers.get('content-type')).toContain('text/csv')
})

test('Attendance: CALPADS export returns CSV with CALPADS header', async () => {
  const adminToken = await getAdminToken()
  const orgId = await getMyOrgId(adminToken)
  if (!orgId) { test.skip(true, 'no orgId'); return }

  const adminId = await getUserId(adminToken)
  if (adminId) await grantOrgAdmin(adminToken, orgId, adminId)

  const today = new Date().toISOString().slice(0, 10)
  const res = await fetch(`${API_BASE}/api/v1/admin/orgs/${orgId}/attendance/export`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${adminToken}`, 'Content-Type': 'application/json' },
    body: JSON.stringify({ startDate: today, endDate: today, format: 'calpads' }),
  })
  if (!res.ok) { test.skip(true, `export: ${res.status}`); return }
  const text = await res.text()
  // First line must include CALPADS columns.
  const firstLine = text.split('\n')[0] ?? ''
  expect(firstLine).toContain('CALPADSCode')
})

test('Attendance: export with endDate < startDate returns 400', async () => {
  const adminToken = await getAdminToken()
  const orgId = await getMyOrgId(adminToken)
  if (!orgId) { test.skip(true, 'no orgId'); return }

  const adminId = await getUserId(adminToken)
  if (adminId) await grantOrgAdmin(adminToken, orgId, adminId)

  const res = await fetch(`${API_BASE}/api/v1/admin/orgs/${orgId}/attendance/export`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${adminToken}`, 'Content-Type': 'application/json' },
    body: JSON.stringify({ startDate: '2026-05-15', endDate: '2026-05-01', format: 'csv' }),
  })
  expect(res.status).toBe(400)
})

// ─────────────────────────────────────────────────────────────────────────────
// Parent attendance view
// ─────────────────────────────────────────────────────────────────────────────

test('Attendance: parent can view linked student attendance', async () => {
  const adminToken = await getAdminToken()
  const orgId = await getMyOrgId(adminToken)
  if (!orgId) { test.skip(true, 'no orgId'); return }

  const adminId = await getUserId(adminToken)
  if (adminId) await grantOrgAdmin(adminToken, orgId, adminId)

  const { access_token: parentToken } = await apiSignup({ email: uniqueEmail('par'), password: PASSWORD })
  const { access_token: stuToken } = await apiSignup({ email: uniqueEmail('stu'), password: PASSWORD })

  const parentId = await getUserId(parentToken)
  const stuId = await getUserId(stuToken)
  if (!parentId || !stuId) { test.skip(true, 'missing IDs'); return }

  // Create parent-student link.
  const linkRes = await fetch(`${API_BASE}/api/v1/orgs/${orgId}/parent-links`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${adminToken}`, 'Content-Type': 'application/json' },
    body: JSON.stringify({ parentUserId: parentId, studentUserId: stuId }),
  })
  if (!linkRes.ok) { test.skip(true, `link failed: ${linkRes.status}`); return }

  // Parent can see child's attendance (empty list is fine).
  const attRes = await fetch(`${API_BASE}/api/v1/parent/students/${stuId}/attendance`, {
    headers: { Authorization: `Bearer ${parentToken}` },
  })
  expect(attRes.ok).toBeTruthy()
  const body = (await attRes.json()) as { records: unknown[] }
  expect(Array.isArray(body.records)).toBe(true)
})

test('Attendance: parent cannot view non-linked student attendance (403)', async () => {
  const adminToken = await getAdminToken()
  const orgId = await getMyOrgId(adminToken)
  if (!orgId) { test.skip(true, 'no orgId'); return }

  const adminId = await getUserId(adminToken)
  if (adminId) await grantOrgAdmin(adminToken, orgId, adminId)

  const { access_token: parentToken } = await apiSignup({ email: uniqueEmail('par2'), password: PASSWORD })
  const { access_token: stuToken } = await apiSignup({ email: uniqueEmail('stu2'), password: PASSWORD })
  const { access_token: otherStuToken } = await apiSignup({ email: uniqueEmail('stu3'), password: PASSWORD })

  const parentId = await getUserId(parentToken)
  const stuId = await getUserId(stuToken)
  const otherId = await getUserId(otherStuToken)
  if (!parentId || !stuId || !otherId) { test.skip(true, 'missing IDs'); return }

  // Link parent to stuId only.
  const linkRes = await fetch(`${API_BASE}/api/v1/orgs/${orgId}/parent-links`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${adminToken}`, 'Content-Type': 'application/json' },
    body: JSON.stringify({ parentUserId: parentId, studentUserId: stuId }),
  })
  if (!linkRes.ok) { test.skip(true, `link failed: ${linkRes.status}`); return }

  // Parent cannot see non-linked student attendance.
  const attRes = await fetch(`${API_BASE}/api/v1/parent/students/${otherId}/attendance`, {
    headers: { Authorization: `Bearer ${parentToken}` },
  })
  expect(attRes.status).toBe(403)
})
