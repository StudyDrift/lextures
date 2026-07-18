/**
 * Shared Playwright helpers for E2E.2 platform feature flag contract specs.
 */
import { execSync } from 'node:child_process'
import { closeSync, openSync, unlinkSync } from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import type { Page } from '@playwright/test'
import { expect } from '@playwright/test'
import {
  apiGetPlatformFeatures,
  apiGetPlatformSettings,
  apiPutPlatformSettings,
  apiRestorePlatformBooleanSettings,
  apiSnapshotPlatformBooleanSettings,
  apiWaitForPlatformFeature,
  apiWaitForPlatformSetting,
} from '../fixtures/api.js'
import type { PlatformFeatureMatrixEntry } from './platform-feature-matrix.js'

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..')
const PLATFORM_SETTINGS_LOCK = '/tmp/lextures-platform-settings.lock'

/** Serialize global platform mutations across Playwright workers. */
export async function withPlatformSettingsLock<T>(fn: () => Promise<T>): Promise<T> {
  const started = Date.now()
  while (true) {
    try {
      const fd = openSync(PLATFORM_SETTINGS_LOCK, 'wx')
      closeSync(fd)
      break
    } catch {
      if (Date.now() - started > 240_000) {
        throw new Error('Timed out waiting for platform settings lock')
      }
      await new Promise((r) => setTimeout(r, 250))
    }
  }
  try {
    return await fn()
  } finally {
    try {
      unlinkSync(PLATFORM_SETTINGS_LOCK)
    } catch {
      /* ignore */
    }
  }
}

export function platformFeatureToggleRow(page: Page, uiLabel: string) {
  return page
    .locator('div')
    .filter({ has: page.locator('p').filter({ hasText: new RegExp(`^${escapeRegExp(uiLabel)}`) }) })
    .filter({ has: page.getByRole('switch') })
    .last()
}

function escapeRegExp(value: string): string {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

export function databaseUrl(): string {
  return (
    process.env.DATABASE_URL ??
    'postgres://studydrift:studydrift@localhost:5432/studydrift?sslmode=disable'
  )
}

/** Grant Global Admin to an existing user (requires DATABASE_URL + go toolchain). */
export function bootstrapGlobalAdmin(email: string) {
  execSync(`go run ./cmd/bootstrap-admin -email=${email}`, {
    cwd: path.join(repoRoot, 'server'),
    env: { ...process.env, DATABASE_URL: databaseUrl(), PATH: process.env.PATH },
    stdio: 'pipe',
  })
}

export async function openGlobalPlatformSettings(page: Page) {
  await page.goto('/settings/platform')
  await expect(page.getByRole('heading', { name: /^Global platform$/i })).toBeVisible({
    timeout: 20_000,
  })
  await expect(page.getByRole('heading', { name: /^Platform features$/i })).toBeVisible({
    timeout: 15_000,
  })
}

/**
 * Run a matrix case with boolean snapshot/restore so failed assertions leave platform state intact.
 * Secret fields are never included in the snapshot payload.
 * Acquires a cross-worker lock so parallel specs cannot race on global settings.
 */
export async function withPlatformBooleanRestore<T>(
  token: string,
  fn: () => Promise<T>,
): Promise<T> {
  return withPlatformSettingsLock(async () => {
    const snapshot = await apiSnapshotPlatformBooleanSettings(token)
    try {
      return await fn()
    } finally {
      await apiRestorePlatformBooleanSettings(token, snapshot).catch(() => {})
    }
  })
}

/** Ensure a database-owned flag is off before enabling it through the UI. */
export async function ensurePlatformFeatureOff(
  token: string,
  entry: Pick<PlatformFeatureMatrixEntry, 'key' | 'runtimeKey'>,
) {
  await apiPutPlatformSettings(token, {
    [entry.key]: false,
    updateMask: [entry.key],
  })
  await apiWaitForPlatformSetting(token, entry.key, false)
  if (entry.runtimeKey) {
    await apiWaitForPlatformFeature(token, entry.runtimeKey, false)
  }
}

export async function assertPlatformUiToggleEnableFlow(
  page: Page,
  token: string,
  entry: PlatformFeatureMatrixEntry,
) {
  await ensurePlatformFeatureOff(token, entry)
  await openGlobalPlatformSettings(page)

  // Search narrows the long list for reliability.
  const search = page.getByPlaceholder('Search features…')
  await search.fill(entry.label)
  const row = platformFeatureToggleRow(page, entry.label)
  const toggle = row.getByRole('switch')
  await expect(toggle, `${entry.key} switch`).toHaveAttribute('aria-checked', 'false', {
    timeout: 10_000,
  })
  await expect(toggle).toBeEnabled()

  await toggle.click()
  await expect(page.getByRole('status').filter({ hasText: /Saved/i })).toBeVisible({
    timeout: 15_000,
  })
  await expect(toggle).toHaveAttribute('aria-checked', 'true', { timeout: 8_000 })

  await expect
    .poll(
      async () => {
        const settings = await apiGetPlatformSettings(token)
        return settings[entry.key] === true
      },
      {
        message: `${entry.key} settings API after enable (label=${entry.label})`,
        timeout: 10_000,
      },
    )
    .toBe(true)

  if (entry.runtimeKey) {
    await expect
      .poll(
        async () => {
          const features = await apiGetPlatformFeatures(token)
          return features[entry.runtimeKey!] === true
        },
        {
          message: `${entry.key} runtime ${entry.runtimeKey} after enable`,
          timeout: 10_000,
        },
      )
      .toBe(true)
  }

  await page.reload()
  await expect(page.getByRole('heading', { name: /^Platform features$/i })).toBeVisible({
    timeout: 15_000,
  })
  await page.getByPlaceholder('Search features…').fill(entry.label)
  const reloaded = platformFeatureToggleRow(page, entry.label).getByRole('switch')
  await expect(reloaded, `${entry.key} after reload`).toHaveAttribute('aria-checked', 'true', {
    timeout: 10_000,
  })
}

export async function assertPlatformApiToggleContract(
  token: string,
  entry: PlatformFeatureMatrixEntry,
) {
  const before = await apiGetPlatformSettings(token)
  const original = before[entry.key] === true
  const unrelatedKey =
    entry.key === 'maintenanceBannerEnabled' ? 'virtualClassroomEnabled' : 'maintenanceBannerEnabled'
  const unrelatedBefore = before[unrelatedKey] === true
  const target = !original

  await apiPutPlatformSettings(token, {
    [entry.key]: target,
    updateMask: [entry.key],
  })

  const after = await apiGetPlatformSettings(token)
  expect(
    after[entry.key],
    `${entry.key} settings after toggle (label=${entry.label} source=${entry.ownershipSource})`,
  ).toBe(target)
  expect(after[unrelatedKey], `${entry.key} must not change ${unrelatedKey}`).toBe(unrelatedBefore)

  if (entry.runtimeKey) {
    await apiWaitForPlatformFeature(token, entry.runtimeKey, target)
  }

  // Restore this key immediately; suite-level restore covers the rest.
  await apiPutPlatformSettings(token, {
    [entry.key]: original,
    updateMask: [entry.key],
  })
  await apiWaitForPlatformSetting(token, entry.key, original)
}
