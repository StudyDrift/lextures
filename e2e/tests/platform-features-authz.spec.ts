/**
 * E2E.2 — platform settings authz, omit preservation, save failure, keyboard, env read-only.
 */
import { expect, test, uniqueEmail, injectToken } from '../fixtures/test.js'
import {
  apiSignup,
  apiLogin,
  apiGetPlatformSettings,
  apiGetPlatformSettingsRaw,
  apiPutPlatformSettings,
  apiPutPlatformSettingsRaw,
  apiSnapshotPlatformBooleanSettings,
  apiRestorePlatformBooleanSettings,
} from '../fixtures/api.js'
import {
  bootstrapGlobalAdmin,
  openGlobalPlatformSettings,
  platformFeatureToggleRow,
  withPlatformBooleanRestore,
  withPlatformSettingsLock,
} from '../lib/platform-feature-matrix-helpers.js'

const PASSWORD = process.env.E2E_ADMIN_PASSWORD ?? 'E2eTestPass1!'
const apiBase = process.env.E2E_API_URL ?? 'http://localhost:8080'

async function createGlobalAdmin(): Promise<string> {
  const email = uniqueEmail('plat-ga')
  await apiSignup({ email, password: PASSWORD, displayName: 'Platform GA' })
  try {
    bootstrapGlobalAdmin(email)
  } catch (err) {
    test.skip(true, `bootstrap unavailable: ${err}`)
  }
  const { access_token } = await apiLogin({ email, password: PASSWORD })
  return access_token
}

async function grantOrgAdmin(actorToken: string, orgId: string, userId: string) {
  const res = await fetch(`${apiBase}/api/v1/orgs/${orgId}/role-grants`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${actorToken}`,
    },
    body: JSON.stringify({ userId, role: 'org_admin' }),
  })
  if (!res.ok) {
    throw new Error(`grant org_admin failed: ${await res.text()}`)
  }
}

test.describe.serial('Platform features authz & UI contracts', () => {
  test('anonymous PUT returns 401', async () => {
    const res = await apiPutPlatformSettingsRaw(null, {
      h5pEnabled: true,
      updateMask: ['h5pEnabled'],
    })
    expect(res.status).toBe(401)
  })

  test('learner PUT returns 403 and does not mutate flags', async ({ seededCourse }) => {
    const gaToken = await createGlobalAdmin()
    await withPlatformBooleanRestore(gaToken, async () => {
      await apiPutPlatformSettings(gaToken, {
        h5pEnabled: false,
        updateMask: ['h5pEnabled'],
      })
      const before = await apiGetPlatformSettings(gaToken)
      const res = await apiPutPlatformSettingsRaw(seededCourse.studentToken, {
        h5pEnabled: true,
        updateMask: ['h5pEnabled'],
      })
      expect(res.status, 'learner platform PUT').toBe(403)
      const after = await apiGetPlatformSettings(gaToken)
      expect(after.h5pEnabled).toBe(before.h5pEnabled)
    })
  })

  test('instructor PUT returns 403 and does not mutate flags', async ({ seededCourse }) => {
    const gaToken = await createGlobalAdmin()
    await withPlatformBooleanRestore(gaToken, async () => {
      await apiPutPlatformSettings(gaToken, {
        scormIngestionEnabled: false,
        updateMask: ['scormIngestionEnabled'],
      })
      const before = await apiGetPlatformSettings(gaToken)
      const res = await apiPutPlatformSettingsRaw(seededCourse.instructorToken, {
        scormIngestionEnabled: true,
        updateMask: ['scormIngestionEnabled'],
      })
      expect(res.status, 'instructor platform PUT').toBe(403)
      const after = await apiGetPlatformSettings(gaToken)
      expect(after.scormIngestionEnabled).toBe(before.scormIngestionEnabled)
    })
  })

  test('org admin PUT returns 403 and does not mutate flags', async () => {
    const gaToken = await createGlobalAdmin()
    const orgAdminEmail = uniqueEmail('plat-oa')
    await apiSignup({ email: orgAdminEmail, password: PASSWORD, displayName: 'Org Admin' })
    const orgAdmin = await apiLogin({ email: orgAdminEmail, password: PASSWORD })

    const meRes = await fetch(`${apiBase}/api/v1/me`, {
      headers: { Authorization: `Bearer ${orgAdmin.access_token}` },
    })
    expect(meRes.ok).toBe(true)
    const me = (await meRes.json()) as { id: string; org?: { id: string }; orgId?: string }
    let orgId = me.org?.id ?? me.orgId
    if (!orgId) {
      const caps = await fetch(`${apiBase}/api/v1/me/org-role-capabilities`, {
        headers: { Authorization: `Bearer ${orgAdmin.access_token}` },
      })
      orgId = ((await caps.json()) as { orgId: string }).orgId
    }
    await grantOrgAdmin(gaToken, orgId!, me.id)

    await withPlatformBooleanRestore(gaToken, async () => {
      await apiPutPlatformSettings(gaToken, {
        oerLibraryEnabled: false,
        updateMask: ['oerLibraryEnabled'],
      })
      const before = await apiGetPlatformSettings(gaToken)
      const res = await apiPutPlatformSettingsRaw(orgAdmin.access_token, {
        oerLibraryEnabled: true,
        updateMask: ['oerLibraryEnabled'],
      })
      expect(res.status, 'org admin platform PUT').toBe(403)
      const getRes = await apiGetPlatformSettingsRaw(orgAdmin.access_token)
      expect([401, 403]).toContain(getRes.status)
      const after = await apiGetPlatformSettings(gaToken)
      expect(after.oerLibraryEnabled).toBe(before.oerLibraryEnabled)
    })
  })

  test('single-field update preserves unrelated settings', async () => {
    const gaToken = await createGlobalAdmin()
    await withPlatformBooleanRestore(gaToken, async () => {
      await apiPutPlatformSettings(gaToken, {
        h5pEnabled: true,
        scormIngestionEnabled: true,
        oerLibraryEnabled: false,
        updateMask: ['h5pEnabled', 'scormIngestionEnabled', 'oerLibraryEnabled'],
      })
      const before = await apiGetPlatformSettings(gaToken)
      await apiPutPlatformSettings(gaToken, {
        h5pEnabled: false,
        updateMask: ['h5pEnabled'],
      })
      const after = await apiGetPlatformSettings(gaToken)
      expect(after.h5pEnabled).toBe(false)
      expect(after.scormIngestionEnabled).toBe(before.scormIngestionEnabled)
      expect(after.oerLibraryEnabled).toBe(before.oerLibraryEnabled)
      // Secret placeholders must remain masked strings when present.
      if (typeof after.smtpPassword === 'string' && after.smtpPassword.length > 0) {
        expect(after.smtpPassword).toMatch(/•|^\*+$/)
      }
    })
  })

  test('settings UI surfaces an error alert when PUT fails', async ({ page }) => {
    const gaToken = await createGlobalAdmin()
    await injectToken(page, gaToken)
    await withPlatformSettingsLock(async () => {
    await page.route('**/api/v1/settings/platform', async (route) => {
      if (route.request().method() === 'PUT') {
        await route.fulfill({
          status: 500,
          contentType: 'application/json',
          body: JSON.stringify({ error: { message: 'Simulated platform save failure' } }),
        })
        return
      }
      await route.continue()
    })

    await openGlobalPlatformSettings(page)
    await page.getByPlaceholder('Search features…').fill('Interactive H5P')
    const toggle = platformFeatureToggleRow(page, 'Interactive H5P content').getByRole('switch')
    await expect(toggle).toBeVisible({ timeout: 10_000 })
    await toggle.click()

    await expect(page.getByRole('alert')).toBeVisible({ timeout: 10_000 })
    await expect(page.getByRole('status').filter({ hasText: /Saved/i })).toHaveCount(0)
    await page.unroute('**/api/v1/settings/platform')
    })
  })

  test('feature switch is keyboard activatable with Space', async ({ page }) => {
    const gaToken = await createGlobalAdmin()
    await injectToken(page, gaToken)
    await withPlatformBooleanRestore(gaToken, async () => {
      await apiPutPlatformSettings(gaToken, {
        h5pEnabled: false,
        updateMask: ['h5pEnabled'],
      })
      await openGlobalPlatformSettings(page)
      await page.getByPlaceholder('Search features…').fill('Interactive H5P')
      const toggle = platformFeatureToggleRow(page, 'Interactive H5P content').getByRole('switch')
      await expect(toggle).toHaveAttribute('aria-checked', 'false')
      await toggle.focus()
      await expect(toggle).toBeFocused()
      await page.keyboard.press('Space')
      await expect(page.getByRole('status').filter({ hasText: /Saved/i })).toBeVisible({
        timeout: 15_000,
      })
      await expect(toggle).toHaveAttribute('aria-checked', 'true')
    })
  })

  test('environment-owned toggle is read-only and keeps its value', async ({ page }) => {
    const gaToken = await createGlobalAdmin()
    await injectToken(page, gaToken)

    await withPlatformSettingsLock(async () => {
    await page.route('**/api/v1/settings/platform', async (route) => {
      if (route.request().method() === 'GET') {
        const res = await route.fetch()
        const data = (await res.json()) as Record<string, unknown>
        const sources = {
          ...((data.sources as Record<string, string> | undefined) ?? {}),
          annotationEnabled: 'environment',
        }
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            ...data,
            annotationEnabled: true,
            sources,
          }),
        })
        return
      }
      // Block PUTs so a click cannot claim persistence.
      if (route.request().method() === 'PUT') {
        await route.fulfill({
          status: 500,
          contentType: 'application/json',
          body: JSON.stringify({ error: { message: 'env-owned should not PUT' } }),
        })
        return
      }
      await route.continue()
    })

    await openGlobalPlatformSettings(page)
    await page.getByPlaceholder('Search features…').fill('Annotations')
    const row = platformFeatureToggleRow(page, 'Annotations')
    const toggle = row.getByRole('switch')
    await expect(toggle).toBeDisabled({ timeout: 10_000 })
    await expect(toggle).toHaveAttribute('aria-checked', 'true')
    await expect(row.getByText(/Environment/i)).toBeVisible()
    await expect(page.getByTestId('feature-toggle-disabled-reason')).toContainText(
      /environment configuration/i,
    )
    // Attempted activation must not show Saved.
    await toggle.click({ force: true }).catch(() => {})
    await expect(page.getByRole('status').filter({ hasText: /Saved/i })).toHaveCount(0)
    await page.unroute('**/api/v1/settings/platform')
    })
  })

  test('teardown restore returns boolean snapshot without clearing secrets', async () => {
    const gaToken = await createGlobalAdmin()
    await withPlatformBooleanRestore(gaToken, async () => {
      const snapshot = await apiSnapshotPlatformBooleanSettings(gaToken)
      expect(Object.keys(snapshot).length).toBeGreaterThan(50)
      expect(snapshot).not.toHaveProperty('smtpPassword')
      expect(snapshot).not.toHaveProperty('samlSpPrivateKeyPem')

      await apiPutPlatformSettings(gaToken, {
        itemAnalysisEnabled: !snapshot.itemAnalysisEnabled,
        updateMask: ['itemAnalysisEnabled'],
      })
      await apiRestorePlatformBooleanSettings(gaToken, snapshot)
      const after = await apiGetPlatformSettings(gaToken)
      expect(after.itemAnalysisEnabled).toBe(snapshot.itemAnalysisEnabled)
    })
  })
})
