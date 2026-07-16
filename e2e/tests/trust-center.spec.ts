/**
 * Trust Center page — public access (plan 20.2)
 */
import { test, expect } from '../fixtures/test.js'

const apiBase = process.env.E2E_API_URL ?? 'http://localhost:8080'

test.describe('Trust Center — public access', () => {
  test('loads without login', async ({ page }) => {
    await page.goto('/trust')
    await expect(page.getByRole('heading', { level: 1, name: /trust center/i })).toBeVisible({
      timeout: 8000,
    })
  })

  test('contains security overview section', async ({ page }) => {
    await page.goto('/trust')
    await expect(page.getByRole('button', { name: /security overview/i })).toBeVisible()
    await expect(page.getByText(/encryption/i).first()).toBeVisible()
    await expect(page.getByText(/incident response/i).first()).toBeVisible()
  })

  test('contains certifications section with SOC 2 and FERPA', async ({ page }) => {
    await page.goto('/trust')
    await expect(page.getByRole('button', { name: /certifications/i })).toBeVisible()
    await expect(page.getByText(/SOC 2/i).first()).toBeVisible()
    await expect(page.getByText(/FERPA/i).first()).toBeVisible()
    await expect(page.getByText(/GDPR/i).first()).toBeVisible()
    await expect(page.getByText(/ISO 27701/i).first()).toBeVisible()
  })

  test('trust ISO API returns 93 Annex A controls', async () => {
    const res = await fetch(`${apiBase}/api/v1/trust/iso`)
    expect(res.ok).toBeTruthy()
    const body = (await res.json()) as { soa?: { total: number } }
    expect(body.soa?.total).toBe(93)
  })

  test('sub-processor table lists AI vendors as when-configured and explains BYOK', async ({ page }) => {
    await page.goto('/trust')
    const table = page.getByRole('table', { name: /sub-processor list/i })
    await expect(table).toBeVisible()
    await expect(table.getByRole('cell', { name: 'Anthropic', exact: true })).toBeVisible()
    await expect(table.getByRole('cell', { name: 'OpenAI', exact: true })).toBeVisible()
    await expect(table.getByRole('cell', { name: 'OpenRouter', exact: true })).toBeVisible()
    await expect(table.getByRole('cell', { name: /when configured as an AI provider/i }).first()).toBeVisible()
    await expect(
      table.getByRole('cell', { name: /not used on BYOK-only deployments that omit OpenRouter/i }),
    ).toBeVisible()
    const byokNote = page.getByTestId('ai-byok-subprocessor-note')
    await expect(byokNote).toBeVisible()
    await expect(byokNote).toContainText(/bring-your-own-key/i)
    await expect(byokNote).toContainText(/not automatically a Lextures sub-processor/i)
  })

  test('incident history table is visible', async ({ page }) => {
    await page.goto('/trust')
    const table = page.getByRole('table', { name: /incident history/i })
    await expect(table).toBeVisible()
  })

  test('contact section has security email', async ({ page }) => {
    await page.goto('/trust')
    await expect(page.getByRole('link', { name: /security@lextures\.com/i }).first()).toBeVisible()
  })

  test('subscribe form accepts email and calls API', async ({ page }) => {
    await page.goto('/trust')
    const emailInput = page.getByLabel(/email address/i)
    await expect(emailInput).toBeVisible()
    await emailInput.fill(`trust-e2e-${Date.now()}@test.invalid`)
    // The subscribe button calls POST /api/v1/trust/sub-processor-updates/subscribe.
    // Without SMTP configured the server returns 204 (no-op store).
    await page.getByRole('button', { name: /subscribe/i }).click()
    // After success the form is replaced with a success message.
    await expect(page.getByRole('status')).toBeVisible({ timeout: 8000 })
    await expect(page.getByText(/you're subscribed/i)).toBeVisible()
  })

  test('subscribe API endpoint returns 204 for valid email', async () => {
    const res = await fetch(`${apiBase}/api/v1/trust/sub-processor-updates/subscribe`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email: `trust-api-e2e-${Date.now()}@test.invalid` }),
    })
    expect(res.status).toBe(204)
  })

  test('subscribe API returns 400 for missing email', async () => {
    const res = await fetch(`${apiBase}/api/v1/trust/sub-processor-updates/subscribe`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({}),
    })
    expect(res.status).toBe(400)
  })

  test('has accessible table captions and headings', async ({ page }) => {
    await page.goto('/trust')
    await expect(page.getByRole('table', { name: /sub-processor list/i })).toBeVisible({
      timeout: 15000,
    })
    await expect(page.getByRole('table', { name: /incident history/i })).toBeVisible()
    const tables = page.getByRole('table')
    const count = await tables.count()
    expect(count).toBeGreaterThanOrEqual(2)
    // Spot-check: sub-processor table has column headers.
    const subProcTable = page.getByRole('table', { name: /sub-processor list/i })
    await expect(subProcTable.getByRole('columnheader', { name: /vendor/i })).toBeVisible()
    await expect(subProcTable.getByRole('columnheader', { name: /dpa status/i })).toBeVisible()
  })
})
