import { useEffect } from 'react'
import { useLocation } from 'react-router-dom'
import { syncStickyOffset } from './sticky-offset'

const CHROME_SEL =
  'header.lms-chrome, [data-lx-sticky-chrome], [data-quiz-focus-bar], [data-reading-focus-bar]'

/**
 * Keeps `--lx-sticky-offset` in sync with sticky chrome height (UX.5 FR-9).
 * Mount once in the app shell.
 */
export function useStickyOffset(): void {
  const location = useLocation()

  useEffect(() => {
    let raf = 0
    const run = () => {
      cancelAnimationFrame(raf)
      raf = requestAnimationFrame(() => {
        syncStickyOffset()
      })
    }
    run()

    const ro = typeof ResizeObserver !== 'undefined' ? new ResizeObserver(run) : null
    const observeChrome = () => {
      for (const el of document.querySelectorAll(CHROME_SEL)) {
        ro?.observe(el)
      }
    }
    observeChrome()

    // Chrome swaps between TopBar / quiz / reading focus without unmounting the shell.
    const shell = document.querySelector('.flex.h-dvh') ?? document.body
    const mo =
      typeof MutationObserver !== 'undefined'
        ? new MutationObserver(() => {
            observeChrome()
            run()
          })
        : null
    mo?.observe(shell, { childList: true, subtree: true })

    window.addEventListener('resize', run)
    const t = window.setTimeout(run, 100)

    return () => {
      window.clearTimeout(t)
      cancelAnimationFrame(raf)
      window.removeEventListener('resize', run)
      ro?.disconnect()
      mo?.disconnect()
    }
  }, [location.pathname])
}
