/**
 * xAPI / Caliper emission (plan 9.6)
 *
 *   [x] GET course events without auth returns 401
 *   [x] Student cannot access course event log API (403)
 *   [x] content_open records an xAPI statement visible to instructor
 *   [x] Actor hash hides email when LRS_ANONYMIZE_ACTORS is enabled (unit-tested server-side)
 */
import { test, expect } from '../fixtures/test.js'

const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'

test('GET course events: unauthenticated returns 401', async () => {
  const res = await fetch(`${API_BASE}/api/v1/courses/demo/events`)
  expect(res.status).toBe(401)
})

test('GET course events: student returns 403', async ({ seededCourse }) => {
  const res = await fetch(
    `${API_BASE}/api/v1/courses/${encodeURIComponent(seededCourse.courseCode)}/events`,
    { headers: { Authorization: `Bearer ${seededCourse.studentToken}` } },
  )
  expect(res.status).toBe(403)
})

test('Event log: content_open appears for instructor', async ({ seededCourse }) => {
  const pageId = seededCourse.contentPageId
  const ctxRes = await fetch(
    `${API_BASE}/api/v1/courses/${encodeURIComponent(seededCourse.courseCode)}/course-context`,
    {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${seededCourse.studentToken}`,
      },
      body: JSON.stringify({ kind: 'content_open', structureItemId: pageId }),
    },
  )
  expect(ctxRes.status).toBe(204)

  await expect
    .poll(async () => {
      const res = await fetch(
        `${API_BASE}/api/v1/courses/${encodeURIComponent(seededCourse.courseCode)}/events`,
        { headers: { Authorization: `Bearer ${seededCourse.instructorToken}` } },
      )
      if (!res.ok) return []
      const data = (await res.json()) as { events: { verb: string }[] }
      return data.events ?? []
    })
    .toContainEqual(expect.objectContaining({ verb: expect.stringContaining('experienced') }))

  const listRes = await fetch(
    `${API_BASE}/api/v1/courses/${encodeURIComponent(seededCourse.courseCode)}/events`,
    { headers: { Authorization: `Bearer ${seededCourse.instructorToken}` } },
  )
  expect(listRes.ok).toBe(true)
  const list = (await listRes.json()) as { events: { verb: string; fullJson: unknown }[] }
  expect(list.events.length).toBeGreaterThan(0)
  const body = JSON.stringify(list.events[0]?.fullJson ?? {})
  expect(body).toContain('xapi')
  expect(body).toContain('caliper')
})

test('Event log UI: instructor sees table row', async ({ coursePage: page, seededCourse }) => {
  await fetch(
    `${API_BASE}/api/v1/courses/${encodeURIComponent(seededCourse.courseCode)}/course-context`,
    {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${seededCourse.studentToken}`,
      },
      body: JSON.stringify({
        kind: 'content_open',
        structureItemId: seededCourse.contentPageId,
      }),
    },
  )

  await page.goto(`/courses/${seededCourse.courseCode}/event-log`)
  await expect(page.getByRole('heading', { name: 'Event log', exact: true })).toBeVisible({
    timeout: 10000,
  })
  await expect(page.getByText(/experienced/i).first()).toBeVisible({ timeout: 15000 })
})
