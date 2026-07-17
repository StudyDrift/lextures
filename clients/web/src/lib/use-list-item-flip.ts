/**
 * AN.4 — FLIP-style position transition when an item's layout rect moves.
 */

import { type CSSProperties, useCallback, useEffect, useRef, useState } from 'react'
import { usePrefersReducedMotion } from './motion'

export function useListItemFlip(active: boolean): {
  ref: (node: HTMLElement | null) => void
  style: CSSProperties
} {
  const reduceMotion = usePrefersReducedMotion()
  const nodeRef = useRef<HTMLElement | null>(null)
  const prevRect = useRef<DOMRect | null>(null)
  const [style, setStyle] = useState<CSSProperties>({})

  const ref = useCallback((node: HTMLElement | null) => {
    nodeRef.current = node
  }, [])

  useEffect(() => {
    const node = nodeRef.current
    if (!node || !active || reduceMotion) {
      prevRect.current = node?.getBoundingClientRect() ?? null
      return
    }
    const next = node.getBoundingClientRect()
    const prev = prevRect.current
    prevRect.current = next
    if (!prev) return
    const dx = prev.left - next.left
    const dy = prev.top - next.top
    if (dx === 0 && dy === 0) return

    setStyle({
      transform: `translate(${dx}px, ${dy}px)`,
      transition: 'none',
    })
    const id = requestAnimationFrame(() => {
      setStyle({
        transform: 'translate(0, 0)',
        transition: 'transform var(--dur-base) var(--ease-bubble)',
      })
    })
    return () => cancelAnimationFrame(id)
  }, [active, reduceMotion])

  return { ref, style }
}
