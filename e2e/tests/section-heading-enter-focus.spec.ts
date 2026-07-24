/**
 * Content editor: Enter in the section title focuses the section body.
 */
import { test, expect } from '../fixtures/test.js'
import { apiCreateContentPage } from '../fixtures/api.js'

test.describe('Section heading Enter focuses content', () => {
  test('pressing Enter in the section title moves the caret into the body', async ({
    coursePage: page,
    seededCourse,
  }) => {
    test.setTimeout(60_000)

    const contentPage = await apiCreateContentPage(
      seededCourse.instructorToken,
      seededCourse.courseCode,
      seededCourse.moduleId,
      'Heading enter focus page',
    )

    await page.goto(
      `/courses/${seededCourse.courseCode}/modules/content/${contentPage.id}`,
    )
    const editBtn = page.getByRole('button', { name: /^edit$/i })
    await expect(editBtn).toBeVisible({ timeout: 15000 })
    await editBtn.click()

    const heading = page.locator('[id^="canvas-heading-"]').first()
    await expect(heading).toBeVisible({ timeout: 8000 })
    await heading.click()
    await heading.fill('My section title')
    await expect(heading).toBeFocused()

    await heading.press('Enter')

    const body = page.locator('[id^="canvas-md-"] [contenteditable="true"]').first()
    await expect(body).toBeFocused({ timeout: 5000 })
  })
})
