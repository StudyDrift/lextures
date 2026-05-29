/**
 * Bug bounty / responsible disclosure (plan 10.16)
 */
import { test, expect, injectToken } from '../fixtures/test.js'
import { apiLogin, apiSignup } from '../fixtures/api.js'

const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'
const PASSWORD = 'E2eTestPass1!'

async function securityModuleEnabled(): Promise<boolean> {
  if (process.env.SECURITY_DISCLOSURE_MODULE_ENABLED === 'true' || process.env.FEATURE_SECURITY_DISCLOSURE === 'true') {
    return true
  }
  const res = await fetch(`${API_BASE}/api/v1/compliance/security-reports`)
  return res.status === 401
}

async function adminTokens(): Promise<string> {
  try {
    const { access_token } = await apiSignup({ email: 'admin@e2e.test', password: PASSWORD })
    return access_token
  } catch {
    const { access_token } = await apiLogin({ email: 'admin@e2e.test', password: PASSWORD })
    return access_token
  }
}

test.describe('Security disclosure — public', () => {
  test('GET /api/v1/trust/security returns policy without auth', async () => {
    const res = await fetch(`${API_BASE}/api/v1/trust/security`)
    expect(res.ok).toBeTruthy()
    const body = (await res.json()) as { contactEmail?: string; coordinatedDisclosureDays?: number }
    expect(body.contactEmail).toBe('security@lextures.io')
    expect(body.coordinatedDisclosureDays).toBe(90)
  })

  test('trust center links to responsible disclosure policy on marketing site', async ({ page }) => {
    await page.goto('/trust')
    await expect(page.getByRole('link', { name: /responsible disclosure policy/i })).toHaveAttribute(
      'href',
      'https://lextures.com/security',
    )
  })
})

test.describe('Security disclosure — admin API', () => {
  test('admin endpoints return 404 when feature disabled', async () => {
    if (await securityModuleEnabled()) {
      test.skip(true, 'Security disclosure module enabled')
    }
    const { access_token } = await apiSignup({
      email: `e2e-sec-off-${Date.now()}@test.invalid`,
      password: PASSWORD,
    })
    const res = await fetch(`${API_BASE}/api/v1/compliance/security-reports`, {
      headers: { Authorization: `Bearer ${access_token}` },
    })
    expect(res.status).toBe(404)
  })

  test('bootstrap admin can log and patch critical report with SLA', async () => {
    if (!(await securityModuleEnabled())) {
      test.skip(true, 'Security disclosure module not enabled')
    }
    const access_token = await adminTokens()
    const headers = {
      Authorization: `Bearer ${access_token}`,
      'Content-Type': 'application/json',
    }

    const createRes = await fetch(`${API_BASE}/api/v1/compliance/security-reports`, {
      method: 'POST',
      headers,
      body: JSON.stringify({
        summary: 'E2E reflected XSS in profile',
        severity: 'critical',
        cvssScore: 9.1,
      }),
    })
    expect(createRes.status).toBe(201)
    const { id } = (await createRes.json()) as { id: string }

    const patchRes = await fetch(`${API_BASE}/api/v1/compliance/security-reports/${id}`, {
      method: 'PATCH',
      headers,
      body: JSON.stringify({ status: 'patched', severity: 'critical' }),
    })
    expect(patchRes.status).toBe(200)

    const getRes = await fetch(`${API_BASE}/api/v1/compliance/security-reports/${id}`, { headers })
    expect(getRes.status).toBe(200)
    const report = (await getRes.json()) as { status: string; slaMet?: boolean }
    expect(report.status).toBe('patched')
    expect(report.slaMet).toBe(true)

    const exportRes = await fetch(`${API_BASE}/api/v1/compliance/security-reports/export`, { headers })
    expect(exportRes.status).toBe(200)
    const csv = await exportRes.text()
    expect(csv).toContain('severity')
    expect(csv).toContain(id)
  })
})

test.describe('Security disclosure — admin UI', () => {
  test('admin page loads for global admin when module enabled', async ({ page }) => {
    if (!(await securityModuleEnabled())) {
      test.skip(true, 'Security disclosure module not enabled')
    }
    const access_token = await adminTokens()
    await injectToken(page, access_token)
    await page.goto('/admin/compliance/security-reports')
    await expect(page.getByRole('heading', { name: /security reports/i })).toBeVisible({ timeout: 10000 })
    await expect(page.getByRole('link', { name: /lextures\.com\/security/i })).toBeVisible()
  })
})
