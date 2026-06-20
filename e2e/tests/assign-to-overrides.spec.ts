/**
 * Differentiated assignments / "Assign To" targeting (plan 2.15).
 */
import { test, expect } from '@playwright/test'
import {
  apiSignup,
  apiCreateCourse,
  apiCreateModule,
  apiCreateAssignment,
  apiPatchCourseFeatures,
  apiEnroll,
} from '../fixtures/api.js'

const apiBase = process.env.E2E_API_URL ?? 'http://localhost:8080'
const PASSWORD = 'E2eTestPass1!'

function uid(prefix = 'a2o') {
  return `e2e-${prefix}-${Date.now()}-${Math.random().toString(36).slice(2)}`
}
function authHeaders(token: string) {
  return { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' }
}

async function getAdminToken(): Promise<string> {
  const adminEmail = process.env.E2E_ADMIN_EMAIL ?? 'admin@e2e.test'
  const adminPassword = process.env.E2E_ADMIN_PASSWORD ?? PASSWORD
  const loginRes = await fetch(`${apiBase}/api/v1/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email: adminEmail, password: adminPassword }),
  })
  if (loginRes.ok) {
    const { access_token } = (await loginRes.json()) as { access_token: string }
    return access_token
  }
  const { access_token } = await apiSignup({
    email: adminEmail,
    password: adminPassword,
    displayName: 'E2E Admin',
  })
  return access_token
}

async function enableAssignToOverrides(adminToken: string): Promise<boolean> {
  const res = await fetch(`${apiBase}/api/v1/settings/platform`, {
    method: 'PUT',
    headers: authHeaders(adminToken),
    body: JSON.stringify({
      ffAssignToOverrides: true,
      updateMask: ['ffAssignToOverrides'],
    }),
  })
  return res.ok
}

async function getEnrollmentIdByDisplayName(
  token: string,
  courseCode: string,
  displayName: string,
): Promise<string> {
  const res = await fetch(`${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/enrollments`, {
    headers: { Authorization: `Bearer ${token}` },
  })
  if (!res.ok) throw new Error(`List enrollments failed (${res.status}): ${await res.text()}`)
  const body = (await res.json()) as { enrollments: Array<{ id: string; displayName?: string | null }> }
  const match = body.enrollments.find((e) => e.displayName === displayName)
  if (!match) throw new Error(`No enrollment found with displayName ${displayName}`)
  return match.id
}

async function transferEnrollmentSection(
  token: string,
  enrollmentId: string,
  sectionId: string,
): Promise<void> {
  const res = await fetch(`${apiBase}/api/v1/enrollments/${encodeURIComponent(enrollmentId)}/section`, {
    method: 'PATCH',
    headers: authHeaders(token),
    body: JSON.stringify({ sectionId }),
  })
  if (!res.ok) throw new Error(`Transfer enrollment section failed (${res.status}): ${await res.text()}`)
}

async function createSection(token: string, courseCode: string, sectionCode: string): Promise<string> {
  const res = await fetch(`${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/sections`, {
    method: 'POST',
    headers: authHeaders(token),
    body: JSON.stringify({ sectionCode }),
  })
  if (!res.ok) throw new Error(`Create section failed (${res.status}): ${await res.text()}`)
  const body = (await res.json()) as { id: string }
  return body.id
}

async function putAssignToOverrides(
  token: string,
  courseCode: string,
  itemId: string,
  targets: Array<Record<string, unknown>>,
): Promise<{ targets: unknown[]; orphaned: boolean }> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/items/${encodeURIComponent(itemId)}/overrides`,
    { method: 'PUT', headers: authHeaders(token), body: JSON.stringify({ targets }) },
  )
  if (!res.ok) throw new Error(`Put assign-to overrides failed (${res.status}): ${await res.text()}`)
  return res.json() as Promise<{ targets: unknown[]; orphaned: boolean }>
}

async function getAssignment(
  token: string,
  courseCode: string,
  itemId: string,
): Promise<{ status: number; dueAt: string | null }> {
  const res = await fetch(
    `${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/assignments/${encodeURIComponent(itemId)}`,
    { headers: { Authorization: `Bearer ${token}` } },
  )
  if (res.status === 404) return { status: 404, dueAt: null }
  if (!res.ok) throw new Error(`Get assignment failed (${res.status}): ${await res.text()}`)
  const body = (await res.json()) as { dueAt: string | null }
  return { status: res.status, dueAt: body.dueAt }
}

async function getCourseStructureItemIds(token: string, courseCode: string): Promise<string[]> {
  const res = await fetch(`${apiBase}/api/v1/courses/${encodeURIComponent(courseCode)}/structure`, {
    headers: { Authorization: `Bearer ${token}` },
  })
  if (!res.ok) throw new Error(`Get structure failed (${res.status}): ${await res.text()}`)
  const body = (await res.json()) as { items: Array<{ id: string }> }
  return body.items.map((i) => i.id)
}

test.describe('Differentiated assignments (assign-to overrides)', () => {
  test('section targets give each section its own due date and hide items from non-targeted students', async () => {
    const adminToken = await getAdminToken()
    if (!(await enableAssignToOverrides(adminToken))) {
      test.skip(true, 'Could not enable ff_assign_to_overrides (no platform admin)')
      return
    }

    const instructor = await apiSignup({
      email: `${uid('inst')}@e2e.test`,
      password: PASSWORD,
      displayName: 'Instructor',
    })
    const studentAEmail = `${uid('stuA')}@e2e.test`
    const studentA = await apiSignup({ email: studentAEmail, password: PASSWORD, displayName: 'Student A' })
    const studentBEmail = `${uid('stuB')}@e2e.test`
    const studentB = await apiSignup({ email: studentBEmail, password: PASSWORD, displayName: 'Student B' })

    const course = await apiCreateCourse(instructor.access_token, { title: 'Differentiated 101' })
    await apiPatchCourseFeatures(instructor.access_token, course.courseCode, { sectionsEnabled: true })
    const sectionA = await createSection(instructor.access_token, course.courseCode, 'A')
    const sectionB = await createSection(instructor.access_token, course.courseCode, 'B')

    await apiEnroll(instructor.access_token, course.courseCode, studentAEmail, 'student', {
      memberToken: studentA.access_token,
    })
    await apiEnroll(instructor.access_token, course.courseCode, studentBEmail, 'student', {
      memberToken: studentB.access_token,
    })
    const enrollmentA = await getEnrollmentIdByDisplayName(instructor.access_token, course.courseCode, 'Student A')
    const enrollmentB = await getEnrollmentIdByDisplayName(instructor.access_token, course.courseCode, 'Student B')
    await transferEnrollmentSection(instructor.access_token, enrollmentA, sectionA)
    await transferEnrollmentSection(instructor.access_token, enrollmentB, sectionB)

    const mod = await apiCreateModule(instructor.access_token, course.courseCode, 'Module 1')
    const assignment = await apiCreateAssignment(instructor.access_token, course.courseCode, mod.id, 'Essay 1')

    // AC-1: Section A due Friday, Section B due Monday.
    const dueFriday = '2026-01-09T23:59:00.000Z'
    const dueMonday = '2026-01-12T23:59:00.000Z'
    await putAssignToOverrides(instructor.access_token, course.courseCode, assignment.id, [
      { targetType: 'section', targetId: sectionA, dueAt: dueFriday },
      { targetType: 'section', targetId: sectionB, dueAt: dueMonday },
    ])

    const aView = await getAssignment(studentA.access_token, course.courseCode, assignment.id)
    expect(aView.status).toBe(200)
    expect(new Date(aView.dueAt ?? '').getTime()).toBe(new Date(dueFriday).getTime())

    const bView = await getAssignment(studentB.access_token, course.courseCode, assignment.id)
    expect(bView.status).toBe(200)
    expect(new Date(bView.dueAt ?? '').getTime()).toBe(new Date(dueMonday).getTime())

    // AC-2: targeting only Section A hides the item entirely from Section B.
    await putAssignToOverrides(instructor.access_token, course.courseCode, assignment.id, [
      { targetType: 'section', targetId: sectionA, dueAt: dueFriday },
    ])

    const aIds = await getCourseStructureItemIds(studentA.access_token, course.courseCode)
    expect(aIds).toContain(assignment.id)
    const bIds = await getCourseStructureItemIds(studentB.access_token, course.courseCode)
    expect(bIds).not.toContain(assignment.id)

    const bViewAfterHide = await getAssignment(studentB.access_token, course.courseCode, assignment.id)
    expect(bViewAfterHide.status).toBe(404)
  })
})
