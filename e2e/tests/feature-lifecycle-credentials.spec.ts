/**
 * E2E.3 Priority 1 — credentials shard (transcripts + parent portal).
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
  getParentNotificationPrefs,
  patchParentNotificationPrefs,
  setPlatformFlag,
  withFeatureLifecycleRestore,
} from '../lib/feature-lifecycle-helpers.js'

const PASSWORD = process.env.E2E_ADMIN_PASSWORD ?? 'E2eTestPass1!'
const families = familiesForShard('credentials')

test.describe.serial('E2E.3 credentials lifecycle', () => {
  let gaToken = ''
  let parentToken = ''

  test.beforeAll(async () => {
    const gaEmail = uniqueEmail('life-cred-ga')
    await apiSignup({ email: gaEmail, password: PASSWORD, displayName: 'Lifecycle Cred GA' })
    try {
      bootstrapGlobalAdmin(gaEmail)
    } catch (err) {
      test.skip(true, `bootstrap unavailable: ${err}`)
    }
    gaToken = (await apiLogin({ email: gaEmail, password: PASSWORD })).access_token

    parentToken = (
      await apiSignup({
        email: uniqueEmail('life-parent'),
        password: PASSWORD,
        displayName: 'Lifecycle Parent',
        accountType: 'parent',
      })
    ).access_token
  })

  test('transcripts: master off → 404; inbound parent authority; re-enable restores runtime', async () => {
    test.setTimeout(120_000)
    const family = families.find((f) => f.id === 'transcripts')!
    await withFeatureLifecycleRestore({
      gaToken,
      fn: async () => {
        await setPlatformFlag(gaToken, 'ffTranscripts', true)
        await setPlatformFlag(gaToken, 'ffTranscriptInbound', true)
        await assertRuntimeFlag(gaToken, 'ffTranscripts', true, 'transcripts on')

        const configProbe = family.probes.find((p) => p.id === 'transcripts-config')!
        const inboundProbe = family.probes.find((p) => p.id === 'transcript-inbound-list')!

        // Unauth when off must follow feature-first contract (AC-5).
        await setPlatformFlag(gaToken, 'ffTranscripts', false)
        await assertProbeDisabled(configProbe, {
          token: gaToken,
          label: 'transcripts auth off',
        })
        await assertProbeDisabled(configProbe, {
          token: null,
          label: 'transcripts unauth off',
        })

        await setPlatformFlag(gaToken, 'ffTranscripts', true)
        await assertRuntimeFlag(gaToken, 'ffTranscripts', true, 'transcripts re-enabled')

        const edge = family.edges.find(
          (e) => e.parent.key === 'ffTranscripts' && e.child.key === 'ffTranscriptInbound',
        )!
        await assertDependencyTruthTable({
          family,
          edge,
          gaToken,
          childProbe: inboundProbe,
          // When both on: auth parent account may get 200 empty list or 403 depending on role.
          bothOnAllowedStatuses: [200, 403],
        })
      },
    })
  })

  test('parent portal: prefs survive flag toggles; runtime v1/v2; documented API gap', async () => {
    test.setTimeout(120_000)
    const family = families.find((f) => f.id === 'parent-portal')!
    await withFeatureLifecycleRestore({
      gaToken,
      fn: async () => {
        await setPlatformFlag(gaToken, 'ffParentPortal', true)
        await setPlatformFlag(gaToken, 'ffParentPortalV2', true)
        await assertRuntimeFlag(gaToken, 'ffParentPortal', true, 'parent portal on')
        await assertRuntimeFlag(gaToken, 'ffParentPortalV2', true, 'parent portal v2 on')

        const before = await getParentNotificationPrefs(parentToken)
        const toggled = !(before.gradePosted === true)
        await patchParentNotificationPrefs(parentToken, {
          gradePosted: toggled,
          missingAssignment: true,
        })

        const childrenProbe = family.probes.find((p) => p.id === 'parent-children')!
        // Documented gap: API is account-type gated, not feature-gated.
        await setPlatformFlag(gaToken, 'ffParentPortal', false)
        await assertRuntimeFlag(gaToken, 'ffParentPortal', false, 'parent portal off')
        const resOff = await fetch(
          `${process.env.E2E_API_URL ?? 'http://localhost:8080'}/api/v1/parent/children`,
          { headers: { Authorization: `Bearer ${parentToken}` } },
        )
        // Parent account still reaches handler (200/empty) or 403 for non-parent — not feature 404.
        expect([200, 403], 'parent API not feature-gated').toContain(resOff.status)
        await assertProbeDisabled(childrenProbe, {
          token: null,
          label: 'parent children unauth',
        })

        await setPlatformFlag(gaToken, 'ffParentPortal', true)
        await setPlatformFlag(gaToken, 'ffParentPortalV2', false)
        await assertRuntimeFlag(gaToken, 'ffParentPortalV2', false, 'v2 off')

        const after = await getParentNotificationPrefs(parentToken)
        expect(after.gradePosted, 'prefs preserved across kill switch').toBe(toggled)

        await setPlatformFlag(gaToken, 'ffParentPortalV2', true)
        await assertRuntimeFlag(gaToken, 'ffParentPortalV2', true, 'v2 re-enabled')
      },
    })
  })
})
