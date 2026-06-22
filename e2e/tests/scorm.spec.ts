/**
 * SCORM package ingestion (plan 2.14) — API smoke tests
 */
import { test, expect } from '@playwright/test'
import { apiSignup } from '../fixtures/api.js'
import { isScormIngestionEnabled } from '../fixtures/platform-features.js'

const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'
const PASSWORD = 'E2eTestPass1!'

function uniqueEmail(prefix = 'scorm') {
  return `e2e-${prefix}-${Date.now()}-${Math.random().toString(36).slice(2)}@test.invalid`
}

test('GET scorm-items: feature off returns 404', async () => {
  if (await isScormIngestionEnabled()) {
    test.skip(true, 'skipped when SCORM ingestion enabled')
  }
  const { access_token } = await apiSignup({ email: uniqueEmail(), password: PASSWORD })
  const fakeId = '00000000-0000-0000-0000-000000000099'
  const res = await fetch(`${API_BASE}/api/v1/courses/C-FAKE/scorm-items/${fakeId}`, {
    headers: { Authorization: `Bearer ${access_token}` },
  })
  expect(res.status).toBe(404)
})

test('POST scorm rte commit: unauthenticated returns 401', async () => {
  const fakeReg = '00000000-0000-0000-0000-000000000099'
  const res = await fetch(`${API_BASE}/api/v1/scorm/rte/${fakeReg}/commit`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ cmi: { 'cmi.core.lesson_status': 'passed' } }),
  })
  expect(res.status).toBe(401)
})

test('POST module scorm upload: unauthenticated returns 401', async () => {
  const res = await fetch(
    `${API_BASE}/api/v1/courses/C-FAKE/structure/modules/00000000-0000-0000-0000-000000000099/scorm`,
    { method: 'POST' },
  )
  expect(res.status).toBe(401)
})
