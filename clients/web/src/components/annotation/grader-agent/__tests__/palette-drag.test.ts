import { describe, expect, it } from 'vitest'
import { beginPaletteDrag, consumePaletteDragType, endPaletteDrag, peekPaletteDragType } from '../palette-drag'

describe('palette drag store', () => {
  it('carries node type from drag start through drop', () => {
    beginPaletteDrag('activity')
    expect(peekPaletteDragType()).toBe('activity')
    expect(consumePaletteDragType()).toBe('activity')
    expect(consumePaletteDragType()).toBeNull()
  })

  it('clears pending drag on end', () => {
    beginPaletteDrag('studentSubmission')
    endPaletteDrag()
    expect(consumePaletteDragType()).toBeNull()
  })

  it('carries AI node type through drag', () => {
    beginPaletteDrag('ai')
    expect(consumePaletteDragType()).toBe('ai')
  })
})
