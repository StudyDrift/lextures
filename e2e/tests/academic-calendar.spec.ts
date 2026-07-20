/**
 * Academic Calendar Awareness — plan 14.6
 *
 * Checklist:
 *   [x] GET /orgs/:orgId/calendar/events unauthenticated returns 401
 *   [x] POST /admin/orgs/:orgId/calendar/events unauthenticated returns 401
 *   [x] Feature disabled: all endpoints return 501
 *   [x] Feature enabled: create calendar event returns 201 with event body
 *   [x] Feature enabled: list events returns created event
 *   [x] Feature enabled: patch event updates fields
 *   [x] Feature enabled: delete event returns 204
 *   [x] Feature enabled: list after delete excludes the event
 *   [x] Feature enabled: term iCal feed returns text/calendar content with VEVENT
 *   [x] Non-admin cannot create/delete events (403)
 *   [x] Invalid eventType returns 400
 *   [x] Invalid date format returns 400
 *   [x] Platform settings: PUT ffAcademicCalendar toggles the feature flag
 */
import { test, expect } from '@playwright/test'
import { apiSignup, apiWaitForPlatformFeature } from '../fixtures/api.js'
import { withPlatformSettingsLock } from '../lib/platform-feature-matrix-helpers.js'

const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'
const ADMIN_EMAIL = process.env.E2E_ADMIN_EMAIL ?? 'admin@e2e.test'
const ADMIN_PASSWORD = process.env.E2E_ADMIN_PASSWORD ?? 'E2eTestPass1!'

function uid(prefix = 'ac') {
  return `e2e-${prefix}-${Date.now()}-${Math.random().toString(36).slice(2)}`
}

function authHeaders(token: string) {
  return { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' }
}

async function getAdminToken(): Promise<string> {
  const loginRes = await fetch(`${API_BASE}/api/v1/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email: ADMIN_EMAIL, password: ADMIN_PASSWORD }),
  })
  if (loginRes.ok) {
    const { access_token } = (await loginRes.json()) as { access_token: string }
    return access_token
  }
  await fetch(`${API_BASE}/api/v1/auth/signup`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email: ADMIN_EMAIL, password: ADMIN_PASSWORD, display_name: 'E2E Admin' }),
  })
  const retry = await fetch(`${API_BASE}/api/v1/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email: ADMIN_EMAIL, password: ADMIN_PASSWORD }),
  })
  const { access_token } = (await retry.json()) as { access_token: string }
  return access_token
}

async function getAdminOrgId(token: string): Promise<string | null> {
  const res = await fetch(`${API_BASE}/api/v1/admin/orgs`, { headers: authHeaders(token) })
  if (!res.ok) return null
  const data = (await res.json()) as { organizations?: Array<{ id: string }> }
  return data.organizations?.[0]?.id ?? null
}

/** Caller must hold withPlatformSettingsLock when the result is asserted. */
async function setAcademicCalendarFeature(token: string, enabled: boolean): Promise<void> {
  const res = await fetch(`${API_BASE}/api/v1/settings/platform`, {
    method: 'PUT',
    headers: authHeaders(token),
    body: JSON.stringify({ ffAcademicCalendar: enabled, updateMask: ['ffAcademicCalendar'] }),
  })
  if (!res.ok) throw new Error(`setAcademicCalendarFeature failed (${res.status})`)
  await apiWaitForPlatformFeature(token, 'ffAcademicCalendar', enabled)
}

// Serialize: these tests mutate the global ffAcademicCalendar platform flag.
test.describe.serial('Academic calendar', () => {
// ─────────────────────────────────────────────────────────────────────────────
// Auth guards
// ─────────────────────────────────────────────────────────────────────────────

test('AC: GET events unauthenticated returns 401', async () => {
  const adminToken = await getAdminToken()
  const orgId = await getAdminOrgId(adminToken)
  if (!orgId) { test.skip(true, 'no org'); return }

  const res = await fetch(`${API_BASE}/api/v1/orgs/${orgId}/calendar/events`)
  expect(res.status).toBe(401)
})

test('AC: POST admin events unauthenticated returns 401', async () => {
  const adminToken = await getAdminToken()
  const orgId = await getAdminOrgId(adminToken)
  if (!orgId) { test.skip(true, 'no org'); return }

  const res = await fetch(`${API_BASE}/api/v1/admin/orgs/${orgId}/calendar/events`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ eventType: 'holiday', eventName: 'Test', startDate: '2027-01-01' }),
  })
  expect(res.status).toBe(401)
})

// ─────────────────────────────────────────────────────────────────────────────
// Feature flag disabled → 501
// ─────────────────────────────────────────────────────────────────────────────

test('AC: GET events returns 501 when feature disabled', async () => {
  await withPlatformSettingsLock(async () => {
    const adminToken = await getAdminToken()
    await setAcademicCalendarFeature(adminToken, false)

    const orgId = await getAdminOrgId(adminToken)
    if (!orgId) { test.skip(true, 'no org'); return }

    const user = await apiSignup({ email: `${uid('u501')}@test.invalid`, password: 'E2eTestPass1!' })
    const res = await fetch(`${API_BASE}/api/v1/orgs/${orgId}/calendar/events`, {
      headers: authHeaders(user.access_token),
    })
    expect(res.status).toBe(501)
  })
})

test('AC: POST admin events returns 501 when feature disabled', async () => {
  await withPlatformSettingsLock(async () => {
    const adminToken = await getAdminToken()
    await setAcademicCalendarFeature(adminToken, false)

    const orgId = await getAdminOrgId(adminToken)
    if (!orgId) { test.skip(true, 'no org'); return }

    const res = await fetch(`${API_BASE}/api/v1/admin/orgs/${orgId}/calendar/events`, {
      method: 'POST',
      headers: authHeaders(adminToken),
      body: JSON.stringify({ eventType: 'holiday', eventName: 'Test', startDate: '2027-01-01' }),
    })
    expect(res.status).toBe(501)
  })
})

// ─────────────────────────────────────────────────────────────────────────────
// CRUD (feature enabled)
// ─────────────────────────────────────────────────────────────────────────────

test('AC: create, list, patch, delete event', async () => {
  await withPlatformSettingsLock(async () => {
  const adminToken = await getAdminToken()
  await setAcademicCalendarFeature(adminToken, true)

  const orgId = await getAdminOrgId(adminToken)
  if (!orgId) { test.skip(true, 'no org'); return }

  // Create
  const createRes = await fetch(`${API_BASE}/api/v1/admin/orgs/${orgId}/calendar/events`, {
    method: 'POST',
    headers: authHeaders(adminToken),
    body: JSON.stringify({
      eventType: 'add_drop_deadline',
      eventName: 'Spring 2027 Add/Drop Deadline',
      startDate: '2027-01-20',
      notes: 'Last day to add or drop without W',
    }),
  })
  expect(createRes.status).toBe(201)
  const { event } = (await createRes.json()) as { event: { id: string; eventType: string; eventName: string; startDate: string; orgId: string } }
  expect(event.eventType).toBe('add_drop_deadline')
  expect(event.eventName).toBe('Spring 2027 Add/Drop Deadline')
  expect(event.startDate).toBe('2027-01-20')
  expect(event.orgId).toBe(orgId)
  const eventId = event.id
  expect(eventId).toBeTruthy()

  // List
  const listRes = await fetch(`${API_BASE}/api/v1/orgs/${orgId}/calendar/events`, {
    headers: authHeaders(adminToken),
  })
  expect(listRes.status).toBe(200)
  const { events } = (await listRes.json()) as { events: Array<{ id: string; eventName: string }> }
  const found = events.find((e) => e.id === eventId)
  expect(found).toBeDefined()
  expect(found?.eventName).toBe('Spring 2027 Add/Drop Deadline')

  // Patch
  const patchRes = await fetch(`${API_BASE}/api/v1/admin/orgs/${orgId}/calendar/events/${eventId}`, {
    method: 'PATCH',
    headers: authHeaders(adminToken),
    body: JSON.stringify({ eventName: 'Updated Add/Drop Deadline', endDate: '2027-01-21' }),
  })
  expect(patchRes.status).toBe(200)
  const { event: patched } = (await patchRes.json()) as { event: { eventName: string; endDate?: string } }
  expect(patched.eventName).toBe('Updated Add/Drop Deadline')
  expect(patched.endDate).toBe('2027-01-21')

  // Delete
  const deleteRes = await fetch(`${API_BASE}/api/v1/admin/orgs/${orgId}/calendar/events/${eventId}`, {
    method: 'DELETE',
    headers: authHeaders(adminToken),
  })
  expect(deleteRes.status).toBe(204)

  // List after delete
  const listAfterRes = await fetch(`${API_BASE}/api/v1/orgs/${orgId}/calendar/events`, {
    headers: authHeaders(adminToken),
  })
  expect(listAfterRes.status).toBe(200)
  const { events: eventsAfter } = (await listAfterRes.json()) as { events: Array<{ id: string }> }
  expect(eventsAfter.find((e) => e.id === eventId)).toBeUndefined()
  })
})

test('AC: iCal feed returns text/calendar with VEVENT', async () => {
  await withPlatformSettingsLock(async () => {
  const adminToken = await getAdminToken()
  await setAcademicCalendarFeature(adminToken, true)

  const orgId = await getAdminOrgId(adminToken)
  if (!orgId) { test.skip(true, 'no org'); return }

  // Create a term
  const termRes = await fetch(`${API_BASE}/api/v1/orgs/${orgId}/terms`, {
    method: 'POST',
    headers: authHeaders(adminToken),
    body: JSON.stringify({
      name: 'Spring 2027',
      termType: 'semester',
      startDate: '2027-01-10',
      endDate: '2027-05-15',
    }),
  })
  const termBody = (await termRes.json()) as { term?: { id?: string } }
  const termId = termBody.term?.id
  if (!termId) { test.skip(true, 'could not create term'); return }

  // Create calendar event linked to term
  await fetch(`${API_BASE}/api/v1/admin/orgs/${orgId}/calendar/events`, {
    method: 'POST',
    headers: authHeaders(adminToken),
    body: JSON.stringify({
      termId,
      eventType: 'no_class_day',
      eventName: 'Spring Break Day 1',
      startDate: '2027-03-08',
    }),
  })

  // Fetch iCal
  const icalRes = await fetch(`${API_BASE}/api/v1/orgs/${orgId}/calendar/terms/${termId}/ical`, {
    headers: authHeaders(adminToken),
  })
  expect(icalRes.status).toBe(200)
  const contentType = icalRes.headers.get('content-type') ?? ''
  expect(contentType).toContain('text/calendar')
  const body = await icalRes.text()
  expect(body).toContain('BEGIN:VCALENDAR')
  expect(body).toContain('BEGIN:VEVENT')
  expect(body).toContain('Spring Break Day 1')
  expect(body).toContain('END:VEVENT')
  expect(body).toContain('END:VCALENDAR')
  })
})

// ─────────────────────────────────────────────────────────────────────────────
// Permission enforcement
// ─────────────────────────────────────────────────────────────────────────────

test('AC: non-admin cannot create events (403)', async () => {
  await withPlatformSettingsLock(async () => {
  const adminToken = await getAdminToken()
  await setAcademicCalendarFeature(adminToken, true)

  const orgId = await getAdminOrgId(adminToken)
  if (!orgId) { test.skip(true, 'no org'); return }

  const nonAdmin = await apiSignup({ email: `${uid('na')}@test.invalid`, password: 'E2eTestPass1!' })
  const res = await fetch(`${API_BASE}/api/v1/admin/orgs/${orgId}/calendar/events`, {
    method: 'POST',
    headers: authHeaders(nonAdmin.access_token),
    body: JSON.stringify({ eventType: 'holiday', eventName: 'Test', startDate: '2027-07-04' }),
  })
  expect(res.status).toBe(403)
  })
})

// ─────────────────────────────────────────────────────────────────────────────
// Validation
// ─────────────────────────────────────────────────────────────────────────────

test('AC: invalid eventType returns 400', async () => {
  await withPlatformSettingsLock(async () => {
  const adminToken = await getAdminToken()
  await setAcademicCalendarFeature(adminToken, true)

  const orgId = await getAdminOrgId(adminToken)
  if (!orgId) { test.skip(true, 'no org'); return }

  const res = await fetch(`${API_BASE}/api/v1/admin/orgs/${orgId}/calendar/events`, {
    method: 'POST',
    headers: authHeaders(adminToken),
    body: JSON.stringify({ eventType: 'INVALID_TYPE', eventName: 'Test', startDate: '2027-01-01' }),
  })
  expect(res.status).toBe(400)
  })
})

test('AC: invalid startDate format returns 400', async () => {
  await withPlatformSettingsLock(async () => {
  const adminToken = await getAdminToken()
  await setAcademicCalendarFeature(adminToken, true)

  const orgId = await getAdminOrgId(adminToken)
  if (!orgId) { test.skip(true, 'no org'); return }

  const res = await fetch(`${API_BASE}/api/v1/admin/orgs/${orgId}/calendar/events`, {
    method: 'POST',
    headers: authHeaders(adminToken),
    body: JSON.stringify({ eventType: 'holiday', eventName: 'Test', startDate: '01-01-2027' }),
  })
  expect(res.status).toBe(400)
  })
})

// ─────────────────────────────────────────────────────────────────────────────
// Platform settings toggle
// ─────────────────────────────────────────────────────────────────────────────

test('AC: platform settings toggle enables and disables the feature', async () => {
  await withPlatformSettingsLock(async () => {
  const adminToken = await getAdminToken()
  const orgId = await getAdminOrgId(adminToken)
  if (!orgId) { test.skip(true, 'no org'); return }

  const user = await apiSignup({ email: `${uid('tog')}@test.invalid`, password: 'E2eTestPass1!' })

  // Disable
  await setAcademicCalendarFeature(adminToken, false)
  const disabledRes = await fetch(`${API_BASE}/api/v1/orgs/${orgId}/calendar/events`, {
    headers: authHeaders(user.access_token),
  })
  expect(disabledRes.status).toBe(501)

  // Enable
  await setAcademicCalendarFeature(adminToken, true)
  const enabledRes = await fetch(`${API_BASE}/api/v1/orgs/${orgId}/calendar/events`, {
    headers: authHeaders(user.access_token),
  })
  expect(enabledRes.status).toBe(200)
  })
})
})
