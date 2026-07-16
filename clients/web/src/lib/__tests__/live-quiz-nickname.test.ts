import { describe, expect, it } from 'vitest'
import { validateNickname } from '../live-quiz-nickname'

describe('validateNickname', () => {
  it('accepts trimmed valid nicknames', () => {
    expect(validateNickname('  Ada  ')).toEqual({ ok: true, nickname: 'Ada' })
  })

  it('rejects empty, long, and illegal charset', () => {
    expect(validateNickname('')).toEqual({ ok: false, reason: 'empty' })
    expect(validateNickname('x'.repeat(25))).toEqual({ ok: false, reason: 'too_long' })
    expect(validateNickname('bad@name')).toEqual({ ok: false, reason: 'charset' })
  })
})
