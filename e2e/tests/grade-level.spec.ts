/**
 * Grade-Level Scoping (docs/plan/13-k12-specific/13.6-grade-level-scoping.md)
 *
 *   [x] Unauthenticated access returns 401
 *   [x] Create course with grade level stored correctly
 *   [x] Create course without grade level returns null
 *   [x] GET /api/v1/courses?grade_level=5 filters by grade level
 *   [x] GET /api/v1/courses?grade_level=INVALID returns 400
 *   [x] PUT course sets / clears grade level
 *   [x] Grade level preserved in GET /api/v1/courses/:code
 */
import { test, expect } from '@playwright/test'
import { apiSignup, apiCreateCourse } from '../fixtures/api.js'

const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'
const PASSWORD = 'E2eTestPass1!'

function uniqueEmail(prefix = 'gl') {
  return `e2e-${prefix}-${Date.now()}-${Math.random().toString(36).slice(2)}@test.invalid`
}

function authHeaders(token: string) {
  return { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' }
}

async function apiCreateCourseWithGradeLevel(
  token: string,
  payload: { title: string; gradeLevel?: string | null },
): Promise<Record<string, unknown>> {
  const res = await fetch(`${API_BASE}/api/v1/courses`, {
    method: 'POST',
    headers: authHeaders(token),
    body: JSON.stringify(payload),
  })
  if (!res.ok) {
    const body = await res.text()
    throw new Error(`Create course failed (${res.status}): ${body}`)
  }
  return res.json() as Promise<Record<string, unknown>>
}

async function apiPutCourse(
  token: string,
  courseCode: string,
  payload: Record<string, unknown>,
): Promise<Record<string, unknown>> {
  const res = await fetch(`${API_BASE}/api/v1/courses/${encodeURIComponent(courseCode)}`, {
    method: 'PUT',
    headers: authHeaders(token),
    body: JSON.stringify(payload),
  })
  if (!res.ok) {
    const body = await res.text()
    throw new Error(`PUT course failed (${res.status}): ${body}`)
  }
  return res.json() as Promise<Record<string, unknown>>
}

async function apiGetCourse(
  token: string,
  courseCode: string,
): Promise<Record<string, unknown>> {
  const res = await fetch(`${API_BASE}/api/v1/courses/${encodeURIComponent(courseCode)}`, {
    headers: { Authorization: `Bearer ${token}` },
  })
  if (!res.ok) {
    const body = await res.text()
    throw new Error(`GET course failed (${res.status}): ${body}`)
  }
  return res.json() as Promise<Record<string, unknown>>
}

async function apiListCourses(
  token: string,
  params?: { grade_level?: string },
): Promise<{ courses: Record<string, unknown>[] }> {
  const qs = params?.grade_level ? `?grade_level=${encodeURIComponent(params.grade_level)}` : ''
  const res = await fetch(`${API_BASE}/api/v1/courses${qs}`, {
    headers: { Authorization: `Bearer ${token}` },
  })
  if (!res.ok) {
    const body = await res.text()
    throw new Error(`List courses failed (${res.status}): ${body}`)
  }
  return res.json() as Promise<{ courses: Record<string, unknown>[] }>
}

test.describe('Grade-Level Scoping API', () => {
  test('unauthenticated access returns 401', async ({ request }) => {
    const res = await request.fetch(`${API_BASE}/api/v1/courses?grade_level=5`)
    expect(res.status()).toBe(401)
  })

  test('create course with grade level stores it correctly', async () => {
    const { access_token: token } = await apiSignup({ email: uniqueEmail(), password: PASSWORD })
    const course = await apiCreateCourseWithGradeLevel(token, { title: 'Grade 5 Science', gradeLevel: '5' })
    expect(course.gradeLevel).toBe('5')
    const code = course.courseCode as string
    const fetched = await apiGetCourse(token, code)
    expect(fetched.gradeLevel).toBe('5')
  })

  test('create course without grade level returns null gradeLevel', async () => {
    const { access_token: token } = await apiSignup({ email: uniqueEmail(), password: PASSWORD })
    const course = await apiCreateCourse(token, { title: 'No Grade Level Course' })
    const fetched = await apiGetCourse(token, course.courseCode)
    expect(fetched.gradeLevel == null || fetched.gradeLevel === undefined).toBe(true)
  })

  test('GET courses filtered by grade_level returns only matching courses', async () => {
    const { access_token: token } = await apiSignup({ email: uniqueEmail(), password: PASSWORD })
    await apiCreateCourseWithGradeLevel(token, { title: 'Grade 5 Math', gradeLevel: '5' })
    await apiCreateCourseWithGradeLevel(token, { title: 'Grade 3 Reading', gradeLevel: '3' })
    await apiCreateCourse(token, { title: 'Higher Ed Course' })

    const filtered = await apiListCourses(token, { grade_level: '5' })
    expect(filtered.courses.length).toBeGreaterThanOrEqual(1)
    for (const c of filtered.courses) {
      expect(c.gradeLevel).toBe('5')
    }
    const grade5Titles = filtered.courses.map((c) => c.title)
    expect(grade5Titles).toContain('Grade 5 Math')
    expect(grade5Titles).not.toContain('Grade 3 Reading')
    expect(grade5Titles).not.toContain('Higher Ed Course')
  })

  test('GET courses with invalid grade_level returns 400', async () => {
    const { access_token: token } = await apiSignup({ email: uniqueEmail(), password: PASSWORD })
    const res = await fetch(`${API_BASE}/api/v1/courses?grade_level=INVALID_GRADE`, {
      headers: { Authorization: `Bearer ${token}` },
    })
    expect(res.status()).toBe(400)
    const body = (await res.json()) as { error?: { code?: string } }
    expect(body.error?.code).toBe('INVALID_INPUT')
  })

  test('PUT course sets grade level', async () => {
    const { access_token: token } = await apiSignup({ email: uniqueEmail(), password: PASSWORD })
    const created = await apiCreateCourse(token, { title: 'Update Grade Level' })
    const code = created.courseCode

    const updated = await apiPutCourse(token, code, {
      title: 'Update Grade Level',
      description: '',
      published: false,
      startsAt: null,
      endsAt: null,
      visibleFrom: null,
      hiddenAt: null,
      scheduleMode: 'fixed',
      relativeEndAfter: null,
      relativeHiddenAfter: null,
      gradeLevel: '7',
    })
    expect(updated.gradeLevel).toBe('7')

    const fetched = await apiGetCourse(token, code)
    expect(fetched.gradeLevel).toBe('7')
  })

  test('PUT course clears grade level when set to null', async () => {
    const { access_token: token } = await apiSignup({ email: uniqueEmail(), password: PASSWORD })
    const created = await apiCreateCourseWithGradeLevel(token, { title: 'Clear Grade Level', gradeLevel: '8' })
    const code = created.courseCode as string

    await apiPutCourse(token, code, {
      title: 'Clear Grade Level',
      description: '',
      published: false,
      startsAt: null,
      endsAt: null,
      visibleFrom: null,
      hiddenAt: null,
      scheduleMode: 'fixed',
      relativeEndAfter: null,
      relativeHiddenAfter: null,
      gradeLevel: '',
    })

    const fetched = await apiGetCourse(token, code)
    expect(fetched.gradeLevel == null || fetched.gradeLevel === undefined).toBe(true)
  })

  test('grade level band values are accepted (K-2, 3-5, 6-8, 9-12, K-12)', async () => {
    const { access_token: token } = await apiSignup({ email: uniqueEmail(), password: PASSWORD })
    const bands = ['K', 'K-2', '3-5', '6-8', '9-12', 'K-12']
    for (const band of bands) {
      const course = await apiCreateCourseWithGradeLevel(token, {
        title: `Band ${band} Course`,
        gradeLevel: band,
      })
      expect(course.gradeLevel).toBe(band)
    }
  })
})
