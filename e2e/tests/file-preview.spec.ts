/**
 * Image & PDF Previewing (plan 8.7) — End-to-end test suite
 *
 * Checklist coverage:
 *   [x] Unauthenticated GET course-file content returns 401
 *   [x] GET course-file content with invalid UUID returns 400 (requires course enrollment)
 *   [x] GET course-file content for non-existent file returns 404
 *   [x] User not enrolled in a course gets 404 (server hides existence)
 *   [x] POST course-files: unauthenticated returns 401
 *   [x] POST course-files: student without item:create permission returns 403
 *   [x] DELETE course-files: unauthenticated returns 401
 *   [x] DELETE course-files: invalid UUID returns 400 (requires course enrollment)
 *   [x] DELETE course-files: non-existent file returns 404 (after course access check)
 *   [x] OPTIONS preflight returns 204 on content endpoint
 *
 * Note: requireCourseAccess returns 404 (not 403) for both non-existent courses
 * and non-enrolled users — this is intentional to avoid disclosing course existence.
 * UUID validation only runs after the access check, so a course-enrolled user is
 * needed to reach the 400 path.
 */

import { test, expect } from '@playwright/test'
import { apiSignup, apiCreateCourse, apiEnroll } from '../fixtures/api.js'

const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'
const PASSWORD = 'E2eTestPass1!'

let _seq = 0
function uniqueEmail(prefix = 'fp') {
  return `e2e-${prefix}-${Date.now()}-${++_seq}@test.invalid`
}

async function getToken(prefix = 'fp'): Promise<string> {
  const { access_token } = await apiSignup({ email: uniqueEmail(prefix), password: PASSWORD })
  return access_token
}

async function authedGet(url: string, token: string) {
  return fetch(url, { headers: { Authorization: `Bearer ${token}` } })
}

const FAKE_FILE_ID = '00000000-0000-0000-0000-000000000099'
const BAD_UUID = 'not-a-uuid'

// ── Auth guard tests ──────────────────────────────────────────────────────────

test('GET course-file content: unauthenticated returns 401', async () => {
  const res = await fetch(
    `${API_BASE}/api/v1/courses/C-FAKE/course-files/${FAKE_FILE_ID}/content`,
  )
  expect(res.status).toBe(401)
})

test('POST course-files: unauthenticated returns 401', async () => {
  const res = await fetch(`${API_BASE}/api/v1/courses/C-FAKE/course-files`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/octet-stream' },
    body: new Uint8Array([1, 2, 3]),
  })
  expect(res.status).toBe(401)
})

test('DELETE course-files: unauthenticated returns 401', async () => {
  const res = await fetch(
    `${API_BASE}/api/v1/courses/C-FAKE/course-files/${FAKE_FILE_ID}`,
    { method: 'DELETE' },
  )
  expect(res.status).toBe(401)
})

// ── Input validation ──────────────────────────────────────────────────────────
// UUID validation runs AFTER requireCourseAccess, so the user must be enrolled
// to reach the parse error. An unenrolled user gets 404 from the access check.

test('GET course-file content: invalid UUID returns 400 for enrolled user', async () => {
  const instructorEmail = uniqueEmail('fp-bad-inst')
  const { access_token: token } = await apiSignup({ email: instructorEmail, password: PASSWORD })
  const course = await apiCreateCourse(token, { title: 'FP Invalid UUID Course' })
  await apiEnroll(token, course.courseCode, instructorEmail, 'teacher')

  const res = await authedGet(
    `${API_BASE}/api/v1/courses/${course.courseCode}/course-files/${BAD_UUID}/content`,
    token,
  )
  expect(res.status).toBe(400)
})

test('DELETE course-files: invalid UUID returns 400 for enrolled user', async () => {
  const instructorEmail = uniqueEmail('fp-del-bad-inst')
  const { access_token: token } = await apiSignup({ email: instructorEmail, password: PASSWORD })
  const course = await apiCreateCourse(token, { title: 'FP Delete Invalid UUID Course' })
  await apiEnroll(token, course.courseCode, instructorEmail, 'teacher')

  const res = await fetch(
    `${API_BASE}/api/v1/courses/${course.courseCode}/course-files/${BAD_UUID}`,
    { method: 'DELETE', headers: { Authorization: `Bearer ${token}` } },
  )
  expect(res.status).toBe(400)
})

// ── OPTIONS preflight ─────────────────────────────────────────────────────────

test('OPTIONS on course-file content returns 204', async () => {
  const res = await fetch(
    `${API_BASE}/api/v1/courses/C-FAKE/course-files/${FAKE_FILE_ID}/content`,
    { method: 'OPTIONS' },
  )
  expect(res.status).toBe(204)
})

// ── Course access & file not found ────────────────────────────────────────────

test('GET course-file content: non-existent course returns 404', async () => {
  const token = await getToken('fp-nocourse')
  // Server returns 404 for both non-existent courses and non-enrolled users
  // to avoid disclosing whether a course exists.
  const res = await authedGet(
    `${API_BASE}/api/v1/courses/C-DOESNOTEXIST/course-files/${FAKE_FILE_ID}/content`,
    token,
  )
  expect(res.status).toBe(404)
})

test('GET course-file content: enrolled user gets 404 for unknown file', async () => {
  // Create instructor + course
  const instructorEmail = uniqueEmail('fp-inst')
  const { access_token: instructorToken } = await apiSignup({
    email: instructorEmail,
    password: PASSWORD,
  })
  const course = await apiCreateCourse(instructorToken, { title: 'FP Test Course' })
  await apiEnroll(instructorToken, course.courseCode, instructorEmail, 'teacher')

  // Instructor can access the course but the file doesn't exist → 404
  const res = await authedGet(
    `${API_BASE}/api/v1/courses/${course.courseCode}/course-files/${FAKE_FILE_ID}/content`,
    instructorToken,
  )
  expect(res.status).toBe(404)
})

test('GET course-file content: non-enrolled user returns 404', async () => {
  // Create course owned by instructor
  const instructorEmail = uniqueEmail('fp-inst2')
  const { access_token: instructorToken } = await apiSignup({
    email: instructorEmail,
    password: PASSWORD,
  })
  const course = await apiCreateCourse(instructorToken, { title: 'FP Private Course' })
  await apiEnroll(instructorToken, course.courseCode, instructorEmail, 'teacher')

  // Outsider: server returns 404 to avoid revealing course existence
  const outsiderToken = await getToken('fp-outsider')
  const res = await authedGet(
    `${API_BASE}/api/v1/courses/${course.courseCode}/course-files/${FAKE_FILE_ID}/content`,
    outsiderToken,
  )
  expect(res.status).toBe(404)
})

test('POST course-files: student (no item:create) returns 403', async () => {
  const instructorEmail = uniqueEmail('fp-inst3')
  const { access_token: instructorToken } = await apiSignup({
    email: instructorEmail,
    password: PASSWORD,
  })
  const studentEmail = uniqueEmail('fp-stu')
  const { access_token: studentToken } = await apiSignup({
    email: studentEmail,
    password: PASSWORD,
  })
  const course = await apiCreateCourse(instructorToken, { title: 'FP Upload Course' })
  await apiEnroll(instructorToken, course.courseCode, instructorEmail, 'teacher')
  await apiEnroll(instructorToken, course.courseCode, studentEmail, 'student')

  const res = await fetch(
    `${API_BASE}/api/v1/courses/${course.courseCode}/course-files`,
    {
      method: 'POST',
      headers: {
        Authorization: `Bearer ${studentToken}`,
        'Content-Type': 'text/plain',
      },
      body: 'hello',
    },
  )
  expect(res.status).toBe(403)
})

test('DELETE course-files: enrolled user gets 404 for unknown file', async () => {
  const instructorEmail = uniqueEmail('fp-inst4')
  const { access_token: instructorToken } = await apiSignup({
    email: instructorEmail,
    password: PASSWORD,
  })
  const course = await apiCreateCourse(instructorToken, { title: 'FP Delete Course' })
  await apiEnroll(instructorToken, course.courseCode, instructorEmail, 'teacher')

  const res = await fetch(
    `${API_BASE}/api/v1/courses/${course.courseCode}/course-files/${FAKE_FILE_ID}`,
    {
      method: 'DELETE',
      headers: { Authorization: `Bearer ${instructorToken}` },
    },
  )
  expect(res.status).toBe(404)
})
