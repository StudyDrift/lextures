import { describe, expect, it } from 'vitest'
import { mathSymbolCount, MATH_SYMBOL_CATEGORY_ORDER, MATH_SYMBOL_PALETTE } from '../math-symbol-palette'

describe('math-symbol-palette', () => {
  it('exposes at least 50 STEM symbols across categories', () => {
    expect(mathSymbolCount()).toBeGreaterThanOrEqual(50)
  })

  it('includes Greek theta for palette AC-4', () => {
    const greek = MATH_SYMBOL_PALETTE.greek
    expect(greek.some((s) => s.latex === '\\theta' && s.label === 'θ')).toBe(true)
  })

  it('covers all documented categories', () => {
    expect(MATH_SYMBOL_CATEGORY_ORDER).toEqual(['general', 'greek', 'calculus', 'logic', 'chemistry'])
  })
})
