import type { CSSProperties } from 'react'

export type MenuPlacement = 'bottom-start' | 'bottom-end' | 'top-start' | 'top-end'

const MENU_GAP = 4
const MENU_PAD = 8
const MENU_MIN_WIDTH = 160

/** Fixed-position style that prefers `placement` but stays inside the viewport. */
export function menuPositionStyle(
  anchor: Pick<DOMRect, 'top' | 'right' | 'bottom' | 'left' | 'width'>,
  menuSize: { width: number; height: number },
  placement: MenuPlacement,
  viewport: { width: number; height: number } = typeof window === 'undefined'
    ? { width: 1024, height: 768 }
    : { width: window.innerWidth, height: window.innerHeight },
): CSSProperties {
  const minWidth = Math.max(anchor.width, MENU_MIN_WIDTH)
  const measured = menuSize.width > 0
  const width = measured ? Math.max(menuSize.width, minWidth) : minWidth
  const height = menuSize.height
  const vw = viewport.width
  const vh = viewport.height

  let vertical: 'bottom' | 'top' = placement.startsWith('bottom') ? 'bottom' : 'top'
  let horizontal: 'start' | 'end' = placement.endsWith('start') ? 'start' : 'end'

  if (height > 0) {
    const below = vh - anchor.bottom - MENU_GAP
    const above = anchor.top - MENU_GAP
    if (vertical === 'bottom' && height > below - MENU_PAD && above > below) vertical = 'top'
    else if (vertical === 'top' && height > above - MENU_PAD && below > above) vertical = 'bottom'
  }
  if (width > 0) {
    const startFits = anchor.left + width <= vw - MENU_PAD
    const endFits = anchor.right - width >= MENU_PAD
    if (horizontal === 'start' && !startFits && endFits) horizontal = 'end'
    else if (horizontal === 'end' && !endFits && startFits) horizontal = 'start'
  }

  let top = vertical === 'bottom' ? anchor.bottom + MENU_GAP : anchor.top - MENU_GAP - height
  const maxTop = vh - MENU_PAD - Math.min(Math.max(height, 0), vh - MENU_PAD * 2)
  top = Math.min(Math.max(MENU_PAD, top), Math.max(MENU_PAD, maxTop))

  const style: CSSProperties = {
    position: 'fixed',
    zIndex: 460,
    top,
    minWidth,
    maxWidth: vw - MENU_PAD * 2,
    maxHeight: vh - MENU_PAD * 2,
    overflowY: height > vh - MENU_PAD * 2 ? 'auto' : undefined,
  }

  if (!measured && horizontal === 'end') {
    // Width unknown: pin the end edge so a first paint cannot overflow right.
    style.right = Math.max(MENU_PAD, vw - anchor.right)
  } else {
    let left = horizontal === 'start' ? anchor.left : anchor.right - width
    left = Math.min(left, vw - MENU_PAD - width)
    style.left = Math.max(MENU_PAD, left)
  }

  return style
}
