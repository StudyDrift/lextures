/**
 * What-if grades (plan 3.16)
 *
 * Checklist:
 *   [x] Feature flag enables what-if mode on My Grades
 *   [x] Student enters hypothetical score and sees projected course grade
 *   [x] Reset to actual clears overrides
 */
import { test, expect, injectToken } from '../fixtures/test.js'
import {
  apiCreateAssignment,
  apiGetCourseGrading,
  apiListEnrollments,
  apiPatchAssignment,
  apiPutCourseGrading,
  apiPutGradebookGrades,
} from '../fixtures/api.js'

test.describe('What-if grades', () => {
  test('student edits hypothetical final and sees projected grade', async ({
    page,
    seededCourse,
  }) => {
    const { instructorToken, courseCode, moduleId, studentToken } = seededCourse

    await apiPutCourseGrading(instructorToken, courseCode, {
      gradingScale: 'percentage',
      assignmentGroups: [
        { name: 'Coursework', sortOrder: 0, weightPercent: 40 },
        { name: 'Final', sortOrder: 1, weightPercent: 60 },
      ],
    })

    const grading = await apiGetCourseGrading(instructorToken, courseCode)
    const courseworkGroup = grading.assignmentGroups.find((g) => g.name === 'Coursework')
    const finalGroup = grading.assignmentGroups.find((g) => g.name === 'Final')
    if (!courseworkGroup || !finalGroup) throw new Error('Expected weighted assignment groups')

    const hw = await apiCreateAssignment(
      instructorToken,
      courseCode,
      moduleId,
      'E2E what-if homework',
    )
    const finalExam = await apiCreateAssignment(
      instructorToken,
      courseCode,
      moduleId,
      'E2E what-if final',
    )

    await apiPatchAssignment(instructorToken, courseCode, hw.id, {
      pointsWorth: 100,
      postingPolicy: 'automatic',
      assignmentGroupId: courseworkGroup.id,
    })
    await apiPatchAssignment(instructorToken, courseCode, finalExam.id, {
      pointsWorth: 100,
      postingPolicy: 'automatic',
      assignmentGroupId: finalGroup.id,
    })

    const roster = await apiListEnrollments(instructorToken, courseCode)
    const student = roster.find((e) => e.role === 'student')
    if (!student) throw new Error('Expected student enrollment')

    await apiPutGradebookGrades(instructorToken, courseCode, {
      [student.userId]: { [hw.id]: '80' },
    })

    await injectToken(page, studentToken)
    await page.goto(`/courses/${courseCode}/my-grades`)

    await page.getByRole('button', { name: /what-if grades/i }).click()
    await expect(page.getByText(/what-if grades are private/i)).toBeVisible()

    const finalInput = page.getByLabel(/hypothetical score for E2E what-if final/i)
    await finalInput.fill('90')
    await finalInput.blur()

    await expect(page.getByText(/projected course grade/i)).toBeVisible()
    await expect(page.getByLabel(/hypothetical projected grade/i)).toBeVisible()
    await expect(page.getByText(/86%/)).toBeVisible({ timeout: 5000 })

    await page.getByRole('button', { name: /reset to actual/i }).click()
    await expect(finalInput).toHaveValue('')
    await expect(page.getByText(/actual course grade/i)).not.toBeVisible()
  })
})
