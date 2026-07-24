/**
 * E2E.1 — unit-level validation of the course feature matrix metadata.
 * Does not require authenticated fixtures (stack may still be up via e2e-local).
 */
import { test, expect } from '@playwright/test'
import {
  COURSE_FEATURE_MATRIX,
  UI_COURSE_FEATURE_ENTRIES,
  validateCourseFeatureMatrix,
  uiEntriesForShard,
} from '../lib/course-feature-matrix.js'

test.describe('Course feature matrix metadata', () => {
  test('matrix keys and labels are unique with required fields', () => {
    const errors = validateCourseFeatureMatrix()
    expect(errors, errors.join('; ')).toEqual([])
  })

  test('UI shards partition all 25 settings rows without overlap', () => {
    const a = uiEntriesForShard('a')
    const b = uiEntriesForShard('b')
    const c = uiEntriesForShard('c')
    expect(a.length + b.length + c.length).toBe(UI_COURSE_FEATURE_ENTRIES.length)
    expect(a.length).toBeGreaterThan(0)
    expect(b.length).toBeGreaterThan(0)
    expect(c.length).toBeGreaterThan(0)

    const keys = [...a, ...b, ...c].map((e) => e.key)
    expect(new Set(keys).size).toBe(keys.length)
  })

  test('groupSpacesEnabled remains API-only until a settings row exists', () => {
    const group = COURSE_FEATURE_MATRIX.find((e) => e.key === 'groupSpacesEnabled')
    expect(group?.uiLabel).toBeNull()
    expect(group?.uiShard).toBeNull()
    expect(group?.nav?.route).toBe('groups')
  })
})
