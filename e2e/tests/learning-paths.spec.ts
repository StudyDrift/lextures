/**
 * Learning paths / course bundles (plan 15.4).
 */
import { test, expect, injectToken } from '../fixtures/test.js'
import { apiGetCourse } from '../fixtures/api.js'

const apiBase = process.env.E2E_API_URL ?? 'http://localhost:8080'

test.describe('Learning paths — API auth', () => {
  test('GET /api/v1/me/paths returns 401 without auth', async () => {
    const res = await fetch(`${apiBase}/api/v1/me/paths`)
    expect(res.status).toBe(401)
  })

  test('POST /api/v1/creator/learning-paths returns 401 without auth', async () => {
    const res = await fetch(`${apiBase}/api/v1/creator/learning-paths`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ title: 'Unauthorized path' }),
    })
    expect(res.status).toBe(401)
  })
})

test.describe('Learning paths — public catalog', () => {
  test('GET /api/v1/catalog/paths is reachable without auth', async () => {
    const res = await fetch(`${apiBase}/api/v1/catalog/paths`)
    if (res.status === 404) {
      test.skip(true, 'ff_learning_paths not enabled in this environment')
    }
    expect(res.status).toBe(200)
    const body = (await res.json()) as { paths?: unknown[] }
    expect(Array.isArray(body.paths)).toBe(true)
  })
})

test.describe('Learning paths — authenticated API', () => {
  test('student can list their enrolled paths when feature enabled', async ({ seededCourse }) => {
    const res = await fetch(`${apiBase}/api/v1/me/paths`, {
      headers: { Authorization: `Bearer ${seededCourse.studentToken}` },
    })
    if (res.status === 404) {
      test.skip(true, 'ff_learning_paths not enabled in this environment')
    }
    expect(res.status).toBe(200)
    const body = (await res.json()) as { paths: unknown[] }
    expect(Array.isArray(body.paths)).toBe(true)
  })

  test('creator can create, list, and delete a learning path', async ({ seededCourse }) => {
    const featRes = await fetch(`${apiBase}/api/v1/platform/features`, {
      headers: { Authorization: `Bearer ${seededCourse.instructorToken}` },
    })
    if (!featRes.ok) {
      test.skip(true, 'platform features unavailable')
    }
    const feats = (await featRes.json()) as { ffLearningPaths?: boolean }
    if (!feats.ffLearningPaths) {
      test.skip(true, 'ff_learning_paths not enabled in this environment')
    }

    const course = (await apiGetCourse(seededCourse.instructorToken, seededCourse.courseCode)) as {
      id: string
    }

    const createRes = await fetch(`${apiBase}/api/v1/creator/learning-paths`, {
      method: 'POST',
      headers: {
        Authorization: `Bearer ${seededCourse.instructorToken}`,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        title: `E2E Path ${Date.now()}`,
        description: 'Created by e2e test',
        courseIds: [course.id],
      }),
    })
    expect(createRes.status).toBe(201)
    const created = (await createRes.json()) as { id: string }
    expect(created.id).toBeTruthy()

    const listRes = await fetch(`${apiBase}/api/v1/creator/learning-paths`, {
      headers: { Authorization: `Bearer ${seededCourse.instructorToken}` },
    })
    expect(listRes.status).toBe(200)
    const listed = (await listRes.json()) as { paths: { id: string }[] }
    expect(listed.paths.some((p) => p.id === created.id)).toBe(true)

    const deleteRes = await fetch(`${apiBase}/api/v1/creator/learning-paths/${created.id}`, {
      method: 'DELETE',
      headers: { Authorization: `Bearer ${seededCourse.instructorToken}` },
    })
    expect(deleteRes.status).toBe(204)
  })
})

test.describe('Learning paths — UI', () => {
  test('my paths page loads when feature enabled', async ({ page, seededCourse }) => {
    const featRes = await fetch(`${apiBase}/api/v1/platform/features`, {
      headers: { Authorization: `Bearer ${seededCourse.studentToken}` },
    })
    if (!featRes.ok) {
      test.skip(true, 'platform features unavailable')
    }
    const feats = (await featRes.json()) as { ffLearningPaths?: boolean }
    if (!feats.ffLearningPaths) {
      test.skip(true, 'ff_learning_paths not enabled in this environment')
    }

    await injectToken(page, seededCourse.studentToken)
    await page.goto('/my-paths')
    await expect(page.getByRole('heading', { name: /my learning paths/i })).toBeVisible({
      timeout: 10_000,
    })
  })
})
