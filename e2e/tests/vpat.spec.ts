/**
 * Sidebar footer links to marketing-site accessibility pages.
 */
import { test, expect } from '@playwright/test'

test.describe('Sidebar footer Accessibility link (authenticated)', () => {
  test('authenticated sidebar footer links to lextures.com/accessibility', async ({ page }) => {
    const signupRes = await page.request.post(
      `${process.env.E2E_API_URL ?? 'http://localhost:8080'}/api/v1/auth/signup`,
      {
        data: {
          email: `a11y-footer-e2e-${Date.now()}@test.invalid`,
          password: 'E2eTestPass1!',
          displayName: 'A11y Footer E2E',
        },
      },
    )
    const { access_token } = (await signupRes.json()) as { access_token: string }

    await page.goto('/')
    await page.evaluate((token: string) => {
      localStorage.setItem('studydrift_access_token', token)
    }, access_token)
    await page.goto('/')
    await expect(page.getByRole('navigation', { name: 'Main' })).toBeVisible({ timeout: 15000 })

    // The accessibility link lives inside the "Legal Agreements" dropdown — open it first.
    await page.locator('footer').getByRole('button', { name: /legal agreements/i }).click()

    const sideNavFooter = page.locator('footer').filter({ hasText: /accessibility/i })
    const link = sideNavFooter.getByRole('link', { name: /accessibility/i })
    await expect(link).toBeVisible()
    await expect(link).toHaveAttribute('href', 'https://lextures.com/accessibility')
  })
})
