/** Number of courses to hydrate on narrow viewports before deferring the rest (LH.2 FR-6). */
export const MOBILE_COURSE_PREFETCH_LIMIT = 3

export function isMobileViewport(): boolean {
  if (typeof window === 'undefined') return false
  return window.matchMedia('(max-width: 768px)').matches
}

/** Split courses into an initial batch and a deferred batch on mobile. */
export function splitCoursesForPrefetch<T>(items: readonly T[]): { initial: T[]; deferred: T[] } {
  if (!isMobileViewport() || items.length <= MOBILE_COURSE_PREFETCH_LIMIT) {
    return { initial: [...items], deferred: [] }
  }
  return {
    initial: items.slice(0, MOBILE_COURSE_PREFETCH_LIMIT),
    deferred: items.slice(MOBILE_COURSE_PREFETCH_LIMIT),
  }
}
