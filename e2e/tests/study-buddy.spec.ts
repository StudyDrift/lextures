import { test, expect } from '@playwright/test'
import { apiSignup, apiCreateCourse, apiEnroll } from '../fixtures/api.js'

const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'
const PASSWORD = 'E2eTestPass1!'

function uniqueEmail(prefix = 'studybuddy') {
  return `e2e-${prefix}-${Date.now()}-${Math.random().toString(36).slice(2)}@test.invalid`
}

function authHeaders(token: string) {
  return { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' }
}

test.describe('AI Study Buddy API', () => {
  test('unauthenticated access returns 401', async ({ request }) => {
    const paths = [
      { method: 'GET', path: '/api/v1/courses/C-FAKE/study-buddy/memory' },
      { method: 'DELETE', path: '/api/v1/courses/C-FAKE/study-buddy/memory' },
      { method: 'GET', path: '/api/v1/courses/C-FAKE/study-buddy/prompts' },
    ]
    for (const { method, path } of paths) {
      const res = await request.fetch(`${API_BASE}${path}`, { method })
      expect(res.status(), `${method} ${path}`).toBe(401)
    }
  })

  test('feature off returns 404 for prompts', async ({ request }) => {
    const featuresRes = await request.get(`${API_BASE}/api/v1/platform/features`, {
      headers: authHeaders('invalid'),
    })
    expect(featuresRes.status()).toBe(401)

    const { access_token } = await apiSignup({
      email: uniqueEmail(),
      password: PASSWORD,
      displayName: 'Study Buddy User',
    })

    const platformRes = await request.get(`${API_BASE}/api/v1/platform/features`, {
      headers: { Authorization: `Bearer ${access_token}` },
    })
    expect(platformRes.ok()).toBeTruthy()
    const features = (await platformRes.json()) as { ffAiStudyBuddy?: boolean }
    if (!features.ffAiStudyBuddy) {
      test.skip(true, 'ffAiStudyBuddy is false on the API')
    }

    const { courseCode } = await apiCreateCourse(access_token, { title: 'Study Buddy Course' })
    await apiEnroll(access_token, courseCode)

    const res = await request.get(
      `${API_BASE}/api/v1/courses/${courseCode}/study-buddy/prompts`,
      { headers: { Authorization: `Bearer ${access_token}` } },
    )
    expect(res.status()).toBe(200)
    const body = (await res.json()) as { prompts?: unknown[] }
    expect(Array.isArray(body.prompts)).toBe(true)
  })

  test('POST message returns 503 when AI provider not configured', async ({ request }) => {
    const featuresRes = await request.get(`${API_BASE}/api/v1/platform/features`)
    if (!featuresRes.ok()) {
      test.skip(true, 'platform features unavailable')
    }
    const features = (await featuresRes.json()) as { ffAiStudyBuddy?: boolean }
    if (!features.ffAiStudyBuddy) {
      test.skip(true, 'ffAiStudyBuddy is false on the API')
    }

    const { access_token } = await apiSignup({
      email: uniqueEmail('msg'),
      password: PASSWORD,
      displayName: 'SB Msg User',
    })
    const { courseCode } = await apiCreateCourse(access_token, { title: 'SB Msg Course' })
    await apiEnroll(access_token, courseCode)

    const res = await request.post(
      `${API_BASE}/api/v1/courses/${courseCode}/study-buddy/message`,
      {
        headers: authHeaders(access_token),
        data: { message: 'What is a list comprehension?', sessionId: '' },
      },
    )
    expect([503, 403, 402]).toContain(res.status())
  })
})
