/**
 * ISO 27001 / 27701 ISMS (plan 10.10)
 */
import { test, expect, injectToken } from '../fixtures/test.js'
import { apiLogin, apiSignup } from '../fixtures/api.js'

async function adminTokens(): Promise<string> {
  try {
    const { access_token } = await apiSignup({ email: 'admin@e2e.test', password: PASSWORD })
    return access_token
  } catch {
    const { access_token } = await apiLogin({ email: 'admin@e2e.test', password: PASSWORD })
    return access_token
  }
}

const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'
const PASSWORD = 'E2eTestPass1!'
const ISO_ISMS_ENABLED =
  process.env.FEATURE_ISO_ISMS === 'true' || process.env.ISO_ISMS_ENABLED === 'true'

function uniqueEmail(prefix = 'iso') {
  return `e2e-${prefix}-${Date.now()}-${Math.random().toString(36).slice(2)}@test.invalid`
}

test.describe('ISO ISMS — public trust endpoint', () => {
  test('GET /api/v1/trust/iso returns program summary without auth', async () => {
    const res = await fetch(`${API_BASE}/api/v1/trust/iso`)
    expect(res.ok).toBeTruthy()
    const body = (await res.json()) as {
      scopeStatement?: string
      iso27001Status?: string
      soa?: { total: number }
    }
    expect(typeof body.scopeStatement).toBe('string')
    expect(body.scopeStatement!.length).toBeGreaterThan(0)
    expect(body.soa?.total).toBe(93)
  })
})

test.describe('ISO ISMS — admin API', () => {
  test('admin endpoints return 404 when feature disabled', async () => {
    test.skip(ISO_ISMS_ENABLED, 'skipped when ISO_ISMS_ENABLED=true')
    const { access_token } = await apiSignup({ email: uniqueEmail('off'), password: PASSWORD })
    const res = await fetch(`${API_BASE}/api/v1/compliance/iso/dashboard`, {
      headers: { Authorization: `Bearer ${access_token}` },
    })
    expect(res.status).toBe(404)
  })

  test('dashboard unauthenticated returns 401', async () => {
    test.skip(!ISO_ISMS_ENABLED, 'requires ISO_ISMS_ENABLED=true')
    const res = await fetch(`${API_BASE}/api/v1/compliance/iso/dashboard`)
    expect(res.status).toBe(401)
  })

  test('bootstrap admin can create audit finding and risk', async () => {
    test.skip(!ISO_ISMS_ENABLED, 'requires ISO_ISMS_ENABLED=true')
    const access_token = await adminTokens()
    const headers = {
      Authorization: `Bearer ${access_token}`,
      'Content-Type': 'application/json',
    }

    const dashRes = await fetch(`${API_BASE}/api/v1/compliance/iso/dashboard`, { headers })
    expect(dashRes.status).toBe(200)
    const dash = (await dashRes.json()) as { program?: { soa?: { total: number } } }
    expect(dash.program?.soa?.total).toBe(93)

    const findingRes = await fetch(`${API_BASE}/api/v1/compliance/iso/audit-findings`, {
      method: 'POST',
      headers,
      body: JSON.stringify({
        auditCycle: 'e2e-internal',
        findingType: 'observation',
        isoClause: 'A.8.15',
        description: 'E2E test finding',
      }),
    })
    expect(findingRes.status).toBe(201)

    const riskRes = await fetch(`${API_BASE}/api/v1/compliance/iso/risk-register`, {
      method: 'POST',
      headers,
      body: JSON.stringify({
        riskTitle: 'E2E test risk',
        likelihood: 2,
        impact: 3,
        treatment: 'mitigate',
      }),
    })
    expect(riskRes.status).toBe(201)

    const listRes = await fetch(`${API_BASE}/api/v1/compliance/iso/audit-findings`, { headers })
    expect(listRes.status).toBe(200)
    const list = (await listRes.json()) as { findings: { description: string }[] }
    expect(list.findings.some((f) => f.description === 'E2E test finding')).toBe(true)
  })
})

test.describe('ISO ISMS — admin UI', () => {
  test('admin page loads dashboard for global admin', async ({ page }) => {
    test.skip(!ISO_ISMS_ENABLED, 'requires ISO_ISMS_ENABLED=true')
    const token = await adminTokens()
    await injectToken(page, token)
    const dashboardResponse = page.waitForResponse(
      (r) => r.url().includes('/api/v1/compliance/iso/dashboard') && r.status() === 200,
    )
    await page.goto('/admin/compliance/iso')
    const ack = page.getByRole('button', { name: /I acknowledge/i })
    if (await ack.isVisible().catch(() => false)) {
      await ack.click()
    }
    await dashboardResponse
    await expect(page.getByText(/Open findings/i)).toBeVisible({ timeout: 15000 })
    await expect(page.getByRole('heading', { name: 'Statement of Applicability' })).toBeVisible()
  })
})
