import { describe, expect, it } from 'vitest'
import { labelsNearDuplicate, levenshtein, normaliseLabel } from '../collisions'

describe('collisions helpers', () => {
  it('levenshtein basics', () => {
    expect(levenshtein('kit', 'sit')).toBe(1)
    expect(levenshtein('same', 'same')).toBe(0)
  })

  it('near-duplicate labels', () => {
    expect(labelsNearDuplicate('My credentials', 'My credential')).toBe(true)
    expect(labelsNearDuplicate('Gradebook', 'Modules')).toBe(false)
    expect(normaliseLabel('My Grades')).toBe('grades')
  })
})
