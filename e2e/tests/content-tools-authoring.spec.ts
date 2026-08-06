/**
 * CT.2 — Content Tools authoring: Tools dropdown, insert, config, slash.
 *
 * Checklist:
 *   [x] Flag off → no Tools control in content page editor
 *   [x] Flag on → Tools dropdown → insert noop_probe → fence in saved markdown
 *   [x] Config panel save prompt → reload shows card
 *   [x] Slash /noop filters and inserts tool
 */
import { test, expect, injectToken } from '../fixtures/test.js'
import {
  apiCreateContentPage,
  apiCreateModule,
  apiPatchCourseFeatures,
} from '../fixtures/api.js'
import { withCourseFeatureRestore } from '../lib/course-feature-matrix-helpers.js'

const apiBase = process.env.E2E_API_URL ?? 'http://localhost:8080'

async function fetchContentPageMarkdown(
  token: string,
  courseCode: string,
  itemId: string,
): Promise<string> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/content-pages/${encodeURIComponent(itemId)}`,
    { headers: { Authorization: `Bearer ${token}` } },
  )
  if (!res.ok) {
    throw new Error(`Get content page failed (${res.status})`)
  }
  const json = (await res.json()) as { markdown?: string }
  return json.markdown ?? ''
}

async function putSettings(
  token: string,
  courseCode: string,
  body: {
    allowedToolIds: string[]
    studentResetAllowed: boolean
    maxInstancesPerItem: number
  },
): Promise<number> {
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
  return res.status
}

test.describe('Content Tools authoring (CT.2)', () => {
  test('flag off: Tools control absent on content page editor', async ({
    page,
    seededCourse,
  }) => {
    const { instructorToken, courseCode } = seededCourse

    await withCourseFeatureRestore(instructorToken, courseCode, async () => {
      await apiPatchCourseFeatures(instructorToken, courseCode, {
        contentToolsEnabled: false,
      })

      const module = await apiCreateModule(instructorToken, courseCode, 'CT2 Off Module')
      const contentPage = await apiCreateContentPage(
        instructorToken,
        courseCode,
        module.id,
        'CT2 Off Page',
      )

      await injectToken(page, instructorToken)
      await page.goto(`/courses/${courseCode}/modules/content/${contentPage.id}`)
      const editBtn = page.getByRole('button', { name: /^edit$/i })
      await expect(editBtn).toBeVisible({ timeout: 15_000 })
      await editBtn.click()

      const sectionBody = page.locator('[id^="canvas-md-"]').first()
      await sectionBody.click()
      await expect(page.getByRole('button', { name: /^tools$/i })).toHaveCount(0)
    })
  })

  test('flag on: Tools dropdown inserts noop_probe; config save survives reload', async ({
    page,
    seededCourse,
  }) => {
    const { instructorToken, courseCode } = seededCourse

    await withCourseFeatureRestore(instructorToken, courseCode, async () => {
      await apiPatchCourseFeatures(instructorToken, courseCode, {
        contentToolsEnabled: true,
      })
      const settingsStatus = await putSettings(instructorToken, courseCode, {
        allowedToolIds: ['noop_probe'],
        studentResetAllowed: false,
        maxInstancesPerItem: 50,
      })
      expect(settingsStatus).toBe(200)

      const module = await apiCreateModule(instructorToken, courseCode, 'CT2 On Module')
      const contentPage = await apiCreateContentPage(
        instructorToken,
        courseCode,
        module.id,
        'CT2 On Page',
      )

      await injectToken(page, instructorToken)
      await page.goto(`/courses/${courseCode}/modules/content/${contentPage.id}`)
      const editBtn = page.getByRole('button', { name: /^edit$/i })
      await expect(editBtn).toBeVisible({ timeout: 15_000 })
      await editBtn.click()

      const sectionBody = page.locator('[id^="canvas-md-"]').first()
      await sectionBody.click()

      const toolsBtn = page.getByRole('button', { name: /^tools$/i })
      await expect(toolsBtn).toBeVisible({ timeout: 10_000 })
      await toolsBtn.click()

      const probe = page.getByRole('option', { name: /no-op probe|noop/i }).first()
      await expect(probe).toBeVisible({ timeout: 8_000 })
      await probe.click()

      await expect(page.locator('[data-content-tool-instance]').first()).toBeVisible({
        timeout: 12_000,
      })
      await expect(page.locator('[data-content-tool-config]').first()).toBeVisible({
        timeout: 8_000,
      })

      const configPanel = page.locator('[data-content-tool-config]').first()
      const promptField = configPanel.getByLabel(/student-visible prompt|prompt/i).first()
      await expect(promptField).toBeVisible({ timeout: 8_000 })
      await promptField.fill('CT2 authoring prompt')
      await configPanel.getByRole('button', { name: /^save$/i }).click()

      // Page-level Save in the content page header (semantic primary).
      // Do not match generic "Saved" from tool sync chips — wait for PATCH + "Page saved".
      const pageSave = page.waitForResponse(
        (res) =>
          res.request().method() === 'PATCH' &&
          res.url().includes(`/content-pages/${contentPage.id}`) &&
          res.ok(),
        { timeout: 20_000 },
      )
      await page.locator('button.bg-accent-solid').filter({ hasText: /^Save$/ }).click()
      await pageSave
      await expect(page.getByText(/^Page saved$/i).first()).toBeVisible({
        timeout: 12_000,
      })

      await expect
        .poll(
          async () =>
            fetchContentPageMarkdown(instructorToken, courseCode, contentPage.id),
          { timeout: 15_000, message: 'content page markdown includes lex-tool fence' },
        )
        .toMatch(/```lex-tool/)
      const markdown = await fetchContentPageMarkdown(
        instructorToken,
        courseCode,
        contentPage.id,
      )
      expect(markdown).toMatch(/"toolId":"noop_probe"/)

      await page.reload()
      await page.getByRole('button', { name: /^edit$/i }).click()
      await expect(page.locator('[data-content-tool-instance]').first()).toBeVisible({
        timeout: 12_000,
      })
      await expect(page.getByText(/CT2 authoring prompt/i).first()).toBeVisible({
        timeout: 8_000,
      })
    })
  })

  test('slash /noop inserts noop_probe tool', async ({ page, seededCourse }) => {
    const { instructorToken, courseCode } = seededCourse

    await withCourseFeatureRestore(instructorToken, courseCode, async () => {
      await apiPatchCourseFeatures(instructorToken, courseCode, {
        contentToolsEnabled: true,
      })
      await putSettings(instructorToken, courseCode, {
        allowedToolIds: ['noop_probe'],
        studentResetAllowed: false,
        maxInstancesPerItem: 50,
      })

      const module = await apiCreateModule(instructorToken, courseCode, 'CT2 Slash Module')
      const contentPage = await apiCreateContentPage(
        instructorToken,
        courseCode,
        module.id,
        'CT2 Slash Page',
      )

      await injectToken(page, instructorToken)
      await page.goto(`/courses/${courseCode}/modules/content/${contentPage.id}`)
      await page.getByRole('button', { name: /^edit$/i }).click()

      const sectionBody = page.locator('[id^="canvas-md-"]').first()
      await sectionBody.click()
      await sectionBody.pressSequentially('/noop', { delay: 30 })

      const slashOption = page.getByRole('option', { name: /no-op probe|noop/i }).first()
      await expect(slashOption).toBeVisible({ timeout: 8_000 })
      await slashOption.click()

      await expect(page.locator('[data-content-tool-instance]').first()).toBeVisible({
        timeout: 12_000,
      })
    })
  })
})
