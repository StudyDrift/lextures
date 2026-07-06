/**
 * M11.3 — Instructor insights & at-risk (mobile API parity)
 *
 *   [x] GET analytics/insights: unauthenticated returns 401
 *   [x] GET analytics/insights: student returns 403
 *   [x] GET enrollments/{id}/progress: student cannot read another student's progress
 */
import { test, expect, uniqueEmail } from '../fixtures/test.js'
import { apiSignup, apiEnroll } from '../fixtures/api.js'
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
  const meRes = await fetch(`${API_BASE}/api/v1/me`, {
    headers: { Authorization: `Bearer ${seededCourse.studentToken}` },
  })
  expect(meRes.ok).toBeTruthy()
  const me = (await meRes.json()) as { id: string }

  const peerEmail = uniqueEmail('peer')
  const { access_token: peerToken } = await apiSignup({
    email: peerEmail,
    password: 'E2eTestPass1!',
  })
  await apiEnroll(
    seededCourse.instructorToken,
    seededCourse.courseCode,
    peerEmail,
    'student',
    peerToken,
  )

  const rosterRes = await fetch(
    `${API_BASE}/api/v1/courses/${encodeURIComponent(seededCourse.courseCode)}/enrollments`,
    { headers: { Authorization: `Bearer ${seededCourse.instructorToken}` } },
  )
  expect(rosterRes.ok).toBeTruthy()
  const roster = (await rosterRes.json()) as {
    enrollments?: Array<{ id: string; role: string; userId: string }>
  }
  const peerEnrollment = (roster.enrollments ?? []).find(
    (e) => e.role === 'student' && e.userId !== me.id,
  )
  if (!peerEnrollment) {
    test.skip(true, 'no peer student enrollment in seeded course')
  }

  const res = await fetch(
    `${API_BASE}/api/v1/courses/${encodeURIComponent(seededCourse.courseCode)}/enrollments/${encodeURIComponent(peerEnrollment!.id)}/progress`,
    { headers: { Authorization: `Bearer ${seededCourse.studentToken}` } },
  )
  expect([403, 404]).toContain(res.status)
})
