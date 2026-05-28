/**
 * i18n framework (plan 11.1): locale detection, switcher, lazy-loaded bundles.
 */
import { test, expect } from '../fixtures/test.js'
import { apiSignup } from '../fixtures/api.js'
import { injectToken } from '../fixtures/test.js'

const PASSWORD = 'E2eTestPass1!'

test.describe('i18n — login page', () => {
  test('browser locale es renders Spanish login and html lang', async ({ browser }) => {
    const context = await browser.newContext({ locale: 'es-ES' })
    const page = await context.newPage()
    await page.addInitScript(() => {
      localStorage.removeItem('lextures.locale')
    })
    await page.goto('/login')
    await expect(page.locator('html')).toHaveAttribute('lang', 'es')
    await expect(page.getByRole('heading', { name: 'Iniciar sesión' })).toBeVisible()
    await context.close()
  })

  test('stored locale es renders Spanish without loading fr bundles', async ({ page }) => {
    const localeRequests: string[] = []
    page.on('request', (req) => {
      const url = req.url()
      if (url.includes('/locales/') && url.endsWith('.json')) {
        localeRequests.push(url)
      }
    })
    await page.addInitScript(() => {
      localStorage.setItem('lextures.locale', 'es')
    })
    await page.goto('/login')
    await expect(page.getByRole('heading', { name: 'Iniciar sesión' })).toBeVisible()
    expect(localeRequests.some((u) => u.includes('/locales/es/'))).toBeTruthy()
    expect(localeRequests.filter((u) => u.includes('/locales/fr/')).length).toBe(0)
  })
})

test.describe('i18n — settings locale switcher', () => {
  test('switching to French updates UI and persists profile locale', async ({ authedPage: page }) => {
    await page.goto('/settings/account')
    const select = page.getByTestId('locale-switcher')
    await expect(select).toBeVisible({ timeout: 10_000 })
    await select.selectOption('fr')
    await expect(page.locator('html')).toHaveAttribute('lang', 'fr', { timeout: 8_000 })

    const token = await page.evaluate(() => localStorage.getItem('studydrift_access_token'))
    const res = await page.request.get('/api/v1/settings/locale', {
      headers: { Authorization: `Bearer ${token}` },
    })
    expect(res.ok()).toBeTruthy()
    const body = (await res.json()) as { locale?: string }
    expect(body.locale).toBe('fr')
  })
})

test.describe('i18n — profile locale on session', () => {
  test('user with es locale preference loads Spanish in settings', async ({ page }) => {
    const email = `e2e-i18n-${Date.now()}@test.invalid`
    const { access_token } = await apiSignup({ email, password: PASSWORD })
    await page.request.put('/api/v1/settings/locale', {
      headers: {
        Authorization: `Bearer ${access_token}`,
        'Content-Type': 'application/json',
      },
      data: { locale: 'es' },
    })
    await injectToken(page, access_token)
    await page.goto('/settings/account')
    await expect(page.locator('html')).toHaveAttribute('lang', 'es', { timeout: 10_000 })
    await expect(page.getByTestId('locale-switcher')).toHaveValue('es')
  })
})
