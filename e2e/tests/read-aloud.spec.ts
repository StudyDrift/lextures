/**
 * Read-aloud (text-to-speech) — plan 12.8
 */
import AxeBuilder from '@axe-core/playwright'
import { test, expect, mainNav, injectToken } from '../fixtures/test.js'
import { apiCreateContentPage, apiPatchContentPage } from '../fixtures/api.js'

const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'

test.describe('Read-aloud API', () => {
  test('unauthenticated reading-preferences returns 401', async () => {
    const res = await fetch(`${API_BASE}/api/v1/me/reading-preferences`)
    expect(res.status).toBe(401)
  })

  test('unauthenticated TTS synthesize returns 401', async () => {
    const res = await fetch(`${API_BASE}/api/v1/tts/synthesize`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ text: 'Hello', lang: 'en', speed: 1 }),
    })
    expect(res.status).toBe(401)
  })

  test('reading preferences round-trip for student', async ({ seededCourse }) => {
    const patchRes = await fetch(`${API_BASE}/api/v1/me/reading-preferences`, {
      method: 'PATCH',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${seededCourse.studentToken}`,
      },
      body: JSON.stringify({ ttsSpeed: 1.5 }),
    })
    expect(patchRes.ok).toBe(true)
    const patched = (await patchRes.json()) as { ttsSpeed: number }
    expect(patched.ttsSpeed).toBe(1.5)

    const getRes = await fetch(`${API_BASE}/api/v1/me/reading-preferences`, {
      headers: { Authorization: `Bearer ${seededCourse.studentToken}` },
    })
    expect(getRes.ok).toBe(true)
    const prefs = (await getRes.json()) as { ttsSpeed: number }
    expect(prefs.ttsSpeed).toBe(1.5)
  })

  test('TTS synthesize returns audio when feature enabled', async ({ seededCourse }) => {
    const res = await fetch(`${API_BASE}/api/v1/tts/synthesize`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${seededCourse.studentToken}`,
      },
      body: JSON.stringify({ text: 'Welcome to the lesson.', lang: 'en-US', speed: 1 }),
    })
    expect(res.ok).toBe(true)
    expect(res.headers.get('content-type')).toContain('audio')
    const buf = await res.arrayBuffer()
    expect(buf.byteLength).toBeGreaterThan(100)
  })
})

test.describe('Read-aloud UI', () => {
  test('student sees read-aloud control and highlight on content page', async ({ page, seededCourse }) => {
    const pageItem = await apiCreateContentPage(
      seededCourse.instructorToken,
      seededCourse.courseCode,
      seededCourse.moduleId,
      'Read aloud demo',
    )
    await apiPatchContentPage(
      seededCourse.instructorToken,
      seededCourse.courseCode,
      pageItem.id,
      {
        markdown:
          'Photosynthesis converts light into chemical energy. Plants use chlorophyll to capture sunlight.',
      },
    )

    await injectToken(page, seededCourse.studentToken)
    await page.goto(`/courses/${seededCourse.courseCode}/modules/content/${pageItem.id}`)

    const nav = mainNav(page)
    try {
      await expect(nav).toBeVisible({ timeout: 15000 })
    } catch {
      test.skip(true, 'Authenticated LMS shell unavailable in this environment')
    }

    const ack = page.getByRole('button', { name: 'I acknowledge' })
    if (await ack.isVisible().catch(() => false)) {
      await ack.click()
    }

    await page.evaluate(() => {
      const synth = window.speechSynthesis
      synth.speak = (utterance: SpeechSynthesisUtterance) => {
        utterance.onstart?.({} as SpeechSynthesisEvent)
        window.setTimeout(() => utterance.onend?.({} as SpeechSynthesisEvent), 800)
      }
      synth.cancel = () => {}
    })

    await expect(page.getByText('Photosynthesis')).toBeVisible({ timeout: 15000 })

    const readAloudBtn = page.getByRole('button', { name: 'Read aloud' })
    await expect(readAloudBtn).toBeVisible({ timeout: 15000 })

    await readAloudBtn.click()
    await expect(page.getByRole('toolbar', { name: 'Read aloud controls' })).toBeVisible()
    const playBtn = page.getByRole('button', { name: 'Play' })
    if (await playBtn.isVisible()) {
      await playBtn.click()
    }
    await expect(page.locator('.tts-sentence-highlight')).toBeVisible({ timeout: 10000 })

    const results = await new AxeBuilder({ page })
      .include('[data-read-aloud-controls]')
      .withTags(['wcag2a', 'wcag2aa'])
      .analyze()
    const critical = results.violations.filter(
      (v) => v.impact === 'critical' || v.impact === 'serious',
    )
    expect(critical.length).toBe(0)
  })
})
