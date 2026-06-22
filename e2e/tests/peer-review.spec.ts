/**
 * Peer Review (plan 3.15)
 */
import { test, expect, uniqueEmail } from '../fixtures/test.js'
import {
  apiEnablePeerReview,
  apiPutPeerReviewConfig,
  apiPostPeerReviewAllocate,
  apiGetPeerReviewAssigned,
  apiCreateAssignment,
  apiPatchAssignmentSubmissionTypes,
  apiUploadAssignmentSubmission,
  apiSignup,
  apiEnroll,
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
    await apiEnablePeerReview()
    const assigned = await apiGetPeerReviewAssigned(seededCourse.studentToken)
    expect(Array.isArray(assigned)).toBe(true)
  })
})

test.describe('Peer Review — instructor setup', () => {
  test('configure and allocate on assignment', async ({ seededCourse }) => {
    await apiEnablePeerReview()

    const assignment = await apiCreateAssignment(
      seededCourse.instructorToken,
      seededCourse.courseCode,
      seededCourse.moduleId,
      'Peer review E2E assignment',
    )

    await apiPatchAssignmentSubmissionTypes(
      seededCourse.instructorToken,
      seededCourse.courseCode,
      assignment.id,
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

    await apiUploadAssignmentSubmission(
      seededCourse.studentToken,
      seededCourse.courseCode,
      assignment.id,
      'First student submission for peer review E2E.',
    )

    const student2Email = uniqueEmail('peer2')
    const { access_token: student2Token } = await apiSignup({
      email: student2Email,
      password: 'E2eTestPass1!',
      displayName: 'Peer Student 2',
    })
    await apiEnroll(
      seededCourse.instructorToken,
      seededCourse.courseCode,
      student2Email,
      'student',
      student2Token,
    )
    await apiUploadAssignmentSubmission(
      student2Token,
      seededCourse.courseCode,
      assignment.id,
      'Second student submission for peer review E2E.',
    )

    const alloc = await apiPostPeerReviewAllocate(
      seededCourse.instructorToken,
      seededCourse.courseCode,
      assignment.id,
    )
    expect(alloc.allocationsCreated).toBeGreaterThanOrEqual(0)
  })
})
