import { describe, expect, it } from 'vitest'
import { lineCount } from '../types'

describe('code_sandbox helpers', () => {
  it('counts lines', () => {
    expect(lineCount('')).toBe(1)
    expect(lineCount('a')).toBe(1)
    expect(lineCount('a\nb\nc')).toBe(3)
  })
})
