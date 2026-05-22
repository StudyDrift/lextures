/**
 * Course-level outcomes reporting (plan 9.5)
 *
 * Checklist coverage:
 *   [x] Outcomes report page loads for instructor
 *   [x] Align assignment to outcome, grade student, report shows pct_met
 *   [x] Student gets 403 on analytics/outcomes API
 *   [x] Improvement note persists via API
 */
import { test, expect, injectToken } from '../fixtures/test.js'
import {
  apiCreateAssignment,
  apiCreateOutcome,
  apiCreateOutcomeLink,
  apiGetCourseEnrollments,
  apiGetOutcomesReport,
  apiPatchAssignment,
  apiPostOutcomeImprovementNote,
  apiPutGradebookGrades,
  apiRefreshOutcomesReport,
} from '../fixtures/api.js'

const apiBase = process.env.E2E_API_URL ?? 'http://localhost:8080'

test.describe('Outcomes report — UI', () => {
  test('report page loads for instructor', async ({ coursePage: page, seededCourse }) => {
    await page.goto(`/courses/${seededCourse.courseCode}/outcomes-report`)
    await expect(page.getByRole('heading', { name: /outcomes report/i })).toBeVisible({
      timeout: 12000,
    })
    await expect(page.getByLabel(/mastery threshold/i)).toBeVisible()
  })

  test('shows cohort pct_met after aligning and grading', async ({
    coursePage: page,
    seededCourse,
  }) => {
    const outcome = await apiCreateOutcome(
      seededCourse.instructorToken,
      seededCourse.courseCode,
      'E2E Report Outcome',
    )
    const assignment = await apiCreateAssignment(
      seededCourse.instructorToken,
      seededCourse.courseCode,
      seededCourse.moduleId,
      'E2E Report Assignment',
    )
    await apiPatchAssignment(
      seededCourse.instructorToken,
      seededCourse.courseCode,
      assignment.id,
      { pointsWorth: 100 },
    )
    await apiCreateOutcomeLink(
      seededCourse.instructorToken,
      seededCourse.courseCode,
      outcome.id,
      { structureItemId: assignment.id, targetKind: 'assignment' },
    )

    const enrollments = await apiGetCourseEnrollments(
      seededCourse.instructorToken,
      seededCourse.courseCode,
    )
    const student = enrollments.find((e) => e.role === 'student')
    expect(student?.userId).toBeTruthy()

    await apiPutGradebookGrades(seededCourse.instructorToken, seededCourse.courseCode, {
      [student!.userId]: { [assignment.id]: '85' },
    })
    await apiRefreshOutcomesReport(seededCourse.instructorToken, seededCourse.courseCode)

    await page.goto(`/courses/${seededCourse.courseCode}/outcomes-report`)
    await expect(page.getByRole('heading', { name: /outcomes report/i })).toBeVisible({
      timeout: 12000,
    })
    await expect(page.getByText('E2E Report Outcome')).toBeVisible()
    await expect(page.getByText(/100% met/i)).toBeVisible({ timeout: 10000 })
    await expect(page.getByText(/1 \/ 1/)).toBeVisible()
  })
})

test.describe('Outcomes report — API', () => {
  test('GET analytics/outcomes returns cohort stats for instructor', async ({ seededCourse }) => {
    const outcome = await apiCreateOutcome(
      seededCourse.instructorToken,
      seededCourse.courseCode,
      'E2E API Outcome',
    )
    const assignment = await apiCreateAssignment(
      seededCourse.instructorToken,
      seededCourse.courseCode,
      seededCourse.moduleId,
      'E2E API Assignment',
    )
    await apiPatchAssignment(
      seededCourse.instructorToken,
      seededCourse.courseCode,
      assignment.id,
      { pointsWorth: 100 },
    )
    await apiCreateOutcomeLink(
      seededCourse.instructorToken,
      seededCourse.courseCode,
      outcome.id,
      { structureItemId: assignment.id, targetKind: 'assignment' },
    )
    const enrollments = await apiGetCourseEnrollments(
      seededCourse.instructorToken,
      seededCourse.courseCode,
    )
    const student = enrollments.find((e) => e.role === 'student')
    await apiPutGradebookGrades(seededCourse.instructorToken, seededCourse.courseCode, {
      [student!.userId]: { [assignment.id]: '90' },
    })
    await apiRefreshOutcomesReport(seededCourse.instructorToken, seededCourse.courseCode)

    const report = await apiGetOutcomesReport(
      seededCourse.instructorToken,
      seededCourse.courseCode,
    )
    const row = report.outcomes.find((o) => o.title === 'E2E API Outcome')
    expect(row).toBeTruthy()
    expect(row!.nAssessed).toBe(1)
    expect(row!.pctMet).toBe(100)
    expect(row!.noAlignments).toBe(false)
  })

  test('student gets 403 on analytics/outcomes', async ({ seededCourse }) => {
    const res = await fetch(
      `${apiBase}/api/v1/courses/${seededCourse.courseCode}/analytics/outcomes`,
      { headers: { Authorization: `Bearer ${seededCourse.studentToken}` } },
    )
    expect(res.status).toBe(403)
  })

  test('improvement note persists', async ({ seededCourse }) => {
    const outcome = await apiCreateOutcome(
      seededCourse.instructorToken,
      seededCourse.courseCode,
      'E2E Note Outcome',
    )
    await apiPostOutcomeImprovementNote(
      seededCourse.instructorToken,
      seededCourse.courseCode,
      outcome.id,
      'Strengthen lab report instructions for Outcome 2',
    )
    await apiRefreshOutcomesReport(seededCourse.instructorToken, seededCourse.courseCode)
    const report = await apiGetOutcomesReport(
      seededCourse.instructorToken,
      seededCourse.courseCode,
    )
    const row = report.outcomes.find((o) => o.outcomeId === outcome.id)
    expect(row?.improvementNote).toContain('lab report instructions')
  })
})

test.describe('Outcomes report — student UI blocked', () => {
  test('student does not see outcomes report nav target', async ({ page, seededCourse }) => {
    await injectToken(page, seededCourse.studentToken)
    await page.goto(`/courses/${seededCourse.courseCode}`)
    await expect(page.getByRole('link', { name: 'Outcomes report' })).not.toBeVisible({
      timeout: 5000,
    })
  })
})
