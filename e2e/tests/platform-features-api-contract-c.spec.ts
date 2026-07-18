/**
 * E2E.2 — platform feature API contract (shard C).
 * Toggles each database-owned flag via PUT, asserts settings (+ runtime when mapped), restores.
 */
import { test, uniqueEmail } from '../fixtures/test.js'
import { apiSignup, apiLogin } from '../fixtures/api.js'
import { apiContractShards } from '../lib/platform-feature-matrix.js'
import {
  assertPlatformApiToggleContract,
  bootstrapGlobalAdmin,
  withPlatformBooleanRestore,
} from '../lib/platform-feature-matrix-helpers.js'

const PASSWORD = process.env.E2E_ADMIN_PASSWORD ?? 'E2eTestPass1!'
const entries = apiContractShards(3)[2]!

test.describe.serial('Platform features API contract C', () => {
  let gaToken = ''

  test.beforeAll(async () => {
    const email = uniqueEmail('plat-api-c')
    await apiSignup({ email, password: PASSWORD, displayName: 'Platform API GA' })
    try {
      bootstrapGlobalAdmin(email)
    } catch (err) {
      test.skip(true, `bootstrap unavailable: ${err}`)
    }
    const { access_token } = await apiLogin({ email, password: PASSWORD })
    gaToken = access_token
  })

  test(`shard c toggles ${entries.length} database-owned flags`, async () => {
    test.setTimeout(Math.max(120_000, entries.length * 2_500))
    await withPlatformBooleanRestore(gaToken, async () => {
      for (const entry of entries) {
        test.info().annotations.push({
          type: 'platform-feature',
          description: `${entry.key}:${entry.category}:${entry.ownershipSource}`,
        })
        await assertPlatformApiToggleContract(gaToken, entry)
      }
    })
  })
})
