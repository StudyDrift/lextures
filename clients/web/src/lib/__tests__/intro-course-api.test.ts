import { describe, expect, it } from 'vitest'
import {
  introCourseCardState,
  shouldShowIntroCelebration,
  shouldShowIntroWelcomeBanner,
  type IntroCourseProgress,
} from '../intro-course-api'

const base: IntroCourseProgress = {
  enrolled: true,
  courseCode: 'C-WLCOME',
  modulesComplete: 0,
  modulesTotal: 7,
  percent: 0,
}

describe('intro-course-api helpers', () => {
  it('maps card states for enrolled learners', () => {
    expect(introCourseCardState(null, true, false)).toBe('loading')
    expect(introCourseCardState(null, false, true)).toBe('error')
    expect(introCourseCardState({ ...base, enrolled: false }, false, false)).toBe('hidden')
    expect(introCourseCardState(base, false, false)).toBe('not-started')
    expect(
      introCourseCardState({ ...base, modulesComplete: 2, percent: 28 }, false, false),
    ).toBe('in-progress')
    expect(
      introCourseCardState(
        { ...base, modulesComplete: 7, percent: 100, completedAt: '2026-01-01T00:00:00Z' },
        false,
        false,
      ),
    ).toBe('completed')
  })

  it('gates welcome banner until started or dismissed', () => {
    expect(shouldShowIntroWelcomeBanner(null)).toBe(false)
    expect(shouldShowIntroWelcomeBanner(base)).toBe(true)
    expect(shouldShowIntroWelcomeBanner({ ...base, welcomeBannerDismissed: true })).toBe(false)
    expect(shouldShowIntroWelcomeBanner({ ...base, modulesComplete: 1, percent: 14 })).toBe(false)
    expect(
      shouldShowIntroWelcomeBanner({
        ...base,
        completedAt: '2026-01-01T00:00:00Z',
      }),
    ).toBe(false)
  })

  it('shows celebration once when completed and not yet seen', () => {
    expect(shouldShowIntroCelebration(null)).toBe(false)
    expect(shouldShowIntroCelebration(base)).toBe(false)
    expect(
      shouldShowIntroCelebration({
        ...base,
        completedAt: '2026-01-01T00:00:00Z',
        celebrationSeen: false,
      }),
    ).toBe(true)
    expect(
      shouldShowIntroCelebration({
        ...base,
        completedAt: '2026-01-01T00:00:00Z',
        celebrationSeen: true,
      }),
    ).toBe(false)
  })
})