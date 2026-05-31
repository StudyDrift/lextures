import { describe, expect, it } from 'vitest'
import { computeCourseFinalPercent } from '../compute-course-final-percent'

describe('computeCourseFinalPercent', () => {
  it('returns null when no columns have max points', () => {
    expect(
      computeCourseFinalPercent(
        [{ id: 'a', maxPoints: null, assignmentGroupId: 'g1' }],
        { a: '10' },
        [{ id: 'g1', weightPercent: 100 }],
      ),
    ).toBeNull()
  })

  it('uses straight points when assignment group weights sum to 0', () => {
    const pct = computeCourseFinalPercent(
      [
        { id: 'a', maxPoints: 100, assignmentGroupId: null },
        { id: 'b', maxPoints: 50, assignmentGroupId: null },
      ],
      { a: '80', b: '40' },
      [],
    )
    expect(pct).toBeCloseTo((120 / 150) * 100, 5)
  })

  it('applies a single 100% group', () => {
    const pct = computeCourseFinalPercent(
      [
        { id: 'a', maxPoints: 50, assignmentGroupId: 'hw' },
        { id: 'b', maxPoints: 50, assignmentGroupId: 'hw' },
      ],
      { a: '40', b: '30' },
      [{ id: 'hw', weightPercent: 100 }],
    )
    expect(pct).toBeCloseTo(70, 5)
  })

  it('weights two groups 50/50', () => {
    const pct = computeCourseFinalPercent(
      [
        { id: 'a', maxPoints: 100, assignmentGroupId: 'hw' },
        { id: 'b', maxPoints: 100, assignmentGroupId: 'ex' },
      ],
      { a: '90', b: '70' },
      [
        { id: 'hw', weightPercent: 50 },
        { id: 'ex', weightPercent: 50 },
      ],
    )
    expect(pct).toBeCloseTo(0.5 * 90 + 0.5 * 70, 5)
  })

  it('treats blank cells as 0 earned when past due (missing work)', () => {
    const past = '2000-01-01T00:00:00Z'
    const pct = computeCourseFinalPercent(
      [{ id: 'a', maxPoints: 100, assignmentGroupId: 'g', dueAt: past }],
      { a: '' },
      [{ id: 'g', weightPercent: 100 }],
    )
    expect(pct).toBe(0)
  })

  it('routes unknown group ids to the ungrouped bucket', () => {
    const pct = computeCourseFinalPercent(
      [{ id: 'x', maxPoints: 100, assignmentGroupId: 'not-in-settings' }],
      { x: '80' },
      [{ id: 'hw', weightPercent: 100 }],
    )
    expect(pct).toBeCloseTo(80, 5)
  })

  it('applies drop lowest 1 in a 100% group (plan 3.9)', () => {
    const pct = computeCourseFinalPercent(
      [
        { id: 'a', maxPoints: 100, assignmentGroupId: 'g', neverDrop: false, replaceWithFinal: false },
        { id: 'b', maxPoints: 100, assignmentGroupId: 'g' },
        { id: 'c', maxPoints: 100, assignmentGroupId: 'g' },
        { id: 'd', maxPoints: 100, assignmentGroupId: 'g' },
      ],
      { a: '60', b: '70', c: '80', d: '90' },
      [{ id: 'g', weightPercent: 100, dropLowest: 1, dropHighest: 0, replaceLowestWithFinal: false }],
    )
    expect(pct).toBeCloseTo(80, 5)
  })

  it('excludes future-due assignments with no grade from the final calculation', () => {
    const future = '2099-01-01T00:00:00Z'
    const pct = computeCourseFinalPercent(
      [
        { id: 'past', maxPoints: 100, assignmentGroupId: 'g', dueAt: '2000-01-01T00:00:00Z' },
        { id: 'future', maxPoints: 100, assignmentGroupId: 'g', dueAt: future },
      ],
      { past: '80', future: '' }, // no grade on future
      [{ id: 'g', weightPercent: 100 }],
    )
    // only the past one (80/100) should count; future excluded entirely
    expect(pct).toBeCloseTo(80, 5)
  })

  it('excludes non-due assignments with no grade (no dueAt at all)', () => {
    const pct = computeCourseFinalPercent(
      [{ id: 'nodue', maxPoints: 100, assignmentGroupId: 'g' }],
      { nodue: '' },
      [{ id: 'g', weightPercent: 100 }],
    )
    expect(pct).toBeNull() // nothing qualified for the average
  })

  it('includes a graded assignment even if its due date is in the future', () => {
    const future = '2099-01-01T00:00:00Z'
    const pct = computeCourseFinalPercent(
      [{ id: 'early', maxPoints: 100, assignmentGroupId: 'g', dueAt: future }],
      { early: '95' },
      [{ id: 'g', weightPercent: 100 }],
    )
    expect(pct).toBeCloseTo(95, 5)
  })

  it('respects explicit now parameter for due date decisions', () => {
    const borderline = '2025-06-01T12:00:00Z'
    // now is before due => no grade => exclude
    const before = computeCourseFinalPercent(
      [{ id: 'x', maxPoints: 100, dueAt: borderline }],
      { x: '' },
      [],
      {},
      '2025-05-01T00:00:00Z',
    )
    expect(before).toBeNull()

    // now after due => include as 0
    const after = computeCourseFinalPercent(
      [{ id: 'x', maxPoints: 100, dueAt: borderline }],
      { x: '' },
      [],
      {},
      '2025-07-01T00:00:00Z',
    )
    expect(after).toBe(0)
  })
})
