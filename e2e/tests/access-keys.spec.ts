import { test, expect } from '@playwright/test'
import { apiSignup } from '../fixtures/api.js'

const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'
const PASSWORD = 'E2eTestPass1!'

function uniqueEmail(prefix = 'accesskeys') {
  return `e2e-${prefix}-${Date.now()}-${Math.random().toString(36).slice(2)}@test.invalid`
}

function authHeaders(token: string) {
  return { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' }
}

test.describe('API access keys', () => {
  test('unauthenticated management routes return 401', async ({ request }) => {
    const paths = [
      { method: 'GET', path: '/api/v1/me/access-keys/scopes' },
      { method: 'GET', path: '/api/v1/me/access-keys' },
      { method: 'POST', path: '/api/v1/me/access-keys' },
    ]
    for (const { method, path } of paths) {
      const res = await request.fetch(`${API_BASE}${path}`, { method })
      expect(res.status(), `${method} ${path}`).toBe(401)
    }
  })

  test('feature on: create key, list masked, revoke', async ({ request }) => {
    const email = uniqueEmail()
    const { access_token } = await apiSignup({
      email,
      password: PASSWORD,
      displayName: 'Access Key User',
    })

    const platformRes = await request.get(`${API_BASE}/api/v1/platform/features`, {
      headers: { Authorization: `Bearer ${access_token}` },
    })
    expect(platformRes.ok()).toBeTruthy()
    const features = (await platformRes.json()) as { ffApiTokens?: boolean }
    if (!features.ffApiTokens) {
      test.skip(true, 'ffApiTokens is false on the API')
    }

    const createRes = await request.post(`${API_BASE}/api/v1/me/access-keys`, {
      headers: authHeaders(access_token),
      data: { label: 'E2E key', scopes: ['courses:read'] },
    })
    expect(createRes.status()).toBe(201)
    const created = (await createRes.json()) as { token?: string; id?: string }
    expect(created.token).toMatch(/^ltk_/)
    expect(created.id).toBeTruthy()

    const listRes = await request.get(`${API_BASE}/api/v1/me/access-keys`, {
      headers: authHeaders(access_token),
    })
    expect(listRes.ok()).toBeTruthy()
    const listed = (await listRes.json()) as {
      tokens?: Array<{ id: string; tokenMask: string }>
    }
    const row = listed.tokens?.find((t) => t.id === created.id)
    expect(row?.tokenMask).toBeTruthy()
    expect(row?.tokenMask).not.toBe(created.token)

    const useRes = await request.get(`${API_BASE}/api/v1/courses`, {
      headers: authHeaders(created.token ?? ''),
    })
    expect(useRes.ok()).toBeTruthy()

    const revokeRes = await request.delete(`${API_BASE}/api/v1/me/access-keys/${created.id}`, {
      headers: authHeaders(access_token),
    })
    expect(revokeRes.ok()).toBeTruthy()

    const afterRevoke = await request.get(`${API_BASE}/api/v1/courses`, {
      headers: authHeaders(created.token ?? ''),
    })
    expect(afterRevoke.status()).toBe(401)
  })
})
