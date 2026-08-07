/**
 * WAI-ARIA APG Tabs keyboard handler for hand-rolled tablists (UX.4 FR-1).
 * Prefer `components/ui/tabs` for new code; this helps migrate legacy markup.
 */

export type TablistOrientation = 'horizontal' | 'vertical'

function isRtl(el: HTMLElement): boolean {
  const dir = el.closest('[dir]')?.getAttribute('dir') ?? document.documentElement.dir
  return dir === 'rtl'
}

/**
 * Attach to a tablist's onKeyDown. Moves focus/selection among `[role="tab"]`
 * children with Arrow keys, Home, End. Invokes `onActivate(tabEl)` after move
 * (default: click).
 */
export function handleTablistKeyDown(
  e: { key: string; currentTarget: EventTarget & HTMLElement; preventDefault: () => void },
  options: {
    orientation?: TablistOrientation
    onActivate?: (tab: HTMLElement) => void
  } = {},
): void {
  const orientation = options.orientation ?? 'horizontal'
  const onActivate = options.onActivate ?? ((tab: HTMLElement) => tab.click())

  const tabs = Array.from(
    e.currentTarget.querySelectorAll<HTMLElement>('[role="tab"]:not([disabled])'),
  )
  if (tabs.length === 0) return
  const current = document.activeElement as HTMLElement | null
  const idx = current ? tabs.indexOf(current) : -1
  if (idx < 0) return

  const horizontal = orientation === 'horizontal'
  const rtl = horizontal && isRtl(e.currentTarget)
  let next = idx

  const goNext =
    (horizontal && (rtl ? e.key === 'ArrowLeft' : e.key === 'ArrowRight')) ||
    (!horizontal && e.key === 'ArrowDown')
  const goPrev =
    (horizontal && (rtl ? e.key === 'ArrowRight' : e.key === 'ArrowLeft')) ||
    (!horizontal && e.key === 'ArrowUp')

  if (goNext) {
    e.preventDefault()
    next = (idx + 1) % tabs.length
  } else if (goPrev) {
    e.preventDefault()
    next = (idx - 1 + tabs.length) % tabs.length
  } else if (e.key === 'Home') {
    e.preventDefault()
    next = 0
  } else if (e.key === 'End') {
    e.preventDefault()
    next = tabs.length - 1
  } else {
    return
  }

  const tab = tabs[next]
  if (!tab) return
  tab.focus()
  onActivate(tab)
}
