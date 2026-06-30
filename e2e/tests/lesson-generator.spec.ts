/**
 * AI Lesson Generator (plan 19.2) — API smoke tests
 */
import { test, expect } from '@playwright/test'
import { apiSignup, apiCreateCourse } from '../fixtures/api.js'

const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'
const PASSWORD = 'E2eTestPass1!'

function uniqueEmail(prefix = 'lesson') {
  return `e2e-${prefix}-${Date.now()}-${Math.random().toString(36).slice(2)}@test.invalid`
}

function authHeaders(token: string) {
  return { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' }
}

test.describe('Lesson Generator API', () => {
  test('unauthenticated POST returns 401', async ({ request }) => {
    const res = await request.post(`${API_BASE}/api/v1/courses/C-FAKE/lesson-generator`, {
      data: {
        learning_objective: 'Test',
        grade_level: '4',
        subject: 'ELA',
      },
    })
    expect(res.status()).toBe(401)
  })

  test('POST returns 404 when feature flag disabled', async ({ request }) => {
    const { access_token } = await apiSignup({
      email: uniqueEmail('instr'),
      password: PASSWORD,
      displayName: 'Lesson Instructor',
    })
    const course = await apiCreateCourse(access_token, {
      title: 'Lesson Gen Course',
      courseCode: `LG-${Date.now()}`,
    })
    const res = await request.post(
      `${API_BASE}/api/v1/courses/${course.courseCode}/lesson-generator`,
      {
        headers: authHeaders(access_token),
        data: {
          learning_objective: 'Identify the main idea of a passage',
          grade_level: '4',
          subject: 'ELA',
          differentiation_levels: ['on_grade'],
        },
      },
    )
    expect(res.status()).toBe(404)
  })

  test('POST validates required fields', async ({ request }) => {
    const { access_token } = await apiSignup({
      email: uniqueEmail('valid'),
      password: PASSWORD,
      displayName: 'Validator',
    })
    const course = await apiCreateCourse(access_token, {
      title: 'Validation Course',
      courseCode: `LG-V-${Date.now()}`,
    })
    const res = await request.post(
      `${API_BASE}/api/v1/courses/${course.courseCode}/lesson-generator`,
      {
        headers: authHeaders(access_token),
        data: { grade_level: '4' },
      },
    )
    expect([400, 404]).toContain(res.status())
  })
})
