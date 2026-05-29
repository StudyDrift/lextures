/**
 * Speech-to-text input — plan 12.9
 */
import AxeBuilder from '@axe-core/playwright'
import { test, expect } from '../fixtures/test.js'

const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'

async function enableSpeechToText(token: string): Promise<void> {
  const res = await fetch(`${API_BASE}/api/v1/settings/platform`, {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify({
      speechToTextEnabled: true,
      updateMask: ['speechToTextEnabled'],
    }),
  })
  expect(res.ok).toBe(true)
}

test.describe('Speech-to-text API', () => {
  test('reading-preferences returns 404 when feature disabled', async ({ seededCourse }) => {
    const res = await fetch(`${API_BASE}/api/v1/me/reading-preferences`, {
      headers: { Authorization: `Bearer ${seededCourse.studentToken}` },
    })
    expect(res.status).toBe(404)
  })

  test('reading-preferences round-trip when enabled', async ({ seededCourse }) => {
    await enableSpeechToText(seededCourse.instructorToken)
    const getRes = await fetch(`${API_BASE}/api/v1/me/reading-preferences`, {
      headers: { Authorization: `Bearer ${seededCourse.studentToken}` },
    })
    expect(getRes.ok).toBe(true)
    const body = (await getRes.json()) as { sttEnabled: boolean; sttLanguage: string }
    expect(body.sttLanguage).toBeTruthy()

    const patchRes = await fetch(`${API_BASE}/api/v1/me/reading-preferences`, {
      method: 'PATCH',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${seededCourse.studentToken}`,
      },
      body: JSON.stringify({ sttEnabled: true, sttLanguage: 'en-US' }),
    })
    expect(patchRes.ok).toBe(true)
    const patched = (await patchRes.json()) as { sttEnabled: boolean }
    expect(patched.sttEnabled).toBe(true)
  })

  test('stt transcribe requires auth', async () => {
    const res = await fetch(`${API_BASE}/api/v1/stt/transcribe`, { method: 'POST' })
    expect(res.status).toBe(401)
  })

  test('platform features includes speechToTextEnabled', async () => {
    const loginRes = await fetch(`${API_BASE}/api/v1/auth/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        email: process.env.E2E_ADMIN_EMAIL ?? 'admin@e2e.test',
        password: process.env.E2E_ADMIN_PASSWORD ?? 'E2eTestPass1!',
      }),
    })
    expect(loginRes.ok).toBe(true)
    const { access_token: token } = (await loginRes.json()) as { access_token: string }
    const res = await fetch(`${API_BASE}/api/v1/platform/features`, {
      headers: { Authorization: `Bearer ${token}` },
    })
    expect(res.ok).toBe(true)
    const body = (await res.json()) as { speechToTextEnabled?: boolean }
    expect(typeof body.speechToTextEnabled).toBe('boolean')
  })
})

test.describe('Speech-to-text UI', () => {
  test.beforeEach(async ({ seededCourse }) => {
    await enableSpeechToText(seededCourse.instructorToken)
  })

  test('dictation inserts text into quiz short-answer field', async ({ page, seededCourse, injectToken }) => {
    await injectToken(page, seededCourse.studentToken)

    await page.addInitScript(() => {
      class MockSpeechRecognition {
        continuous = false
        interimResults = true
        lang = 'en-US'
        maxAlternatives = 1
        onstart: (() => void) | null = null
        onend: (() => void) | null = null
        onerror: ((event: { error: string }) => void) | null = null
        onresult:
          | ((event: {
              resultIndex: number
              results: Array<{ isFinal: boolean; 0: { transcript: string } }>
            }) => void)
          | null = null
        start() {
          this.onstart?.()
          queueMicrotask(() => {
            this.onresult?.({
              resultIndex: 0,
              results: [{ isFinal: true, 0: { transcript: 'Hello world' } }],
            })
            this.onend?.()
          })
        }
        stop() {
          this.onend?.()
        }
        abort() {}
      }
      // @ts-expect-error test mock
      window.SpeechRecognition = MockSpeechRecognition
      // @ts-expect-error test mock
      window.webkitSpeechRecognition = MockSpeechRecognition
      Object.defineProperty(navigator, 'mediaDevices', {
        configurable: true,
        value: {
          getUserMedia: async () => ({ getTracks: () => [{ stop: () => {} }] }),
        },
      })
    })

    await page.goto(`/courses/${encodeURIComponent(seededCourse.courseCode)}/modules`)
    await page.getByRole('link', { name: /quiz/i }).first().click()
    await page.getByRole('button', { name: /take quiz|start/i }).first().click()

    const dictationBtn = page.getByRole('button', { name: 'Start dictation' }).first()
    await expect(dictationBtn).toBeVisible({ timeout: 15_000 })
    await dictationBtn.click()

    const answer = page.getByPlaceholder('Your answer').first()
    await expect(answer).toHaveValue(/Hello world/, { timeout: 10_000 })
  })

  test('dictation button passes axe when visible', async ({ page, seededCourse, injectToken }) => {
    await injectToken(page, seededCourse.studentToken)
    await page.addInitScript(() => {
      class MockSpeechRecognition {
        continuous = false
        interimResults = true
        lang = 'en-US'
        maxAlternatives = 1
        onstart: (() => void) | null = null
        onend: (() => void) | null = null
        onerror: ((event: { error: string }) => void) | null = null
        onresult: (() => void) | null = null
        start() {
          this.onstart?.()
        }
        stop() {
          this.onend?.()
        }
        abort() {}
      }
      // @ts-expect-error test mock
      window.SpeechRecognition = MockSpeechRecognition
      Object.defineProperty(navigator, 'mediaDevices', {
        configurable: true,
        value: {
          getUserMedia: async () => ({ getTracks: () => [{ stop: () => {} }] }),
        },
      })
    })

    await page.goto(`/courses/${encodeURIComponent(seededCourse.courseCode)}/modules`)
    await page.getByRole('link', { name: /quiz/i }).first().click()
    await page.getByRole('button', { name: /take quiz|start/i }).first().click()

    const dictationBtn = page.getByRole('button', { name: 'Start dictation' }).first()
    await expect(dictationBtn).toBeVisible({ timeout: 15_000 })

    const results = await new AxeBuilder({ page }).include('main').analyze()
    expect(results.violations).toEqual([])
  })
})
