import { loadCanvasImportCredentials } from '../../lib/canvas-import-credentials'
import { queueQuizGradeSyncToCanvas } from '../../lib/canvas-quiz-grade-sync-api'
import type { CourseCanvasLinkApi } from '../../lib/courses-api'

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