import type { ReactNode } from 'react'

type LiveRegionProps = {
  /** The content to announce. Changing this value triggers a screen-reader announcement. */
  children: ReactNode
  /** Default "polite" (waits for the user to finish) — use "assertive" only for errors/urgent alerts. */
  politeness?: 'polite' | 'assertive'
  /** When true the region is hidden from the visual layout (sr-only). Default true. */
  visuallyHidden?: boolean
  className?: string
}

/**
 * Reusable aria-live region.  Swap in `politeness="assertive"` only for time-sensitive alerts;
 * "polite" is the right default for status updates and search result counts.
 */
export function LiveRegion({
  children,
  politeness = 'polite',
  visuallyHidden = true,
  className,
}: LiveRegionProps) {
  return (
    <div
      aria-live={politeness}
      aria-atomic="true"
      role="status"
      className={
        visuallyHidden
          ? `sr-only ${className ?? ''}`.trim()
          : className
      }
    >
      {children}
    </div>
  )
}
