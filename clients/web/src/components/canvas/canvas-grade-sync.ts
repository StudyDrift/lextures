import { loadCanvasImportCredentials } from '../../lib/canvas-import-credentials'
import {
  queueQuizGradeSyncToCanvas,
  queueSubmissionSyncToCanvas,
  type CourseCanvasLinkApi,
  type SubmissionGradeApi,
} from '../../lib/courses-api'

export type CanvasGradePushPayload = {
  pointsEarned?: number
  rubricScores?: Record<string, number>
  instructorComment?: string | null
}

type QueueCanvasGradeSyncOptions = {
  courseCode: string
  itemId: string
  submissionId: string
  canvasLink: CourseCanvasLinkApi
  gradePayload: CanvasGradePushPayload
  accessToken?: string
  onComplete?: (grade: SubmissionGradeApi) => void
  onError?: (message: string) => void
}

/** Returns null when sync is not applicable; otherwise an abort handle for in-flight syncs. */
export function queueCanvasGradeSync({
  courseCode,
  itemId,
  submissionId,
  canvasLink,
  gradePayload,
  accessToken,
  onComplete,
  onError,
}: QueueCanvasGradeSyncOptions): { abort: () => void } | null {
  if (!canvasLink.linked || !canvasLink.gradeSyncEnabled) return null
  const token = accessToken?.trim() || loadCanvasImportCredentials()?.accessToken?.trim()
  if (!token) {
    onError?.(
      'Grade saved. Add a Canvas access token with grade permissions in Import settings to sync back to Canvas.',
    )
    return null
  }
  return queueSubmissionSyncToCanvas(courseCode, itemId, submissionId, {
    accessToken: token,
    canvasBaseUrl: canvasLink.canvasBaseUrl,
    ...gradePayload,
  }, {
    onComplete,
    onError,
  })
}

type QueueCanvasQuizGradeSyncOptions = {
  courseCode: string
  itemId: string
  attemptId: string
  canvasLink: CourseCanvasLinkApi
  pointsEarned?: number
  accessToken?: string
  onComplete?: () => void
  onError?: (message: string) => void
}

/** Returns null when sync is not applicable; otherwise an abort handle for the in-flight sync. */
export function queueCanvasQuizGradeSync({
  courseCode,
  itemId,
  attemptId,
  canvasLink,
  pointsEarned,
  accessToken,
  onComplete,
  onError,
}: QueueCanvasQuizGradeSyncOptions): { abort: () => void } | null {
  if (!canvasLink.linked || !canvasLink.gradeSyncEnabled) return null
  const token = accessToken?.trim() || loadCanvasImportCredentials()?.accessToken?.trim()
  if (!token) {
    onError?.(
      'Scores saved. Add a Canvas access token with grade permissions in Import settings to sync back to Canvas.',
    )
    return null
  }
  return queueQuizGradeSyncToCanvas(
    courseCode,
    itemId,
    attemptId,
    { accessToken: token, canvasBaseUrl: canvasLink.canvasBaseUrl, pointsEarned },
    { onComplete, onError },
  )
}