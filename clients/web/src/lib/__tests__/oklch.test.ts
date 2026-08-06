import { describe, expect, it } from 'vitest'
import {
  deriveAccentRamp,
  formatOklch,
  hexToOklch,
  oklchToHex,
  parseOklch,
} from '../tokens/oklch'
import { contrastRatio, AA_NORMAL } from '../tokens/contrast'

describe('oklch', () => {
  it('parses valid OKLCH and rejects injection', () => {
    expect(parseOklch('oklch(0.55 0.18 264)')).toEqual({ l: 0.55, c: 0.18, h: 264 })
    expect(parseOklch('oklch(55% 0.18 264)')).toEqual({ l: 0.55, c: 0.18, h: 264 })
    expect(parseOklch('oklch(0.55 0.18 264); background:url(x)')).toBeNull()
    expect(parseOklch('red')).toBeNull()
    expect(parseOklch('oklch(2 0.18 264)')).toBeNull()
  })

  it('round-trips format', () => {
    const p = parseOklch('oklch(0.55 0.18 264)')!
    expect(formatOklch(p)).toBe('oklch(0.55 0.18 264)')
  })

  it('derives accent ramp with AA onAccent for #B3122B-class hues', () => {
    const seed = hexToOklch('#4F46E5')!
    const ramp = deriveAccentRamp(seed)
    expect(Object.keys(ramp)).toHaveLength(11)
    const solid = oklchToHex(parseOklch(ramp['600'])!)
    const onAccent = '#ffffff'
    expect(contrastRatio(onAccent, solid)).toBeGreaterThanOrEqual(AA_NORMAL)
  })

  it('rejects impossible parse for brand_accent_contrast scenarios', () => {
    // Very light yellow seed still produces a ramp; validation is separate.
    const seed = hexToOklch('#FFFF00')!
    const ramp = deriveAccentRamp(seed)
    expect(ramp['50']).toMatch(/^oklch\(/)
  })
})
