/**
 * E2E.2 — representative Global platform UI samples (one per category).
 */
import { test, uniqueEmail, injectToken } from '../fixtures/test.js'
import { apiSignup, apiLogin } from '../fixtures/api.js'
import { UI_SAMPLE_PLATFORM_FEATURES } from '../lib/platform-feature-matrix.js'
import {
  assertPlatformUiToggleEnableFlow,
  bootstrapGlobalAdmin,
  withPlatformBooleanRestore,
} from '../lib/platform-feature-matrix-helpers.js'

const PASSWORD = process.env.E2E_ADMIN_PASSWORD ?? 'E2eTestPass1!'

test.describe.serial('Platform features UI sample', () => {
  let gaToken = ''

  test.beforeAll(async () => {
    const email = uniqueEmail('plat-ui')
    await apiSignup({ email, password: PASSWORD, displayName: 'Platform UI GA' })
    try {
      bootstrapGlobalAdmin(email)
    } catch (err) {
      test.skip(true, `bootstrap unavailable: ${err}`)
    }
    const { access_token } = await apiLogin({ email, password: PASSWORD })
    gaToken = access_token
  })

  for (const entry of UI_SAMPLE_PLATFORM_FEATURES) {
    test(`enable ${entry.key} via Global platform UI (${entry.category})`, async ({ page }) => {
      test.info().annotations.push({
        type: 'platform-feature',
        description: `${entry.key}:${entry.label}:${entry.category}`,
      })
      await injectToken(page, gaToken)
      await withPlatformBooleanRestore(gaToken, async () => {
        await assertPlatformUiToggleEnableFlow(page, gaToken, entry)
      })
    })
  }
})
