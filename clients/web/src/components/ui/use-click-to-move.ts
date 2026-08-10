/**
 * UX.5 — click-to-move hook (WCAG 2.2 SC 2.5.7 single-pointer alternative).
 * Lives separate from reorderable components so Fast Refresh stays happy.
 */
import { useCallback, useEffect, useState } from 'react'
import { announce } from '../../lib/a11y'
import {
  initialClickToMoveState,
  isClickToMoveActive,
  isValidClickToMoveTarget,
  reduceClickToMove,
  type ClickToMoveState,
} from '../../lib/reorderable/click-to-move'

export type UseClickToMoveOptions = {
  /** Called when the user picks a destination after selecting a source. */
  onMove: (sourceId: string, targetId: string) => void
  /** Optional live-region copy. Defaults are English fallbacks. */
  messages?: {
    enter?: string
    cancel?: string
  }
}

export function useClickToMove(options: UseClickToMoveOptions) {
  const { onMove, messages } = options
  const [state, setState] = useState<ClickToMoveState>(initialClickToMoveState)

  const apply = useCallback(
    (event: Parameters<typeof reduceClickToMove>[1]) => {
      setState((prev) => {
        const { state: next, effect } = reduceClickToMove(prev, event)
        if (effect.type === 'commit') {
          onMove(effect.sourceId, effect.targetId)
        } else if (effect.type === 'announce') {
          const text =
            effect.messageKey === 'enter'
              ? (messages?.enter ??
                'Move mode. Use arrow keys or select a destination. Escape cancels.')
              : (messages?.cancel ?? 'Move cancelled.')
          announce(text)
        }
        return next
      })
    },
    [messages?.cancel, messages?.enter, onMove],
  )

  useEffect(() => {
    if (!isClickToMoveActive(state)) return
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') {
        e.preventDefault()
        apply({ type: 'cancel' })
      }
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [apply, state])

  return {
    state,
    active: isClickToMoveActive(state),
    sourceId: state.mode === 'idle' ? null : state.sourceId,
    selectSource: (id: string) => apply({ type: 'select-source', id }),
    selectTarget: (id: string) => apply({ type: 'select-target', id }),
    cancel: () => apply({ type: 'cancel' }),
    isValidTarget: (id: string) => isValidClickToMoveTarget(state, id),
  }
}
