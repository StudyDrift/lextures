/**
 * Hall pass + classroom signals (plan 13.9).
 *
 *   [x] Unauthenticated routes return 401
 *   [x] Invalid UUIDs return 400
 *   [x] Student can request a hall pass; teacher approves it; student marks returned
 *   [x] Teacher sees the active pass in the section's "currently out" list
 *   [x] Anonymous question hides the author from the submission response
 *   [x] Teacher can see questions in the queue (includes authorId for moderation)
 */
import { test, expect } from '@playwright/test'
import { apiSignup, apiCreateCourse, apiEnroll } from '../fixtures/api.js'

const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'
const PASSWORD = 'E2eTestPass1!'

function uid(prefix = 'cs') {
  return `e2e-${prefix}-${Date.now()}-${Math.random().toString(36).slice(2)}`
}
function uniqueEmail(prefix = 'cs') {
  return `${uid(prefix)}@test.invalid`
}
function authHeaders(token: string) {
  return { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' }
}

// ─────────────────────────────────────────────────────────────────────────────
// Auth guards
// ─────────────────────────────────────────────────────────────────────────────

test('ClassroomSignals: POST hall-passes unauthenticated returns 401', async () => {
  const res = await fetch(
    `${API_BASE}/api/v1/sections/00000000-0000-0000-0000-000000000001/hall-passes`,
    { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: '{}' },
  )
  expect(res.status).toBe(401)
})

test('ClassroomSignals: GET active hall-passes unauthenticated returns 401', async () => {
  const res = await fetch(
    `${API_BASE}/api/v1/sections/00000000-0000-0000-0000-000000000001/hall-passes/active`,
  )
  expect(res.status).toBe(401)
})

test('ClassroomSignals: PATCH hall-pass unauthenticated returns 401', async () => {
  const res = await fetch(
    `${API_BASE}/api/v1/hall-passes/00000000-0000-0000-0000-000000000001`,
    { method: 'PATCH', headers: { 'Content-Type': 'application/json' }, body: '{}' },
  )
  expect(res.status).toBe(401)
})

test('ClassroomSignals: POST course question unauthenticated returns 401', async () => {
  const res = await fetch(
    `${API_BASE}/api/v1/courses/00000000-0000-0000-0000-000000000001/questions`,
    { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: '{}' },
  )
  expect(res.status).toBe(401)
})

test('ClassroomSignals: GET course questions unauthenticated returns 401', async () => {
  const res = await fetch(
    `${API_BASE}/api/v1/courses/00000000-0000-0000-0000-000000000001/questions`,
  )
  expect(res.status).toBe(401)
})

// ─────────────────────────────────────────────────────────────────────────────
// Input validation
// ─────────────────────────────────────────────────────────────────────────────

test('ClassroomSignals: invalid section UUID returns 400', async () => {
  const { access_token: token } = await apiSignup({
    email: uniqueEmail('valid'),
    password: PASSWORD,
  })
  const res = await fetch(`${API_BASE}/api/v1/sections/not-a-uuid/hall-passes`, {
    method: 'POST',
    headers: authHeaders(token),
    body: JSON.stringify({ destination: 'bathroom' }),
  })
  expect(res.status).toBe(400)
})

test('ClassroomSignals: invalid destination returns 400', async () => {
  const { access_token: token } = await apiSignup({
    email: uniqueEmail('dest'),
    password: PASSWORD,
  })
  const res = await fetch(
    `${API_BASE}/api/v1/sections/00000000-0000-0000-0000-000000000099/hall-passes`,
    {
      method: 'POST',
      headers: authHeaders(token),
      body: JSON.stringify({ destination: 'playground' }),
    },
  )
  expect(res.status).toBe(400)
})

test('ClassroomSignals: empty question returns 400', async () => {
  const { access_token: token } = await apiSignup({
    email: uniqueEmail('emptyq'),
    password: PASSWORD,
  })
  const res = await fetch(
    `${API_BASE}/api/v1/courses/00000000-0000-0000-0000-000000000099/questions`,
    {
      method: 'POST',
      headers: authHeaders(token),
      body: JSON.stringify({ question: '   ' }),
    },
  )
  expect(res.status).toBe(400)
})

// ─────────────────────────────────────────────────────────────────────────────
// Full hall-pass flow (request → approve → return)
// ─────────────────────────────────────────────────────────────────────────────

interface SeededSection {
  teacherToken: string
  studentToken: string
  studentEmail: string
  studentId: string
  courseCode: string
  courseId: string
  sectionId: string
}

async function getUserId(token: string): Promise<string> {
  const res = await fetch(`${API_BASE}/api/v1/me`, {
    headers: { Authorization: `Bearer ${token}` },
  })
  const data = (await res.json()) as { id: string }
  return data.id
}

async function seedTeacherWithSection(): Promise<SeededSection | null> {
  const teacherEmail = uniqueEmail('teach')
  const { access_token: teacherToken } = await apiSignup({
    email: teacherEmail,
    password: PASSWORD,
  })
  const studentEmail = uniqueEmail('stud')
  const { access_token: studentToken } = await apiSignup({
    email: studentEmail,
    password: PASSWORD,
  })

  const course = await apiCreateCourse(teacherToken, { title: `CS-${uid()}` })
  const courseCode = course.courseCode
  const courseId = course.id

  // Enable sections for the course.
  await fetch(`${API_BASE}/api/v1/courses/${courseCode}/features`, {
    method: 'PATCH',
    headers: authHeaders(teacherToken),
    body: JSON.stringify({ sectionsEnabled: true }),
  })

  await apiEnroll(teacherToken, courseCode, studentEmail, 'student', studentToken)

  const teacherId = await getUserId(teacherToken)
  const sectionRes = await fetch(`${API_BASE}/api/v1/courses/${courseCode}/sections`, {
    method: 'POST',
    headers: authHeaders(teacherToken),
    body: JSON.stringify({ sectionCode: `S-${uid('sec').slice(-6)}`, instructorUserId: teacherId }),
  })
  if (!sectionRes.ok) return null
  const section = (await sectionRes.json()) as { id?: string }
  if (!section.id) return null

  // Move student into section.
  await apiEnroll(teacherToken, courseCode, studentEmail, 'student', {
    memberToken: studentToken,
    sectionId: section.id,
  })

  const studentId = await getUserId(studentToken)

  return {
    teacherToken,
    studentToken,
    studentEmail,
    studentId,
    courseCode,
    courseId,
    sectionId: section.id,
  }
}

interface HallPass {
  id: string
  studentId?: string
  sectionId: string
  destination: string
  estimatedMins: number | null
  status: string
  requestedAt: string
  approvedAt: string | null
  returnedAt: string | null
  approvedBy: string | null
  overdue: boolean
}

test('ClassroomSignals: full hall pass lifecycle (request → approve → return)', async () => {
  const seed = await seedTeacherWithSection()
  if (!seed) {
    test.skip(true, 'could not seed teacher/student/section')
    return
  }

  // Student requests pass.
  const reqRes = await fetch(`${API_BASE}/api/v1/sections/${seed.sectionId}/hall-passes`, {
    method: 'POST',
    headers: authHeaders(seed.studentToken),
    body: JSON.stringify({ destination: 'bathroom', estimatedMins: 5 }),
  })
  expect(reqRes.status).toBe(201)
  const { pass } = (await reqRes.json()) as { pass: HallPass }
  expect(pass.status).toBe('requested')
  expect(pass.destination).toBe('bathroom')
  expect(pass.estimatedMins).toBe(5)
  expect(pass.studentId).toBe(seed.studentId)

  // Teacher sees it in active list.
  const activeRes = await fetch(
    `${API_BASE}/api/v1/sections/${seed.sectionId}/hall-passes/active`,
    { headers: authHeaders(seed.teacherToken) },
  )
  expect(activeRes.status).toBe(200)
  const { passes } = (await activeRes.json()) as { passes: HallPass[] }
  const found = passes.find((p) => p.id === pass.id)
  expect(found).toBeTruthy()

  // Teacher approves.
  const approveRes = await fetch(`${API_BASE}/api/v1/hall-passes/${pass.id}`, {
    method: 'PATCH',
    headers: authHeaders(seed.teacherToken),
    body: JSON.stringify({ status: 'approved' }),
  })
  expect(approveRes.status).toBe(200)
  const approved = (await approveRes.json()) as { pass: HallPass }
  expect(approved.pass.status).toBe('approved')
  expect(approved.pass.approvedAt).not.toBeNull()

  // Student marks themselves returned.
  const returnRes = await fetch(`${API_BASE}/api/v1/hall-passes/${pass.id}`, {
    method: 'PATCH',
    headers: authHeaders(seed.studentToken),
    body: JSON.stringify({ status: 'returned' }),
  })
  expect(returnRes.status).toBe(200)
  const returned = (await returnRes.json()) as { pass: HallPass }
  expect(returned.pass.status).toBe('returned')
  expect(returned.pass.returnedAt).not.toBeNull()

  // No longer in active list.
  const activeAfterRes = await fetch(
    `${API_BASE}/api/v1/sections/${seed.sectionId}/hall-passes/active`,
    { headers: authHeaders(seed.teacherToken) },
  )
  const { passes: afterPasses } = (await activeAfterRes.json()) as { passes: HallPass[] }
  expect(afterPasses.find((p) => p.id === pass.id)).toBeFalsy()
})

test('ClassroomSignals: pass cannot be re-approved (invalid transition → 409)', async () => {
  const seed = await seedTeacherWithSection()
  if (!seed) {
    test.skip(true, 'could not seed')
    return
  }
  const reqRes = await fetch(`${API_BASE}/api/v1/sections/${seed.sectionId}/hall-passes`, {
    method: 'POST',
    headers: authHeaders(seed.studentToken),
    body: JSON.stringify({ destination: 'office' }),
  })
  expect(reqRes.status).toBe(201)
  const { pass } = (await reqRes.json()) as { pass: HallPass }

  // Approve once.
  const ok1 = await fetch(`${API_BASE}/api/v1/hall-passes/${pass.id}`, {
    method: 'PATCH',
    headers: authHeaders(seed.teacherToken),
    body: JSON.stringify({ status: 'approved' }),
  })
  expect(ok1.status).toBe(200)

  // Re-approve → 409.
  const ok2 = await fetch(`${API_BASE}/api/v1/hall-passes/${pass.id}`, {
    method: 'PATCH',
    headers: authHeaders(seed.teacherToken),
    body: JSON.stringify({ status: 'approved' }),
  })
  expect(ok2.status).toBe(409)
})

// ─────────────────────────────────────────────────────────────────────────────
// Anonymous question queue (AC-3: name hidden from peers)
// ─────────────────────────────────────────────────────────────────────────────

interface AnonQuestion {
  id: string
  courseId: string
  question: string
  addressed: boolean
  createdAt: string
  authorId?: string
}

test('ClassroomSignals: student submission response strips authorId (AC-3)', async () => {
  const seed = await seedTeacherWithSection()
  if (!seed) {
    test.skip(true, 'could not seed')
    return
  }
  const res = await fetch(`${API_BASE}/api/v1/courses/${seed.courseId}/questions`, {
    method: 'POST',
    headers: authHeaders(seed.studentToken),
    body: JSON.stringify({ question: 'Can you re-explain question 3?' }),
  })
  expect(res.status).toBe(201)
  const data = (await res.json()) as { question: AnonQuestion }
  expect(data.question.question).toBe('Can you re-explain question 3?')
  expect(data.question.authorId).toBeUndefined()
})

test('ClassroomSignals: teacher sees questions including authorId (moderation)', async () => {
  const seed = await seedTeacherWithSection()
  if (!seed) {
    test.skip(true, 'could not seed')
    return
  }
  await fetch(`${API_BASE}/api/v1/courses/${seed.courseId}/questions`, {
    method: 'POST',
    headers: authHeaders(seed.studentToken),
    body: JSON.stringify({ question: 'Mod-test question' }),
  })
  const listRes = await fetch(`${API_BASE}/api/v1/courses/${seed.courseId}/questions`, {
    headers: authHeaders(seed.teacherToken),
  })
  expect(listRes.status).toBe(200)
  const data = (await listRes.json()) as { questions: AnonQuestion[] }
  const found = data.questions.find((q) => q.question === 'Mod-test question')
  expect(found).toBeTruthy()
  expect(found?.authorId).toBe(seed.studentId)
})

test('ClassroomSignals: non-staff cannot read course question queue (403)', async () => {
  const seed = await seedTeacherWithSection()
  if (!seed) {
    test.skip(true, 'could not seed')
    return
  }
  const res = await fetch(`${API_BASE}/api/v1/courses/${seed.courseId}/questions`, {
    headers: authHeaders(seed.studentToken),
  })
  expect(res.status).toBe(403)
})
