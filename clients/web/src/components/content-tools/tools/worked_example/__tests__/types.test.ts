import { describe, expect, it } from 'vitest'
import { prefersReducedMotion } from '../types'

describe('worked_example types', () => {
  it('exposes prefersReducedMotion helper', () => {
    expect(typeof prefersReducedMotion()).toBe('boolean')
  })
})
