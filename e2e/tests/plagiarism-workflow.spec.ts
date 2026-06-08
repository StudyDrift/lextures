/**
 * Plagiarism workflow (plan 14.8)
 *
 *   [x] GET originality unauthenticated returns 401
 *   [x] Endpoints return 501 when feature disabled
 *   [x] POST retry returns 501 when feature disabled
 */
import { test, expect } from '../fixtures/test.js'

const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'

test.describe('Plagiarism workflow API', () => {
  test('GET originality unauthenticated returns 401', async () => {
    const res = await fetch(
      `${API_BASE}/api/v1/courses/E2E-TEST/assignments/00000000-0000-0000-0000-000000000001/submissions/00000000-0000-0000-0000-000000000002/originality`,
    )
    expect(res.status).toBe(401)
  })

  test('GET originality returns 501 when feature disabled', async ({ seededCourse }) => {
    const res = await fetch(
      `${API_BASE}/api/v1/courses/${encodeURIComponent(seededCourse.courseCode)}/assignments/00000000-0000-0000-0000-000000000001/submissions/00000000-0000-0000-0000-000000000002/originality`,
      { headers: { Authorization: `Bearer ${seededCourse.instructorToken}` } },
    )
    expect(res.status).toBe(501)
  })

  test('POST originality retry returns 501 when feature disabled', async ({ seededCourse }) => {
    const res = await fetch(
      `${API_BASE}/api/v1/courses/${encodeURIComponent(seededCourse.courseCode)}/assignments/00000000-0000-0000-0000-000000000001/submissions/00000000-0000-0000-0000-000000000002/originality/retry`,
      {
        method: 'POST',
        headers: { Authorization: `Bearer ${seededCourse.instructorToken}` },
      },
    )
    expect(res.status).toBe(501)
  })
})
