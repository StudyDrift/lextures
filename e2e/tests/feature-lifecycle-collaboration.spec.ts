/**
 * E2E.3 Priority 1 — collaboration shard (boards + live quizzes).
 * Kill-switch, data preservation, parent/child truth tables, auth-first probes.
 */
import { expect } from '@playwright/test'
import { test, uniqueEmail, injectToken } from '../fixtures/test.js'
import { apiSignup, apiLogin, apiCreateCourse, apiEnroll } from '../fixtures/api.js'
import { familiesForShard } from '../lib/feature-lifecycle-manifest.js'
import {
  assertCourseWebOffState,
  assertCourseWebOnState,
  assertDependencyTruthTable,
  assertProbeDisabled,
  bootstrapGlobalAdmin,
  createBoard,
  createQuizKit,
  listBoards,
  listQuizKits,
  setCourseFlag,
  setPlatformFlag,
  withFeatureLifecycleRestore,
} from '../lib/feature-lifecycle-helpers.js'

const PASSWORD = process.env.E2E_ADMIN_PASSWORD ?? 'E2eTestPass1!'
const families = familiesForShard('collaboration')

test.describe.serial('E2E.3 collaboration lifecycle', () => {
  let gaToken = ''
  let instructorToken = ''
  let courseCode = ''

  test.beforeAll(async () => {
    const gaEmail = uniqueEmail('life-collab-ga')
    await apiSignup({ email: gaEmail, password: PASSWORD, displayName: 'Lifecycle GA' })
    try {
      bootstrapGlobalAdmin(gaEmail)
    } catch (err) {
      test.skip(true, `bootstrap unavailable: ${err}`)
    }
    gaToken = (await apiLogin({ email: gaEmail, password: PASSWORD })).access_token

    const instrEmail = uniqueEmail('life-collab-instr')
    instructorToken = (
      await apiSignup({ email: instrEmail, password: PASSWORD, displayName: 'Lifecycle Instr' })
    ).access_token
    const course = await apiCreateCourse(instructorToken, { title: 'E2E.3 Collaboration' })
    courseCode = course.courseCode
    await apiEnroll(instructorToken, courseCode, instrEmail, 'teacher')
  })

  test('visual boards: course kill switch, data preservation, web off-state', async ({ page }) => {
    test.setTimeout(120_000)
    const family = families.find((f) => f.id === 'visual-boards')!
    await withFeatureLifecycleRestore({
      gaToken,
      instructorToken,
      courseCode,
      fn: async () => {
        await setCourseFlag(instructorToken, courseCode, 'visualBoardsEnabled', true)
        const board = await createBoard(instructorToken, courseCode, `Lifecycle Board ${Date.now()}`)
        expect(board.id).toBeTruthy()

        const listProbe = family.probes.find((p) => p.id === 'list-boards')!
        await assertProbeDisabled(listProbe, {
          token: null,
          courseCode,
          label: 'boards unauth',
        })

        await setCourseFlag(instructorToken, courseCode, 'visualBoardsEnabled', false)
        await assertProbeDisabled(listProbe, {
          token: instructorToken,
          courseCode,
          label: 'boards course-off',
        })

        await injectToken(page, instructorToken)
        await assertCourseWebOffState(page, family, courseCode)

        await setCourseFlag(instructorToken, courseCode, 'visualBoardsEnabled', true)
        const restored = await listBoards(instructorToken, courseCode)
        expect(
          restored.some((b) => b.id === board.id),
          'board preserved after re-enable',
        ).toBe(true)
        await assertCourseWebOnState(page, family, courseCode)

        // Realtime child: platform off → WS 404 even with course on.
        const rtProbe = family.probes.find((p) => p.id === 'boards-realtime-ws-upgrade')!
        await setPlatformFlag(gaToken, 'ffBoardsRealtime', false)
        await assertProbeDisabled(rtProbe, {
          token: instructorToken,
          courseCode,
          label: 'boards realtime off',
        })
        await assertProbeDisabled(rtProbe, {
          token: null,
          courseCode,
          label: 'boards realtime unauth feature-first',
        })
      },
    })
  })

  test('live quizzes: course + hosting kill switches, kit preservation, truth tables', async ({
    page,
  }) => {
    test.setTimeout(180_000)
    const family = families.find((f) => f.id === 'interactive-quizzes')!
    await withFeatureLifecycleRestore({
      gaToken,
      instructorToken,
      courseCode,
      fn: async () => {
        await setCourseFlag(instructorToken, courseCode, 'interactiveQuizzesEnabled', true)
        await setPlatformFlag(gaToken, 'ffIqLiveHosting', true)
        await setPlatformFlag(gaToken, 'ffIqAiGeneration', true)

        const kit = await createQuizKit(instructorToken, courseCode, `Lifecycle Kit ${Date.now()}`)
        expect(kit.id).toBeTruthy()

        const kitsProbe = family.probes.find((p) => p.id === 'list-kits')!
        const gamesProbe = family.probes.find((p) => p.id === 'create-game')!
        const aiProbe = family.probes.find((p) => p.id === 'ai-generation')!

        await assertProbeDisabled(kitsProbe, {
          token: null,
          courseCode,
          label: 'kits unauth',
        })

        // Course off: kits unavailable (AC-3 course-scoped).
        await setCourseFlag(instructorToken, courseCode, 'interactiveQuizzesEnabled', false)
        await assertProbeDisabled(kitsProbe, {
          token: instructorToken,
          courseCode,
          label: 'kits course-off',
        })
        await injectToken(page, instructorToken)
        await assertCourseWebOffState(page, family, courseCode)

        await setCourseFlag(instructorToken, courseCode, 'interactiveQuizzesEnabled', true)
        const kits = await listQuizKits(instructorToken, courseCode)
        expect(
          kits.some((k) => k.id === kit.id),
          'kit preserved after re-enable',
        ).toBe(true)
        await assertCourseWebOnState(page, family, courseCode)

        // Hosting off with course on: games 404, kits still list.
        await setPlatformFlag(gaToken, 'ffIqLiveHosting', false)
        await assertProbeDisabled(gamesProbe, {
          token: instructorToken,
          courseCode,
          label: 'games hosting-off',
        })
        const kitsWhileHostingOff = await listQuizKits(instructorToken, courseCode)
        expect(kitsWhileHostingOff.some((k) => k.id === kit.id)).toBe(true)
        await setPlatformFlag(gaToken, 'ffIqLiveHosting', true)

        // Parent course off / child AI on → parent authoritative (AC-2).
        const courseToAi = family.edges.find(
          (e) => e.parent.key === 'interactiveQuizzesEnabled' && e.child.key === 'ffIqAiGeneration',
        )!
        await assertDependencyTruthTable({
          family,
          edge: courseToAi,
          gaToken,
          instructorToken,
          courseCode,
          childProbe: aiProbe,
          bothOnAllowedStatuses: [400, 404, 503],
        })

        // Course parent / hosting child (games surface requires both).
        const courseToHosting = family.edges.find(
          (e) => e.parent.key === 'interactiveQuizzesEnabled' && e.child.key === 'ffIqLiveHosting',
        )!
        await assertDependencyTruthTable({
          family,
          edge: courseToHosting,
          gaToken,
          instructorToken,
          courseCode,
          childProbe: gamesProbe,
          bothOnAllowedStatuses: [400, 404, 403],
        })
      },
    })
  })
})
