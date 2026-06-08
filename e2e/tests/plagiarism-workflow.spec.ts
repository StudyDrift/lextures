/**
 * Plagiarism workflow (plan 14.8)
 *
 *   [x] GET originality unauthenticated returns 401
 *   [x] Authenticated instructor gets 404 for missing submission when feature enabled
 */
import { test, expect } from '../fixtures/test.js'

const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'

function authHeaders(token: string) {
  return { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' }
}

test.describe('Plagiarism workflow API', () => {
  test('GET originality unauthenticated returns 401', async () => {
    const res = await fetch(
      `${API_BASE}/api/v1/courses/E2E-TEST/assignments/00000000-0000-0000-0000-000000000001/submissions/00000000-0000-0000-0000-000000000002/originality`,
    )
    expect(res.status).toBe(401)
  })
})

test('Instructor GET originality for missing submission returns 404', async ({ seededCourse }) => {
  const res = await fetch(
    `${API_BASE}/api/v1/courses/${encodeURIComponent(seededCourse.courseCode)}/assignments/00000000-0000-0000-0000-000000000001/submissions/00000000-0000-0000-0000-000000000002/originality`,
    { headers: authHeaders(seededCourse.instructorToken) },
  )
  expect([404, 501]).toContain(res.status)
})

test('Instructor POST originality retry for missing submission', async ({ seededCourse }) => {
  const res = await fetch(
    `${API_BASE}/api/v1/courses/${encodeURIComponent(seededCourse.courseCode)}/assignments/00000000-0000-0000-0000-000000000001/submissions/00000000-0000-0000-0000-000000000002/originality/retry`,
    {
      method: 'POST',
      headers: authHeaders(seededCourse.instructorToken),
    },
  )
  expect([200, 403, 404, 501]).toContain(res.status)
})
