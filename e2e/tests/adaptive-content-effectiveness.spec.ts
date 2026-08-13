/**
 * AC.7 — Post-assessment binding & effectiveness API/UI.
 *
 * Checklist:
 *   [x] Instructor binds post-assessment on a unit
 *   [x] Effectiveness GET/refresh return verdict payload
 *   [x] Workspace shows post-assessment picker + effectiveness chip
 *   [x] Student cannot refresh effectiveness (403)
 */
import { test, expect, injectToken } from '../fixtures/test.js'
import {
  apiCreateContentPage,
  apiCreateTimedQuiz,
  apiPatchCourseFeatures,
} from '../fixtures/api.js'

const apiBase = process.env.E2E_API_URL ?? 'http://localhost:8080'

async function apiCreateAdaptiveUnit(
  token: string,
  courseCode: string,
  body: Record<string, unknown>,
): Promise<{ id: string; postAssessmentItemId?: string | null }> {
  const deadline = Date.now() + 15_000
  for (;;) {
    const res = await fetch(
      `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/adaptive-content/units`,
      {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify(body),
      },
    )
    if (res.ok) {
      return res.json() as Promise<{ id: string; postAssessmentItemId?: string | null }>
    }
    const text = await res.text()
    const killSwitchBusy = res.status === 503 && text.includes('kill-switch engaged')
    if (!killSwitchBusy || Date.now() >= deadline) {
      throw new Error(`Create ACE unit failed (${res.status}): ${text}`)
    }
    await new Promise((r) => setTimeout(r, 400))
  }
}

async function apiPatchAdaptiveUnit(
  token: string,
  courseCode: string,
  unitId: string,
  body: Record<string, unknown>,
): Promise<void> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/adaptive-content/units/${encodeURIComponent(unitId)}`,
    {
      method: 'PATCH',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify(body),
    },
  )
  if (!res.ok) {
    throw new Error(`Patch ACE unit failed (${res.status}): ${await res.text()}`)
  }
}

test.describe('Adaptive content effectiveness (AC.7)', () => {
  test('API: bind post-assessment, refresh effectiveness, student denied', async ({
    seededCourse,
  }) => {
    await apiPatchCourseFeatures(seededCourse.instructorToken, seededCourse.courseCode, {
      adaptiveContentEnabled: true,
    })

    const page = await apiCreateContentPage(
      seededCourse.instructorToken,
      seededCourse.courseCode,
      seededCourse.moduleId,
      'ACE E2E Content',
    )
    const pre = await apiCreateTimedQuiz(
      seededCourse.instructorToken,
      seededCourse.courseCode,
      seededCourse.moduleId,
      10,
    )
    const post = await apiCreateTimedQuiz(
      seededCourse.instructorToken,
      seededCourse.courseCode,
      seededCourse.moduleId,
      10,
    )

    const unit = await apiCreateAdaptiveUnit(
      seededCourse.instructorToken,
      seededCourse.courseCode,
      {
        targetKind: 'module',
        targetModuleItemId: seededCourse.moduleId,
        baseContentItemId: page.id,
        preAssessmentItemId: pre.id,
        postAssessmentItemId: post.id,
        allowedAxes: ['emphasis', 'scaffolding'],
        status: 'active',
        triggerMode: 'pre_quiz',
      },
    )
    expect(unit.id).toBeTruthy()
    expect(unit.postAssessmentItemId).toBe(post.id)

    await apiPatchAdaptiveUnit(
      seededCourse.instructorToken,
      seededCourse.courseCode,
      unit.id,
      { status: 'active' },
    )

    const refreshRes = await fetch(
      `${apiBase}/api/v1/courses/${encodeURIComponent(seededCourse.courseCode)}/adaptive-content/effectiveness/refresh`,
      {
        method: 'POST',
        headers: { Authorization: `Bearer ${seededCourse.instructorToken}` },
      },
    )
    expect(refreshRes.status).toBe(200)
    const refreshBody = (await refreshRes.json()) as { refreshedUnits: number }
    expect(refreshBody.refreshedUnits).toBeGreaterThanOrEqual(1)

    const unitEffRes = await fetch(
      `${apiBase}/api/v1/courses/${encodeURIComponent(seededCourse.courseCode)}/adaptive-content/units/${encodeURIComponent(unit.id)}/effectiveness`,
      { headers: { Authorization: `Bearer ${seededCourse.instructorToken}` } },
    )
    expect(unitEffRes.status).toBe(200)
    const eff = (await unitEffRes.json()) as {
      unitId: string
      verdict: string
      nTreatment: number
      nHoldout: number
      byMode: unknown[]
      smallCellMinN: number
      minNPerArm: number
    }
    expect(eff.unitId).toBe(unit.id)
    expect(eff.verdict).toBe('insufficient_data')
    expect(eff.smallCellMinN).toBeGreaterThanOrEqual(1)
    expect(eff.minNPerArm).toBeGreaterThanOrEqual(1)

    const studentRefresh = await fetch(
      `${apiBase}/api/v1/courses/${encodeURIComponent(seededCourse.courseCode)}/adaptive-content/effectiveness/refresh`,
      {
        method: 'POST',
        headers: { Authorization: `Bearer ${seededCourse.studentToken}` },
      },
    )
    expect([401, 403]).toContain(studentRefresh.status)
  })

  test('UI: post-assessment picker and effectiveness chip in workspace', async ({
    coursePage: page,
    seededCourse,
  }) => {
    await apiPatchCourseFeatures(seededCourse.instructorToken, seededCourse.courseCode, {
      adaptiveContentEnabled: true,
    })

    const content = await apiCreateContentPage(
      seededCourse.instructorToken,
      seededCourse.courseCode,
      seededCourse.moduleId,
      'ACE UI Content',
    )
    const post = await apiCreateTimedQuiz(
      seededCourse.instructorToken,
      seededCourse.courseCode,
      seededCourse.moduleId,
      10,
    )
    const unit = await apiCreateAdaptiveUnit(
      seededCourse.instructorToken,
      seededCourse.courseCode,
      {
        targetKind: 'module',
        targetModuleItemId: seededCourse.moduleId,
        baseContentItemId: content.id,
        postAssessmentItemId: post.id,
        allowedAxes: ['emphasis'],
        status: 'draft',
        triggerMode: 'pre_quiz',
      },
    )
    await fetch(
      `${apiBase}/api/v1/courses/${encodeURIComponent(seededCourse.courseCode)}/adaptive-content/effectiveness/refresh`,
      {
        method: 'POST',
        headers: { Authorization: `Bearer ${seededCourse.instructorToken}` },
      },
    )

    await injectToken(page, seededCourse.instructorToken)
    await page.goto(`/courses/${seededCourse.courseCode}/settings/adaptive-content`)
    await expect(page.getByText(/Adaptive Content workspace/i)).toBeVisible({ timeout: 15000 })
    await expect(page.getByTestId('ace-new-post-assessment')).toBeVisible({ timeout: 10000 })
    await expect(page.getByTestId('ace-refresh-effectiveness')).toBeVisible()
    await expect(page.getByTestId('ace-effectiveness-chip').first()).toBeVisible()

    await page.getByRole('button', { name: /^Open$/i }).first().click()
    await expect(page.getByTestId('ace-edit-post-assessment')).toBeVisible({ timeout: 10000 })
    await expect(page.getByTestId('ace-verdict-banner')).toBeVisible()
    expect(unit.id).toBeTruthy()
  })
})
