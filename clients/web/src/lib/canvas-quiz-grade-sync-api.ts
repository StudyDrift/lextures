import { authorizedFetch } from './api'
import { getAccessToken } from './auth'
import {
  awaitCanvasSyncJob,
  type CanvasSubmissionSyncQueuedResponse,
} from './canvas-sync-job-wait'
import { readApiErrorMessage } from './errors'

async function parseJson(res: Response): Promise<unknown> {
  return res.json().catch(() => ({}))
}

/** Queues a Canvas grade push for one quiz attempt and waits for completion over a job WebSocket. */
export async function syncQuizAttemptToCanvas(
  courseCode: string,
  itemId: string,
  attemptId: string,
  body: {
    canvasBaseUrl?: string
    accessToken: string
    pointsEarned?: number
  },
  options?: { signal?: AbortSignal },
): Promise<void> {
  const authToken = getAccessToken()
  if (!authToken) {
    throw new Error('Sign in to sync grades to Canvas.')
  }

  const signal = options?.signal
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/quizzes/${encodeURIComponent(itemId)}/attempts/${encodeURIComponent(attemptId)}/sync-canvas`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
      signal,
    },
  )
  const raw = await parseJson(res)
  if (!res.ok) throw new Error(readApiErrorMessage(raw))

  await awaitCanvasSyncJob(raw as CanvasSubmissionSyncQueuedResponse, authToken, signal)
}

/** Queues a quiz attempt Canvas grade push and invokes callbacks when the background job finishes. */
export function queueQuizGradeSyncToCanvas(
  courseCode: string,
  itemId: string,
  attemptId: string,
  body: { canvasBaseUrl?: string; accessToken: string; pointsEarned?: number },
  handlers: { onComplete?: () => void; onError?: (message: string) => void },
): { abort: () => void } {
  const controller = new AbortController()
  void syncQuizAttemptToCanvas(courseCode, itemId, attemptId, body, { signal: controller.signal })
    .then(() => handlers.onComplete?.())
    .catch((e) => {
      if (controller.signal.aborted) return
      handlers.onError?.(e instanceof Error ? e.message : 'Could not sync to Canvas.')
    })
  return { abort: () => controller.abort() }
}