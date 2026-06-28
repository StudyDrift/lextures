import { test, expect } from '../fixtures/test'

const API_BASE = process.env.API_BASE ?? 'http://localhost:8080'

// Plan 17.4 NFR security: the scheduler exposes no unauthenticated HTTP trigger;
// every admin scheduler endpoint must reject anonymous callers before doing work.
function expectAuthRejected(status: number) {
  expect([401, 403, 501]).toContain(status)
}

test('Scheduler: list endpoint unauthenticated is rejected', async () => {
  const res = await fetch(`${API_BASE}/api/v1/admin/scheduler`)
  expectAuthRejected(res.status)
})

test('Scheduler: history endpoint unauthenticated is rejected', async () => {
  const res = await fetch(
    `${API_BASE}/api/v1/admin/scheduler/late_submission_sweep/history`,
  )
  expectAuthRejected(res.status)
})

test('Scheduler: enable endpoint unauthenticated is rejected', async () => {
  const res = await fetch(
    `${API_BASE}/api/v1/admin/scheduler/late_submission_sweep/enable`,
    { method: 'POST' },
  )
  expectAuthRejected(res.status)
})

test('Scheduler: disable endpoint unauthenticated is rejected', async () => {
  const res = await fetch(
    `${API_BASE}/api/v1/admin/scheduler/late_submission_sweep/disable`,
    { method: 'POST' },
  )
  expectAuthRejected(res.status)
})

test('Scheduler: manual trigger unauthenticated is rejected', async () => {
  const res = await fetch(
    `${API_BASE}/api/v1/admin/scheduler/late_submission_sweep/trigger`,
    { method: 'POST' },
  )
  expectAuthRejected(res.status)
})
