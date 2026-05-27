/**
 * DOM-based ARIA live region announcer.
 *
 * Call `announce()` from anywhere to push a message to the live region
 * rendered by <AriaAnnouncer />. Works outside React's render cycle.
 *
 * Uses a double-clear trick: empty → rAF → set text, which forces screen
 * readers to re-announce even when the same string is repeated.
 */

const POLITE_ID = 'a11y-polite-announcer'
const ASSERTIVE_ID = 'a11y-assertive-announcer'

export type Politeness = 'polite' | 'assertive'

export function announce(message: string, politeness: Politeness = 'polite'): void {
  const id = politeness === 'assertive' ? ASSERTIVE_ID : POLITE_ID
  const el = document.getElementById(id)
  if (!el) return
  el.textContent = ''
  requestAnimationFrame(() => {
    el.textContent = message
  })
}
