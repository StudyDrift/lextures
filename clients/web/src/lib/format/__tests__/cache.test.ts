import { describe, expect, it } from 'vitest'
import { FormatterCache } from '../cache'

describe('FormatterCache', () => {
  it('returns the same instance for repeated keys', () => {
    const cache = new FormatterCache<object>()
    let built = 0
    const a = cache.get('de|UTC|{"dateStyle":"medium"}', () => {
      built++
      return { id: built }
    })
    const b = cache.get('de|UTC|{"dateStyle":"medium"}', () => {
      built++
      return { id: built }
    })
    expect(a).toBe(b)
    expect(built).toBe(1)
  })

  it('evicts oldest entry after max size', () => {
    const cache = new FormatterCache<string>()
    let n = 0
    for (let i = 0; i < 25; i++) {
      cache.get(`k${i}`, () => `v${n++}`)
    }
    expect(cache.get('k0', () => 'new')).toBe('new')
    expect(cache.get('k24', () => 'missing')).toBe('v24')
  })
})
