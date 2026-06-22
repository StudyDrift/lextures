/**
 * Peer Review (plan 3.15)
 */
import { test, expect } from '../fixtures/test.js'
import {
  apiEnablePeerReview,
  apiPutPeerReviewConfig,
  apiPostPeerReviewAllocate,
  apiGetPeerReviewAssigned,
  apiCreateAssignment,
} from '../fixtures/api.js'

test.describe('Peer Review', () => {
  test('assigned endpoint returns 404 when feature is disabled', async ({ seededCourse }) => {
    const apiBase = process.env.E2E_API_URL ?? 'http://localhost:8080'
    const res = await fetch(`${apiBase}/api/v1/peer-review/assigned`, {
      headers: { Authorization: `Bearer ${seededCourse.studentToken}` },
    })
    expect(res.status).toBe(404)
  })

  test('API round-trip after enabling feature', async ({ seededCourse }) => {
    await apiEnablePeerReview(seededCourse.instructorToken)
    const assigned = await apiGetPeerReviewAssigned(seededCourse.studentToken)
    expect(Array.isArray(assigned)).toBe(true)
  })
})

test.describe('Peer Review — instructor setup', () => {
  test('configure and allocate on assignment', async ({ seededCourse }) => {
    await apiEnablePeerReview(seededCourse.instructorToken)

    const assignment = await apiCreateAssignment(
      seededCourse.instructorToken,
      seededCourse.courseCode,
      seededCourse.moduleId,
      'Peer review E2E assignment',
    )

    await apiPutPeerReviewConfig(
      seededCourse.instructorToken,
      seededCourse.courseCode,
      assignment.id,
      {
        reviewsPerReviewer: 1,
        anonymity: 'double_blind',
        gradeMode: 'none',
        aggregation: 'median',
      },
    )

    const alloc = await apiPostPeerReviewAllocate(
      seededCourse.instructorToken,
      seededCourse.courseCode,
      assignment.id,
    )
    expect(typeof alloc.allocationsCreated).toBe('number')
  })
})
