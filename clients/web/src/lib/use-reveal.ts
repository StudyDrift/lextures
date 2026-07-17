/**
 * AN.3 — per-region reveal state for skeleton→content choreography.
 */

import { useEffect, useState } from 'react'
import { usePrefersReducedMotion } from './motion'

export type UseRevealOptions = {
  /** Data region is ready (loading finished, including error/empty). */
  ready: boolean
  /** Feature kill-switch; when false, behave as an instant swap. */
  enabled?: boolean
}

export type UseRevealResult = {
  /** Show skeleton / placeholder. */
  showSkeleton: boolean
  /** Show real content (stays true after first reveal even if ready flickers). */
  showContent: boolean
  /** Region has completed its first reveal; refresh must not re-animate. */
  hasRevealed: boolean
  /** True only for the first ready→content handoff (apply entrance class once). */
  playEntrance: boolean
  reduceMotion: boolean
  enabled: boolean
}

/**
 * Per-region reveal state. Call once at the data boundary (dashboard, list page, etc.).
 */
export function useReveal({ ready, enabled = true }: UseRevealOptions): UseRevealResult {
  const reduceMotion = usePrefersReducedMotion()
  const [hasRevealed, setHasRevealed] = useState(false)
  const [playEntrance, setPlayEntrance] = useState(false)

  useEffect(() => {
    if (!ready || hasRevealed) return
    setHasRevealed(true)
    if (enabled) setPlayEntrance(true)
  }, [ready, hasRevealed, enabled])

  const showContent = hasRevealed || ready
  const showSkeleton = !showContent

  return {
    showSkeleton,
    showContent,
    hasRevealed,
    playEntrance: enabled && playEntrance,
    reduceMotion,
    enabled,
  }
}
