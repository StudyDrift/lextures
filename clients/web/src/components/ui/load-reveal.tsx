/**
 * AN.3 — Load choreography primitives.
 *
 * - `<LoadReveal>` crossfades skeleton → content (FR-1).
 * - `<StaggerReveal index>` bubble-enters peers with capped stagger (FR-2 / FR-3).
 *
 * Content stays interactive during fade (FR-9). Reduced motion → opacity ≤100ms, no stagger.
 * Region state: `useReveal` from `@/lib/use-reveal`.
 */

import {
  type CSSProperties,
  type ReactNode,
  useEffect,
  useRef,
  useState,
} from 'react'
import { durations, revealDelayMs, usePrefersReducedMotion } from '../../lib/motion'
import { useReveal } from '../../lib/use-reveal'

export type LoadRevealProps = {
  ready: boolean
  enabled?: boolean
  skeleton: ReactNode
  children: ReactNode
  className?: string
  /** Optional aria label for the busy region. */
  'aria-label'?: string
}

/**
 * Crossfade skeleton → content. Once revealed, content stays mounted through refreshes.
 */
export function LoadReveal({
  ready,
  enabled = true,
  skeleton,
  children,
  className,
  'aria-label': ariaLabel,
}: LoadRevealProps) {
  const { showSkeleton, showContent, playEntrance, reduceMotion, hasRevealed } = useReveal({
    ready,
    enabled,
  })
  const busy = !ready && !hasRevealed

  const contentClass =
    playEntrance
      ? reduceMotion
        ? 'lx-motion-fade-in'
        : 'lx-load-reveal-in'
      : undefined

  return (
    <div
      className={['lx-load-reveal', className].filter(Boolean).join(' ')}
      aria-busy={busy || undefined}
      aria-label={ariaLabel}
      data-motion-reveal={enabled ? 'on' : 'off'}
    >
      {showSkeleton ? (
        <div className="lx-load-reveal-skeleton">{skeleton}</div>
      ) : null}
      {showContent ? (
        <div className={['lx-load-reveal-content', contentClass].filter(Boolean).join(' ')}>
          {children}
        </div>
      ) : null}
    </div>
  )
}

export type StaggerRevealProps = {
  /** 0-based peer index; delays cap at stagger.maxItems. */
  index: number
  /** When false, render children with no entrance (kill-switch). */
  enabled?: boolean
  /** Parent region ready — entrance runs once when this becomes true. */
  ready?: boolean
  children: ReactNode
  className?: string
  as?: 'div' | 'section' | 'li' | 'article'
  'aria-label'?: string
}

/**
 * Staggered bubble entrance for a peer item. Runs once per mount when `ready` is true.
 * Items beyond the stagger cap share the max delay (group-fade).
 */
export function StaggerReveal({
  index,
  enabled = true,
  ready = true,
  children,
  className,
  as: Tag = 'div',
  'aria-label': ariaLabel,
}: StaggerRevealProps) {
  const reduceMotion = usePrefersReducedMotion()
  const [entered, setEntered] = useState(false)
  const didEnter = useRef(false)

  useEffect(() => {
    if (!ready || !enabled || didEnter.current) return
    didEnter.current = true
    const id = requestAnimationFrame(() => setEntered(true))
    return () => cancelAnimationFrame(id)
  }, [ready, enabled])

  if (!enabled) {
    return (
      <Tag className={className} data-lx-reveal="off" aria-label={ariaLabel}>
        {children}
      </Tag>
    )
  }

  const delay = revealDelayMs(index, reduceMotion)
  const style: CSSProperties = {
    ['--lx-reveal-delay' as string]: `${delay}ms`,
    pointerEvents: 'auto',
  }

  const animClass = !ready || !entered
    ? 'lx-stagger-reveal-pending'
    : reduceMotion
      ? 'lx-motion-fade-in'
      : 'lx-stagger-reveal'

  return (
    <Tag
      className={[animClass, className].filter(Boolean).join(' ')}
      style={style}
      data-lx-reveal={entered ? 'in' : 'pending'}
      data-lx-reveal-index={index}
      aria-label={ariaLabel}
    >
      {children}
    </Tag>
  )
}

/** Crossfade duration token (ms) for tests / docs. */
export const LOAD_REVEAL_DURATION_MS = durations.base
