/**
 * E2E.1 — Course Settings → Features UI matrix (shard B).
 */
import { test } from '../fixtures/test.js'
import { uiEntriesForShard } from '../lib/course-feature-matrix.js'
import {
  assertUiToggleEnableFlow,
  withCourseFeatureRestore,
} from '../lib/course-feature-matrix-helpers.js'

const entries = uiEntriesForShard('b')

test.describe('Course features UI matrix B', () => {
  for (const entry of entries) {
    test(`enable ${entry.key} via settings UI (${entry.uiLabel})`, async ({
      coursePage: page,
      seededCourse,
    }) => {
      test.info().annotations.push({
        type: 'course-feature',
        description: `${seededCourse.courseCode}:${entry.key}`,
      })
      await withCourseFeatureRestore(
        seededCourse.instructorToken,
        seededCourse.courseCode,
        async () => {
          await assertUiToggleEnableFlow(
            page,
            seededCourse.instructorToken,
            seededCourse.courseCode,
            entry,
          )
        },
      )
    })
  }
})
