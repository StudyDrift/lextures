/**
 * Skip navigation link (WCAG 2.1 SC 2.4.1 — Bypass Blocks).
 *
 * Visually hidden until focused, then overlays the top of the page so
 * keyboard/screen-reader users can bypass repeated navigation.
 */

interface SkipLinkProps {
  target?: string
  label?: string
}

export function SkipLink({ target = '#main-content', label = 'Skip to main content' }: SkipLinkProps) {
  return (
    <a
      href={target}
      className={[
        // Hidden by default
        'absolute left-2 top-2 z-[9999]',
        '-translate-y-20 opacity-0',
        // Visible on focus
        'focus:translate-y-0 focus:opacity-100',
        // Styling
        'rounded-md bg-indigo-600 px-4 py-2 text-sm font-semibold text-white',
        'transition-all duration-150',
        'focus:outline-none focus:ring-2 focus:ring-indigo-400 focus:ring-offset-2',
      ].join(' ')}
    >
      {label}
    </a>
  )
}
