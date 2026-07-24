import { describe, expect, it } from 'vitest'
import { isSectionHeadingEnterToContentKey } from '../section-heading-enter'

describe('isSectionHeadingEnterToContentKey', () => {
  it('returns true for plain Enter', () => {
    expect(
      isSectionHeadingEnterToContentKey({ key: 'Enter', shiftKey: false, isComposing: false }),
    ).toBe(true)
  })

  it('returns false for Shift+Enter', () => {
    expect(
      isSectionHeadingEnterToContentKey({ key: 'Enter', shiftKey: true, isComposing: false }),
    ).toBe(false)
  })

  it('returns false while IME is composing', () => {
    expect(
      isSectionHeadingEnterToContentKey({ key: 'Enter', shiftKey: false, isComposing: true }),
    ).toBe(false)
    expect(
      isSectionHeadingEnterToContentKey({
        key: 'Enter',
        shiftKey: false,
        nativeEvent: { isComposing: true },
      }),
    ).toBe(false)
  })

  it('returns false for other keys', () => {
    expect(
      isSectionHeadingEnterToContentKey({ key: 'Tab', shiftKey: false, isComposing: false }),
    ).toBe(false)
  })
})
