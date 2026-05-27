/**
 * Moves focus to the main content area on every client-side route change
 * (WCAG 2.1 SC 2.4.3 — Focus Order; SPA best-practice).
 *
 * Targets the element with id="main-content", falling back to the <main>
 * landmark. Adds tabIndex=-1 temporarily so the element is focusable
 * without disrupting natural tab order.
 */
import { useEffect } from 'react'
import { useLocation } from 'react-router-dom'

export function useFocusOnRoute(): void {
  const { pathname } = useLocation()

  useEffect(() => {
    const target =
      (document.getElementById('main-content') as HTMLElement | null) ??
      (document.querySelector('main') as HTMLElement | null)

    if (!target) return

    const hadTabIndex = target.hasAttribute('tabindex')
    const prevTabIndex = target.getAttribute('tabindex')

    target.setAttribute('tabindex', '-1')
    target.focus({ preventScroll: true })

    if (!hadTabIndex) {
      target.removeAttribute('tabindex')
    } else if (prevTabIndex !== null) {
      target.setAttribute('tabindex', prevTabIndex)
    }
  }, [pathname])
}
