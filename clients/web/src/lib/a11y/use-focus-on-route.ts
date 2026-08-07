/**
 * Moves focus to the new page on every client-side route change
 * (WCAG 2.1 SC 2.4.3 — Focus Order; SPA best-practice).
 *
 * Prefer the first `h1` inside main content (announces the new page), then
 * `#main-content`, then the `<main>` landmark. Adds tabIndex=-1 temporarily so
 * the element is focusable without disrupting natural tab order (UX.4 FR-9).
 */
import { useEffect } from 'react'
import { useLocation } from 'react-router-dom'

export function useFocusOnRoute(): void {
  const { pathname } = useLocation()

  useEffect(() => {
    const main =
      (document.getElementById('main-content') as HTMLElement | null) ??
      (document.querySelector('main') as HTMLElement | null)

    const heading =
      (main?.querySelector('h1') as HTMLElement | null) ??
      (document.querySelector('h1') as HTMLElement | null)

    const target = heading ?? main
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
