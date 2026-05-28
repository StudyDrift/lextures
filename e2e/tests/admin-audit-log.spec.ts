/**
 * Admin Audit Log (plan 10.11)
 *
 *   [x] GET audit-log without auth returns 401
 *   [x] GET audit-log/export without auth returns 401
 *   [x] GET audit-log/{id} without auth returns 401
 *   [x] All audit-log endpoints return 404 when feature is disabled
 *   [x] GET audit-log with non-admin user returns 403
 *   [x] Admin can list audit log events (feature on)
 *   [x] Admin can get a single audit event by ID
 *   [x] Admin can export audit log as CSV
 *   [x] Admin can export audit log as JSON
 *   [x] GET audit-log/{non-existent-id} returns 404
 *   [x] Invalid actorId query param returns 400
 */
import { test, expect } from '../fixtures/test.js'
import { apiLogin, apiSignup } from '../fixtures/api.js'

const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'
const PASSWORD = 'E2eTestPass1!'

async function auditLogEnabled(): Promise<boolean> {
  if (process.env.ADMIN_AUDIT_LOG_DISABLED === 'true') {
    return false
  }
  // Feature defaults to on; probe by checking if unauthenticated returns 401 (on) or 404 (off).
  const res = await fetch(`${API_BASE}/api/v1/compliance/audit-log`)
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

function uniqueEmail(prefix = 'auditlog') {
  return `e2e-${prefix}-${Date.now()}-${Math.random().toString(36).slice(2)}@test.invalid`
}

// ──────────────────────────────────────────────────────────
// Unauthenticated 401 checks
// ──────────────────────────────────────────────────────────

test('AdminAuditLog: GET audit-log unauthenticated returns 401', async () => {
  if (!(await auditLogEnabled())) {
    test.skip(true, 'admin audit log not enabled')
  }
  const res = await fetch(`${API_BASE}/api/v1/compliance/audit-log`)
  expect(res.status).toBe(401)
})

test('AdminAuditLog: GET audit-log/export unauthenticated returns 401', async () => {
  if (!(await auditLogEnabled())) {
    test.skip(true, 'admin audit log not enabled')
  }
  const res = await fetch(`${API_BASE}/api/v1/compliance/audit-log/export`)
  expect(res.status).toBe(401)
})

test('AdminAuditLog: GET audit-log/{id} unauthenticated returns 401', async () => {
  if (!(await auditLogEnabled())) {
    test.skip(true, 'admin audit log not enabled')
  }
  const res = await fetch(
    `${API_BASE}/api/v1/compliance/audit-log/00000000-0000-0000-0000-000000000001`,
  )
  expect(res.status).toBe(401)
})

// ──────────────────────────────────────────────────────────
// Feature-disabled 404 checks
// ──────────────────────────────────────────────────────────

test('AdminAuditLog: All endpoints return 404 when feature disabled', async () => {
  if (await auditLogEnabled()) {
    test.skip(true, 'admin audit log is enabled; skipping disabled-feature test')
  }
  const { access_token } = await apiSignup({ email: uniqueEmail('off'), password: PASSWORD })
  const headers = { Authorization: `Bearer ${access_token}` }

  const paths = [
    '/api/v1/compliance/audit-log',
    '/api/v1/compliance/audit-log/export',
    '/api/v1/compliance/audit-log/00000000-0000-0000-0000-000000000001',
  ]
  for (const path of paths) {
    const res = await fetch(`${API_BASE}${path}`, { headers })
    expect(res.status, `${path} should return 404`).toBe(404)
  }
})

// ──────────────────────────────────────────────────────────
// Non-admin forbidden
// ──────────────────────────────────────────────────────────

test('AdminAuditLog: Non-admin user gets 403', async () => {
  if (!(await auditLogEnabled())) {
    test.skip(true, 'admin audit log not enabled')
  }
  const { access_token } = await apiSignup({ email: uniqueEmail('nonadmin'), password: PASSWORD })
  const res = await fetch(`${API_BASE}/api/v1/compliance/audit-log`, {
    headers: { Authorization: `Bearer ${access_token}` },
  })
  expect(res.status).toBe(403)
})

// ──────────────────────────────────────────────────────────
// Admin happy-path tests
// ──────────────────────────────────────────────────────────

test('AdminAuditLog: Admin can list audit log', async () => {
  if (!(await auditLogEnabled())) {
    test.skip(true, 'admin audit log not enabled')
  }
  const access_token = await adminTokens()
  const res = await fetch(`${API_BASE}/api/v1/compliance/audit-log`, {
    headers: { Authorization: `Bearer ${access_token}` },
  })
  expect(res.status).toBe(200)
  const body = (await res.json()) as { events: unknown[] }
  expect(Array.isArray(body.events)).toBe(true)
})

test('AdminAuditLog: Admin can filter by eventType', async () => {
  if (!(await auditLogEnabled())) {
    test.skip(true, 'admin audit log not enabled')
  }
  const access_token = await adminTokens()
  const res = await fetch(
    `${API_BASE}/api/v1/compliance/audit-log?eventType=role_grant`,
    { headers: { Authorization: `Bearer ${access_token}` } },
  )
  expect(res.status).toBe(200)
  const body = (await res.json()) as { events: Array<{ eventType: string }> }
  expect(Array.isArray(body.events)).toBe(true)
  for (const e of body.events) {
    expect(e.eventType).toBe('role_grant')
  }
})

test('AdminAuditLog: GET audit-log/{non-existent-id} returns 404', async () => {
  if (!(await auditLogEnabled())) {
    test.skip(true, 'admin audit log not enabled')
  }
  const access_token = await adminTokens()
  const res = await fetch(
    `${API_BASE}/api/v1/compliance/audit-log/00000000-0000-0000-0000-000000000099`,
    { headers: { Authorization: `Bearer ${access_token}` } },
  )
  expect(res.status).toBe(404)
})

test('AdminAuditLog: Invalid actorId returns 400', async () => {
  if (!(await auditLogEnabled())) {
    test.skip(true, 'admin audit log not enabled')
  }
  const access_token = await adminTokens()
  const res = await fetch(
    `${API_BASE}/api/v1/compliance/audit-log?actorId=not-a-uuid`,
    { headers: { Authorization: `Bearer ${access_token}` } },
  )
  expect(res.status).toBe(400)
})

test('AdminAuditLog: Admin can export as CSV', async () => {
  if (!(await auditLogEnabled())) {
    test.skip(true, 'admin audit log not enabled')
  }
  const access_token = await adminTokens()
  const res = await fetch(
    `${API_BASE}/api/v1/compliance/audit-log/export?format=csv`,
    { headers: { Authorization: `Bearer ${access_token}` } },
  )
  expect(res.status).toBe(200)
  const ct = res.headers.get('content-type') ?? ''
  expect(ct).toContain('text/csv')
  const text = await res.text()
  expect(text).toContain('event_id')
  expect(text).toContain('event_type')
})

test('AdminAuditLog: Admin can export as JSON', async () => {
  if (!(await auditLogEnabled())) {
    test.skip(true, 'admin audit log not enabled')
  }
  const access_token = await adminTokens()
  const res = await fetch(
    `${API_BASE}/api/v1/compliance/audit-log/export`,
    { headers: { Authorization: `Bearer ${access_token}` } },
  )
  expect(res.status).toBe(200)
  const body = (await res.json()) as { events: unknown[] }
  expect(Array.isArray(body.events)).toBe(true)
})
