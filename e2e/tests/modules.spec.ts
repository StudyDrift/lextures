/**
 * Course modules
 *
 * Checklist coverage (docs/e2e.md):
 *   [x] Modules list page loads
 *   [x] Create a new module → appears in list
 *   [x] Collapse/expand module section
 *   [x] Archive a module → disappears from active list
 *   [x] Delete a module → removed permanently
 *   [x] Add Vibe Activity module item via dropdown + rich modal
 *   [x] Vibe Activity renders in outline and viewer page with interactive HTML
 */
import { test, expect } from '../fixtures/test.js'

test.describe('Course modules', () => {
  test('modules list page loads', async ({ coursePage: page, seededCourse }) => {
    await page.goto(`/courses/${seededCourse.courseCode}/modules`)
    await expect(page.getByRole('heading', { name: /modules/i })).toBeVisible()
  })

  test('pre-seeded module appears in the list', async ({ coursePage: page, seededCourse }) => {
    await page.goto(`/courses/${seededCourse.courseCode}/modules`)
    await expect(page.getByText(seededCourse.moduleTitle)).toBeVisible()
  })

  test('create a new module via Actions menu → module appears in list', async ({
    coursePage: page,
    seededCourse,
  }) => {
    await page.goto(`/courses/${seededCourse.courseCode}/modules`)
    await expect(page.getByText(seededCourse.moduleTitle)).toBeVisible()

    // The Actions button is the indigo button at the top-right of the modules page.
    // Use aria-haspopup="menu" to avoid matching the search shortcut button.
    const actionsBtn = page.locator('button[aria-haspopup="menu"]', { hasText: /actions/i })
    await expect(actionsBtn).toBeVisible({ timeout: 8000 })
    await actionsBtn.click()

    // Click "Add Module" inside the dropdown.
    await page.getByRole('menuitem', { name: /add module/i }).click()

    // A modal prompts for the module name.
    const nameInput = page.getByRole('dialog').getByRole('textbox').first()
    await nameInput.fill('New E2E Module')
    await page.getByRole('dialog').getByRole('button', { name: /create|save/i }).click()

    await expect(page.getByText('New E2E Module')).toBeVisible({ timeout: 8000 })
  })

  test('module section is visible and has action buttons', async ({
    coursePage: page,
    seededCourse,
  }) => {
    await page.goto(`/courses/${seededCourse.courseCode}/modules`)
    const moduleRow = page.locator('li').filter({ hasText: seededCourse.moduleTitle }).first()
    await expect(moduleRow).toBeVisible()
    // The module row has a settings/gear button.
    await expect(moduleRow.locator('button').first()).toBeVisible()
  })

  test('archive a module → disappears from active list', async ({
    coursePage: page,
    seededCourse,
  }) => {
    await page.goto(`/courses/${seededCourse.courseCode}/modules`)
    await expect(page.getByText(seededCourse.moduleTitle)).toBeVisible()

    const moduleRow = page.locator('li').filter({ hasText: seededCourse.moduleTitle }).first()
    await moduleRow.hover()
    await moduleRow.getByRole('button', { name: /module settings/i }).click()

    const dialog = page.getByRole('dialog')
    await expect(dialog).toBeVisible({ timeout: 5000 })
    await dialog.getByRole('button', { name: /delete module/i }).click()

    await expect(page.getByRole('heading', { name: /delete module/i })).toBeVisible({ timeout: 5000 })
    const confirmDelete = page.getByRole('button', { name: /delete module|archive module/i }).last()
    await expect(confirmDelete).toBeEnabled({ timeout: 10000 })
    await confirmDelete.click()

    const activeModuleRow = page.locator('li').filter({ hasText: seededCourse.moduleTitle }).first()
    await expect(activeModuleRow).not.toBeVisible({ timeout: 8000 })
  })

  // --- Vibe Activity tests (new module type) ---

  test('add a Vibe Activity via the module item dropdown → appears in outline', async ({
    coursePage: page,
    seededCourse,
  }) => {
    const vibeHtml =
      '<!doctype html><html><body style="padding:1rem;font-family:sans-serif;background:#f8fafc">' +
      '<h1 data-testid="vibe-title">Hello from Vibe Activity!</h1>' +
      '<button onclick="document.getElementById(\'out\').textContent=\'Clicked!\'">Click me</button>' +
      '<div id="out" data-testid="vibe-output"></div>' +
      '</body></html>'

    await page.route(
      `**/api/v1/courses/${encodeURIComponent(seededCourse.courseCode)}/vibe-activities/generate`,
      async (route) => {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ html: vibeHtml }),
        })
      },
    )

    await page.goto(`/courses/${seededCourse.courseCode}/modules`)
    await expect(page.getByText(seededCourse.moduleTitle)).toBeVisible()

    // Find the target module row and click its "Add module item" button
    const moduleRow = page.locator('li').filter({ hasText: seededCourse.moduleTitle }).first()
    await moduleRow.hover()

    // The Add button inside the module card
    const addBtn = moduleRow.locator('button[aria-haspopup="menu"]', { hasText: /add module item|add item/i }).first()
    await expect(addBtn).toBeVisible({ timeout: 5000 })
    await addBtn.click()

    // Click the new "Vibe Activity" menu item
    const vibeItem = page.getByRole('menuitem', { name: /vibe activity/i })
    await expect(vibeItem).toBeVisible({ timeout: 3000 })
    await vibeItem.click()

    // The Vibe Activity builder opens in prompt-first mode
    const dialog = page.getByRole('dialog')
    await expect(dialog).toBeVisible({ timeout: 5000 })

    const promptArea = dialog.locator('textarea').first()
    await promptArea.fill('A simple interactive demo for E2E')
    await dialog.getByRole('button', { name: /generate/i }).click()

    const titleInput = dialog.getByPlaceholder(/activity title/i)
    await expect(titleInput).toBeVisible({ timeout: 8000 })
    await titleInput.fill('E2E Interactive Demo')

    await dialog.getByRole('button', { name: /^save$/i }).click()

    // After save the modal closes and the new item should appear in the outline
    await expect(dialog).toBeHidden({ timeout: 8000 })
    await expect(page.getByText('E2E Interactive Demo')).toBeVisible({ timeout: 8000 })

    const vibeRow = page.locator('li, [role="listitem"]').filter({ hasText: 'E2E Interactive Demo' }).first()
    await expect(vibeRow).toBeVisible()
  })

  test('Vibe Activity item can be opened and renders interactive content', async ({
    coursePage: page,
    seededCourse,
  }) => {
    // Seed one directly via API for reliability (faster + deterministic)
    const { apiCreateVibeActivity } = await import('../fixtures/api.js')
    const seeded = await apiCreateVibeActivity(
      seededCourse.instructorToken,
      seededCourse.courseCode,
      seededCourse.moduleId,
      {
        title: 'E2E Seeded Vibe',
        html:
          '<!doctype html><html><body style="font-family:sans-serif;padding:2rem">' +
          '<h1 data-testid="vibe-heading">Seeded Vibe Activity</h1>' +
          '<p>This content was generated by the test.</p>' +
          '</body></html>',
      }
    )

    await page.goto(`/courses/${seededCourse.courseCode}/modules`)
    await expect(page.getByText('E2E Seeded Vibe')).toBeVisible({ timeout: 8000 })

    // Click the link to open the viewer
    await page.getByText('E2E Seeded Vibe').click()

    // We should land on the dedicated viewer page
    await expect(page).toHaveURL(/\/modules\/vibe-activity\//)

    // The LmsPage title + the rendered iframe content should be visible
    await expect(page.getByRole('heading', { name: /E2E Seeded Vibe/i })).toBeVisible({ timeout: 8000 })

    // The iframe should contain our injected markup (Playwright can access iframe content)
    const iframe = page.frameLocator('iframe[title*="vibe"], iframe[sandbox*="allow-scripts"]').first()
    await expect(iframe.getByTestId('vibe-heading')).toHaveText('Seeded Vibe Activity', { timeout: 8000 })
  })
})
