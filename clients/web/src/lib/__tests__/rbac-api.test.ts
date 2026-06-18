import { describe, expect, it } from 'vitest'
import { canCreateCourses, isValidPermissionString, PERM_COURSE_CREATE } from '../rbac-api'

describe('isValidPermissionString', () => {
  it('accepts exactly four colon-separated non-empty parts', () => {
    expect(isValidPermissionString('a:b:c:d')).toBe(true)
    expect(isValidPermissionString('global:app:course:create')).toBe(true)
  })

  it('rejects wrong segment count', () => {
    expect(isValidPermissionString('a:b:c')).toBe(false)
    expect(isValidPermissionString('a:b:c:d:e')).toBe(false)
  })

  it('rejects empty segments', () => {
    expect(isValidPermissionString('a:b::d')).toBe(false)
    expect(isValidPermissionString(' :b:c:d')).toBe(false)
  })

  it('trims whitespace before validating', () => {
    expect(isValidPermissionString('  a:b:c:d  ')).toBe(true)
  })
})

describe('canCreateCourses', () => {
  it('is false while permissions are loading', () => {
    const allows = (p: string) => p === PERM_COURSE_CREATE
    expect(canCreateCourses(allows, true)).toBe(false)
  })

  it('is true when PERM_COURSE_CREATE is granted', () => {
    const allows = (p: string) => p === PERM_COURSE_CREATE
    expect(canCreateCourses(allows, false)).toBe(true)
  })

  it('is false without PERM_COURSE_CREATE', () => {
    expect(canCreateCourses(() => false, false)).toBe(false)
  })
})
