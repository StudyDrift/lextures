/**
 * UX.4 — central overlay stack for nested modals.
 *
 * Counts active modal layers and applies `inert` to `#root` while any modal is
 * open. Portaled overlays live under `document.body` as siblings of `#root`, so
 * they remain interactive. Nested open → close must not clear inert early.
 *
 * Toasts and non-modal popovers MUST NOT register here.
 */

let depth = 0

function applyInert() {
  if (typeof document === 'undefined') return
  const root = document.getElementById('root')
  if (!root) return
  root.inert = depth > 0
}

/** Register a modal overlay as open. Returns a disposer that unregisters it. */
export function pushModalOverlay(): () => void {
  depth += 1
  applyInert()
  let released = false
  return () => {
    if (released) return
    released = true
    depth = Math.max(0, depth - 1)
    applyInert()
  }
}

/** Test-only: current stack depth. */
export function getModalOverlayDepth(): number {
  return depth
}

/** Test-only: reset stack (e.g. between unit tests). */
export function resetModalOverlayStack(): void {
  depth = 0
  applyInert()
}
