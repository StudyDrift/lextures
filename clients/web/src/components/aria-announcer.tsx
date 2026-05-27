/**
 * Global ARIA live regions (WCAG 2.1 SC 4.1.3 — Status Messages).
 *
 * Mount once near the root of the app. Use the `announce()` helper from
 * `lib/a11y` to push messages to the appropriate region.
 *
 * Two regions:
 *  - polite:    queued announcements (default) — does not interrupt current speech
 *  - assertive: urgent announcements — interrupts immediately (use sparingly)
 */

export function AriaAnnouncer() {
  return (
    <>
      <div
        id="a11y-polite-announcer"
        role="status"
        aria-live="polite"
        aria-atomic="true"
        className="pointer-events-none fixed -left-[10000px] top-auto h-px w-px overflow-hidden"
      />
      <div
        id="a11y-assertive-announcer"
        role="alert"
        aria-live="assertive"
        aria-atomic="true"
        className="pointer-events-none fixed -left-[10000px] top-auto h-px w-px overflow-hidden"
      />
    </>
  )
}
