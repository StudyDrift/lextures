/**
 * Recover from stale SPA deploys where a lazy chunk URL 404s or returns HTML
 * (MIME type "text/html" instead of JavaScript). One hard reload usually picks
 * up the new index + asset set.
 */

const RELOAD_KEY = 'lextures.chunk-load-reload'

function isChunkLoadError(error: unknown): boolean {
  if (!(error instanceof Error)) return false
  const msg = error.message || ''
  return (
    error.name === 'ChunkLoadError' ||
    /Failed to fetch dynamically imported module/i.test(msg) ||
    /Loading chunk [\d]+ failed/i.test(msg) ||
    /Importing a module script failed/i.test(msg) ||
    /error loading dynamically imported module/i.test(msg)
  )
}

function alreadyReloadedRecently(): boolean {
  try {
    const raw = sessionStorage.getItem(RELOAD_KEY)
    if (!raw) return false
    const ts = Number(raw)
    if (!Number.isFinite(ts)) return false
    // Avoid reload loops if the deploy is still broken.
    return Date.now() - ts < 30_000
  } catch {
    return false
  }
}

function markReloaded(): void {
  try {
    sessionStorage.setItem(RELOAD_KEY, String(Date.now()))
  } catch {
    // ignore quota / private mode
  }
}

/** Hard-reload once when a deploy left the browser on a stale chunk graph. */
export function reloadForStaleChunkOnce(): boolean {
  if (alreadyReloadedRecently()) return false
  markReloaded()
  window.location.reload()
  return true
}

export function isStaleChunkError(error: unknown): boolean {
  return isChunkLoadError(error)
}

/**
 * Wrap a dynamic import so chunk-load failures trigger a single recovery reload.
 * Use with React.lazy(() => lazyImport(() => import('...'))).
 */
export function lazyImport<T>(factory: () => Promise<T>): Promise<T> {
  return factory().catch((error: unknown) => {
    if (isChunkLoadError(error) && reloadForStaleChunkOnce()) {
      // Never resolves — page is reloading.
      return new Promise<T>(() => {})
    }
    throw error
  })
}

/** Install Vite preload + unhandled rejection handlers for chunk failures. */
export function installChunkLoadRecovery(): void {
  if (typeof window === 'undefined') return

  window.addEventListener('vite:preloadError', (event) => {
    // Prevent the default unhandled error path when we handle it.
    event.preventDefault()
    reloadForStaleChunkOnce()
  })

  window.addEventListener('unhandledrejection', (event) => {
    if (isChunkLoadError(event.reason) && reloadForStaleChunkOnce()) {
      event.preventDefault()
    }
  })
}
