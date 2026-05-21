/**
 * Interactive H5P content (plan 8.12) — End-to-end test suite
 */
import { test, expect } from '@playwright/test'
import { apiSignup, apiCreateCourse, apiEnroll } from '../fixtures/api.js'

const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'
const PASSWORD = 'E2eTestPass1!'
const H5P_ENABLED = process.env.FEATURE_H5P === 'true'

function uniqueEmail(prefix = 'h5p') {
  return `e2e-${prefix}-${Date.now()}-${Math.random().toString(36).slice(2)}@test.invalid`
}

test('GET h5p package: feature off returns 404', async () => {
  test.skip(H5P_ENABLED, 'skipped when FEATURE_H5P=true')
  const { access_token } = await apiSignup({ email: uniqueEmail(), password: PASSWORD })
  const fakeId = '00000000-0000-0000-0000-000000000099'
  const res = await fetch(`${API_BASE}/api/v1/courses/C-FAKE/h5p/${fakeId}`, {
    headers: { Authorization: `Bearer ${access_token}` },
  })
  expect(res.status).toBe(404)
})

test('POST xapi/statements: unauthenticated returns 401', async () => {
  const res = await fetch(`${API_BASE}/api/v1/xapi/statements`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      courseCode: 'C-FAKE',
      packageId: '00000000-0000-0000-0000-000000000099',
      statement: { verb: { id: 'http://adlnet.gov/expapi/verbs/completed' } },
    }),
  })
  expect(res.status).toBe(401)
})

test('POST xapi/statements: feature off returns 404', async () => {
  test.skip(H5P_ENABLED, 'skipped when FEATURE_H5P=true')
  const { access_token } = await apiSignup({ email: uniqueEmail(), password: PASSWORD })
  const res = await fetch(`${API_BASE}/api/v1/xapi/statements`, {
    method: 'POST',
    headers: {
      Authorization: `Bearer ${access_token}`,
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      courseCode: 'C-FAKE',
      packageId: '00000000-0000-0000-0000-000000000099',
      statement: { verb: { id: 'http://adlnet.gov/expapi/verbs/completed' } },
    }),
  })
  expect(res.status).toBe(404)
})

test('GET h5p completions: student without gradebook permission returns 403', async () => {
  test.skip(!H5P_ENABLED, 'requires FEATURE_H5P=true')
  const instructorEmail = uniqueEmail('inst')
  const studentEmail = uniqueEmail('stu')
  const { access_token: instToken } = await apiSignup({ email: instructorEmail, password: PASSWORD })
  const { access_token: stuToken } = await apiSignup({ email: studentEmail, password: PASSWORD })
  const course = await apiCreateCourse(instToken, { title: 'H5P Gradebook Course' })
  await apiEnroll(instToken, course.courseCode, instructorEmail, 'teacher')
  await apiEnroll(instToken, course.courseCode, studentEmail, 'student')
  const fakeId = '00000000-0000-0000-0000-000000000099'
  const res = await fetch(
    `${API_BASE}/api/v1/courses/${course.courseCode}/h5p/${fakeId}/completions`,
    { headers: { Authorization: `Bearer ${stuToken}` } },
  )
  expect(res.status).toBe(403)
})

test('POST module h5p upload: unauthenticated returns 401', async () => {
  const res = await fetch(
    `${API_BASE}/api/v1/courses/C-FAKE/structure/modules/00000000-0000-0000-0000-000000000001/h5p`,
    { method: 'POST' },
  )
  expect(res.status).toBe(401)
})

test('OPTIONS h5p render returns 204', async () => {
  test.skip(!H5P_ENABLED, 'requires FEATURE_H5P=true')
  const res = await fetch(
    `${API_BASE}/api/v1/courses/C-FAKE/h5p/00000000-0000-0000-0000-000000000099/render`,
    { method: 'OPTIONS' },
  )
  expect(res.status).toBe(204)
})
