import { useEffect } from 'react'

/**
 * Marks the app root `inert` while a modal overlay is open so background
 * content is removed from the a11y tree (FR-4). Portal content lives under
 * `document.body` as a sibling of `#root`.
 */
export function useInertBackground(active: boolean) {
  useEffect(() => {
    if (!active || typeof document === 'undefined') return
    const root = document.getElementById('root')
    if (!root) return
    const prev = root.inert
    root.inert = true
    return () => {
      root.inert = prev
    }
  }, [active])
}
