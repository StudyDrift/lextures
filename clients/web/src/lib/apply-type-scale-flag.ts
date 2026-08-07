/**
 * UX.3 — apply ffTypeScale (16px body base) via data-type-scale on <html>.
 * Persists to localStorage so the index.html bootstrap can restore before paint.
 */
export function applyTypeScaleFlag(enabled: boolean): void {
  if (typeof document === 'undefined') return
  const root = document.documentElement
  if (enabled) {
    root.setAttribute('data-type-scale', 'on')
  } else {
    root.removeAttribute('data-type-scale')
  }
  try {
    localStorage.setItem('lextures.typeScale', enabled ? '1' : '0')
  } catch {
    /* ignore */
  }
}
