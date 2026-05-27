/**
 * Focus trap utility for modal dialogs and overlays (WCAG 2.1 SC 2.1.2).
 *
 * Returns activate/deactivate functions. Activate moves focus to the first
 * focusable element in `container` and constrains Tab/Shift+Tab within it.
 * Deactivate restores focus to the element that was focused before activation.
 */

const FOCUSABLE =
  'a[href], button:not([disabled]), input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"]), details > summary'

function getFocusable(container: HTMLElement): HTMLElement[] {
  return Array.from(container.querySelectorAll<HTMLElement>(FOCUSABLE)).filter(
    (el) => !el.closest('[inert]'),
  )
}

export interface FocusTrap {
  activate: () => void
  deactivate: () => void
}

export function createFocusTrap(container: HTMLElement): FocusTrap {
  let restoreTo: HTMLElement | null = null

  function handleKeyDown(e: KeyboardEvent) {
    if (e.key !== 'Tab') return
    const focusable = getFocusable(container)
    if (focusable.length === 0) return
    const first = focusable[0]
    const last = focusable[focusable.length - 1]
    if (e.shiftKey) {
      if (document.activeElement === first) {
        e.preventDefault()
        last.focus()
      }
    } else {
      if (document.activeElement === last) {
        e.preventDefault()
        first.focus()
      }
    }
  }

  return {
    activate() {
      restoreTo = document.activeElement as HTMLElement | null
      const focusable = getFocusable(container)
      if (focusable.length > 0) focusable[0].focus()
      document.addEventListener('keydown', handleKeyDown)
    },
    deactivate() {
      document.removeEventListener('keydown', handleKeyDown)
      restoreTo?.focus()
      restoreTo = null
    },
  }
}
