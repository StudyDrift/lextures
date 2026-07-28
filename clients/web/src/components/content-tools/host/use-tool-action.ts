import { useRef, useState } from 'react'
import {
  runContentToolAction,
  type ContentToolActionResponse,
  type ContentToolState,
} from '../../../lib/courses-api'

export type UseToolActionOptions = {
  courseCode: string
  instanceId: string
  onState?: (state: ContentToolState) => void
}

export type UseToolActionResult = {
  runAction: (
    name: string,
    input: Record<string, unknown>,
    opts?: { idempotencyKey?: string },
  ) => Promise<ContentToolActionResponse>
  busy: boolean
  error: string | null
}

function newIdempotencyKey(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID()
  }
  return `ct-action-${Date.now()}-${Math.random().toString(36).slice(2)}`
}

export function useToolAction(opts: UseToolActionOptions): UseToolActionResult {
  const { courseCode, instanceId, onState } = opts
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const onStateRef = useRef(onState)
  onStateRef.current = onState

  async function runAction(
    name: string,
    input: Record<string, unknown>,
    actionOpts?: { idempotencyKey?: string },
  ): Promise<ContentToolActionResponse> {
    // Read-only status probes should not flip busy — that re-renders the host and
    // used to cancel in-flight mount fetches when tools depended on runAction identity.
    const trackBusy = name !== 'status'
    if (trackBusy) setBusy(true)
    setError(null)
    try {
      const res = await runContentToolAction(courseCode, instanceId, name, {
        input,
        idempotencyKey: actionOpts?.idempotencyKey ?? newIdempotencyKey(),
      })
      onStateRef.current?.(res.state)
      return res
    } catch (e) {
      const message = e instanceof Error ? e.message : 'Action failed.'
      setError(message)
      throw e
    } finally {
      if (trackBusy) setBusy(false)
    }
  }

  return { runAction, busy, error }
}
