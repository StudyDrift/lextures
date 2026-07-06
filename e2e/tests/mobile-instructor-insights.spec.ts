/**
 * M11.3 — Instructor insights & at-risk (mobile API parity)
 *
 *   [x] GET analytics/insights: unauthenticated returns 401
 *   [x] GET analytics/insights: student returns 403
 *   [x] GET enrollments/{id}/progress: student cannot read another student's progress
 */
import { test, expect } from '../fixtures/test.js'
import { isAtRiskEnabled } from '../fixtures/platform-features.js'

const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'

test('GET analytics/insights: unauthenticated returns 401', async () => {
  const res = await fetch(`${API_BASE}/api/v1/courses/demo/analytics/insights`)
  expect(res.status).toBe(401)
})

test('GET analytics/insights: student returns 403', async ({ seededCourse }) => {
  if (!(await isAtRiskEnabled())) {
    test.skip(true, 'requires instructor insights enabled')
  }
  const res = await fetch(
    `${API_BASE}/api/v1/courses/${encodeURIComponent(seededCourse.courseCode)}/analytics/insights`,
    { headers: { Authorization: `Bearer ${seededCourse.studentToken}` } },
  )
  expect(res.status).toBe(403)
})

test('GET student progress: student cannot view peer progress', async ({ seededCourse }) => {
  const rosterRes = await fetch(
    `${API_BASE}/api/v1/courses/${encodeURIComponent(seededCourse.courseCode)}/enrollments`,
    { headers: { Authorization: `Bearer ${seededCourse.instructorToken}` } },
  )
  expect(rosterRes.ok).toBeTruthy()
  const roster = (await rosterRes.json()) as { enrollments?: Array<{ id: string; role: string }> }
  const studentEnrollment = (roster.enrollments ?? []).find((e) => e.role === 'student')
  if (!studentEnrollment) {
    test.skip(true, 'no student enrollment in seeded course')
  }

  const res = await fetch(
    `${API_BASE}/api/v1/courses/${encodeURIComponent(seededCourse.courseCode)}/enrollments/${encodeURIComponent(studentEnrollment!.id)}/progress`,
    { headers: { Authorization: `Bearer ${seededCourse.studentToken}` } },
  )
  expect([403, 404]).toContain(res.status)
})
