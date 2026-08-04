/**
 * Course Checklist state API (CC.2) + foundation rule pack (CC.3).
 *
 * Checklist coverage:
 *   [x] Unauthenticated GET /checklist returns 401
 *   [x] Student GET /checklist returns 403
 *   [x] Teacher GET /checklist returns 200 with summary
 *   [x] Teacher dismisses/restores item (CC.3: all rules recommended — badge may stay 0)
 *   [x] Bare course shows orientation.welcome-message + people.students-enrolled as todo
 *   [x] Post welcome announcement → orientation.welcome-message becomes done
 *   [x] Enroll a student → people.students-enrolled becomes done
 */

import { test, expect } from '@playwright/test'
import {
  apiSignup,
  apiCreateCourse,
  apiEnroll,
  apiGetFeedChannels,
  apiPostFeedMessage,
} from '../fixtures/api.js'

const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'
const PASSWORD = 'E2eTestPass1!'

let _seq = 0
function uniqueEmail(prefix = 'cc3') {
  return `e2e-${prefix}-${Date.now()}-${++_seq}@test.invalid`
}

type ChecklistItem = {
  id: string
  status: string
  finding?: { status?: string }
}

type ChecklistResponse = {
  engineVersion: number
  catalogVersion: string
  summary: {
    outstandingEssential: number
    dismissed: number
    todo?: number
  }
  categories?: Array<{ items?: ChecklistItem[] }>
  items?: ChecklistItem[]
}

function findItem(list: ChecklistResponse, id: string): ChecklistItem | undefined {
  for (const cat of list.categories ?? []) {
    for (const it of cat.items ?? []) {
      if (it.id === id) return it
    }
  }
  for (const it of list.items ?? []) {
    if (it.id === id) return it
  }
  return undefined
}

function itemStatus(it: ChecklistItem | undefined): string {
  return it?.status ?? it?.finding?.status ?? ''
}

async function fetchChecklist(token: string, courseCode: string): Promise<ChecklistResponse> {
  const res = await fetch(`${API_BASE}/api/v1/courses/${courseCode}/checklist`, {
    headers: { Authorization: `Bearer ${token}` },
  })
  expect(res.status).toBe(200)
  return res.json() as Promise<ChecklistResponse>
}

test('Course checklist: unauthenticated GET returns 401', async () => {
  const res = await fetch(`${API_BASE}/api/v1/courses/C-FAKE/checklist`)
  expect(res.status).toBe(401)
})

test('Course checklist: student receives 403; teacher dismiss/restore updates dismissed count', async () => {
  const teacherEmail = uniqueEmail('cc3-teacher')
  const studentEmail = uniqueEmail('cc3-student')
  const { access_token: teacherTok } = await apiSignup({ email: teacherEmail, password: PASSWORD })
  const { access_token: studentTok } = await apiSignup({ email: studentEmail, password: PASSWORD })
  const course = await apiCreateCourse(teacherTok, { title: 'CC3 Checklist Course' })
  await apiEnroll(teacherTok, course.courseCode, teacherEmail, 'teacher')
  await apiEnroll(teacherTok, course.courseCode, studentEmail, 'student', studentTok)

  const studentRes = await fetch(`${API_BASE}/api/v1/courses/${course.courseCode}/checklist`, {
    headers: { Authorization: `Bearer ${studentTok}` },
  })
  expect(studentRes.status).toBe(403)
  const studentBody = await studentRes.text()
  expect(studentBody).not.toContain('Set course start')

  const list = await fetchChecklist(teacherTok, course.courseCode)
  expect(list.engineVersion).toBeGreaterThanOrEqual(1)
  expect(typeof list.catalogVersion).toBe('string')
  // CC.3 FR-37: all foundation rules ship as recommended — essentials may be 0.
  const beforeEssential = list.summary.outstandingEssential
  expect(beforeEssential).toBeGreaterThanOrEqual(0)

  const dismissRes = await fetch(
    `${API_BASE}/api/v1/courses/${course.courseCode}/checklist/items/course.dates/dismiss`,
    {
      method: 'POST',
      headers: {
        Authorization: `Bearer ${teacherTok}`,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ reason: 'not_applicable', note: 'e2e dismiss' }),
    },
  )
  expect(dismissRes.status).toBe(200)

  const summaryAfterDismiss = await fetch(
    `${API_BASE}/api/v1/courses/${course.courseCode}/checklist/summary`,
    { headers: { Authorization: `Bearer ${teacherTok}` } },
  )
  expect(summaryAfterDismiss.status).toBe(200)
  const summary1 = await summaryAfterDismiss.json()
  expect(summary1.dismissed).toBe(1)
  expect(summary1.outstandingEssential).toBe(beforeEssential)

  const restoreRes = await fetch(
    `${API_BASE}/api/v1/courses/${course.courseCode}/checklist/items/course.dates/restore`,
    {
      method: 'POST',
      headers: { Authorization: `Bearer ${teacherTok}` },
    },
  )
  expect(restoreRes.status).toBe(200)

  const summaryAfterRestore = await fetch(
    `${API_BASE}/api/v1/courses/${course.courseCode}/checklist/summary`,
    { headers: { Authorization: `Bearer ${teacherTok}` } },
  )
  const summary2 = await summaryAfterRestore.json()
  expect(summary2.dismissed).toBe(0)
  expect(summary2.outstandingEssential).toBe(beforeEssential)
})

test('Course checklist CC.3: welcome message and enrollment flip to done', async () => {
  const teacherEmail = uniqueEmail('cc3-rules-teacher')
  const studentEmail = uniqueEmail('cc3-rules-student')
  const { access_token: teacherTok } = await apiSignup({ email: teacherEmail, password: PASSWORD })
  const { access_token: studentTok } = await apiSignup({ email: studentEmail, password: PASSWORD })
  const course = await apiCreateCourse(teacherTok, { title: 'CC3 Rules Course' })
  await apiEnroll(teacherTok, course.courseCode, teacherEmail, 'teacher')

  async function refreshChecklist(token: string, courseCode: string): Promise<ChecklistResponse> {
    const res = await fetch(`${API_BASE}/api/v1/courses/${courseCode}/checklist/refresh`, {
      method: 'POST',
      headers: { Authorization: `Bearer ${token}` },
    })
    expect(res.status).toBe(200)
    return res.json() as Promise<ChecklistResponse>
  }

  let list = await refreshChecklist(teacherTok, course.courseCode)
  expect(itemStatus(findItem(list, 'orientation.welcome-message'))).toBe('todo')
  expect(itemStatus(findItem(list, 'people.students-enrolled'))).toBe('todo')

  // apiGetFeedChannels historically typed as an array; the API returns { channels }.
  const channelsRaw = (await apiGetFeedChannels(teacherTok, course.courseCode)) as
    | Array<{ id: string; name: string }>
    | { channels?: Array<{ id: string; name: string }> }
  const channels = Array.isArray(channelsRaw) ? channelsRaw : (channelsRaw.channels ?? [])
  const announcements = channels.find((c) => c.name.toLowerCase() === 'announcements')
  expect(announcements, 'announcements channel should exist').toBeTruthy()
  const welcomeBody = 'Welcome to the course! '.repeat(20) // ≥ 200 chars
  await apiPostFeedMessage(teacherTok, course.courseCode, announcements!.id, welcomeBody)

  list = await refreshChecklist(teacherTok, course.courseCode)
  expect(itemStatus(findItem(list, 'orientation.welcome-message'))).toBe('done')

  await apiEnroll(teacherTok, course.courseCode, studentEmail, 'student', studentTok)
  list = await refreshChecklist(teacherTok, course.courseCode)
  expect(itemStatus(findItem(list, 'people.students-enrolled'))).toBe('done')
})
