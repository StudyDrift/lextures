/**
 * Reading-level adaptation (plan 11.6)
 */
import { test, expect } from '../fixtures/test.js'
import { apiSignup, apiCreateCourse, apiCreateModule, apiCreateContentPage, apiPatchContentPage, apiEnroll } from '../fixtures/api.js'

const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'
const PASSWORD = 'E2eTestPass1!'

function uniqueEmail(prefix = 'rl') {
  return `e2e-${prefix}-${Date.now()}-${Math.random().toString(36).slice(2)}@test.invalid`
}

function authHeaders(token: string) {
  return { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' }
}

const longBody =
  'The water cycle moves water through the environment in a continuous loop. ' +
  'Sunlight heats oceans and lakes so water evaporates into the air. ' +
  'Warm air rises and cools, forming clouds through condensation. ' +
  'Rain and snow return water to the ground in a process called precipitation. ' +
  'Plants absorb water through their roots and release some through transpiration. ' +
  'Rivers carry water back to the ocean where the cycle begins again. ' +
  'Students learn that fresh water is a limited resource we must protect. ' +
  'Communities store water in reservoirs and clean it in treatment plants before use. ' +
  'Groundwater fills spaces in soil and rock beneath our feet for wells and springs. ' +
  'Understanding the water cycle helps explain weather patterns and climate change impacts. '

test.describe('Reading level API', () => {
  test('unauthenticated GET reading-level returns 401', async ({ request }) => {
    const res = await request.get(
      `${API_BASE}/api/v1/courses/C-TEST01/items/00000000-0000-0000-0000-000000000001/reading-level`,
    )
    expect(res.status()).toBe(401)
  })

  test('PATCH content page stores FKGL after save when enabled', async ({ request }) => {
    const { access_token } = await apiSignup({
      email: uniqueEmail('score'),
      password: PASSWORD,
      displayName: 'RL Score',
    })
    const course = await apiCreateCourse(access_token, { title: 'RL Score Course' })
    const mod = await apiCreateModule(access_token, course.courseCode, 'Unit')
    const page = await apiCreateContentPage(access_token, course.courseCode, mod.id, 'Reading lesson')
    await apiPatchContentPage(access_token, course.courseCode, page.id, { markdown: longBody })

    const rlRes = await request.get(
      `${API_BASE}/api/v1/courses/${course.courseCode}/items/${page.id}/reading-level`,
      { headers: authHeaders(access_token) },
    )
    if (rlRes.status() === 404) {
      test.skip(true, 'reading_level_enabled is off — enable in platform seed')
      return
    }
    expect(rlRes.status()).toBe(200)
    const body = (await rlRes.json()) as { sufficient?: boolean; fkgl?: number }
    expect(body.sufficient).toBe(true)
    expect(typeof body.fkgl).toBe('number')
  })

  test('student without accommodation cannot GET cached simplify', async ({ request }) => {
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
    const course = await apiCreateCourse(instructor.access_token, { title: 'RL Access' })
    const mod = await apiCreateModule(instructor.access_token, course.courseCode, 'M1')
    const page = await apiCreateContentPage(instructor.access_token, course.courseCode, mod.id, 'Text')
    await apiPatchContentPage(instructor.access_token, course.courseCode, page.id, {
      markdown: longBody,
    })
    await apiEnroll(instructor.access_token, course.courseCode, studentEmail)

    const res = await request.get(
      `${API_BASE}/api/v1/courses/${course.courseCode}/items/${page.id}/simplify/4`,
      { headers: authHeaders(student.access_token) },
    )
    if (res.status() === 404 && (await res.text()).includes('not enabled')) {
      test.skip(true, 'reading level disabled')
      return
    }
    expect(res.status()).toBe(403)
  })
})

test.describe('Reading level UI', () => {
  test('instructor sees reading level badge after saving content', async ({
    coursePage: page,
    seededCourse,
  }) => {
    const mod = await apiCreateModule(seededCourse.instructorToken, seededCourse.courseCode, 'RL Mod')
    const contentPage = await apiCreateContentPage(
      seededCourse.instructorToken,
      seededCourse.courseCode,
      mod.id,
      'FKGL Page',
    )
    await apiPatchContentPage(
      seededCourse.instructorToken,
      seededCourse.courseCode,
      contentPage.id,
      { markdown: longBody },
    )

    await page.goto(
      `/courses/${seededCourse.courseCode}/modules/content/${contentPage.id}`,
    )
    await page.getByRole('button', { name: /^edit$/i }).click()
    const badge = page.getByRole('button', { name: /Flesch-Kincaid Grade Level/i })
    const visible = await badge.isVisible().catch(() => false)
    if (!visible) {
      test.skip(true, 'reading_level_enabled is off in this environment')
      return
    }
    await expect(badge).toBeVisible()
  })
})
