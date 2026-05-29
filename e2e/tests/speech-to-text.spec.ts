/**
 * Speech-to-text input — plan 12.9
 */
import AxeBuilder from '@axe-core/playwright'
import { test, expect } from '../fixtures/test.js'

const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'
const E2E_ADMIN_EMAIL = process.env.E2E_ADMIN_EMAIL ?? 'admin@e2e.test'
const E2E_ADMIN_PASSWORD = process.env.E2E_ADMIN_PASSWORD ?? 'E2eTestPass1!'

/** Platform settings require global admin; global E2E seed enables STT by default. */
async function adminToken(): Promise<string> {
  const loginRes = await fetch(`${API_BASE}/api/v1/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email: E2E_ADMIN_EMAIL, password: E2E_ADMIN_PASSWORD }),
  })
  expect(loginRes.ok).toBe(true)
  const { access_token } = (await loginRes.json()) as { access_token: string }
  return access_token
}

async function patchSpeechToTextEnabled(token: string, enabled: boolean): Promise<void> {
  const res = await fetch(`${API_BASE}/api/v1/settings/platform`, {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify({
      speechToTextEnabled: enabled,
      updateMask: ['speechToTextEnabled'],
    }),
  })
  expect(res.ok).toBe(true)
}

async function enableSpeechToText(): Promise<void> {
  await patchSpeechToTextEnabled(await adminToken(), true)
}

function installSpeechRecognitionMock(page: import('@playwright/test').Page, withFinalResult = false) {
  return page.addInitScript((emitFinal) => {
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
        if (emitFinal) {
          queueMicrotask(() => {
            this.onresult?.({
              resultIndex: 0,
              results: [{ isFinal: true, 0: { transcript: 'Hello world' } }],
            })
            this.onend?.()
          })
        }
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
  }, withFinalResult)
}

async function openSyllabusEditor(page: import('@playwright/test').Page, courseCode: string) {
  await page.goto(`/courses/${encodeURIComponent(courseCode)}/syllabus`)
  await page.getByRole('button', { name: /^edit$/i }).click()
  const editor = page.locator('[contenteditable="true"]').first()
  await expect(editor).toBeVisible({ timeout: 15_000 })
  await editor.click()
  return editor
}

test.describe('Speech-to-text API', () => {
  test('reading-preferences PATCH returns 404 for stt fields when feature disabled', async ({
    seededCourse,
  }) => {
    const admin = await adminToken()
    await patchSpeechToTextEnabled(admin, false)
    try {
      const res = await fetch(`${API_BASE}/api/v1/me/reading-preferences`, {
        method: 'PATCH',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${seededCourse.studentToken}`,
        },
        body: JSON.stringify({ sttEnabled: true, sttLanguage: 'en-US' }),
      })
      expect(res.status).toBe(404)
    } finally {
      await patchSpeechToTextEnabled(admin, true)
    }
  })

  test('reading-preferences round-trip when enabled', async ({ seededCourse }) => {
    await enableSpeechToText()
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
  test.beforeEach(async () => {
    await enableSpeechToText()
  })

  test('dictation inserts text into syllabus block editor', async ({ coursePage: page, seededCourse }) => {
    await installSpeechRecognitionMock(page, true)
    const editor = await openSyllabusEditor(page, seededCourse.courseCode)

    const dictationBtn = page.getByRole('button', { name: 'Start dictation' })
    await expect(dictationBtn).toBeVisible({ timeout: 15_000 })
    await dictationBtn.click()

    await expect(editor).toContainText(/Hello world/, { timeout: 10_000 })
  })

  test('dictation button passes axe when visible', async ({ coursePage: page, seededCourse }) => {
    await installSpeechRecognitionMock(page, false)
    await openSyllabusEditor(page, seededCourse.courseCode)

    const dictationBtn = page.getByRole('button', { name: 'Start dictation' })
    await expect(dictationBtn).toBeVisible({ timeout: 15_000 })

    const results = await new AxeBuilder({ page }).include('main').analyze()
    expect(results.violations).toEqual([])
  })
})
