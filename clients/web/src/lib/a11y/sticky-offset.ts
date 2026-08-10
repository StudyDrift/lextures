/**
 * UX.5 FR-8/FR-9 — Focus Not Obscured (Minimum).
 *
 * Maintains `--lx-sticky-offset` from the rendered sticky chrome height so
 * focusable content can use `scroll-margin-block-start` and never sit entirely
 * under the top bar / focus bars.
 */

export const STICKY_OFFSET_CSS_VAR = '--lx-sticky-offset'
export const STICKY_CHROME_SELECTOR =
  'header.lms-chrome, [data-lx-sticky-chrome], [data-quiz-focus-bar], [data-reading-focus-bar]'

/** Default when chrome has not measured yet (TopBar is h-14 = 3.5rem). */
export const DEFAULT_STICKY_OFFSET_PX = 56

/**
 * Sum the heights of sticky chrome elements that overlap the main scrollport.
 * Only elements currently in the layout (not `display:none`) contribute.
 */
export function measureStickyChromeOffset(root: ParentNode = document): number {
  const nodes = root.querySelectorAll(STICKY_CHROME_SELECTOR)
  let maxBottom = 0
  for (const node of nodes) {
    if (!(node instanceof HTMLElement)) continue
    const style = getComputedStyle(node)
    if (style.display === 'none' || style.visibility === 'hidden') continue
    const rect = node.getBoundingClientRect()
    if (rect.height <= 0) continue
    // Sticky/fixed chrome typically anchors to the top of the viewport.
    if (rect.top >= 0 && rect.top < 120) {
      maxBottom = Math.max(maxBottom, rect.bottom)
    }
  }
  if (maxBottom > 0) return Math.ceil(maxBottom)
  return DEFAULT_STICKY_OFFSET_PX
}

/** Write the CSS custom property on `:root` (or a provided element). */
export function applyStickyOffset(
  offsetPx: number,
  target: HTMLElement = document.documentElement,
): void {
  const px = Math.max(0, Math.round(offsetPx))
  target.style.setProperty(STICKY_OFFSET_CSS_VAR, `${px}px`)
}

export function readStickyOffset(target: HTMLElement = document.documentElement): number {
  const raw = getComputedStyle(target).getPropertyValue(STICKY_OFFSET_CSS_VAR).trim()
  if (!raw) return DEFAULT_STICKY_OFFSET_PX
  const n = Number.parseFloat(raw)
  return Number.isFinite(n) ? n : DEFAULT_STICKY_OFFSET_PX
}

/**
 * Measure and apply in one step. Safe to call from ResizeObserver / layout effects.
 */
export function syncStickyOffset(root: ParentNode = document): number {
  const px = measureStickyChromeOffset(root)
  applyStickyOffset(px)
  return px
}
