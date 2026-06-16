import { useEffect, useRef } from 'react'
import { postSeatTimeHeartbeat } from '../lib/seat-time-api'

const HEARTBEAT_INTERVAL_MS = 60_000

function newSessionToken(): string {
  if (typeof crypto !== 'undefined' && 'randomUUID' in crypto) {
    return crypto.randomUUID()
  }
  return `seat-${Date.now()}-${Math.random().toString(36).slice(2)}`
}

/**
 * Sends seat-time heartbeats every 60s while the content page is visible (plan 14.17).
 */
export function useSeatTimeHeartbeat(contentItemId: string | undefined, enabled: boolean): void {
  const sessionTokenRef = useRef<string>(newSessionToken())

  useEffect(() => {
    if (!enabled || !contentItemId) return

    const token = sessionTokenRef.current

    function tick(): void {
      if (!contentItemId || document.visibilityState !== 'visible') return
      void postSeatTimeHeartbeat(contentItemId, token).catch(() => {
        // Heartbeat loss is acceptable; server under-counts conservatively.
      })
    }

    tick()
    const timer = setInterval(tick, HEARTBEAT_INTERVAL_MS)
    const onVisibility = (): void => {
      if (document.visibilityState === 'visible') tick()
    }
    document.addEventListener('visibilitychange', onVisibility)

    return () => {
      clearInterval(timer)
      document.removeEventListener('visibilitychange', onVisibility)
    }
  }, [contentItemId, enabled])
}
