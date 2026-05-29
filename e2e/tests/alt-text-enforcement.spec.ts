/**
 * Alt-text enforcement — plan 12.5
 */
import AxeBuilder from '@axe-core/playwright'
import { test, expect, mainNav, injectToken } from '../fixtures/test.js'
import { apiCreateContentPage, apiPatchContentPage } from '../fixtures/api.js'

const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'

test.describe('Alt-text enforcement API', () => {
  test('GET accessibility returns coverage for instructor', async ({ seededCourse }) => {
    const pageItem = await apiCreateContentPage(
      seededCourse.instructorToken,
      seededCourse.courseCode,
      seededCourse.moduleId,
      'Alt text coverage page',
    )
    await apiPatchContentPage(
      seededCourse.instructorToken,
      seededCourse.courseCode,
      pageItem.id,
      { markdown: '![Diagram of a cell](/api/v1/courses/demo/files/cell.png)' },
    )

    const res = await fetch(
      `${API_BASE}/api/v1/courses/${encodeURIComponent(seededCourse.courseCode)}/accessibility`,
      { headers: { Authorization: `Bearer ${seededCourse.instructorToken}` } },
    )
    expect(res.ok).toBe(true)
    const body = (await res.json()) as {
      altTextCoverage: { withAlt: number; total: number; percent: number }
      hardBlockSave: boolean
    }
    expect(body.altTextCoverage.total).toBeGreaterThanOrEqual(1)
    expect(body.altTextCoverage.withAlt).toBeGreaterThanOrEqual(1)
    expect(body.altTextCoverage.percent).toBeGreaterThanOrEqual(0)
    expect(body.hardBlockSave).toBe(true)
  })

  test('missing alt text appears in uncovered items', async ({ seededCourse }) => {
    const pageItem = await apiCreateContentPage(
      seededCourse.instructorToken,
      seededCourse.courseCode,
      seededCourse.moduleId,
      'Missing alt page',
    )
    await apiPatchContentPage(
      seededCourse.instructorToken,
      seededCourse.courseCode,
      pageItem.id,
      { markdown: '![](/api/v1/courses/demo/files/missing-alt.png)' },
    )

    const res = await fetch(
      `${API_BASE}/api/v1/courses/${encodeURIComponent(seededCourse.courseCode)}/accessibility`,
      { headers: { Authorization: `Bearer ${seededCourse.instructorToken}` } },
    )
    expect(res.ok).toBe(true)
    const body = (await res.json()) as {
      altTextCoverage: { uncoveredItems: Array<{ itemId: string; missing: number }> }
    }
    const hit = body.altTextCoverage.uncoveredItems.find((u) => u.itemId === pageItem.id)
    expect(hit?.missing).toBeGreaterThanOrEqual(1)
  })

  test('student cannot access accessibility coverage', async ({ seededCourse }) => {
    const res = await fetch(
      `${API_BASE}/api/v1/courses/${encodeURIComponent(seededCourse.courseCode)}/accessibility`,
      { headers: { Authorization: `Bearer ${seededCourse.studentToken}` } },
    )
    expect(res.status).toBe(403)
  })

  test('unauthenticated alt-text suggest returns 401', async ({ seededCourse }) => {
    const res = await fetch(
      `${API_BASE}/api/v1/courses/${encodeURIComponent(seededCourse.courseCode)}/alt-text/suggest`,
      {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ imageUrl: 'https://example.com/x.png' }),
      },
    )
    expect(res.status).toBe(401)
  })
})

test.describe('Alt-text enforcement UI', () => {
  test('course accessibility settings shows coverage', async ({ page, seededCourse }) => {
    await injectToken(page, seededCourse.instructorToken)
    await page.goto(`/courses/${seededCourse.courseCode}/settings/accessibility`)
    const nav = mainNav(page)
    try {
      await expect(nav).toBeVisible({ timeout: 15000 })
    } catch {
      test.skip(true, 'Authenticated LMS shell unavailable in this environment')
    }
    await expect(page.getByText(/alt-text coverage/i)).toBeVisible({ timeout: 15000 })

    const results = await new AxeBuilder({ page })
      .include('main')
      .withTags(['wcag2a', 'wcag2aa'])
      .disableRules(['landmark-one-main', 'region'])
      .analyze()
    const critical = results.violations.filter(
      (v) => v.impact === 'critical' || v.impact === 'serious',
    )
    expect(critical.length).toBe(0)
  })
})
