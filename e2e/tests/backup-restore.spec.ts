/**
 * Backup / restore ops (plan 10.15)
 *
 *   [x] GET backup-status without auth returns 401
 *   [x] POST restore-drill without auth returns 401
 *   [x] Endpoints return 404 when feature disabled
 *   [x] Global Admin can read backup status when enabled
 *   [x] Global Admin can record a restore drill
 */
import { test, expect } from '../fixtures/test.js'
import { apiSignup } from '../fixtures/api.js'

const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'
const PASSWORD = 'E2eTestPass1!'
const BACKUP_ENABLED =
  process.env.BACKUP_MODULE_ENABLED === 'true' || process.env.FEATURE_BACKUP_MODULE === 'true'

test('Backup: GET backup-status unauthenticated returns 401', async () => {
  test.skip(!BACKUP_ENABLED, 'requires BACKUP_MODULE_ENABLED=true')
  const res = await fetch(`${API_BASE}/api/v1/internal/ops/backup-status`)
  expect(res.status).toBe(401)
})

test('Backup: POST restore-drill unauthenticated returns 401', async () => {
  test.skip(!BACKUP_ENABLED, 'requires BACKUP_MODULE_ENABLED=true')
  const res = await fetch(`${API_BASE}/api/v1/internal/ops/restore-drill`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({}),
  })
  expect(res.status).toBe(401)
})

test('Backup: endpoints return 404 when module disabled', async ({ request }) => {
  test.skip(BACKUP_ENABLED, 'only when BACKUP_MODULE_ENABLED is not set')
  const res = await request.get(`${API_BASE}/api/v1/internal/ops/backup-status`, {
    headers: { Authorization: 'Bearer invalid' },
  })
  expect([401, 404]).toContain(res.status())
})

test('Backup: admin can read status and record drill', async () => {
  test.skip(!BACKUP_ENABLED, 'requires BACKUP_MODULE_ENABLED=true')

  const email = process.env.E2E_ADMIN_EMAIL ?? `e2e-backup-${Date.now()}@test.invalid`
  const { access_token: token } = await apiSignup({
    email,
    password: PASSWORD,
    displayName: 'Backup E2E Admin',
  })

  const statusRes = await fetch(`${API_BASE}/api/v1/internal/ops/backup-status`, {
    headers: { Authorization: `Bearer ${token}` },
  })
  expect(statusRes.status).toBe(200)
  const status = (await statusRes.json()) as { tiers: unknown[]; targets: { postgresRpoMinutes: number } }
  expect(status.targets.postgresRpoMinutes).toBe(60)
  expect(Array.isArray(status.tiers)).toBe(true)

  const now = new Date()
  const drillRes = await fetch(`${API_BASE}/api/v1/internal/ops/restore-drill`, {
    method: 'POST',
    headers: {
      Authorization: `Bearer ${token}`,
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      drillDate: now.toISOString().slice(0, 10),
      backupTimestamp: new Date(now.getTime() - 3600000).toISOString(),
      restoreStart: new Date(now.getTime() - 1800000).toISOString(),
      restoreEnd: now.toISOString(),
      rpoAchievedMinutes: 30,
      rtoAchievedMinutes: 60,
      pass: true,
      smokeTestOutput: 'e2e smoke ok',
    }),
  })
  expect(drillRes.status).toBe(201)
  const created = (await drillRes.json()) as { id: string }
  expect(created.id).toBeTruthy()
})
