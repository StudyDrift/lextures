/**
 * Focus trap utility for modal dialogs and overlays (WCAG 2.1 SC 2.1.2).
 *
 * Returns activate/deactivate functions. Activate moves focus to the first
 * focusable element in `container` and constrains Tab/Shift+Tab within it.
 * Deactivate restores focus to the element that was focused before activation,
 * with a fallback chain when the trigger has unmounted (UX.4 FR-4 / reliability):
 * trigger → nearest stable ancestor still in the document → `#main-content` /
 * `main` → `document.body`.
 */

const FOCUSABLE =
  'a[href], button:not([disabled]), input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"]), details > summary'

function getFocusable(container: HTMLElement): HTMLElement[] {
  return Array.from(container.querySelectorAll<HTMLElement>(FOCUSABLE)).filter((el) => {
    if (el.closest('[inert]')) return false
    // Skip truly hidden elements; jsdom has no layout so offsetParent/rects are unreliable —
    // only exclude when the attribute/style explicitly hides the node.
    if (el.getAttribute('aria-hidden') === 'true') return false
    if (el.hasAttribute('hidden')) return false
    const style = el.style
    if (style?.display === 'none' || style?.visibility === 'hidden') return false
    return true
  })
}

/**
 * Focus restoration fallback chain when the original trigger is gone.
 * Prefer a still-connected ancestor, then the main landmark, then body.
 */
export function resolveFocusRestoreTarget(
  preferred: HTMLElement | null | undefined,
): HTMLElement | null {
  if (preferred && preferred.isConnected) return preferred

  // Walk ancestors even when preferred is detached (parent chain may still exist briefly).
  let walk: Element | null = preferred ?? null
  while (walk) {
    const parent = walk.parentElement
    if (!parent) break
    if (
      parent instanceof HTMLElement &&
      parent.isConnected &&
      parent !== document.body &&
      parent !== document.documentElement &&
      (parent.matches(FOCUSABLE) ||
        (parent.tabIndex >= 0 && parent.getAttribute('tabindex') !== '-1'))
    ) {
      return parent
    }
    walk = parent
  }

  const main =
    (document.getElementById('main-content') as HTMLElement | null) ??
    (document.querySelector('main') as HTMLElement | null)
  if (main && main.isConnected) return main

  return document.body
}

export interface FocusTrap {
  activate: () => void
  deactivate: () => void
}

export type CreateFocusTrapOptions = {
  /** Override initial focus (defaults to first focusable in container). */
  initialFocus?: HTMLElement | null
  /** Override restore target captured on activate. */
  restoreFocus?: HTMLElement | null
}

export function createFocusTrap(
  container: HTMLElement,
  options: CreateFocusTrapOptions = {},
): FocusTrap {
  let restoreTo: HTMLElement | null = null

  function handleKeyDown(e: KeyboardEvent) {
    if (e.key !== 'Tab') return
    const focusable = getFocusable(container)
    if (focusable.length === 0) return
    const first = focusable[0]!
    const last = focusable[focusable.length - 1]!
    if (e.shiftKey) {
      if (document.activeElement === first) {
        e.preventDefault()
        last.focus()
      }
    } else if (document.activeElement === last) {
      e.preventDefault()
      first.focus()
    }
  }

  return {
    activate() {
      restoreTo =
        options.restoreFocus ??
        (document.activeElement instanceof HTMLElement ? document.activeElement : null)
      const focusable = getFocusable(container)
      const initial = options.initialFocus
      if (initial && container.contains(initial)) {
        initial.focus()
      } else if (focusable.length > 0) {
        focusable[0]!.focus()
      } else {
        // Make the container itself focusable as a last resort (empty dialog).
        if (!container.hasAttribute('tabindex')) {
          container.setAttribute('tabindex', '-1')
        }
        container.focus()
      }
      document.addEventListener('keydown', handleKeyDown)
    },
    deactivate() {
      document.removeEventListener('keydown', handleKeyDown)
      const target = resolveFocusRestoreTarget(restoreTo)
      restoreTo = null
      if (!target) return
      try {
        // Landmarks / non-interactive restore targets need a temporary tabindex.
        const needsTabIndex =
          !target.matches(
            'a[href], button, input, select, textarea, [tabindex]:not([tabindex="-1"])',
          ) && !target.hasAttribute('tabindex')
        if (needsTabIndex) target.setAttribute('tabindex', '-1')
        target.focus({ preventScroll: true })
        if (needsTabIndex) {
          // Leave tabindex so subsequent restores work; remove only if we didn't need it long-term.
          // Keep -1 so the element remains programmatically focusable (SPA shell pattern).
        }
      } catch {
        /* detached or non-focusable — ignore */
      }
    },
  }
}
