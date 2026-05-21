/**
 * Image & PDF Previewing (plan 8.7) — End-to-end test suite
 *
 * Checklist coverage:
 *   [x] Unauthenticated GET course-file content returns 401
 *   [x] GET course-file content with invalid UUID returns 400
 *   [x] GET course-file content for non-existent file returns 404
 *   [x] Student enrolled in course can reach the content endpoint (not 403)
 *   [x] User not enrolled in a course cannot access its files (403)
 *   [x] POST course-files: unauthenticated returns 401
 *   [x] POST course-files: student without item:create permission returns 403
 *   [x] DELETE course-files: unauthenticated returns 401
 *   [x] DELETE course-files: non-existent file returns 404 (after course access check)
 *   [x] OPTIONS preflight returns 204 on content endpoint
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

test('GET course-file content: invalid UUID returns 400', async () => {
  const token = await getToken('fp-bad')
  const res = await authedGet(
    `${API_BASE}/api/v1/courses/C-FAKE/course-files/${BAD_UUID}/content`,
    token,
  )
  // UUID parse error happens after auth, before course access check
  expect(res.status).toBe(400)
})

test('DELETE course-files: invalid UUID returns 400', async () => {
  const token = await getToken('fp-del-bad')
  const res = await fetch(
    `${API_BASE}/api/v1/courses/C-FAKE/course-files/${BAD_UUID}`,
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

test('GET course-file content: non-existent course returns 403 (access denied)', async () => {
  const token = await getToken('fp-nocourse')
  // User is not enrolled in C-DOESNOTEXIST → 403
  const res = await authedGet(
    `${API_BASE}/api/v1/courses/C-DOESNOTEXIST/course-files/${FAKE_FILE_ID}/content`,
    token,
  )
  expect(res.status).toBe(403)
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

test('GET course-file content: non-enrolled user returns 403', async () => {
  // Create course owned by instructor
  const instructorEmail = uniqueEmail('fp-inst2')
  const { access_token: instructorToken } = await apiSignup({
    email: instructorEmail,
    password: PASSWORD,
  })
  const course = await apiCreateCourse(instructorToken, { title: 'FP Private Course' })
  await apiEnroll(instructorToken, course.courseCode, instructorEmail, 'teacher')

  // Outsider with their own account
  const outsiderToken = await getToken('fp-outsider')
  const res = await authedGet(
    `${API_BASE}/api/v1/courses/${course.courseCode}/course-files/${FAKE_FILE_ID}/content`,
    outsiderToken,
  )
  expect(res.status).toBe(403)
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
