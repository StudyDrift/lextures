import { useCallback, useEffect, useState, type TransitionEvent } from 'react'

type ExitTransitionResult = {
  /** Keep rendering while exit animation plays. */
  rendered: boolean
  exiting: boolean
  onTransitionEnd: (e: TransitionEvent) => void
}

/** Gates unmount behind a CSS exit transition (plan 22.1, rule 6). */
export function useExitTransition(visible: boolean, property = 'opacity'): ExitTransitionResult {
  const [rendered, setRendered] = useState(visible)
  const [exiting, setExiting] = useState(false)

  useEffect(() => {
    if (visible) {
      setRendered(true)
      setExiting(false)
      return
    }
    if (rendered) {
      setExiting(true)
    }
  }, [visible, rendered])

  const onTransitionEnd = useCallback(
    (e: TransitionEvent) => {
      if (e.propertyName !== property || !exiting) return
      setRendered(false)
      setExiting(false)
    },
    [exiting, property],
  )

  return { rendered, exiting, onTransitionEnd }
}