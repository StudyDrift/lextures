/**
 * Plan 12.4 — Captions on uploaded media (accessibility layer)
 *
 * API auth guards and feature-flag behavior (extends plan 8.4 captions.spec.ts).
 */

import { test, expect, injectToken } from '../fixtures/test.js'
import { apiLogin, apiSignup } from '../fixtures/api.js'

const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'
const PASSWORD = 'E2eTestPass1!'

function uniqueEmail(prefix = 'vcap') {
  return `e2e-${prefix}-${Date.now()}-${Math.random().toString(36).slice(2)}@test.invalid`
}

const fakeObjectId = '00000000-0000-0000-0000-000000000001'
const fakeCaptionId = '00000000-0000-0000-0000-000000000002'

test('GET admin captions compliance: unauthenticated returns 401', async () => {
  const res = await fetch(`${API_BASE}/api/v1/admin/captions/compliance`)
  expect(res.status).toBe(401)
})

test('POST caption import: unauthenticated returns 401', async () => {
  const res = await fetch(`${API_BASE}/api/v1/files/${fakeObjectId}/captions/import`, {
    method: 'POST',
  })
  expect(res.status).toBe(401)
})

test('PATCH caption VTT: unauthenticated returns 401', async () => {
  const res = await fetch(
    `${API_BASE}/api/v1/files/${fakeObjectId}/captions/${fakeCaptionId}`,
    {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ vtt_content: 'WEBVTT\n\n1\n00:00:00.000 --> 00:00:02.000\nHi\n' }),
    },
  )
  expect(res.status).toBe(401)
})

test('DELETE caption: unauthenticated returns 401', async () => {
  const res = await fetch(
    `${API_BASE}/api/v1/files/${fakeObjectId}/captions/${fakeCaptionId}`,
    { method: 'DELETE' },
  )
  expect(res.status).toBe(401)
})

test('GET caption export: unauthenticated returns 401', async () => {
  const res = await fetch(
    `${API_BASE}/api/v1/files/${fakeObjectId}/captions/${fakeCaptionId}/export?format=vtt`,
  )
  expect(res.status).toBe(401)
})

test('PATCH course caption-policy: unauthenticated returns 401', async () => {
  const res = await fetch(`${API_BASE}/api/v1/courses/C-TEST01/caption-policy`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ requireCaptions: true }),
  })
  expect(res.status).toBe(401)
})

test('GET admin captions compliance: non-admin returns 403', async () => {
  const { access_token } = await apiSignup({ email: uniqueEmail(), password: PASSWORD })
  const res = await fetch(`${API_BASE}/api/v1/admin/captions/compliance`, {
    headers: { Authorization: `Bearer ${access_token}` },
  })
  expect(res.status).toBe(403)
})

test('platform features includes videoCaptionsEnabled field', async () => {
  const { access_token } = await apiSignup({ email: uniqueEmail('pfeat'), password: PASSWORD })
  const res = await fetch(`${API_BASE}/api/v1/platform/features`, {
    headers: { Authorization: `Bearer ${access_token}` },
  })
  expect(res.status).toBe(200)
  const body = (await res.json()) as Record<string, unknown>
  expect(body).toHaveProperty('videoCaptionsEnabled')
  expect(body).toHaveProperty('autoCaptioningEnabled')
})

test('Caption compliance page loads for global admin', async ({ page }) => {
  let token: string
  try {
    ;({ access_token: token } = await apiSignup({
      email: process.env.E2E_ADMIN_EMAIL ?? 'admin@e2e.test',
      password: process.env.E2E_ADMIN_PASSWORD ?? PASSWORD,
    }))
  } catch {
    ;({ access_token: token } = await apiLogin({
      email: process.env.E2E_ADMIN_EMAIL ?? 'admin@e2e.test',
      password: process.env.E2E_ADMIN_PASSWORD ?? PASSWORD,
    }))
  }
  await injectToken(page, token)
  await page.goto('/admin/caption-compliance')
  await expect(page.getByRole('heading', { name: /caption compliance report/i })).toBeVisible({
    timeout: 15000,
  })
})
