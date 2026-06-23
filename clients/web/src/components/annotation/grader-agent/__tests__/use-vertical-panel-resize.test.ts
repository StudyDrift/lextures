import { describe, expect, it } from 'vitest'
import { clampResizeHeight } from '../use-vertical-panel-resize'

describe('clampResizeHeight', () => {
  it('clamps within bounds', () => {
    expect(clampResizeHeight(50, 96, 480)).toBe(96)
    expect(clampResizeHeight(200, 96, 480)).toBe(200)
    expect(clampResizeHeight(600, 96, 480)).toBe(480)
  })
})