/**
 * AC.9 — Analytics, reporting & operability.
 *
 * Checklist:
 *   [x] Instructor report empty/KPI surface + CSV export
 *   [x] Student denied report (403)
 *   [x] Admin org report + drill-down link
 *   [x] Report tab visible in workspace UI
 */
import { execSync } from 'node:child_process'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { test, expect, injectToken, uniqueEmail } from '../fixtures/test.js'
import {
  apiCreateContentPage,
  apiCreateTimedQuiz,
  apiLogin,
  apiPatchCourseFeatures,
  apiSignup,
} from '../fixtures/api.js'

const apiBase = process.env.E2E_API_URL ?? 'http://localhost:8080'
const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..')
const ADMIN_PASSWORD = 'E2eTestPass1!LongRandomAceReports'

async function apiCreateAdaptiveUnit(
  token: string,
  courseCode: string,
  body: Record<string, unknown>,
): Promise<{ id: string }> {
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
      return res.json() as Promise<{ id: string }>
    }
    const text = await res.text()
    const killSwitchBusy = res.status === 503 && text.includes('kill-switch engaged')
    if (!killSwitchBusy || Date.now() >= deadline) {
      throw new Error(`Create ACE unit failed (${res.status}): ${text}`)
    }
    await new Promise((r) => setTimeout(r, 400))
  }
}

function bootstrapGlobalAdmin(email: string) {
  const databaseURL =
    process.env.DATABASE_URL ??
    'postgres://studydrift:studydrift@localhost:5432/studydrift?sslmode=disable'
  execSync(`go run ./cmd/bootstrap-admin -email=${email}`, {
    cwd: path.join(repoRoot, 'server'),
    stdio: 'pipe',
    env: {
      ...process.env,
      PATH: `/usr/local/go/bin:${process.env.PATH ?? ''}`,
      DATABASE_URL: databaseURL,
    },
  })
}

test.describe('Adaptive content reports (AC.9)', () => {
  test('API: instructor report + CSV; student denied; admin org report', async ({
    seededCourse,
  }) => {
    await apiPatchCourseFeatures(seededCourse.instructorToken, seededCourse.courseCode, {
      adaptiveContentEnabled: true,
    })

    const page = await apiCreateContentPage(
      seededCourse.instructorToken,
      seededCourse.courseCode,
      seededCourse.moduleId,
      'ACE Report Content',
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
    await apiCreateAdaptiveUnit(seededCourse.instructorToken, seededCourse.courseCode, {
      targetKind: 'module',
      targetModuleItemId: seededCourse.moduleId,
      baseContentItemId: page.id,
      preAssessmentItemId: pre.id,
      postAssessmentItemId: post.id,
      allowedAxes: ['emphasis'],
      status: 'active',
      triggerMode: 'pre_quiz',
    })

    await fetch(
      `${apiBase}/api/v1/courses/${encodeURIComponent(seededCourse.courseCode)}/adaptive-content/effectiveness/refresh`,
      {
        method: 'POST',
        headers: { Authorization: `Bearer ${seededCourse.instructorToken}` },
      },
    )

    const reportRes = await fetch(
      `${apiBase}/api/v1/courses/${encodeURIComponent(seededCourse.courseCode)}/adaptive-content/report`,
      { headers: { Authorization: `Bearer ${seededCourse.instructorToken}` } },
    )
    expect(reportRes.status).toBe(200)
    const report = (await reportRes.json()) as {
      empty: boolean
      nUnits: number
      coverage: { eligibleContentItems: number }
      unitsToReview: unknown[]
      cost: { tokensUsedPeriod: number }
    }
    expect(report.nUnits).toBeGreaterThanOrEqual(1)
    expect(report.coverage.eligibleContentItems).toBeGreaterThanOrEqual(1)
    expect(Array.isArray(report.unitsToReview)).toBe(true)

    const exportRes = await fetch(
      `${apiBase}/api/v1/courses/${encodeURIComponent(seededCourse.courseCode)}/adaptive-content/report/export`,
      { headers: { Authorization: `Bearer ${seededCourse.instructorToken}` } },
    )
    expect(exportRes.status).toBe(200)
    expect(exportRes.headers.get('content-type') ?? '').toContain('text/csv')
    const csv = await exportRes.text()
    expect(csv).toContain('section')
    expect(csv).toContain('summary')

    const studentRes = await fetch(
      `${apiBase}/api/v1/courses/${encodeURIComponent(seededCourse.courseCode)}/adaptive-content/report`,
      { headers: { Authorization: `Bearer ${seededCourse.studentToken}` } },
    )
    expect(studentRes.status).toBe(403)

    const adminEmail = uniqueEmail('ace-report-admin')
    await apiSignup({
      email: adminEmail,
      password: ADMIN_PASSWORD,
      displayName: 'ACE Report Admin',
    })
    try {
      bootstrapGlobalAdmin(adminEmail)
    } catch (err) {
      test.skip(true, `bootstrap unavailable: ${err}`)
    }
    const { access_token: adminToken } = await apiLogin({
      email: adminEmail,
      password: ADMIN_PASSWORD,
    })

    const adminReportRes = await fetch(`${apiBase}/api/v1/admin/adaptive-content/report`, {
      headers: { Authorization: `Bearer ${adminToken}` },
    })
    expect(adminReportRes.status).toBe(200)
    const adminReport = (await adminReportRes.json()) as {
      coursesUsingAce: number
      courses: Array<{ courseCode: string; reportUrl: string }>
    }
    expect(adminReport.coursesUsingAce).toBeGreaterThanOrEqual(1)
    const row = adminReport.courses.find((c) => c.courseCode === seededCourse.courseCode)
    expect(row?.reportUrl).toContain('tab=report')

    const adminExport = await fetch(`${apiBase}/api/v1/admin/adaptive-content/report/export`, {
      headers: { Authorization: `Bearer ${adminToken}` },
    })
    expect(adminExport.status).toBe(200)
    expect(await adminExport.text()).toContain('courses_using_ace')
  })

  test('UI: instructor report tab shows KPIs or empty state', async ({ page, seededCourse }) => {
    await apiPatchCourseFeatures(seededCourse.instructorToken, seededCourse.courseCode, {
      adaptiveContentEnabled: true,
    })
    await injectToken(page, seededCourse.instructorToken)
    await page.goto(
      `/courses/${encodeURIComponent(seededCourse.courseCode)}/settings/adaptive-content?tab=report`,
    )
    await expect(page.getByTestId('ace-report-tab')).toBeVisible()
    await page.getByTestId('ace-report-tab').click()
    await expect(
      page.getByTestId('ace-course-report').or(page.getByTestId('ace-report-empty')),
    ).toBeVisible({ timeout: 15000 })
  })
})
