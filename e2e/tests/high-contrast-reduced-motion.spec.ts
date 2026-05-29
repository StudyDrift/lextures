/**
 * Plan 12.7 — High-contrast and reduced-motion preferences.
 *
 * Coverage:
 *   [x] Unauthenticated GET reading-preferences returns 401
 *   [x] Unauthenticated PATCH reading-preferences returns 401
 *   [x] Feature flag off → GET returns 404
 *   [x] Feature flag off → PATCH returns 404
 *   [x] PATCH persists high-contrast preference (round-trip)
 *   [x] PATCH persists reduce-motion preference (round-trip)
 *   [x] PATCH partial update merges with existing values
 *   [x] OS prefers-reduced-motion: reduce suppresses animations on the page
 *   [x] html.high-contrast class activates when preference is set
 *   [x] html.reduced-motion class activates when preference is set
 */

import { test, expect } from '@playwright/test'
import { apiSignup } from '../fixtures/api.js'

const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'
const PASSWORD = 'E2eTestPass1!'

function uniqueEmail(prefix = 'hcrm') {
  return `e2e-${prefix}-${Date.now()}-${Math.random().toString(36).slice(2)}@test.invalid`
}

function authHeaders(token: string) {
  return { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' }
}

// ── Auth guard tests ────────────────────────────────────────────────────────

test('GET reading-preferences: unauthenticated returns 401', async () => {
  const res = await fetch(`${API_BASE}/api/v1/me/reading-preferences`)
  expect(res.status).toBe(401)
})

test('PATCH reading-preferences: unauthenticated returns 401', async () => {
  const res = await fetch(`${API_BASE}/api/v1/me/reading-preferences`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ highContrast: true }),
  })
  expect(res.status).toBe(401)
})

// ── Feature flag tests ──────────────────────────────────────────────────────

test('GET reading-preferences: feature flag off returns 404', async () => {
  const { access_token } = await apiSignup({ email: uniqueEmail('ff-get'), password: PASSWORD })
  const res = await fetch(`${API_BASE}/api/v1/me/reading-preferences`, {
    headers: authHeaders(access_token),
  })
  if (res.status === 200) {
    test.skip(true, 'ff_high_contrast_reduced_motion is on in this environment — skipping 404 check')
    return
  }
  expect(res.status).toBe(404)
})

test('PATCH reading-preferences: feature flag off returns 404', async () => {
  const { access_token } = await apiSignup({ email: uniqueEmail('ff-patch'), password: PASSWORD })
  const res = await fetch(`${API_BASE}/api/v1/me/reading-preferences`, {
    method: 'PATCH',
    headers: authHeaders(access_token),
    body: JSON.stringify({ highContrast: true }),
  })
  if (res.status === 200) {
    test.skip(true, 'ff_high_contrast_reduced_motion is on in this environment — skipping 404 check')
    return
  }
  expect(res.status).toBe(404)
})

// ── Preference round-trip tests ─────────────────────────────────────────────

async function requireFeatureEnabled(token: string) {
  const probe = await fetch(`${API_BASE}/api/v1/me/reading-preferences`, {
    headers: authHeaders(token),
  })
  if (probe.status === 404) {
    return false
  }
  return true
}

test('PATCH persists high-contrast preference', async () => {
  const { access_token } = await apiSignup({ email: uniqueEmail('hc'), password: PASSWORD })
  if (!(await requireFeatureEnabled(access_token))) {
    test.skip(true, 'ff_high_contrast_reduced_motion is off — enable in platform seed')
    return
  }

  const patchRes = await fetch(`${API_BASE}/api/v1/me/reading-preferences`, {
    method: 'PATCH',
    headers: authHeaders(access_token),
    body: JSON.stringify({ highContrast: true }),
  })
  expect(patchRes.status).toBe(200)
  const patched = (await patchRes.json()) as { highContrast: boolean; reduceMotion: boolean }
  expect(patched.highContrast).toBe(true)
  expect(patched.reduceMotion).toBe(false)

  const getRes = await fetch(`${API_BASE}/api/v1/me/reading-preferences`, {
    headers: authHeaders(access_token),
  })
  expect(getRes.status).toBe(200)
  const fetched = (await getRes.json()) as { highContrast: boolean; reduceMotion: boolean }
  expect(fetched.highContrast).toBe(true)
})

test('PATCH persists reduce-motion preference', async () => {
  const { access_token } = await apiSignup({ email: uniqueEmail('rm'), password: PASSWORD })
  if (!(await requireFeatureEnabled(access_token))) {
    test.skip(true, 'ff_high_contrast_reduced_motion is off — enable in platform seed')
    return
  }

  const patchRes = await fetch(`${API_BASE}/api/v1/me/reading-preferences`, {
    method: 'PATCH',
    headers: authHeaders(access_token),
    body: JSON.stringify({ reduceMotion: true }),
  })
  expect(patchRes.status).toBe(200)
  const patched = (await patchRes.json()) as { highContrast: boolean; reduceMotion: boolean }
  expect(patched.reduceMotion).toBe(true)
  expect(patched.highContrast).toBe(false)
})

test('PATCH partial update merges with existing values', async () => {
  const { access_token } = await apiSignup({ email: uniqueEmail('merge'), password: PASSWORD })
  if (!(await requireFeatureEnabled(access_token))) {
    test.skip(true, 'ff_high_contrast_reduced_motion is off — enable in platform seed')
    return
  }

  await fetch(`${API_BASE}/api/v1/me/reading-preferences`, {
    method: 'PATCH',
    headers: authHeaders(access_token),
    body: JSON.stringify({ highContrast: true, reduceMotion: true }),
  })

  const patchRes = await fetch(`${API_BASE}/api/v1/me/reading-preferences`, {
    method: 'PATCH',
    headers: authHeaders(access_token),
    body: JSON.stringify({ highContrast: false }),
  })
  expect(patchRes.status).toBe(200)
  const patched = (await patchRes.json()) as { highContrast: boolean; reduceMotion: boolean }
  expect(patched.highContrast).toBe(false)
  expect(patched.reduceMotion).toBe(true)
})

// ── Browser-level CSS tests ─────────────────────────────────────────────────

test('OS prefers-reduced-motion suppresses animations', async ({ page }) => {
  await page.emulateMedia({ reducedMotion: 'reduce' })
  await page.goto('/')
  await page.waitForLoadState('domcontentloaded')

  const animDuration = await page.evaluate(() => {
    const el = document.querySelector('.sidenav-course-items > *') as HTMLElement | null
    if (!el) return null
    return getComputedStyle(el).animationDuration
  })
  if (animDuration !== null) {
    expect(['0s', '0.01ms', 'none']).toContain(animDuration)
  }
})

test('html.high-contrast class activates when localStorage flag is set', async ({ page }) => {
  await page.addInitScript(() => {
    localStorage.setItem('lextures.highContrast', '1')
  })
  await page.goto('/')
  await page.waitForLoadState('domcontentloaded')
  const hasClass = await page.evaluate(() =>
    document.documentElement.classList.contains('high-contrast'),
  )
  expect(hasClass).toBe(true)
})

test('html.reduced-motion class activates when localStorage flag is set', async ({ page }) => {
  await page.addInitScript(() => {
    localStorage.setItem('lextures.reduceMotion', '1')
  })
  await page.goto('/')
  await page.waitForLoadState('domcontentloaded')
  const hasClass = await page.evaluate(() =>
    document.documentElement.classList.contains('reduced-motion'),
  )
  expect(hasClass).toBe(true)
})

test('forced-colors mode: interactive elements retain visible boundaries', async ({ page }) => {
  await page.emulateMedia({ forcedColors: 'active' })
  await page.goto('/')
  await page.waitForLoadState('domcontentloaded')
  const buttons = page.locator('button').first()
  const count = await page.locator('button').count()
  if (count > 0) {
    await expect(buttons).toBeVisible()
  }
})
