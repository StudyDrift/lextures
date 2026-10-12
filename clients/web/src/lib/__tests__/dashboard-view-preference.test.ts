import { beforeEach, describe, expect, it } from 'vitest'
import {
  formatTimeSpent,
  getCachedHasParentChildren,
  setCachedHasParentChildren,
  getDashboardViewPreference,
  setDashboardViewPreference,
  shouldShowParentHome,
} from '../dashboard-view-preference'

describe('dashboard view preference', () => {
  beforeEach(() => localStorage.clear())

  it('remembers the chosen view', () => {
    expect(getDashboardViewPreference()).toBeNull()
    setDashboardViewPreference('teacher')
    expect(getDashboardViewPreference()).toBe('teacher')
    setDashboardViewPreference('parent')
    expect(getDashboardViewPreference()).toBe('parent')
  })

  it('defaults parents with children to Parent home unless they chose teacher', () => {
    expect(shouldShowParentHome(true, null)).toBe(true)
    expect(shouldShowParentHome(true, 'parent')).toBe(true)
    expect(shouldShowParentHome(true, 'teacher')).toBe(false)
    expect(shouldShowParentHome(false, 'parent')).toBe(false)
  })

  it('formats time spent', () => {
    expect(formatTimeSpent(0)).toBe('0m')
    expect(formatTimeSpent(59)).toBe('0m')
    expect(formatTimeSpent(720)).toBe('12m')
    expect(formatTimeSpent(3900)).toBe('1h 05m')
  })

  it('caches whether the account has children', () => {
    expect(getCachedHasParentChildren()).toBe(false)
    setCachedHasParentChildren(true)
    expect(getCachedHasParentChildren()).toBe(true)
    setCachedHasParentChildren(false)
    expect(getCachedHasParentChildren()).toBe(false)
  })
})
