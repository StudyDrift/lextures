import { describe, expect, it, vi } from 'vitest'
import { MOBILE_COURSE_PREFETCH_LIMIT, splitCoursesForPrefetch } from './dashboard-course-prefetch'

describe('splitCoursesForPrefetch', () => {
  it('returns all items on desktop viewports', () => {
    vi.stubGlobal('matchMedia', () => ({ matches: false }))
    const items = [1, 2, 3, 4, 5]
    expect(splitCoursesForPrefetch(items)).toEqual({ initial: items, deferred: [] })
  })

  it('splits after the mobile prefetch limit on narrow viewports', () => {
    vi.stubGlobal('matchMedia', () => ({ matches: true }))
    const items = Array.from({ length: MOBILE_COURSE_PREFETCH_LIMIT + 2 }, (_, i) => i)
    const { initial, deferred } = splitCoursesForPrefetch(items)
    expect(initial).toHaveLength(MOBILE_COURSE_PREFETCH_LIMIT)
    expect(deferred).toHaveLength(2)
  })
})
