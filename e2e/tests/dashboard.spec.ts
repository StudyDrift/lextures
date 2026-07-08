/**
 * Dashboard
 *
 * Checklist coverage (docs/e2e.md):
 *   [x] Dashboard loads after login with "My Courses" section visible
 *   [x] Intro course onboarding shown for fresh users (auto-enrolled in C-WLCOME)
 */
import { test, expect } from '../fixtures/test.js'
import { mainNav } from '../fixtures/test.js'

test.describe('Dashboard', () => {
  test('loads after login and shows main UI sections', async ({ authedPage: page }) => {
    // authedPage lands on / after injecting the token
    await expect(page).toHaveURL('/')
    await expect(mainNav(page)).toBeVisible()

    // Page heading
    await expect(page.getByRole('heading', { name: /dashboard/i })).toBeVisible()
  })

  test('shows quick links when the instructor has courses', async ({ coursePage: page }) => {
    await page.goto('/')
    const dashboardMain = page.locator('[data-onboarding="dashboard-main"]')
    await expect(dashboardMain).toBeVisible({ timeout: 30000 })
    await expect(dashboardMain.getByRole('link', { name: /inbox/i })).toBeVisible()
    await expect(dashboardMain.getByRole('link', { name: /all courses/i })).toBeVisible()
  })

  test('shows intro course onboarding for a fresh user', async ({ authedPage: page }) => {
    await expect(page).toHaveURL('/')
    // New users are auto-enrolled in the platform intro course when intro_course_enabled is on.
    await expect(page.getByRole('region', { name: /intro course onboarding/i })).toBeVisible({
      timeout: 30000,
    })
  })
})
