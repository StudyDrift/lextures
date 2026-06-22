/**
 * Differentiated assignments — assign-to targeting & multiple due dates (plan 2.15)
 */
import { test, expect, uniqueEmail } from '../fixtures/test.js'
import {
  apiSignup,
  apiCreateCourse,
  apiCreateModule,
  apiCreateAssignment,
  apiEnroll,
  apiPatchCourseFeatures,
  apiCreateCourseSection,
  apiPutAssignToTargets,
  apiGetAssignToTargets,
  apiBulkExtendAssignToDueDate,
  apiGetCourseStructure,
  apiGetCourseEnrollments,
} from '../fixtures/api.js'

const PASSWORD = 'E2eTestPass1!'

const apiBase = process.env.E2E_API_URL ?? 'http://localhost:8080'

async function apiMeUserId(token: string): Promise<string> {
  const res = await fetch(`${apiBase}/api/v1/me`, {
    headers: { Authorization: `Bearer ${token}` },
  })
  if (!res.ok) throw new Error(`GET /api/v1/me failed (${res.status})`)
  const data = (await res.json()) as { id?: string }
  if (!data.id) throw new Error('GET /api/v1/me missing id')
  return data.id
}

function dueIso(year: number, month: number, day: number): string {
  return new Date(Date.UTC(year, month - 1, day, 23, 59, 0)).toISOString()
}

function sameCalendarDay(a: string | null | undefined, b: string): boolean {
  if (!a) return false
  const da = new Date(a)
  const db = new Date(b)
  return (
    da.getUTCFullYear() === db.getUTCFullYear() &&
    da.getUTCMonth() === db.getUTCMonth() &&
    da.getUTCDate() === db.getUTCDate()
  )
}

async function setupDifferentiatedCourse() {
  const instructorEmail = uniqueEmail('assign-inst')
  const { access_token: instructorToken } = await apiSignup({
    email: instructorEmail,
    password: PASSWORD,
    displayName: 'Assign-To Instructor',
  })

  const studentAEmail = uniqueEmail('assign-stu-a')
  const { access_token: studentAToken } = await apiSignup({
    email: studentAEmail,
    password: PASSWORD,
    displayName: 'Section A Student',
    accountType: 'parent',
  })

  const studentBEmail = uniqueEmail('assign-stu-b')
  const { access_token: studentBToken } = await apiSignup({
    email: studentBEmail,
    password: PASSWORD,
    displayName: 'Section B Student',
    accountType: 'parent',
  })

  const course = await apiCreateCourse(instructorToken, { title: 'Assign-To E2E Course' })
  await apiEnroll(instructorToken, course.courseCode, instructorEmail, 'teacher')
  await apiPatchCourseFeatures(instructorToken, course.courseCode, { sectionsEnabled: true })

  const sectionA = await apiCreateCourseSection(instructorToken, course.courseCode, 'SEC-A', 'Section A')
  const sectionB = await apiCreateCourseSection(instructorToken, course.courseCode, 'SEC-B', 'Section B')

  await apiEnroll(instructorToken, course.courseCode, studentAEmail, 'student', {
    memberToken: studentAToken,
    sectionId: sectionA.id,
  })
  await apiEnroll(instructorToken, course.courseCode, studentBEmail, 'student', {
    memberToken: studentBToken,
    sectionId: sectionB.id,
  })

  const mod = await apiCreateModule(instructorToken, course.courseCode, 'Unit 1')
  const assignment = await apiCreateAssignment(
    instructorToken,
    course.courseCode,
    mod.id,
    'Differentiated Essay',
  )

  const [userIdA, userIdB] = await Promise.all([
    apiMeUserId(studentAToken),
    apiMeUserId(studentBToken),
  ])
  const enrollments = await apiGetCourseEnrollments(instructorToken, course.courseCode)
  const enrollmentA = enrollments.find((e) => e.role === 'student' && e.userId === userIdA)
  const enrollmentB = enrollments.find((e) => e.role === 'student' && e.userId === userIdB)

  return {
    instructorToken,
    studentAToken,
    studentBToken,
    courseCode: course.courseCode,
    sectionA,
    sectionB,
    assignment,
    enrollmentA,
    enrollmentB,
  }
}

test.describe('Differentiated assignments (2.15)', () => {
  test('section-specific due dates resolve per student (AC-1)', async () => {
    const ctx = await setupDifferentiatedCourse()
    const dueA = dueIso(2026, 6, 13) // Friday
    const dueB = dueIso(2026, 6, 16) // Monday

    await apiPutAssignToTargets(ctx.instructorToken, ctx.courseCode, ctx.assignment.id, [
      { targetType: 'section', targetId: ctx.sectionA.id, dueAt: dueA },
      { targetType: 'section', targetId: ctx.sectionB.id, dueAt: dueB },
    ])

    const structA = await apiGetCourseStructure(ctx.studentAToken, ctx.courseCode)
    const itemA = structA.find((i) => i.id === ctx.assignment.id)
    expect(itemA).toBeTruthy()
    expect(sameCalendarDay(itemA?.dueAt, dueA)).toBe(true)

    const structB = await apiGetCourseStructure(ctx.studentBToken, ctx.courseCode)
    const itemB = structB.find((i) => i.id === ctx.assignment.id)
    expect(itemB).toBeTruthy()
    expect(sameCalendarDay(itemB?.dueAt, dueB)).toBe(true)
  })

  test('non-targeted student does not see section-only assignment (AC-2)', async () => {
    const ctx = await setupDifferentiatedCourse()

    await apiPutAssignToTargets(ctx.instructorToken, ctx.courseCode, ctx.assignment.id, [
      { targetType: 'section', targetId: ctx.sectionA.id, dueAt: dueIso(2026, 6, 13) },
    ])

    const structA = await apiGetCourseStructure(ctx.studentAToken, ctx.courseCode)
    expect(structA.some((i) => i.id === ctx.assignment.id)).toBe(true)

    const structB = await apiGetCourseStructure(ctx.studentBToken, ctx.courseCode)
    expect(structB.some((i) => i.id === ctx.assignment.id)).toBe(false)
  })

  test('student override wins over section target (AC-4)', async () => {
    const ctx = await setupDifferentiatedCourse()
    expect(ctx.enrollmentA?.id).toBeTruthy()

    const sectionDue = dueIso(2026, 6, 13)
    const studentDue = dueIso(2026, 6, 20)

    await apiPutAssignToTargets(ctx.instructorToken, ctx.courseCode, ctx.assignment.id, [
      { targetType: 'section', targetId: ctx.sectionA.id, dueAt: sectionDue },
      {
        targetType: 'student',
        targetId: ctx.enrollmentA!.id,
        dueAt: studentDue,
      },
    ])

    const structA = await apiGetCourseStructure(ctx.studentAToken, ctx.courseCode)
    const itemA = structA.find((i) => i.id === ctx.assignment.id)
    expect(itemA).toBeTruthy()
    expect(sameCalendarDay(itemA?.dueAt, studentDue)).toBe(true)
    expect(sameCalendarDay(itemA?.dueAt, sectionDue)).toBe(false)
  })

  test('orphaned targets are flagged when no student matches (AC-5)', async () => {
    const ctx = await setupDifferentiatedCourse()
    const emptySection = await apiCreateCourseSection(
      ctx.instructorToken,
      ctx.courseCode,
      'EMPTY',
      'Empty Section',
    )

    await apiPutAssignToTargets(ctx.instructorToken, ctx.courseCode, ctx.assignment.id, [
      { targetType: 'section', targetId: emptySection.id, dueAt: dueIso(2026, 6, 13) },
    ])

    const res = await apiGetAssignToTargets(ctx.instructorToken, ctx.courseCode, ctx.assignment.id)
    expect(res.orphaned).toBe(true)
  })

  test('bulk extend sets student-level due date override', async () => {
    const ctx = await setupDifferentiatedCourse()
    expect(ctx.enrollmentB?.id).toBeTruthy()

    const extendedDue = dueIso(2026, 7, 1)
    await apiBulkExtendAssignToDueDate(
      ctx.instructorToken,
      ctx.courseCode,
      ctx.assignment.id,
      [ctx.enrollmentB!.id],
      extendedDue,
    )

    const structB = await apiGetCourseStructure(ctx.studentBToken, ctx.courseCode)
    const itemB = structB.find((i) => i.id === ctx.assignment.id)
    expect(itemB).toBeTruthy()
    expect(sameCalendarDay(itemB?.dueAt, extendedDue)).toBe(true)
  })

  test('assign-to editor is visible on assignment settings', async ({ coursePage, seededCourse }) => {
    const assignment = await apiCreateAssignment(
      seededCourse.instructorToken,
      seededCourse.courseCode,
      seededCourse.moduleId,
      'Assign-To UI Check',
    )

    await coursePage.goto(
      `/courses/${seededCourse.courseCode}/modules/assignment/${assignment.id}`,
    )
    await coursePage.getByRole('button', { name: /^Edit$/i }).click()
    await expect(coursePage.getByRole('button', { name: /^Assign to$/i })).toBeVisible({
      timeout: 12000,
    })
    await coursePage.getByRole('button', { name: /^Assign to$/i }).click()
    await expect(coursePage.getByRole('button', { name: /Save assign-to targets/i })).toBeVisible()
    await expect(coursePage.getByRole('button', { name: /Add audience/i })).toBeVisible()
  })
})
