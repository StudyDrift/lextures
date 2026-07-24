/**
 * TipTap markdown editor tables: insert renders as <table>, toolbar + controls.
 */
import { test, expect } from '../fixtures/test.js'
import {
  apiCreateContentPage,
  apiCreateModule,
  apiPatchContentPage,
} from '../fixtures/api.js'

test.describe('Editor tables', () => {
  test('insert table from toolbar renders a real table and supports column controls', async ({
    coursePage: page,
    seededCourse,
  }) => {
    const module = await apiCreateModule(
      seededCourse.instructorToken,
      seededCourse.courseCode,
      'Tables Module',
    )
    const contentPage = await apiCreateContentPage(
      seededCourse.instructorToken,
      seededCourse.courseCode,
      module.id,
      'Table Authoring',
    )

    await page.goto(
      `/courses/${seededCourse.courseCode}/modules/content/${contentPage.id}`,
    )
    const editBtn = page.getByRole('button', { name: /^edit$/i })
    await expect(editBtn).toBeVisible({ timeout: 15000 })
    await editBtn.click()

    const sectionBody = page.locator('[id^="canvas-md-"]').first()
    await sectionBody.click()

    const insertTable = page.getByRole('button', { name: /insert table/i })
    await expect(insertTable).toBeVisible({ timeout: 8000 })
    await insertTable.click()

    const editorTable = sectionBody.locator('table')
    await expect(editorTable).toBeVisible({ timeout: 5000 })
    await expect(editorTable.locator('th')).toHaveCount(3)

    await editorTable.locator('td').first().click()
    const addColumn = page.getByRole('button', { name: /^add column$/i })
    await expect(addColumn).toBeVisible({ timeout: 5000 })
    await addColumn.click()
    await expect(editorTable.locator('th')).toHaveCount(4)

    const widen = page.getByRole('button', { name: /widen column/i })
    await expect(widen).toBeVisible()
    await widen.click()

    await page.getByRole('button', { name: /^save$/i }).click()
    await expect(page.getByText(/saved|updated/i).first()).toBeVisible({ timeout: 12000 })

    await page.reload()
    await expect(page.locator('.syllabus-md table').first()).toBeVisible({ timeout: 10000 })
    await expect(page.locator('.syllabus-md th')).toHaveCount(4)
  })

  test('assignment body with blank-line pipe tables renders as a table for students', async ({
    coursePage: page,
    seededCourse,
  }) => {
    const contentPage = await apiCreateContentPage(
      seededCourse.instructorToken,
      seededCourse.courseCode,
      seededCourse.moduleId,
      'Broken Table Heal',
    )

    await apiPatchContentPage(
      seededCourse.instructorToken,
      seededCourse.courseCode,
      contentPage.id,
      {
        markdown: `Traditional software vs AI.

| Feature | Traditional Software | AI Systems |

|-----------------|---------------------------------|---------------------------------|

| How it works | Follows fixed rules | Learns patterns |

**Quick check:**
Explain the difference.
`,
      },
    )

    await page.goto(
      `/courses/${seededCourse.courseCode}/modules/content/${contentPage.id}`,
    )
    await expect(page.locator('.syllabus-md table').first()).toBeVisible({ timeout: 10000 })
    await expect(page.getByRole('columnheader', { name: /feature/i })).toBeVisible()
    await expect(page.getByText(/\| Feature \|/)).toHaveCount(0)
  })
})
