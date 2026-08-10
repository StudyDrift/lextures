import { useEffect } from 'react'

/**
 * UX.6 FR-10 — warn before leaving a dirty form.
 *
 * Uses `beforeunload` for tab close / refresh. In-app navigation should pair this
 * with `UnsavedChangesBanner` (save/discard) until the app moves to a data router
 * where `useBlocker` can intercept SPA transitions.
 */
export function useUnsavedChanges(dirty: boolean, _message?: string) {
  useEffect(() => {
    if (!dirty) return
    const onBeforeUnload = (e: BeforeUnloadEvent) => {
      e.preventDefault()
      // Chromium requires returnValue to be set for the dialog to appear.
      e.returnValue = ''
    }
    window.addEventListener('beforeunload', onBeforeUnload)
    return () => window.removeEventListener('beforeunload', onBeforeUnload)
  }, [dirty])
}
