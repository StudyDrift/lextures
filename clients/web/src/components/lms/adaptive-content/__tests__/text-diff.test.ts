import { describe, expect, it } from 'vitest'
import { diffLines } from '../../../../lib/text-diff'

describe('diffLines', () => {
  it('marks added and removed lines', () => {
    const d = diffLines('a\nb\nc', 'a\nx\nc')
    expect(d.some((l) => l.type === 'remove' && l.text === 'b')).toBe(true)
    expect(d.some((l) => l.type === 'add' && l.text === 'x')).toBe(true)
    expect(d.filter((l) => l.type === 'same').map((l) => l.text)).toEqual(['a', 'c'])
  })

  it('handles empty base', () => {
    const d = diffLines('', 'hello')
    expect(d).toEqual([{ type: 'add', text: 'hello' }])
  })
})
