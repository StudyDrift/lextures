/**
 * Editor image/file insert modal — select existing course files or upload, then Insert.
 */
import { test, expect } from '../fixtures/test.js'
import {
  apiCreateContentPage,
  apiUploadCourseManagedFile,
} from '../fixtures/api.js'

/** Minimal 1×1 PNG */
const TINY_PNG = Uint8Array.from(
  atob(
    'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg==',
  ),
  (c) => c.charCodeAt(0),
)

test.describe('Editor image insert modal', () => {
  test('toolbar image button opens modal; selecting a course file and Insert embeds it', async ({
    coursePage: page,
    seededCourse,
  }) => {
    test.setTimeout(60_000)

    const uploaded = await apiUploadCourseManagedFile(
      seededCourse.instructorToken,
      seededCourse.courseCode,
      { name: 'e2e-diagram.png', mimeType: 'image/png', bytes: TINY_PNG },
    )

    const contentPage = await apiCreateContentPage(
      seededCourse.instructorToken,
      seededCourse.courseCode,
      seededCourse.moduleId,
      'Image insert modal page',
    )

    await page.goto(
      `/courses/${seededCourse.courseCode}/modules/content/${contentPage.id}`,
    )
    const editBtn = page.getByRole('button', { name: /^edit$/i })
    await expect(editBtn).toBeVisible({ timeout: 15000 })
    await editBtn.click()

    const sectionBody = page.locator('[id^="canvas-md-"]').first()
    await sectionBody.click()

    const imageBtn = page.getByRole('button', { name: /insert image/i }).first()
    await expect(imageBtn).toBeVisible({ timeout: 8000 })
    await imageBtn.click()

    const dialog = page.getByRole('dialog', { name: /insert file or image/i })
    await expect(dialog).toBeVisible({ timeout: 8000 })
    await expect(dialog.getByText(uploaded.displayName || 'e2e-diagram.png')).toBeVisible({
      timeout: 8000,
    })

    await dialog.getByRole('option', { name: /e2e-diagram\.png/i }).click()
    await dialog.getByRole('button', { name: /^insert$/i }).click()
    await expect(dialog).toBeHidden({ timeout: 8000 })

    // TipTap course images may load via authenticated blob URL; wait for the node.
    await expect(sectionBody.locator('img, figure').first()).toBeVisible({ timeout: 12000 })
  })

  test('modal upload zone stages a file and Insert embeds it', async ({
    coursePage: page,
    seededCourse,
  }) => {
    test.setTimeout(60_000)

    const contentPage = await apiCreateContentPage(
      seededCourse.instructorToken,
      seededCourse.courseCode,
      seededCourse.moduleId,
      'Image upload insert page',
    )

    await page.goto(
      `/courses/${seededCourse.courseCode}/modules/content/${contentPage.id}`,
    )
    await page.getByRole('button', { name: /^edit$/i }).click()
    const sectionBody = page.locator('[id^="canvas-md-"]').first()
    await sectionBody.click()

    await page.getByRole('button', { name: /insert image/i }).first().click()
    const dialog = page.getByRole('dialog', { name: /insert file or image/i })
    await expect(dialog).toBeVisible({ timeout: 8000 })

    const fileInput = dialog.locator('input[type="file"]')
    await fileInput.setInputFiles({
      name: 'upload-from-modal.png',
      mimeType: 'image/png',
      buffer: Buffer.from(TINY_PNG),
    })

    await expect(dialog.getByText('upload-from-modal.png')).toBeVisible()
    await dialog.getByRole('button', { name: /^insert$/i }).click()
    await expect(dialog).toBeHidden({ timeout: 10000 })
    await expect(sectionBody.locator('img, figure').first()).toBeVisible({ timeout: 12000 })
  })
})
