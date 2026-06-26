import { apiBaseUrl } from './api'
import { getAccessToken } from './auth'

export type CanvasSubmissionSyncQueuedResponse = {
  jobId: string
  message: string
}

function canvasSubmissionSyncWebSocketUrl(jobId: string): string | null {
  if (!getAccessToken()) return null
  const base = apiBaseUrl()
  const u = new URL(base)
  u.protocol = u.protocol === 'https:' ? 'wss:' : 'ws:'
  return `${u.origin}/api/v1/ws/canvas-submission-sync/${encodeURIComponent(jobId)}`
}

/** Waits for a queued Canvas sync job to finish over its WebSocket. Resolves with the synced grade
 * (when the server returns one) or undefined on a plain completion. */
export function awaitCanvasSyncJob(
  queued: CanvasSubmissionSyncQueuedResponse,
  authToken: string,
  signal?: AbortSignal,
): Promise<Record<string, unknown> | undefined> {
  if (!queued.jobId?.trim()) {
    return Promise.reject(new Error('Server did not return a sync job id.'))
  }
  const url = canvasSubmissionSyncWebSocketUrl(queued.jobId)
  if (!url) {
    return Promise.reject(new Error('Sign in to sync grades to Canvas.'))
  }

  return new Promise<Record<string, unknown> | undefined>((resolve, reject) => {
    const ws = new WebSocket(url)
    let settled = false
    let aborted = false

    const fail = (msg: string) => {
      if (settled) return
      settled = true
      reject(new Error(msg))
    }

    const onAbort = () => {
      aborted = true
      if (settled) return
      settled = true
      ws.close()
      reject(new Error('Canvas sync cancelled.'))
    }

    signal?.addEventListener('abort', onAbort, { once: true })

    ws.onopen = () => {
      if (signal?.aborted) {
        onAbort()
        return
      }
      ws.send(JSON.stringify({ authToken }))
    }

    ws.onmessage = (ev) => {
      let rawMsg: unknown
      try {
        rawMsg = JSON.parse(String(ev.data))
      } catch {
        fail('Unexpected message from server.')
        ws.close()
        return
      }
      const o = rawMsg as { type?: string; message?: string; grade?: Record<string, unknown> }
      if (o.type === 'complete') {
        if (!settled) {
          settled = true
          ws.close()
          resolve(o.grade && typeof o.grade === 'object' ? o.grade : undefined)
        }
        return
      }
      if (o.type === 'error') {
        fail(typeof o.message === 'string' ? o.message : 'Could not sync to Canvas.')
        ws.close()
      }
    }

    ws.onerror = () => {
      fail('Connection error during Canvas sync.')
    }

    ws.onclose = () => {
      if (!settled && !aborted) {
        fail('Connection closed before Canvas sync finished.')
      }
    }
  })
}