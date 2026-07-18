/**
 * E2E.1 — navigation + direct-route gates for navigation-producing course flags.
 */
import { test } from '../fixtures/test.js'
import { injectToken } from '../fixtures/test.js'
import { NAV_COURSE_FEATURE_ENTRIES } from '../lib/course-feature-matrix.js'
import {
  assertNavGate,
  withCourseFeatureRestore,
} from '../lib/course-feature-matrix-helpers.js'

test.describe('Course features navigation matrix', () => {
  for (const entry of NAV_COURSE_FEATURE_ENTRIES) {
    // groupSpacesEnabled has a dedicated API-only spec for runtime gate + persistence.
    if (entry.key === 'groupSpacesEnabled') continue

    test(`nav gate for ${entry.key}`, async ({ page, seededCourse }) => {
      test.info().annotations.push({
        type: 'course-feature-nav',
        description: `${seededCourse.courseCode}:${entry.key}`,
      })

      const token =
        entry.nav.audience === 'instructor'
          ? seededCourse.instructorToken
          : seededCourse.studentToken

      await injectToken(page, token)

      await withCourseFeatureRestore(
        seededCourse.instructorToken,
        seededCourse.courseCode,
        async () => {
          await assertNavGate(
            page,
            seededCourse.instructorToken,
            seededCourse.courseCode,
            entry.key,
            entry.nav,
            entry.uiDefaultOn,
          )
        },
      )
    })
  }
})
