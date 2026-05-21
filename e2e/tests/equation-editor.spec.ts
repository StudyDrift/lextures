/**
 * MathML / equation editor (plan 8.11)
 */
import { test, expect } from '../fixtures/test.js'
import {
  apiCreateContentPage,
  apiCreateModule,
  apiPatchContentPage,
} from '../fixtures/api.js'

test.describe('Equation editor', () => {
  test('instructor inserts equation on content page and student sees rendered math', async ({
    coursePage: page,
    seededCourse,
  }) => {
    const module = await apiCreateModule(
      seededCourse.instructorToken,
      seededCourse.courseCode,
      'Math Module',
    )
    const contentPage = await apiCreateContentPage(
      seededCourse.instructorToken,
      seededCourse.courseCode,
      module.id,
      'Calculus Intro',
    )

    await page.goto(
      `/courses/${seededCourse.courseCode}/modules/content/${contentPage.id}`,
    )
    await page.getByRole('button', { name: /^edit$/i }).click()

    const sectionBody = page.locator('[id^="canvas-md-"]').first()
    await sectionBody.click()
    await sectionBody.pressSequentially('/equation', { delay: 30 })

    await expect(page.getByRole('dialog', { name: /equation editor/i })).toBeVisible({
      timeout: 8000,
    })

    const latexInput = page.getByLabel(/latex source/i)
    await latexInput.fill('\\frac{d}{dx}\\sin(x) = \\cos(x)')

    await page.getByRole('radio', { name: /display block/i }).check()
    await page.getByRole('button', { name: /^insert$/i }).click()

    await page.getByRole('button', { name: /^save$/i }).click()
    await expect(page.getByText(/saved|updated/i).first()).toBeVisible({ timeout: 12000 })

    await page.reload()
    await expect(page.locator('.katex').first()).toBeVisible({ timeout: 10000 })
    await expect(page.locator('math').first()).toBeAttached({ timeout: 5000 })
  })

  test('saved markdown with display math renders on content page', async ({
    coursePage: page,
    seededCourse,
  }) => {
    const contentPage = await apiCreateContentPage(
      seededCourse.instructorToken,
      seededCourse.courseCode,
      seededCourse.moduleId,
      'Equation View Test',
    )

    await apiPatchContentPage(
      seededCourse.instructorToken,
      seededCourse.courseCode,
      contentPage.id,
      {
        markdown: '$$\n\\frac{a}{b}\n$$\n',
      },
    )

    await page.goto(
      `/courses/${seededCourse.courseCode}/modules/content/${contentPage.id}`,
    )
    await expect(page.locator('.katex').first()).toBeVisible({ timeout: 10000 })
    await expect(page.locator('math').first()).toBeAttached()
  })
})
