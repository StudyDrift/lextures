/**
 * WAI-ARIA APG Menu keyboard handler for hand-rolled menus (UX.4 FR-2).
 * Prefer `components/ui/menu` for new code; this helps migrate legacy markup.
 *
 * Expects a container with `[role="menuitem"]` children (buttons or links).
 */

export type MenuKeyHandlers = {
  /** Close menu and restore focus to trigger. */
  onClose: () => void
  /** Optional: after Enter/Space on an item (default: click). */
  onActivate?: (item: HTMLElement) => void
}

function menuitems(container: HTMLElement): HTMLElement[] {
  return Array.from(
    container.querySelectorAll<HTMLElement>('[role="menuitem"]:not([disabled]):not([aria-disabled="true"])'),
  )
}

/**
 * Attach to a menu container's onKeyDown. Implements arrows, Home/End,
 * typeahead, Escape, and Tab-to-close per APG.
 */
export function handleMenuKeyDown(
  e: {
    key: string
    ctrlKey: boolean
    metaKey: boolean
    altKey: boolean
    currentTarget: EventTarget & HTMLElement
    preventDefault: () => void
  },
  handlers: MenuKeyHandlers,
  typeaheadState: { buffer: string; at: number },
): void {
  const items = menuitems(e.currentTarget)
  if (items.length === 0) return

  const current = document.activeElement as HTMLElement | null
  let idx = current ? items.indexOf(current) : -1
  if (idx < 0) idx = 0

  if (e.key === 'ArrowDown') {
    e.preventDefault()
    const next = items[(idx + 1) % items.length]
    next?.focus()
    return
  }
  if (e.key === 'ArrowUp') {
    e.preventDefault()
    const next = items[(idx - 1 + items.length) % items.length]
    next?.focus()
    return
  }
  if (e.key === 'Home') {
    e.preventDefault()
    items[0]?.focus()
    return
  }
  if (e.key === 'End') {
    e.preventDefault()
    items[items.length - 1]?.focus()
    return
  }
  if (e.key === 'Escape') {
    e.preventDefault()
    handlers.onClose()
    return
  }
  if (e.key === 'Tab') {
    // APG: Tab closes; browser continues with restored focus.
    handlers.onClose()
    return
  }
  if (e.key === 'Enter' || e.key === ' ') {
    e.preventDefault()
    const item = items[idx]
    if (!item) return
    if (handlers.onActivate) handlers.onActivate(item)
    else item.click()
    return
  }
  if (e.key.length === 1 && !e.ctrlKey && !e.metaKey && !e.altKey) {
    const now = Date.now()
    if (now - typeaheadState.at > 500) typeaheadState.buffer = ''
    typeaheadState.at = now
    typeaheadState.buffer += e.key.toLowerCase()
    const match = items.find((el) => {
      const text = (el.getAttribute('data-text-value') ?? el.textContent ?? '').trim().toLowerCase()
      return text.startsWith(typeaheadState.buffer)
    })
    match?.focus()
  }
}

/** Focus the first enabled menuitem inside a menu container. */
export function focusFirstMenuitem(container: HTMLElement | null): void {
  if (!container) return
  menuitems(container)[0]?.focus()
}
