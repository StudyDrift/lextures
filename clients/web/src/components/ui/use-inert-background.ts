import { useEffect } from 'react'
import { pushModalOverlay } from '../../lib/a11y/overlay-stack'

/**
 * Marks the app root `inert` while a modal overlay is open so background
 * content is removed from the a11y tree (UX.4 FR-4). Nested modals share a
 * ref-counted stack — inert stays on until the last modal closes.
 *
 * Portal content lives under `document.body` as a sibling of `#root`.
 */
export function useInertBackground(active: boolean) {
  useEffect(() => {
    if (!active || typeof document === 'undefined') return
    return pushModalOverlay()
  }, [active])
}
