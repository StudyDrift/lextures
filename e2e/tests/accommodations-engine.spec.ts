/**
 * Accommodations engine — plan 12.10
 */
import { test, expect, injectToken } from '../fixtures/test.js'
import {
  apiCreateStudentAccommodation,
  apiCreateTimedQuiz,
  apiFetchAccommodationAuditLog,
  apiFetchEnrollmentAccommodationSummary,
  apiListEnrollments,
  apiLogin,
  apiPatchPlatformSettings,
  apiSignup,
  apiStartQuiz,
  effectiveTimeLimitSecondsFromStart,
} from '../fixtures/api.js'

const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'
const ADMIN_EMAIL = process.env.E2E_ADMIN_EMAIL ?? 'admin@e2e.test'
const ADMIN_PASSWORD = process.env.E2E_ADMIN_PASSWORD ?? 'E2eTestPass1!'

async function adminToken(): Promise<string> {
  try {
    const { access_token } = await apiSignup({
      email: ADMIN_EMAIL,
      password: ADMIN_PASSWORD,
      displayName: 'E2E Admin',
    })
    return access_token
  } catch {
    const { access_token } = await apiLogin({ email: ADMIN_EMAIL, password: ADMIN_PASSWORD })
    return access_token
  }
}

async function enableAccommodationsEngine(token: string): Promise<void> {
  await apiPatchPlatformSettings(token, {
    accommodationsEngineEnabled: true,
    ffAccommodationsEngine: true,
    updateMask: ['accommodationsEngineEnabled', 'ffAccommodationsEngine'],
  })
}

test.describe('Accommodations engine', () => {
  test('extended time, audit log, and enrollment indicator', async ({
    coursePage: page,
    seededCourse,
  }) => {
    const admin = await adminToken()
    await enableAccommodationsEngine(admin)

    const enrollments = await apiListEnrollments(seededCourse.instructorToken, seededCourse.courseCode)
    const studentEnrollment = enrollments.find((e) => e.role === 'student')
    expect(studentEnrollment?.userId).toBeTruthy()

    await apiCreateStudentAccommodation(admin, studentEnrollment!.userId, {
      timeMultiplier: 1.5,
    })

    const quiz = await apiCreateTimedQuiz(
      seededCourse.instructorToken,
      seededCourse.courseCode,
      seededCourse.moduleId,
      40,
    )

    const start = await apiStartQuiz(seededCourse.studentToken, seededCourse.courseCode, quiz.id)
    expect(start.extendedTimeActive).toBe(true)
    const effectiveSeconds = effectiveTimeLimitSecondsFromStart(start)
    expect(effectiveSeconds).toBe(3600)

    const audit = await apiFetchAccommodationAuditLog(admin, {
      studentId: studentEnrollment!.userId,
      limit: 50,
    })
    const timeEntry = audit.find((e) => e.accommodationType === 'time_extension')
    expect(timeEntry).toBeDefined()
    expect(timeEntry?.context).toBe('quiz_attempt')

    const summary = await apiFetchEnrollmentAccommodationSummary(
      seededCourse.instructorToken,
      studentEnrollment!.id,
    )
    expect(summary.hasAccommodation).toBe(true)
    expect(summary.flags.length).toBeGreaterThan(0)

    await page.goto(`/courses/${seededCourse.courseCode}/enrollments`)
    await expect(page.getByText('Has active accommodations')).toBeVisible({ timeout: 10000 })
  })

  test('audit endpoint returns 404 when engine disabled', async () => {
    const admin = await adminToken()
    await apiPatchPlatformSettings(admin, {
      accommodationsEngineEnabled: false,
      updateMask: ['accommodationsEngineEnabled'],
    })
    const res = await fetch(`${API_BASE}/api/v1/admin/accommodations/audit`, {
      headers: { Authorization: `Bearer ${admin}` },
    })
    expect(res.status).toBe(404)
  })
})
