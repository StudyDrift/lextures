/**
 * RTL locale support (plan 11.2)
 */
import { test, expect } from '../fixtures/test.js'
import { apiSignup } from '../fixtures/api.js'
import { injectToken } from '../fixtures/test.js'

const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'
const PASSWORD = 'E2eTestPass1!'

function uniqueEmail(label = 'rtl'): string {
  return `e2e-rtl-${label}-${Date.now()}-${Math.random().toString(36).slice(2)}@test.invalid`
}

test.describe('RTL locale API', () => {
  test('GET /api/v1/public/locale-defaults respects Accept-Language', async () => {
    const res = await fetch(`${API_BASE}/api/v1/public/locale-defaults`, {
      headers: { 'Accept-Language': 'ar-SA,ar;q=0.9' },
    })
    expect(res.status).toBe(200)
    const body = (await res.json()) as { locale?: string }
    expect(body.locale).toBe('ar-SA')
  })

  test('authenticated user can set Arabic locale', async () => {
    const { access_token } = await apiSignup({ email: uniqueEmail(), password: PASSWORD })
    const putRes = await fetch(`${API_BASE}/api/v1/settings/locale`, {
      method: 'PUT',
      headers: {
        Authorization: `Bearer ${access_token}`,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ locale: 'ar' }),
    })
    expect(putRes.status).toBe(200)
    const patched = (await putRes.json()) as { locale?: string }
    expect(patched.locale).toBe('ar')

    const getRes = await fetch(`${API_BASE}/api/v1/settings/locale`, {
      headers: { Authorization: `Bearer ${access_token}` },
    })
    expect(getRes.status).toBe(200)
    const account = (await getRes.json()) as { locale?: string }
    expect(account.locale).toBe('ar')
  })
})

test.describe('RTL document direction in browser', () => {
  test('Arabic locale sets html dir=rtl when rtl feature enabled', async ({ page }) => {
    const { access_token } = await apiSignup({ email: uniqueEmail('ui'), password: PASSWORD })
    await injectToken(page, access_token)

    await page.evaluate(() => {
      localStorage.setItem('lextures.locale', 'ar')
      localStorage.setItem('lextures.rtlEnabled', '1')
    })

    await page.goto('/settings/account')
    const localeSelect = page.getByTestId('locale-switcher')
    await expect(localeSelect).toBeVisible({ timeout: 15_000 })
    await localeSelect.selectOption('ar')

    await expect
      .poll(async () => page.evaluate(() => document.documentElement.getAttribute('dir')))
      .toBe('rtl')
    await expect
      .poll(async () => page.evaluate(() => document.documentElement.getAttribute('data-locale')))
      .toBe('ar')

    await localeSelect.selectOption('en')
    await expect
      .poll(async () => page.evaluate(() => document.documentElement.getAttribute('dir')))
      .toBe('ltr')
  })

  test('main navigation remains visible in RTL mode', async ({ page }) => {
    const { access_token } = await apiSignup({ email: uniqueEmail('nav'), password: PASSWORD })
    await injectToken(page, access_token)
    await page.evaluate(() => {
      localStorage.setItem('lextures.locale', 'ar')
      localStorage.setItem('lextures.rtlEnabled', '1')
    })
    await page.goto('/dashboard')
    const nav = page.getByRole('navigation', { name: 'Main' })
    await expect(nav).toBeVisible()
    await expect
      .poll(async () => page.evaluate(() => document.documentElement.dir))
      .toBe('rtl')
  })
})
