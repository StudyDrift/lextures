import { describe, expect, it } from 'vitest'
import { clampResizeWidth } from '../use-horizontal-panel-resize'

describe('clampResizeWidth', () => {
  it('clamps width between min and max', () => {
    expect(clampResizeWidth(200, 240, 480)).toBe(240)
    expect(clampResizeWidth(320, 240, 480)).toBe(320)
    expect(clampResizeWidth(520, 240, 480)).toBe(480)
  })
})