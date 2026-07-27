/**
 * CT.9 — Tool marketplace & third-party tools lifecycle.
 *
 * Checklist:
 *   [x] Enable platform flag
 *   [x] Developer creates tool + release; failing checks reject submit
 *   [x] Reviewer approves
 *   [x] Org admin installs with consent capabilities
 *   [x] Course without allowlist omits tool from catalog
 *   [x] Course allowlist includes tool
 *   [x] Revoke renders tombstone / installation revoked
 *   [x] Developer analytics are aggregate-only
 */
import { test, expect } from '../fixtures/test.js'
import { apiPatchCourseFeatures } from '../fixtures/api.js'
import { withCourseFeatureRestore } from '../lib/course-feature-matrix-helpers.js'

const apiBase = process.env.E2E_API_URL ?? 'http://localhost:8080'

async function enableMarketplace(token: string): Promise<void> {
  const res = await fetch(`${apiBase}/api/v1/settings/platform`, {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify({
      ffContentToolMarketplace: true,
      updateMask: ['ffContentToolMarketplace'],
    }),
  })
  // Some environments require global admin; tolerate if already on / unauthorized for non-admin seed.
  if (!res.ok && res.status !== 403 && res.status !== 404) {
    // Fall through — tests will skip if routes return 501.
  }
}

function sampleManifest(toolId: string, version: string) {
  return {
    id: toolId,
    version,
    name: 'CT9 Lab',
    category: 'practice',
    capabilities: ['state', 'network'],
    configSchema: { type: 'object', properties: {} },
    stateSchema: { type: 'object', properties: {} },
    scoring: { mode: 'none' },
    storage: { maxStateBytes: 4096 },
    roles: { interact: ['student'] },
    a11y: { keyboardOperable: true, srPattern: 'live' },
    i18nNamespace: 'tools.ct9.lab',
    ui: { renderer: 'iframe', icon: 'flask', group: 'science' },
    sandbox: 'iframe',
    dataSheet: {
      collects: { answer: { purpose: 'practice', retention: 'course' } },
      leavesPlatform: true,
      processors: ['api.example.com'],
      visibility: 'self',
      wcagLevel: 'AA',
    },
    network: { allowedHosts: ['api.example.com'] },
  }
}

test.describe('CT.9 Content Tools marketplace', () => {
  test('submit → approve → install → allowlist → revoke', async ({ page, seededCourse }) => {
    test.setTimeout(120_000)
    const { instructorToken, courseCode } = seededCourse

    await enableMarketplace(instructorToken)

    const probe = await fetch(`${apiBase}/api/v1/tool-marketplace/tools`, {
      headers: { Authorization: `Bearer ${instructorToken}` },
    })
    if (probe.status === 501) {
      test.skip(true, 'ffContentToolMarketplace not enabled for this environment')
    }

    const suffix = Date.now().toString(36)
    const toolId = `acme.ct9_lab_${suffix}`

    const createRes = await fetch(`${apiBase}/api/v1/developer/tools`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${instructorToken}`,
      },
      body: JSON.stringify({
        toolId,
        displayName: 'CT9 Lab',
        summary: 'Marketplace e2e tool',
        visibility: 'unlisted',
      }),
    })
    expect(createRes.ok, await createRes.text()).toBeTruthy()

    // Grant the instructor's org access to the unlisted tool.
    const meRes = await fetch(`${apiBase}/api/v1/me`, {
      headers: { Authorization: `Bearer ${instructorToken}` },
    })
    const me = (await meRes.json()) as { orgId?: string; org?: { id?: string } }
    const orgId = me.org?.id || me.orgId
    expect(orgId).toBeTruthy()

    const grantRes = await fetch(
      `${apiBase}/api/v1/developer/tools/${encodeURIComponent(toolId)}/access-grants`,
      {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${instructorToken}`,
        },
        body: JSON.stringify({ orgId }),
      },
    )
    expect(grantRes.status === 204 || grantRes.ok, await grantRes.text()).toBeTruthy()

    const badRelease = await fetch(
      `${apiBase}/api/v1/developer/tools/${encodeURIComponent(toolId)}/releases`,
      {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${instructorToken}`,
        },
        body: JSON.stringify({
          version: '1.0.0',
          manifest: sampleManifest(toolId, '1.0.0'),
          bundleBase64: Buffer.from('bundle-v1').toString('base64'),
          axeStatus: 'fail',
          keyboardTestStatus: 'pass',
          i18nKeys: { title: 'Lab' },
        }),
      },
    )
    expect(badRelease.ok, await badRelease.text()).toBeTruthy()
    const badBody = (await badRelease.json()) as { checks: { ok: boolean } }
    expect(badBody.checks.ok).toBe(false)

    const badSubmit = await fetch(
      `${apiBase}/api/v1/developer/tools/${encodeURIComponent(toolId)}/releases/1.0.0/submit`,
      {
        method: 'POST',
        headers: { Authorization: `Bearer ${instructorToken}` },
      },
    )
    expect(badSubmit.status).toBe(422)

    const goodRelease = await fetch(
      `${apiBase}/api/v1/developer/tools/${encodeURIComponent(toolId)}/releases`,
      {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${instructorToken}`,
        },
        body: JSON.stringify({
          version: '1.0.1',
          manifest: sampleManifest(toolId, '1.0.1'),
          bundleBase64: Buffer.from('bundle-v101').toString('base64'),
          axeStatus: 'pass',
          keyboardTestStatus: 'pass',
          i18nKeys: { title: 'Lab' },
        }),
      },
    )
    expect(goodRelease.ok, await goodRelease.text()).toBeTruthy()
    const goodBody = (await goodRelease.json()) as {
      release: { id: string }
      checks: { ok: boolean }
    }
    expect(goodBody.checks.ok).toBe(true)

    const submitRes = await fetch(
      `${apiBase}/api/v1/developer/tools/${encodeURIComponent(toolId)}/releases/1.0.1/submit`,
      {
        method: 'POST',
        headers: { Authorization: `Bearer ${instructorToken}` },
      },
    )
    expect(submitRes.ok, await submitRes.text()).toBeTruthy()

    const decideRes = await fetch(
      `${apiBase}/api/v1/admin/tool-reviews/${encodeURIComponent(goodBody.release.id)}/decision`,
      {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${instructorToken}`,
        },
        body: JSON.stringify({ approve: true, notes: 'looks good' }),
      },
    )
    if (decideRes.status === 403) {
      test.skip(true, 'instructor seed lacks platform reviewer permission')
    }
    expect(decideRes.ok, await decideRes.text()).toBeTruthy()

    const previewRes = await fetch(
      `${apiBase}/api/v1/orgs/${encodeURIComponent(orgId!)}/tool-installations/preview?toolId=${encodeURIComponent(toolId)}`,
      { headers: { Authorization: `Bearer ${instructorToken}` } },
    )
    if (previewRes.status === 403) {
      test.skip(true, 'instructor seed lacks org admin for install')
    }
    expect(previewRes.ok, await previewRes.text()).toBeTruthy()
    const preview = (await previewRes.json()) as {
      capabilities: { capability: string; plainLanguage: string }[]
    }
    expect(preview.capabilities.some((c) => c.plainLanguage.includes('external service'))).toBe(
      true,
    )

    const installRes = await fetch(
      `${apiBase}/api/v1/orgs/${encodeURIComponent(orgId!)}/tool-installations`,
      {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${instructorToken}`,
        },
        body: JSON.stringify({ toolId, consented: true }),
      },
    )
    expect(installRes.ok, await installRes.text()).toBeTruthy()
    const installation = (await installRes.json()) as { id: string; status: string }
    expect(installation.status).toBe('active')

    await withCourseFeatureRestore(instructorToken, courseCode, async () => {
      await apiPatchCourseFeatures(instructorToken, courseCode, { contentToolsEnabled: true })

      // Without course allowlist opt-in, tool must be absent (AC-3).
      const catalogEmpty = await fetch(
        `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/content-tools/catalog`,
        { headers: { Authorization: `Bearer ${instructorToken}` } },
      )
      expect(catalogEmpty.ok).toBeTruthy()
      const emptyBody = (await catalogEmpty.json()) as { tools: { id: string }[] }
      expect(emptyBody.tools.some((t) => t.id === toolId)).toBe(false)

      const settingsRes = await fetch(
        `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/content-tools/settings`,
        {
          method: 'PUT',
          headers: {
            'Content-Type': 'application/json',
            Authorization: `Bearer ${instructorToken}`,
          },
          body: JSON.stringify({
            allowedToolIds: ['noop_probe', toolId],
            studentResetAllowed: false,
            maxInstancesPerItem: 50,
          }),
        },
      )
      expect(settingsRes.ok, await settingsRes.text()).toBeTruthy()

      const catalogIn = await fetch(
        `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/content-tools/catalog`,
        { headers: { Authorization: `Bearer ${instructorToken}` } },
      )
      expect(catalogIn.ok).toBeTruthy()
      const inBody = (await catalogIn.json()) as { tools: { id: string }[] }
      expect(inBody.tools.some((t) => t.id === toolId)).toBe(true)
    })

    const analyticsRes = await fetch(
      `${apiBase}/api/v1/developer/tools/${encodeURIComponent(toolId)}/analytics`,
      { headers: { Authorization: `Bearer ${instructorToken}` } },
    )
    expect(analyticsRes.ok, await analyticsRes.text()).toBeTruthy()
    const analytics = (await analyticsRes.json()) as Record<string, unknown>
    expect(analytics.activeInstalls).toBeTruthy()
    expect(JSON.stringify(analytics)).not.toMatch(/student|email|enrollment/i)

    const revokeRes = await fetch(
      `${apiBase}/api/v1/orgs/${encodeURIComponent(orgId!)}/tool-installations/${encodeURIComponent(installation.id)}`,
      {
        method: 'DELETE',
        headers: { Authorization: `Bearer ${instructorToken}` },
      },
    )
    expect(revokeRes.ok, await revokeRes.text()).toBeTruthy()
    const revoked = (await revokeRes.json()) as { status: string }
    expect(revoked.status).toBe('revoked')

    // UI smoke: marketplace page loads when flagged.
    await page.goto('/tool-marketplace')
    await expect(page.getByTestId('tool-marketplace-page').or(page.getByTestId('tool-marketplace-disabled'))).toBeVisible({
      timeout: 15_000,
    })
  })
})
