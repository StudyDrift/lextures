import { describe, expect, it } from 'vitest'
import { parseAndValidateConfigJSON } from '../validate'
import type { JsonSchema } from '../types'

const schema: JsonSchema = {
  type: 'object',
  required: ['title'],
  properties: {
    title: { type: 'string', title: 'Title' },
    count: { type: 'integer' },
  },
}

describe('parseAndValidateConfigJSON', () => {
  it('accepts a valid JSON object with required fields', () => {
    const result = parseAndValidateConfigJSON('{"title":"Hello","count":2}', schema)
    expect(result).toEqual({ ok: true, config: { title: 'Hello', count: 2 } })
  })

  it('rejects empty input', () => {
    const result = parseAndValidateConfigJSON('   ', schema)
    expect(result.ok).toBe(false)
    if (!result.ok) {
      expect(result.errors[0]?.message).toMatch(/empty/i)
    }
  })

  it('rejects invalid JSON', () => {
    const result = parseAndValidateConfigJSON('{title:}', schema)
    expect(result.ok).toBe(false)
    if (!result.ok) {
      expect(result.errors[0]?.message).toMatch(/invalid json/i)
    }
  })

  it('rejects arrays and non-objects', () => {
    expect(parseAndValidateConfigJSON('[]', schema).ok).toBe(false)
    expect(parseAndValidateConfigJSON('"string"', schema).ok).toBe(false)
    expect(parseAndValidateConfigJSON('null', schema).ok).toBe(false)
  })

  it('rejects missing required fields', () => {
    const result = parseAndValidateConfigJSON('{"count":1}', schema)
    expect(result.ok).toBe(false)
    if (!result.ok) {
      expect(result.errors).toEqual(
        expect.arrayContaining([expect.objectContaining({ path: 'title' })]),
      )
    }
  })
})
