/**
 * AN.4 — List / collection motion: keyed diffs, concurrent-animation caps,
 * drag lift tokens, and reduced-motion contracts.
 *
 * Pure helpers (unit-tested). React wrappers live in `components/ui/animated-list.tsx`.
 */

import type { CSSProperties } from 'react'
import { CSS, type Transform } from '@dnd-kit/utilities'
import type { DropAnimation } from '@dnd-kit/core'
import { defaultDropAnimation } from '@dnd-kit/core'
import { distances, durations, easings, prefersReducedMotion } from './motion'

/** Max simultaneous enter/exit/move animations (FR-9 / scalability). */
export const LIST_MOTION_MAX_CONCURRENT = 12

/** Slight lift on drag grab (FR-4). */
export const LIST_DRAG_LIFT_SCALE = 1.03

export type ListPhase = 'enter' | 'exit' | 'move' | 'steady'

export type ListDiff = {
  entered: string[]
  exited: string[]
  /** Present in both; index changed. */
  moved: string[]
  /** Present in both; same index. */
  steady: string[]
}

export type ListTransitionItem = {
  key: string
  phase: ListPhase
  index: number
  /** False when kill-switch, reduced-motion (for slide), capped, or off-screen. */
  animate: boolean
}

/**
 * Keyed set-diff between previous and next identity lists (FR-7).
 */
export function diffListKeys(prevKeys: readonly string[], nextKeys: readonly string[]): ListDiff {
  const prevSet = new Set(prevKeys)
  const nextSet = new Set(nextKeys)
  const entered: string[] = []
  const exited: string[] = []
  const moved: string[] = []
  const steady: string[] = []

  for (const key of nextKeys) {
    if (!prevSet.has(key)) {
      entered.push(key)
      continue
    }
    const prevIdx = prevKeys.indexOf(key)
    const nextIdx = nextKeys.indexOf(key)
    if (prevIdx !== nextIdx) moved.push(key)
    else steady.push(key)
  }
  for (const key of prevKeys) {
    if (!nextSet.has(key)) exited.push(key)
  }

  return { entered, exited, moved, steady }
}

export type ComputeListTransitionsOptions = {
  prevKeys: readonly string[]
  nextKeys: readonly string[]
  /** Keys still rendering while exit animation plays (may overlap nextKeys). */
  exitingKeys?: readonly string[]
  reduceMotion?: boolean
  enabled?: boolean
  maxConcurrent?: number
  /**
   * When provided, only these keys may animate (virtualization / viewport) — FR-9.
   * Keys outside the set still transition to their final state without motion.
   */
  visibleKeys?: ReadonlySet<string>
  /**
   * `append` — only newly appended keys at the end animate (fade); no move of existing.
   * `mutate` — full enter/exit/move (default).
   */
  mode?: 'mutate' | 'append'
}

/**
 * Build per-item phases for a keyed list update.
 * Caps concurrent animations; overflow applies without motion (FR-9).
 * Under reduced motion, phases remain but `animate` is false for slide/scale (FR-8).
 */
export function computeListTransitions(opts: ComputeListTransitionsOptions): ListTransitionItem[] {
  const {
    prevKeys,
    nextKeys,
    exitingKeys = [],
    reduceMotion = false,
    enabled = true,
    maxConcurrent = LIST_MOTION_MAX_CONCURRENT,
    visibleKeys,
    mode = 'mutate',
  } = opts

  const diff = diffListKeys(prevKeys, nextKeys)
  const renderKeys: string[] = []
  const seen = new Set<string>()
  for (const key of nextKeys) {
    renderKeys.push(key)
    seen.add(key)
  }
  for (const key of exitingKeys) {
    if (!seen.has(key)) {
      renderKeys.push(key)
      seen.add(key)
    }
  }

  const phaseByKey = new Map<string, ListPhase>()
  for (const key of diff.steady) phaseByKey.set(key, 'steady')
  for (const key of diff.moved) phaseByKey.set(key, mode === 'append' ? 'steady' : 'move')
  for (const key of diff.entered) phaseByKey.set(key, 'enter')
  for (const key of exitingKeys.length ? exitingKeys : diff.exited) {
    if (!nextKeys.includes(key)) phaseByKey.set(key, 'exit')
  }
  for (const key of diff.exited) {
    if (!phaseByKey.has(key)) phaseByKey.set(key, 'exit')
  }

  // Append mode: only brand-new trailing keys enter; ignore moves of existing.
  if (mode === 'append') {
    const prevSet = new Set(prevKeys)
    for (const key of nextKeys) {
      if (!prevSet.has(key)) phaseByKey.set(key, 'enter')
      else phaseByKey.set(key, 'steady')
    }
  }

  let budget = enabled && !reduceMotion ? maxConcurrent : 0
  // Under reduced motion we still allow opacity-only (animate=true for enter/exit fade).
  if (enabled && reduceMotion) {
    budget = maxConcurrent
  }

  return renderKeys.map((key, index) => {
    const phase = phaseByKey.get(key) ?? 'steady'
    const inViewport = !visibleKeys || visibleKeys.has(key)
    let animate = false
    if (enabled && inViewport && phase !== 'steady') {
      if (reduceMotion) {
        // Opacity-only: enter/exit may fade; move is instant (FR-8).
        animate = phase === 'enter' || phase === 'exit'
      } else if (budget > 0) {
        animate = true
        budget -= 1
      }
    }
    return { key, phase, index, animate }
  })
}

/** CSS class for an item phase (declarative). */
export function listPhaseClassName(
  phase: ListPhase,
  animate: boolean,
  reduceMotion: boolean,
): string {
  if (!animate) {
    if (phase === 'exit') return 'lx-list-item-exit-instant'
    return 'lx-list-item'
  }
  if (reduceMotion) {
    if (phase === 'enter') return 'lx-list-item lx-list-item-enter-fade'
    if (phase === 'exit') return 'lx-list-item lx-list-item-exit-fade'
    return 'lx-list-item'
  }
  switch (phase) {
    case 'enter':
      return 'lx-list-item lx-list-item-enter'
    case 'exit':
      return 'lx-list-item lx-list-item-exit'
    case 'move':
      return 'lx-list-item lx-list-item-move'
    case 'steady':
      return 'lx-list-item'
    default: {
      const _exhaustive: never = phase
      return _exhaustive
    }
  }
}

export type ListDragStyleOptions = {
  transform: Transform | null
  transition?: string | null
  isDragging: boolean
  reduceMotion?: boolean
  /** Kill-switch; when false, no lift scale. */
  enabled?: boolean
}

/**
 * @dnd-kit sortable style with lift (scale + shadow) on grab (FR-4).
 * Reduced motion → static elevation, no scale (FR-8).
 */
export function listDragStyle(opts: ListDragStyleOptions): CSSProperties {
  const {
    transform,
    transition,
    isDragging,
    reduceMotion = prefersReducedMotion(),
    enabled = true,
  } = opts

  const scale = enabled && isDragging && !reduceMotion ? LIST_DRAG_LIFT_SCALE : undefined
  const t = transform
    ? {
        ...transform,
        scaleX: scale ?? transform.scaleX,
        scaleY: scale ?? transform.scaleY,
      }
    : null

  return {
    transform: CSS.Transform.toString(t),
    transition:
      transition ??
      (enabled ? `transform ${durations.base}ms ${easings.bubble}` : undefined),
    opacity: isDragging ? 0.95 : undefined,
    zIndex: isDragging ? 10 : undefined,
    boxShadow: isDragging
      ? reduceMotion || !enabled
        ? '0 4px 12px rgb(15 23 42 / 0.12)'
        : '0 10px 28px rgb(15 23 42 / 0.18)'
      : undefined,
    willChange: isDragging ? 'transform' : undefined,
  }
}

/**
 * Drop settle animation using AN.1 bubble curve (FR-4).
 */
export function listDropAnimation(opts?: {
  reduceMotion?: boolean
  enabled?: boolean
}): DropAnimation {
  const reduce = opts?.reduceMotion ?? prefersReducedMotion()
  const enabled = opts?.enabled ?? true
  if (!enabled) {
    return { ...defaultDropAnimation, duration: 0 }
  }
  if (reduce) {
    return {
      ...defaultDropAnimation,
      duration: durations.instant,
      easing: 'linear',
    }
  }
  return {
    ...defaultDropAnimation,
    duration: durations.deliberate,
    easing: easings.bubble,
    keyframes({ transform }) {
      const { initial, final } = transform
      return [
        {
          transform: CSS.Transform.toString({
            ...initial,
            scaleX: LIST_DRAG_LIFT_SCALE,
            scaleY: LIST_DRAG_LIFT_SCALE,
          }),
          opacity: 1,
        },
        {
          transform: CSS.Transform.toString({
            ...final,
            scaleX: 1,
            scaleY: 1,
          }),
          opacity: 1,
        },
      ]
    },
  }
}

/** Enter translate for list items (matches AN.1 distance). */
export const listEnterTranslatePx = distances.enterTranslatePx

/** Keyboard reorder helpers (accessible alternative to drag). */
export function listReorderKeyboardHandlers(opts: {
  index: number
  count: number
  onMove: (from: number, to: number) => void
}): {
  onKeyDown: (e: { key: string; preventDefault: () => void }) => void
} {
  return {
    onKeyDown: (e) => {
      if (e.key === 'ArrowUp' || e.key === 'ArrowLeft') {
        if (opts.index <= 0) return
        e.preventDefault()
        opts.onMove(opts.index, opts.index - 1)
      } else if (e.key === 'ArrowDown' || e.key === 'ArrowRight') {
        if (opts.index >= opts.count - 1) return
        e.preventDefault()
        opts.onMove(opts.index, opts.index + 1)
      }
    },
  }
}
