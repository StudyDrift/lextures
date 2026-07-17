/**
 * AN.4 — `<AnimatedList>` for insert/remove/reorder motion.
 *
 * Initial mount is steady (AN.3 owns first reveal). Mutations animate while
 * `enabled` (typically `ffMotionLists`). Reduced motion → opacity-only.
 */

import {
  type AnimationEvent,
  type ElementType,
  type ReactNode,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react'
import { listPhaseClassName } from '../../lib/list-motion'
import { useListTransition } from '../../lib/use-list-transition'
import { usePrefersReducedMotion } from '../../lib/motion'

export type AnimatedListItemMeta = {
  phase: 'enter' | 'exit' | 'move' | 'steady'
  animate: boolean
  className: string
  index: number
  completeExit: () => void
}

export type AnimatedListProps<T> = {
  items: readonly T[]
  getKey: (item: T) => string
  /** Feature kill-switch (`ff_motion_lists`). */
  enabled?: boolean
  mode?: 'mutate' | 'append'
  as?: 'ul' | 'ol' | 'div'
  /** Wrapper per item; defaults to `li` for lists, `div` otherwise. */
  itemAs?: 'li' | 'div'
  className?: string
  itemClassName?: string
  role?: string
  'aria-label'?: string
  'aria-live'?: 'polite' | 'off' | 'assertive'
  children: (item: T, meta: AnimatedListItemMeta) => ReactNode
  onRemovedFocusKey?: (nextKey: string | null) => void
}

/**
 * Animated keyed list. Retains removed items until their exit animation completes.
 */
export function AnimatedList<T>({
  items,
  getKey,
  enabled = true,
  mode = 'mutate',
  as: Tag = 'ul',
  itemAs,
  className,
  itemClassName,
  role,
  'aria-label': ariaLabel,
  'aria-live': ariaLive = 'polite',
  children,
  onRemovedFocusKey,
}: AnimatedListProps<T>) {
  const reduceMotion = usePrefersReducedMotion()
  const ItemTag: ElementType = itemAs ?? (Tag === 'div' ? 'div' : 'li')
  const cacheRef = useRef(new Map<string, T>())
  const keys = useMemo(() => items.map(getKey), [items, getKey])

  for (const item of items) {
    cacheRef.current.set(getKey(item), item)
  }

  const [announce, setAnnounce] = useState('')
  const handleAnnounce = useCallback(
    (msg: { type: 'added' | 'removed'; keys: string[] }) => {
      if (msg.type === 'removed') {
        setAnnounce(msg.keys.length === 1 ? 'Item removed' : `${msg.keys.length} items removed`)
        onRemovedFocusKey?.(keys[0] ?? null)
      } else {
        setAnnounce(msg.keys.length === 1 ? 'Item added' : `${msg.keys.length} items added`)
      }
    },
    [keys, onRemovedFocusKey],
  )

  const { items: transitions, completeExit } = useListTransition({
    keys,
    enabled,
    mode,
    onAnnounce: handleAnnounce,
  })

  useEffect(() => {
    const live = new Set(transitions.map((t) => t.key))
    for (const key of [...cacheRef.current.keys()]) {
      if (!live.has(key)) cacheRef.current.delete(key)
    }
  }, [transitions])

  return (
    <Tag
      className={['lx-animated-list', className].filter(Boolean).join(' ')}
      role={role}
      aria-label={ariaLabel}
      data-motion-lists={enabled ? 'on' : 'off'}
      data-lx-list-mode={mode}
    >
      <span className="sr-only" aria-live={ariaLive} aria-atomic="true">
        {announce}
      </span>
      {transitions.map((t) => {
        const item = cacheRef.current.get(t.key)
        if (item === undefined) return null
        const phaseClass = listPhaseClassName(t.phase, t.animate, reduceMotion)
        return (
          <ItemTag
            key={t.key}
            className={['lx-list-item-shell', phaseClass, itemClassName].filter(Boolean).join(' ')}
            data-lx-list-phase={t.phase}
            data-lx-list-animate={t.animate ? '1' : '0'}
            onAnimationEnd={(e: AnimationEvent) => {
              if (e.target !== e.currentTarget) return
              if (t.phase === 'exit') completeExit(t.key)
            }}
          >
            <div className="lx-list-item-inner">
              {children(item, {
                phase: t.phase,
                animate: t.animate,
                className: phaseClass,
                index: t.index,
                completeExit: () => completeExit(t.key),
              })}
            </div>
          </ItemTag>
        )
      })}
    </Tag>
  )
}
