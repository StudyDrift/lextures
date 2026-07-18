/**
 * E2E.3 Priority 1 — AI shard (persistent tutor, study buddy, lesson generator).
 * Stubs inference: only asserts gates; no provider calls while flags are off.
 */
import { expect } from '@playwright/test'
import { test, uniqueEmail } from '../fixtures/test.js'
import { apiSignup, apiLogin, apiCreateCourse, apiEnroll } from '../fixtures/api.js'
import { familiesForShard } from '../lib/feature-lifecycle-manifest.js'
import {
  assertProbeDisabled,
  assertRuntimeFlag,
  bootstrapGlobalAdmin,
  setCourseFlag,
  setPlatformFlag,
  withFeatureLifecycleRestore,
} from '../lib/feature-lifecycle-helpers.js'

const PASSWORD = process.env.E2E_ADMIN_PASSWORD ?? 'E2eTestPass1!'
const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'
const families = familiesForShard('ai')

test.describe.serial('E2E.3 AI lifecycle', () => {
  let gaToken = ''
  let instructorToken = ''
  let courseCode = ''

  test.beforeAll(async () => {
    const gaEmail = uniqueEmail('life-ai-ga')
    await apiSignup({ email: gaEmail, password: PASSWORD, displayName: 'Lifecycle AI GA' })
    try {
      bootstrapGlobalAdmin(gaEmail)
    } catch (err) {
      test.skip(true, `bootstrap unavailable: ${err}`)
    }
    gaToken = (await apiLogin({ email: gaEmail, password: PASSWORD })).access_token

    const instrEmail = uniqueEmail('life-ai-instr')
    instructorToken = (
      await apiSignup({ email: instrEmail, password: PASSWORD, displayName: 'Lifecycle AI Instr' })
    ).access_token
    const course = await apiCreateCourse(instructorToken, { title: 'E2E.3 AI Course' })
    courseCode = course.courseCode
    await apiEnroll(instructorToken, courseCode, instrEmail, 'teacher')
  })

  test('AI family: platform kill switches, course tutor gate, re-enable, auth contracts', async () => {
    test.setTimeout(180_000)
    const family = families.find((f) => f.id === 'ai-capabilities')!
    await withFeatureLifecycleRestore({
      gaToken,
      instructorToken,
      courseCode,
      fn: async () => {
        await setPlatformFlag(gaToken, 'ffPersistentTutor', true)
        await setPlatformFlag(gaToken, 'ffAiStudyBuddy', true)
        await setPlatformFlag(gaToken, 'ffLessonGenerator', true)
        await setCourseFlag(instructorToken, courseCode, 'aiTutorEnabled', true)

        const tutorProbe = family.probes.find((p) => p.id === 'tutor-sessions')!
        const buddyProbe = family.probes.find((p) => p.id === 'study-buddy-prompts')!
        const lessonProbe = family.probes.find((p) => p.id === 'lesson-generator')!

        // Feature-first unauth when off (AC-5 documented contract).
        await setPlatformFlag(gaToken, 'ffPersistentTutor', false)
        await assertProbeDisabled(tutorProbe, {
          token: instructorToken,
          courseCode,
          label: 'tutor sessions off',
        })
        await assertProbeDisabled(tutorProbe, {
          token: null,
          courseCode,
          label: 'tutor sessions unauth feature-first',
        })

        await setPlatformFlag(gaToken, 'ffPersistentTutor', true)
        await setCourseFlag(instructorToken, courseCode, 'aiTutorEnabled', true)
        const onRes = await fetch(
          `${API_BASE}/api/v1/courses/${encodeURIComponent(courseCode)}/tutor/sessions`,
          { headers: { Authorization: `Bearer ${instructorToken}` } },
        )
        // Empty list or provider/config errors — not feature-disabled.
        expect([200, 403, 503], 'tutor sessions when on').toContain(onRes.status)

        // Course flag off while platform on → 403 (AC-3).
        await setCourseFlag(instructorToken, courseCode, 'aiTutorEnabled', false)
        const courseOff = await fetch(
          `${API_BASE}/api/v1/courses/${encodeURIComponent(courseCode)}/tutor/sessions`,
          { headers: { Authorization: `Bearer ${instructorToken}` } },
        )
        expect(courseOff.status, 'course aiTutorEnabled off').toBe(403)
        await setCourseFlag(instructorToken, courseCode, 'aiTutorEnabled', true)

        await setPlatformFlag(gaToken, 'ffAiStudyBuddy', false)
        await assertProbeDisabled(buddyProbe, {
          token: instructorToken,
          courseCode,
          label: 'study buddy off',
        })
        await setPlatformFlag(gaToken, 'ffAiStudyBuddy', true)
        await assertRuntimeFlag(gaToken, 'ffAiStudyBuddy', true, 'study buddy re-enabled')

        await setPlatformFlag(gaToken, 'ffLessonGenerator', false)
        await assertProbeDisabled(lessonProbe, {
          token: instructorToken,
          courseCode,
          label: 'lesson generator off',
        })
        // Auth-first: unauthenticated requests get 401 before feature disclosure.
        await assertProbeDisabled(lessonProbe, {
          token: null,
          courseCode,
          label: 'lesson generator unauth auth-first',
        })
        await setPlatformFlag(gaToken, 'ffLessonGenerator', true)

        // Parent/child truth table with distinct disabled statuses (platform 404 vs course 403).
        const combos: Array<{
          platform: boolean
          course: boolean
          expected: number
        }> = [
          { platform: false, course: false, expected: 404 },
          { platform: false, course: true, expected: 404 },
          { platform: true, course: false, expected: 403 },
          { platform: true, course: true, expected: 200 },
        ]
        for (const combo of combos) {
          await setPlatformFlag(gaToken, 'ffPersistentTutor', combo.platform)
          await setCourseFlag(instructorToken, courseCode, 'aiTutorEnabled', combo.course)
          const res = await fetch(
            `${API_BASE}/api/v1/courses/${encodeURIComponent(courseCode)}/tutor/sessions`,
            { headers: { Authorization: `Bearer ${instructorToken}` } },
          )
          if (combo.platform && combo.course) {
            expect([200, 403, 503], `AI both on got ${res.status}`).toContain(res.status)
          } else {
            expect(res.status, `platform=${combo.platform} course=${combo.course}`).toBe(
              combo.expected,
            )
          }
        }
      },
    })
  })
})
