/**
 * E2E.3 Priority 2 — representative off→on→off journeys (one probe or runtime toggle per family).
 */
import { expect } from '@playwright/test'
import { test, uniqueEmail } from '../fixtures/test.js'
import { apiSignup, apiLogin, apiCreateCourse, apiEnroll } from '../fixtures/api.js'
import { familiesForShard } from '../lib/feature-lifecycle-manifest.js'
import {
  assertProbeDisabled,
  assertRuntimeFlag,
  bootstrapGlobalAdmin,
  setPlatformFlag,
  withFeatureLifecycleRestore,
} from '../lib/feature-lifecycle-helpers.js'

const PASSWORD = process.env.E2E_ADMIN_PASSWORD ?? 'E2eTestPass1!'
const families = familiesForShard('priority2')

test.describe.serial('E2E.3 Priority 2 lifecycle samples', () => {
  let gaToken = ''
  let instructorToken = ''
  let courseCode = ''

  test.beforeAll(async () => {
    const gaEmail = uniqueEmail('life-p2-ga')
    await apiSignup({ email: gaEmail, password: PASSWORD, displayName: 'Lifecycle P2 GA' })
    try {
      bootstrapGlobalAdmin(gaEmail)
    } catch (err) {
      test.skip(true, `bootstrap unavailable: ${err}`)
    }
    gaToken = (await apiLogin({ email: gaEmail, password: PASSWORD })).access_token

    const instrEmail = uniqueEmail('life-p2-instr')
    instructorToken = (
      await apiSignup({ email: instrEmail, password: PASSWORD, displayName: 'Lifecycle P2 Instr' })
    ).access_token
    const course = await apiCreateCourse(instructorToken, { title: 'E2E.3 Priority 2' })
    courseCode = course.courseCode
    await apiEnroll(instructorToken, courseCode, instrEmail, 'teacher')
  })

  test('Priority 2 families: documented disabled probes and runtime toggle restore', async () => {
    test.setTimeout(Math.max(180_000, families.length * 20_000))
    await withFeatureLifecycleRestore({
      gaToken,
      instructorToken,
      courseCode,
      fn: async () => {
        for (const family of families) {
          test.info().annotations.push({
            type: 'lifecycle-family',
            description: `${family.id}:${family.label}`,
          })

          const mutableMasters = family.masterFlags.filter(
            (f) => f.kind === 'platform' && !f.alwaysOn,
          )
          if (mutableMasters.length === 0) continue

          for (const master of mutableMasters) {
            await setPlatformFlag(gaToken, master.key, true)
            await assertRuntimeFlag(gaToken, master.key, true, `${family.id} on`).catch(() => {
              // settings-only masters may lack runtime keys
            })
          }

          for (const probe of family.probes) {
            for (const gate of probe.gatedBy) {
              if (gate.kind === 'platform' && !gate.alwaysOn) {
                await setPlatformFlag(gaToken, gate.key, false)
              }
            }
            await assertProbeDisabled(probe, {
              token: instructorToken,
              courseCode,
              label: `${family.id}/${probe.id} auth off`,
            })
            await assertProbeDisabled(probe, {
              token: null,
              courseCode,
              label: `${family.id}/${probe.id} unauth`,
            })
            for (const gate of probe.gatedBy) {
              if (gate.kind === 'platform' && !gate.alwaysOn) {
                await setPlatformFlag(gaToken, gate.key, true)
              }
            }
          }

          // Runtime-only families: prove off → on restore.
          if (family.probes.length === 0) {
            for (const master of mutableMasters) {
              await setPlatformFlag(gaToken, master.key, false)
              const features = await (
                await fetch(
                  `${process.env.E2E_API_URL ?? 'http://localhost:8080'}/api/v1/platform/features`,
                  { headers: { Authorization: `Bearer ${gaToken}` } },
                )
              ).json() as Record<string, unknown>
              if (typeof features[master.key] === 'boolean') {
                expect(features[master.key], `${family.id} ${master.key} off`).toBe(false)
              }
              await setPlatformFlag(gaToken, master.key, true)
              if (typeof features[master.key] === 'boolean') {
                await assertRuntimeFlag(gaToken, master.key, true, `${family.id} restored`)
              }
            }
          }
        }
      },
    })
  })
})
