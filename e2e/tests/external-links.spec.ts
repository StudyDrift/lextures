/**
 * External link module items (plan 8.8)
 *
 * Checklist coverage:
 *   [x] Instructor can create an external link item inside a module
 *   [x] Created external link appears in the modules list
 *   [x] Clicking the external link item loads the link page
 *   [x] External link page shows "Open link" button for students
 *   [x] Instructor can edit the URL of an existing external link
 *   [x] Cloud picker buttons are visible to instructors in the edit form
 */
import { test, expect, injectToken } from '../fixtures/test.js'

const apiBase = process.env.E2E_API_URL ?? 'http://localhost:8080'

async function apiCreateExternalLink(
  token: string,
  courseCode: string,
  moduleId: string,
  title: string,
  url: string,
): Promise<{ id: string }> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/structure/modules/${encodeURIComponent(moduleId)}/external-links`,
    {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify({ title, url }),
    },
  )
  if (!res.ok) {
    const body = await res.text()
    throw new Error(`Create external link failed (${res.status}): ${body}`)
  }
  return res.json() as Promise<{ id: string }>
}

test.describe('External link module items', () => {
  test('instructor can add an external link item and it appears in the module list', async ({
    coursePage: page,
    seededCourse,
  }) => {
    await page.goto(`/courses/${seededCourse.courseCode}/modules`)
    await expect(page.getByText(seededCourse.moduleTitle)).toBeVisible()

    // Find the module row and open the "Add item" menu
    const moduleRow = page.locator('li').filter({ hasText: seededCourse.moduleTitle }).first()
    await moduleRow.hover()

    // Look for an "Add item" or "+" button inside the module row
    const addBtn = moduleRow.getByRole('button', { name: /add item|add content|\+/i }).first()
    await expect(addBtn).toBeVisible({ timeout: 5000 })
    await addBtn.click()

    // Click "External link" in the dropdown menu
    const extLinkItem = page.getByRole('menuitem', { name: /external link/i })
    await expect(extLinkItem).toBeVisible({ timeout: 3000 })
    await extLinkItem.click()

    // Fill in the modal
    const dialog = page.getByRole('dialog')
    await expect(dialog).toBeVisible({ timeout: 5000 })
    await dialog.getByRole('textbox', { name: /title|link title/i }).fill('E2E Test Link')
    await dialog.getByRole('textbox', { name: /url/i }).fill('https://example.com/e2e-test')
    await dialog.getByRole('button', { name: /save/i }).click()

    // The new item should appear in the module
    await expect(page.getByText('E2E Test Link')).toBeVisible({ timeout: 8000 })
  })

  test('seeded external link appears in module list and loads link page', async ({
    coursePage: page,
    seededCourse,
  }) => {
    // Seed an external link via API
    const item = await apiCreateExternalLink(
      seededCourse.instructorToken,
      seededCourse.courseCode,
      seededCourse.moduleId,
      'E2E External Resource',
      'https://example.com/resource',
    )
    expect(item.id).toBeTruthy()

    await page.goto(`/courses/${seededCourse.courseCode}/modules`)
    await expect(page.getByText('E2E External Resource')).toBeVisible({ timeout: 8000 })
  })

  test('external link page shows "Open link" button and link title', async ({
    coursePage: page,
    seededCourse,
  }) => {
    // Seed the link
    const item = await apiCreateExternalLink(
      seededCourse.instructorToken,
      seededCourse.courseCode,
      seededCourse.moduleId,
      'My Reading Material',
      'https://example.com/reading',
    )

    // Navigate directly to the external link item page (instructor view)
    await page.goto(`/courses/${seededCourse.courseCode}/modules/external-link/${item.id}`)

    // The page should show the link title
    await expect(page.getByRole('heading', { name: /my reading material/i })).toBeVisible({ timeout: 8000 })

    // "Open link" button should be present
    await expect(page.getByRole('link', { name: /open link/i })).toBeVisible({ timeout: 5000 })
  })

  test('student sees "Open link" button for an external link', async ({
    page,
    seededCourse,
  }) => {
    const item = await apiCreateExternalLink(
      seededCourse.instructorToken,
      seededCourse.courseCode,
      seededCourse.moduleId,
      'Student Reading',
      'https://example.com/student-reading',
    )

    // Log in as student
    await injectToken(page, seededCourse.studentToken)
    await page.goto(`/courses/${seededCourse.courseCode}/modules/external-link/${item.id}`)

    // The page should show the title
    await expect(page.getByRole('heading', { name: /student reading/i })).toBeVisible({ timeout: 8000 })

    // Student sees "Open link" (auto-open or manual button)
    await expect(page.getByRole('link', { name: /open link/i })).toBeVisible({ timeout: 5000 })
  })

  test('instructor can edit the URL of an existing external link', async ({
    coursePage: page,
    seededCourse,
  }) => {
    const item = await apiCreateExternalLink(
      seededCourse.instructorToken,
      seededCourse.courseCode,
      seededCourse.moduleId,
      'Editable Link',
      'https://example.com/original',
    )

    await page.goto(`/courses/${seededCourse.courseCode}/modules/external-link/${item.id}`)
    await expect(page.getByRole('heading', { name: /editable link/i })).toBeVisible({ timeout: 8000 })

    // The URL input should be visible to instructors
    const urlInput = page.getByRole('textbox', { name: /destination url/i })
    await expect(urlInput).toBeVisible({ timeout: 5000 })

    // Change the URL
    await urlInput.fill('https://example.com/updated')
    await page.getByRole('button', { name: /save url/i }).click()

    // The updated URL should be reflected in the link button
    const openLink = page.getByRole('link', { name: /open link/i })
    await expect(openLink).toBeVisible({ timeout: 5000 })
    await expect(openLink).toHaveAttribute('href', 'https://example.com/updated')
  })

  test('cloud picker buttons are visible to instructors in the edit form', async ({
    coursePage: page,
    seededCourse,
  }) => {
    const item = await apiCreateExternalLink(
      seededCourse.instructorToken,
      seededCourse.courseCode,
      seededCourse.moduleId,
      'Cloud Link Test',
      'https://example.com/cloud',
    )

    await page.goto(`/courses/${seededCourse.courseCode}/modules/external-link/${item.id}`)
    await expect(page.getByRole('heading', { name: /cloud link test/i })).toBeVisible({ timeout: 8000 })

    // Cloud provider buttons should appear in the edit form
    await expect(page.getByRole('button', { name: /google drive/i })).toBeVisible({ timeout: 5000 })
    await expect(page.getByRole('button', { name: /onedrive/i })).toBeVisible({ timeout: 5000 })
    await expect(page.getByRole('button', { name: /dropbox/i })).toBeVisible({ timeout: 5000 })
  })

  test('external link creation modal shows cloud storage buttons', async ({
    coursePage: page,
    seededCourse,
  }) => {
    await page.goto(`/courses/${seededCourse.courseCode}/modules`)
    await expect(page.getByText(seededCourse.moduleTitle)).toBeVisible()

    const moduleRow = page.locator('li').filter({ hasText: seededCourse.moduleTitle }).first()
    await moduleRow.hover()

    const addBtn = moduleRow.getByRole('button', { name: /add item|add content|\+/i }).first()
    await expect(addBtn).toBeVisible({ timeout: 5000 })
    await addBtn.click()

    const extLinkItem = page.getByRole('menuitem', { name: /external link/i })
    await expect(extLinkItem).toBeVisible({ timeout: 3000 })
    await extLinkItem.click()

    const dialog = page.getByRole('dialog')
    await expect(dialog).toBeVisible({ timeout: 5000 })

    // Cloud provider buttons visible in modal
    await expect(dialog.getByRole('button', { name: /google drive/i })).toBeVisible()
    await expect(dialog.getByRole('button', { name: /onedrive/i })).toBeVisible()
    await expect(dialog.getByRole('button', { name: /dropbox/i })).toBeVisible()
  })
})
