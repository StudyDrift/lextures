import { Toaster } from 'sonner'
import { useLmsDarkMode } from '../hooks/use-lms-dark-mode'

/**
 * Global toast queue: top-right, stacks, auto-dismiss. Sonner uses a live region
 * for screen reader announcements (polite updates).
 * AN.5: enter slide+fade / exit fade tuned to AN.1 tokens via `.lx-toaster-motion`
 * (reduced-motion CSS overrides in index.css).
 */
export function LmsToaster() {
  const dark = useLmsDarkMode()
  return (
    <Toaster
      // UX.5 FR-10: top-right stack + scroll-margin-inline-end on focusables so
      // focused content is not entirely obscured by toasts (SC 2.4.11).
      position="top-right"
      closeButton
      richColors
      expand={false}
      visibleToasts={5}
      offset={{ top: 'calc(var(--lx-sticky-offset, 3.5rem) + 0.5rem)', right: '1rem' }}
      theme={dark ? 'dark' : 'light'}
      className="lx-toaster-motion"
      toastOptions={{
        duration: 4500,
        classNames: {
          toast: 'font-sans',
        },
      }}
    />
  )
}
