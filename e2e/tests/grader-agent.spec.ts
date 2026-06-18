import { test, expect, injectToken } from '../fixtures/test.js'
import {
  apiCreateAssignment,
  apiPatchAssignmentSubmissionTypes,
  apiUploadAssignmentSubmission,
} from '../fixtures/api.js'

const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'

test.describe('Grader Agent API', () => {
  test('dry-run without auth returns 401', async () => {
    const res = await fetch(
      `${API_BASE}/api/v1/courses/E2E-TEST/assignments/00000000-0000-0000-0000-000000000001/grader-agent/dry-run`,
      {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ prompt: 'Grade fairly', submissionId: '00000000-0000-0000-0000-000000000002' }),
      },
    )
    expect(res.status).toBe(401)
  })
})

test('Instructor dry-runs and applies mocked agent grade in SpeedGrader', async ({
  coursePage,
  seededCourse,
}) => {
  const assignment = await apiCreateAssignment(
    seededCourse.instructorToken,
    seededCourse.courseCode,
    seededCourse.moduleId,
    'Grader Agent E2E Essay',
  )
  await apiPatchAssignmentSubmissionTypes(
    seededCourse.instructorToken,
    seededCourse.courseCode,
    assignment.id,
  )
  await apiUploadAssignmentSubmission(
    seededCourse.studentToken,
    seededCourse.courseCode,
    assignment.id,
    'My thesis argues that renewable energy adoption accelerates when policy aligns with grid economics.',
  )

  const dryRunBody = {
    suggestedPoints: 42,
    rubricScores: {},
    comment: 'Strong thesis; add more citations.',
    confidence: 0.88,
    promptTokens: 1200,
    completionTokens: 180,
  }

  await coursePage.route(`**/api/v1/courses/${seededCourse.courseCode}/assignments/${assignment.id}/grader-agent`, async (route) => {
    if (route.request().method() === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ config: null }),
      })
      return
    }
    await route.continue()
  })

  await coursePage.route(`**/grader-agent/dry-run`, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(dryRunBody),
    })
  })

  await coursePage.route(`**/submissions/*/grade`, async (route) => {
    if (route.request().method() === 'PUT') {
      const payload = route.request().postDataJSON() as { gradedByAi?: boolean; pointsEarned?: number }
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          pointsEarned: payload.pointsEarned ?? 42,
          posted: false,
          gradedByAi: payload.gradedByAi === true,
          instructorComment: dryRunBody.comment,
        }),
      })
      return
    }
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ posted: false }),
    })
  })

  await coursePage.route('**/api/v1/platform/features', async (route) => {
    const res = await route.fetch()
    const data = (await res.json()) as Record<string, unknown>
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ ...data, graderAgentEnabled: true }),
    })
  })

  await coursePage.goto(
    `/courses/${seededCourse.courseCode}/assignments/${assignment.id}?preview=submissions`,
  )
  await expect(coursePage.getByRole('button', { name: 'Grader Agent' })).toBeVisible({ timeout: 15_000 })
  await coursePage.getByRole('button', { name: 'Grader Agent' }).click()
  await expect(coursePage.getByRole('dialog', { name: 'Grading agent' })).toBeVisible()

  await coursePage.getByPlaceholder(/clear thesis/i).fill('Award full marks for a working thesis.')
  await coursePage.getByRole('button', { name: 'Dry run' }).click()
  await expect(coursePage.getByText('Strong thesis')).toBeVisible({ timeout: 10_000 })
  await coursePage.getByRole('button', { name: 'Apply to this student' }).click()
})

test('Student sees AI disclosure on posted agent grade', async ({ page, seededCourse }) => {
  const assignment = await apiCreateAssignment(
    seededCourse.instructorToken,
    seededCourse.courseCode,
    seededCourse.moduleId,
    'Disclosure Essay',
  )
  await apiPatchAssignmentSubmissionTypes(
    seededCourse.instructorToken,
    seededCourse.courseCode,
    assignment.id,
  )
  await apiUploadAssignmentSubmission(
    seededCourse.studentToken,
    seededCourse.courseCode,
    assignment.id,
    'Student essay body for disclosure test.',
  )

  await page.route('**/api/v1/platform/features', async (route) => {
    const res = await route.fetch()
    const data = (await res.json()) as Record<string, unknown>
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ ...data, graderAgentEnabled: true }),
    })
  })

  await page.route(`**/submissions/*/grade`, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        pointsEarned: 90,
        posted: true,
        gradedByAi: true,
        instructorComment: 'Well argued.',
      }),
    })
  })

  await injectToken(page, seededCourse.studentToken)
  await page.goto(`/courses/${seededCourse.courseCode}/assignments/${assignment.id}`)
  await expect(
    page.getByText(/drafted by an AI grading agent/i),
  ).toBeVisible({ timeout: 15_000 })
  await expect(page.getByRole('button', { name: 'Request human re-grade' })).toBeVisible()
})