import { Toaster } from 'sonner'
import { useSyncExternalStore } from 'react'
import { useLmsDarkMode } from '../hooks/use-lms-dark-mode'
import { motionOverlaysEnabled, subscribePlatformFeatures } from '../lib/platform-features'
import { usePrefersReducedMotion } from '../lib/motion'
import { overlayMotionTokens } from '../lib/overlay-motion'

/**
 * Global toast queue: top-right, stacks, auto-dismiss. Sonner uses a live region
 * for screen reader announcements (polite updates).
 * AN.5: enter slide+fade / exit fade tuned to AN.1 tokens via `.lx-toaster-motion`.
 *
 * Mounted outside PlatformFeaturesProvider — reads the module snapshot.
 */
export function LmsToaster() {
  const dark = useLmsDarkMode()
  const overlaysOn = useSyncExternalStore(
    subscribePlatformFeatures,
    () => motionOverlaysEnabled(),
    () => true,
  )
  const reducedMotion = usePrefersReducedMotion()
  const motionOn = overlaysOn && !reducedMotion

  return (
    <Toaster
      position="top-right"
      closeButton
      richColors
      expand={false}
      visibleToasts={5}
      theme={dark ? 'dark' : 'light'}
      className={overlaysOn ? 'lx-toaster-motion' : undefined}
      duration={4500}
      toastOptions={{
        duration: 4500,
        classNames: {
          toast: motionOn ? 'font-sans lx-overlay-toast-panel' : 'font-sans',
        },
        style: motionOn
          ? {
              ['--lx-toast-enter' as string]: `${overlayMotionTokens.enterMs}ms`,
              ['--lx-toast-exit' as string]: `${overlayMotionTokens.exitMs}ms`,
            }
          : undefined,
      }}
    />
  )
}
