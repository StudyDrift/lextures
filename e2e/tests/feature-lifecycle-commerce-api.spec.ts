/**
 * E2E.3 Priority 1 — commerce/API shard (payments/tax/revenue + public API/tokens).
 */
import { expect } from '@playwright/test'
import { test, uniqueEmail } from '../fixtures/test.js'
import { apiSignup, apiLogin } from '../fixtures/api.js'
import { familiesForShard } from '../lib/feature-lifecycle-manifest.js'
import {
  assertDependencyTruthTable,
  assertProbeDisabled,
  assertRuntimeFlag,
  bootstrapGlobalAdmin,
  createAccessKey,
  setPlatformFlag,
  withFeatureLifecycleRestore,
} from '../lib/feature-lifecycle-helpers.js'

const PASSWORD = process.env.E2E_ADMIN_PASSWORD ?? 'E2eTestPass1!'
const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'
const families = familiesForShard('commerce-api')

test.describe.serial('E2E.3 commerce/API lifecycle', () => {
  let gaToken = ''
  let userToken = ''

  test.beforeAll(async () => {
    const gaEmail = uniqueEmail('life-com-ga')
    await apiSignup({ email: gaEmail, password: PASSWORD, displayName: 'Lifecycle Commerce GA' })
    try {
      bootstrapGlobalAdmin(gaEmail)
    } catch (err) {
      test.skip(true, `bootstrap unavailable: ${err}`)
    }
    gaToken = (await apiLogin({ email: gaEmail, password: PASSWORD })).access_token
    userToken = (
      await apiSignup({
        email: uniqueEmail('life-com-user'),
        password: PASSWORD,
        displayName: 'Lifecycle Commerce User',
      })
    ).access_token
  })

  test('payments/tax/revenue: kill switches and parent→child tax authority', async () => {
    test.setTimeout(120_000)
    const family = families.find((f) => f.id === 'payments-billing')!
    await withFeatureLifecycleRestore({
      gaToken,
      fn: async () => {
        await setPlatformFlag(gaToken, 'ffPaymentsEnabled', true)
        await setPlatformFlag(gaToken, 'ffStripeBilling', true)
        await setPlatformFlag(gaToken, 'ffTaxCollection', true)
        await setPlatformFlag(gaToken, 'ffRevenueShare', true)

        const txProbe = family.probes.find((p) => p.id === 'my-transactions')!
        const taxProbe = family.probes.find((p) => p.id === 'tax-quote')!
        const revProbe = family.probes.find((p) => p.id === 'creator-earnings')!

        await assertProbeDisabled(txProbe, { token: null, label: 'transactions unauth' })

        // Payments kill switch requires BOTH abstraction + Stripe off.
        await setPlatformFlag(gaToken, 'ffPaymentsEnabled', false)
        await setPlatformFlag(gaToken, 'ffStripeBilling', false)
        await assertProbeDisabled(txProbe, {
          token: userToken,
          label: 'transactions both billing flags off',
        })

        await setPlatformFlag(gaToken, 'ffPaymentsEnabled', true)
        await setPlatformFlag(gaToken, 'ffStripeBilling', true)
        await assertRuntimeFlag(gaToken, 'ffStripeBilling', true, 'stripe re-enabled')

        const onRes = await fetch(`${API_BASE}/api/v1/me/transactions`, {
          headers: { Authorization: `Bearer ${userToken}` },
        })
        expect([200, 404], 'transactions when billing on').toContain(onRes.status)

        await setPlatformFlag(gaToken, 'ffRevenueShare', false)
        await assertProbeDisabled(revProbe, {
          token: userToken,
          label: 'revenue share off',
        })
        await setPlatformFlag(gaToken, 'ffRevenueShare', true)

        const stripeToTax = family.edges.find(
          (e) => e.parent.key === 'ffStripeBilling' && e.child.key === 'ffTaxCollection',
        )!
        await assertDependencyTruthTable({
          family,
          edge: stripeToTax,
          gaToken,
          childProbe: taxProbe,
          // Both on: may 400 (bad course) / 404 (provider) / 429 — not feature-disabled.
          bothOnAllowedStatuses: [200, 400, 404, 429, 503],
        })
      },
    })
  })

  test('public API / API tokens: management 501 when off; access-key requests 503 when public API off', async () => {
    test.setTimeout(120_000)
    const family = families.find((f) => f.id === 'public-api-tokens')!
    await withFeatureLifecycleRestore({
      gaToken,
      fn: async () => {
        await setPlatformFlag(gaToken, 'ffApiTokens', true)
        await setPlatformFlag(gaToken, 'ffPublicApi', true)
        await assertRuntimeFlag(gaToken, 'ffApiTokens', true, 'api tokens on')

        const key = await createAccessKey(userToken, `e2e-life-${Date.now()}`)
        expect(key.token).toMatch(/^ltk_/)

        const listProbe = family.probes.find((p) => p.id === 'access-keys-list')!
        await assertProbeDisabled(listProbe, { token: null, label: 'access-keys unauth' })

        await setPlatformFlag(gaToken, 'ffApiTokens', false)
        await assertProbeDisabled(listProbe, {
          token: userToken,
          label: 'access-keys management off',
        })

        // Public API kill switch: ltk_ bearer gets 503 (not JWT SPA path).
        await setPlatformFlag(gaToken, 'ffApiTokens', true)
        await setPlatformFlag(gaToken, 'ffPublicApi', false)
        await assertRuntimeFlag(gaToken, 'ffPublicApi', false, 'public api off')
        const pubOff = await fetch(`${API_BASE}/api/v1/courses`, {
          headers: { Authorization: `Bearer ${key.token}` },
        })
        expect(pubOff.status, 'ltk_ when public API off').toBe(503)

        await setPlatformFlag(gaToken, 'ffPublicApi', true)
        const pubOn = await fetch(`${API_BASE}/api/v1/courses`, {
          headers: { Authorization: `Bearer ${key.token}` },
        })
        expect([200, 401], 'ltk_ when public API on').toContain(pubOn.status)

        // Key still listed after re-enable (non-destructive).
        const listOn = await fetch(`${API_BASE}/api/v1/me/access-keys`, {
          headers: { Authorization: `Bearer ${userToken}` },
        })
        expect(listOn.status).toBe(200)
        const body = (await listOn.json()) as { tokens?: Array<{ id: string }> }
        expect(body.tokens?.some((t) => t.id === key.id)).toBe(true)
      },
    })
  })
})
