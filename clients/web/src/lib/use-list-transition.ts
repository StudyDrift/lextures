/**
 * AN.4 — React hook: keyed list transitions with exit hold + animation budget.
 *
 * Initial mount does not animate (coordinates with AN.3 load reveal). Subsequent
 * key mutations produce enter/exit/move phases while `enabled`.
 */

import { useEffect, useRef, useState } from 'react'
import {
  computeListTransitions,
  LIST_MOTION_MAX_CONCURRENT,
  type ListTransitionItem,
} from './list-motion'
import { durations, usePrefersReducedMotion } from './motion'

export type UseListTransitionOptions = {
  keys: readonly string[]
  enabled?: boolean
  mode?: 'mutate' | 'append'
  maxConcurrent?: number
  visibleKeys?: ReadonlySet<string>
  /** Live-region announcements for add/remove (a11y). */
  onAnnounce?: (message: { type: 'added' | 'removed'; keys: string[] }) => void
}

export type UseListTransitionResult = {
  items: ListTransitionItem[]
  reduceMotion: boolean
  enabled: boolean
  completeExit: (key: string) => void
  /** True after first snapshot; mutations animate only when true. */
  hasSettled: boolean
}

type Snapshot = {
  prevKeys: string[]
  nextKeys: string[]
  exitingKeys: string[]
}

function keysEqual(a: readonly string[], b: readonly string[]): boolean {
  if (a.length !== b.length) return false
  for (let i = 0; i < a.length; i++) {
    if (a[i] !== b[i]) return false
  }
  return true
}

export function useListTransition({
  keys,
  enabled = true,
  mode = 'mutate',
  maxConcurrent = LIST_MOTION_MAX_CONCURRENT,
  visibleKeys,
  onAnnounce,
}: UseListTransitionOptions): UseListTransitionResult {
  const reduceMotion = usePrefersReducedMotion()
  const [hasSettled, setHasSettled] = useState(false)
  const [snap, setSnap] = useState<Snapshot>(() => ({
    prevKeys: [...keys],
    nextKeys: [...keys],
    exitingKeys: [],
  }))
  const announceRef = useRef(onAnnounce)
  announceRef.current = onAnnounce
  const committedRef = useRef<string[]>([...keys])

  useEffect(() => {
    if (!hasSettled) {
      committedRef.current = [...keys]
      setSnap({
        prevKeys: [...keys],
        nextKeys: [...keys],
        exitingKeys: [],
      })
      setHasSettled(true)
    }
  }, [keys, hasSettled])

  useEffect(() => {
    if (!hasSettled) return
    if (keysEqual(keys, committedRef.current)) return

    const prev = committedRef.current
    const next = [...keys]
    const prevSet = new Set(prev)
    const nextSet = new Set(next)
    const entered = next.filter((k) => !prevSet.has(k))
    const exited = prev.filter((k) => !nextSet.has(k))

    if (entered.length) announceRef.current?.({ type: 'added', keys: entered })
    if (exited.length) announceRef.current?.({ type: 'removed', keys: exited })

    const holdExits = enabled && !reduceMotion ? exited : []
    setSnap({
      prevKeys: prev,
      nextKeys: next,
      exitingKeys: holdExits,
    })
    committedRef.current = next

    // After enter/move durations, advance prev so phases become steady.
    const settleMs = !enabled || reduceMotion ? 0 : durations.base
    const settleId = window.setTimeout(() => {
      setSnap((s) => ({
        prevKeys: s.nextKeys,
        nextKeys: s.nextKeys,
        exitingKeys: s.exitingKeys,
      }))
    }, settleMs)
    return () => window.clearTimeout(settleId)
  }, [keys, hasSettled, enabled, reduceMotion])

  useEffect(() => {
    if (!snap.exitingKeys.length) return
    if (!enabled || reduceMotion) {
      setSnap((s) => (s.exitingKeys.length ? { ...s, exitingKeys: [] } : s))
      return
    }
    const ms = durations.fast + 80
    const id = window.setTimeout(() => {
      setSnap((s) => (s.exitingKeys.length ? { ...s, exitingKeys: [] } : s))
    }, ms)
    return () => window.clearTimeout(id)
  }, [snap.exitingKeys, enabled, reduceMotion])

  const completeExit = (key: string) => {
    setSnap((s) => ({
      ...s,
      exitingKeys: s.exitingKeys.filter((k) => k !== key),
    }))
  }

  const animate = enabled && hasSettled
  const items: ListTransitionItem[] = computeListTransitions({
    prevKeys: animate ? snap.prevKeys : snap.nextKeys,
    nextKeys: snap.nextKeys,
    exitingKeys: animate ? snap.exitingKeys : [],
    reduceMotion,
    enabled: animate,
    maxConcurrent,
    visibleKeys,
    mode,
  })

  return {
    items,
    reduceMotion,
    enabled,
    completeExit,
    hasSettled,
  }
}
