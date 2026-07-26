/**
 * CT.1 — Content Tools foundations: course flag, settings, catalog allowlist.
 *
 * Checklist:
 *   [x] Instructor enables contentToolsEnabled via Features UI (persists)
 *   [x] Catalog returns tools (at least noop_probe) when flag on
 *   [x] Allowlist ["noop_probe"] narrows catalog to exactly that tool
 *   [x] Catalog returns 404 when flag off
 */
import { test, expect, injectToken } from '../fixtures/test.js'
import { apiGetCourse, apiPatchCourseFeatures } from '../fixtures/api.js'
import {
  featureToggleRow,
  openCourseFeaturesSettings,
  withCourseFeatureRestore,
} from '../lib/course-feature-matrix-helpers.js'

const apiBase = process.env.E2E_API_URL ?? 'http://localhost:8080'

async function fetchCatalog(
  token: string,
  courseCode: string,
): Promise<{ status: number; json: { tools?: Array<{ id: string }> } | null }> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/content-tools/catalog`,
    { headers: { Authorization: `Bearer ${token}` } },
  )
  const text = await res.text()
  let json: { tools?: Array<{ id: string }> } | null = null
  try {
    json = text ? (JSON.parse(text) as { tools?: Array<{ id: string }> }) : null
  } catch {
    json = null
  }
  return { status: res.status, json }
}

async function putSettings(
  token: string,
  courseCode: string,
  body: {
    allowedToolIds: string[]
    studentResetAllowed: boolean
    maxInstancesPerItem: number
  },
): Promise<{ status: number; json: Record<string, unknown> | null }> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/content-tools/settings`,
    {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify(body),
    },
  )
  const text = await res.text()
  let json: Record<string, unknown> | null = null
  try {
    json = text ? (JSON.parse(text) as Record<string, unknown>) : null
  } catch {
    json = null
  }
  return { status: res.status, json }
}

test.describe('Content Tools foundations (CT.1)', () => {
  test('UI: enable Content Tools in course settings Features; catalog + allowlist', async ({
    page,
    seededCourse,
  }) => {
    const { instructorToken, courseCode } = seededCourse

    await withCourseFeatureRestore(instructorToken, courseCode, async () => {
      await apiPatchCourseFeatures(instructorToken, courseCode, {
        contentToolsEnabled: false,
      })

      const offCatalog = await fetchCatalog(instructorToken, courseCode)
      expect(offCatalog.status, 'catalog when flag off').toBe(404)

      await injectToken(page, instructorToken)
      await openCourseFeaturesSettings(page, courseCode)

      const row = featureToggleRow(page, 'Content Tools')
      const toggle = row.getByRole('switch')
      await expect(toggle).toHaveAttribute('aria-checked', 'false', { timeout: 10_000 })

      await toggle.click()
      await expect(page.getByRole('status').filter({ hasText: /Saved/i })).toBeVisible({
        timeout: 10_000,
      })
      await expect(toggle).toHaveAttribute('aria-checked', 'true', { timeout: 8_000 })

      await expect
        .poll(
          async () => {
            const data = await apiGetCourse(instructorToken, courseCode)
            return data.contentToolsEnabled === true
          },
          { message: `${courseCode} contentToolsEnabled after UI enable`, timeout: 10_000 },
        )
        .toBe(true)

      await expect(page.getByText(/Available tools/i).first()).toBeVisible({ timeout: 10_000 })

      const onCatalog = await fetchCatalog(instructorToken, courseCode)
      expect(onCatalog.status).toBe(200)
      expect(onCatalog.json?.tools?.length).toBeGreaterThanOrEqual(1)
      expect(onCatalog.json?.tools?.some((t) => t.id === 'noop_probe')).toBeTruthy()

      const put = await putSettings(instructorToken, courseCode, {
        allowedToolIds: ['noop_probe'],
        studentResetAllowed: false,
        maxInstancesPerItem: 50,
      })
      expect(put.status).toBe(200)
      expect(put.json?.allowedToolIds).toEqual(['noop_probe'])

      const filtered = await fetchCatalog(instructorToken, courseCode)
      expect(filtered.status).toBe(200)
      expect(filtered.json?.tools?.map((t) => t.id)).toEqual(['noop_probe'])

      await apiPatchCourseFeatures(instructorToken, courseCode, {
        contentToolsEnabled: false,
      })
      const afterOff = await fetchCatalog(instructorToken, courseCode)
      expect(afterOff.status).toBe(404)
    })
  })

  test('API: catalog 404 when disabled; settings + allowlist when enabled', async ({
    seededCourse,
  }) => {
    const { instructorToken, courseCode } = seededCourse

    await withCourseFeatureRestore(instructorToken, courseCode, async () => {
      await apiPatchCourseFeatures(instructorToken, courseCode, {
        contentToolsEnabled: false,
      })
      expect((await fetchCatalog(instructorToken, courseCode)).status).toBe(404)

      await apiPatchCourseFeatures(instructorToken, courseCode, {
        contentToolsEnabled: true,
      })

      const catalog = await fetchCatalog(instructorToken, courseCode)
      expect(catalog.status).toBe(200)
      expect(catalog.json?.tools?.some((t) => t.id === 'noop_probe')).toBeTruthy()

      const settingsRes = await fetch(
        `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/content-tools/settings`,
        { headers: { Authorization: `Bearer ${instructorToken}` } },
      )
      expect(settingsRes.status).toBe(200)
      const settings = (await settingsRes.json()) as {
        allowedToolIds: string[]
        studentResetAllowed: boolean
        maxInstancesPerItem: number
      }
      expect(Array.isArray(settings.allowedToolIds)).toBeTruthy()
      expect(typeof settings.studentResetAllowed).toBe('boolean')
      expect(typeof settings.maxInstancesPerItem).toBe('number')

      const put = await putSettings(instructorToken, courseCode, {
        allowedToolIds: ['noop_probe'],
        studentResetAllowed: true,
        maxInstancesPerItem: 25,
      })
      expect(put.status).toBe(200)
      expect(put.json?.allowedToolIds).toEqual(['noop_probe'])
      expect(put.json?.studentResetAllowed).toBe(true)
      expect(put.json?.maxInstancesPerItem).toBe(25)

      const filtered = await fetchCatalog(instructorToken, courseCode)
      expect(filtered.json?.tools?.map((t) => t.id)).toEqual(['noop_probe'])
    })
  })
})
