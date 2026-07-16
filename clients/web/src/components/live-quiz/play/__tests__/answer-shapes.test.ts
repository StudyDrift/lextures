import { describe, expect, it } from 'vitest'
import { ANSWER_SHAPES, colorForIndex, shapeForIndex } from '../answer-shape-meta'

describe('answer-shapes', () => {
  it('assigns distinct shapes for the first six options', () => {
    const shapes = [0, 1, 2, 3, 4, 5].map(shapeForIndex)
    expect(new Set(shapes).size).toBe(6)
    expect(shapes).toEqual(ANSWER_SHAPES)
  })

  it('cycles colours for more than six options', () => {
    expect(colorForIndex(0)).toContain('rose')
    expect(colorForIndex(6)).toBe(colorForIndex(0))
  })
})
