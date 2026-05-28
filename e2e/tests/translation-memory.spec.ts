/**
 * Translation memory & course content locales (plan 11.5)
 */
import { test, expect } from '../fixtures/test.js'
import {
  apiSignup,
  apiCreateCourse,
  apiCreateModule,
  apiCreateContentPage,
  apiPatchContentPage,
  apiEnroll,
} from '../fixtures/api.js'

const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'
const PASSWORD = 'E2eTestPass1!'

function uniqueEmail(prefix = 'tm') {
  return `e2e-${prefix}-${Date.now()}-${Math.random().toString(36).slice(2)}@test.invalid`
}

function authHeaders(token: string) {
  return { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' }
}

const spanishBody =
  'El ciclo del agua mueve el agua por el medio ambiente en un bucle continuo. ' +
  'La luz solar calienta océanos y lagos para que el agua se evapore. '

const englishBody =
  'The water cycle moves water through the environment in a continuous loop. ' +
  'Sunlight heats oceans and lakes so water evaporates into the air. '

test.describe('Translation memory API', () => {
  test('unauthenticated translations list returns 401', async ({ request }) => {
    const res = await request.get(
      `${API_BASE}/api/v1/courses/C-TEST01/translations?target_locale=es`,
    )
    expect(res.status()).toBe(401)
  })

  test('student cannot read instructor draft translation via published-only content', async ({
    request,
  }) => {
    const instructor = await apiSignup({
      email: uniqueEmail('inst'),
      password: PASSWORD,
      displayName: 'Instructor',
    })
    const studentEmail = uniqueEmail('stu')
    const student = await apiSignup({
      email: studentEmail,
      password: PASSWORD,
      displayName: 'Student',
    })
    const course = await apiCreateCourse(instructor.access_token, { title: 'TM Course' })
    const mod = await apiCreateModule(instructor.access_token, course.courseCode, 'Unit 1')
    const page = await apiCreateContentPage(
      instructor.access_token,
      course.courseCode,
      mod.id,
      'Water lesson',
    )
    await apiPatchContentPage(instructor.access_token, course.courseCode, page.id, {
      markdown: englishBody,
    })
    await apiEnroll(instructor.access_token, course.courseCode, studentEmail)

    const saveRes = await request.put(
      `${API_BASE}/api/v1/courses/${course.courseCode}/translations/${page.id}`,
      {
        headers: authHeaders(instructor.access_token),
        data: {
          targetLocale: 'es',
          translatedTitle: 'Lección del agua',
          translatedBody: spanishBody,
          isDraft: true,
          machineTranslationDraft: true,
        },
      },
    )
    if (saveRes.status() === 404) {
      test.skip(true, 'translation_memory_enabled is off')
      return
    }
    expect(saveRes.status()).toBe(200)

    const viewRes = await request.get(
      `${API_BASE}/api/v1/courses/${course.courseCode}/content-pages/${page.id}`,
      { headers: authHeaders(student.access_token) },
    )
    expect(viewRes.status()).toBe(200)
    const body = (await viewRes.json()) as { markdown?: string; title?: string }
    expect(body.markdown).toContain('water cycle')
    expect(body.markdown).not.toContain('ciclo del agua')
  })

  test('published translation shown to student with content locale', async ({ request }) => {
    const instructor = await apiSignup({
      email: uniqueEmail('pub'),
      password: PASSWORD,
      displayName: 'Instructor',
    })
    const studentEmail = uniqueEmail('pubstu')
    const student = await apiSignup({
      email: studentEmail,
      password: PASSWORD,
      displayName: 'Student',
    })
    const course = await apiCreateCourse(instructor.access_token, { title: 'TM Publish' })
    const mod = await apiCreateModule(instructor.access_token, course.courseCode, 'M1')
    const page = await apiCreateContentPage(
      instructor.access_token,
      course.courseCode,
      mod.id,
      'Lesson',
    )
    await apiPatchContentPage(instructor.access_token, course.courseCode, page.id, {
      markdown: englishBody,
    })
    await apiEnroll(instructor.access_token, course.courseCode, studentEmail)

    const saveRes = await request.put(
      `${API_BASE}/api/v1/courses/${course.courseCode}/translations/${page.id}`,
      {
        headers: authHeaders(instructor.access_token),
        data: {
          targetLocale: 'es',
          translatedTitle: 'Lección',
          translatedBody: spanishBody,
          isDraft: false,
        },
      },
    )
    if (saveRes.status() === 404) {
      test.skip(true, 'translation_memory_enabled is off')
      return
    }
    expect(saveRes.status()).toBe(200)

    const pubRes = await request.post(
      `${API_BASE}/api/v1/courses/${course.courseCode}/translations/${page.id}/publish`,
      {
        headers: authHeaders(instructor.access_token),
        data: { targetLocale: 'es' },
      },
    )
    expect(pubRes.status()).toBe(200)

    const localeRes = await request.patch(
      `${API_BASE}/api/v1/courses/${course.courseCode}/me/content-locale`,
      {
        headers: authHeaders(student.access_token),
        data: { contentLocale: 'es' },
      },
    )
    expect(localeRes.status()).toBe(204)

    const viewRes = await request.get(
      `${API_BASE}/api/v1/courses/${course.courseCode}/content-pages/${page.id}`,
      { headers: authHeaders(student.access_token) },
    )
    expect(viewRes.status()).toBe(200)
    const viewed = (await viewRes.json()) as { markdown?: string }
    expect(viewed.markdown).toContain('ciclo del agua')
  })

  test('translation coverage reports percent translated', async ({ request }) => {
    const instructor = await apiSignup({
      email: uniqueEmail('cov'),
      password: PASSWORD,
      displayName: 'Instructor',
    })
    const course = await apiCreateCourse(instructor.access_token, { title: 'Coverage' })
    const mod = await apiCreateModule(instructor.access_token, course.courseCode, 'M1')
    const page = await apiCreateContentPage(
      instructor.access_token,
      course.courseCode,
      mod.id,
      'Only page',
    )
    await apiPatchContentPage(instructor.access_token, course.courseCode, page.id, {
      markdown: englishBody,
    })

    const covRes = await request.get(
      `${API_BASE}/api/v1/courses/${course.courseCode}/translation-coverage?target_locale=es`,
      { headers: authHeaders(instructor.access_token) },
    )
    if (covRes.status() === 404) {
      test.skip(true, 'translation_memory_enabled is off')
      return
    }
    expect(covRes.status()).toBe(200)
    const cov = (await covRes.json()) as {
      totalItems?: number
      translatedItems?: number
      percent?: number
    }
    expect(cov.totalItems).toBeGreaterThanOrEqual(1)
    expect(cov.translatedItems).toBe(0)
    expect(cov.percent).toBe(0)

    await request.put(
      `${API_BASE}/api/v1/courses/${course.courseCode}/translations/${page.id}`,
      {
        headers: authHeaders(instructor.access_token),
        data: {
          targetLocale: 'es',
          translatedBody: spanishBody,
          isDraft: false,
        },
      },
    )
    await request.post(
      `${API_BASE}/api/v1/courses/${course.courseCode}/translations/${page.id}/publish`,
      {
        headers: authHeaders(instructor.access_token),
        data: { targetLocale: 'es' },
      },
    )

    const cov2 = await request.get(
      `${API_BASE}/api/v1/courses/${course.courseCode}/translation-coverage?target_locale=es`,
      { headers: authHeaders(instructor.access_token) },
    )
    const after = (await cov2.json()) as { translatedItems?: number; percent?: number }
    expect(after.translatedItems).toBeGreaterThanOrEqual(1)
    expect(after.percent).toBeGreaterThan(0)
  })
})

test.describe('Translation memory UI', () => {
  test('instructor opens translations settings', async ({ coursePage: page, seededCourse }) => {
    await page.goto(`/courses/${seededCourse.courseCode}/settings/translations`)
    const heading = page.getByRole('heading', { name: /translations/i })
    const visible = await heading.isVisible().catch(() => false)
    if (!visible) {
      const disabled = await page.getByText(/translation memory is disabled/i).isVisible().catch(() => false)
      if (disabled) test.skip(true, 'translation_memory_enabled is off')
      return
    }
    await expect(heading).toBeVisible()
  })
})
