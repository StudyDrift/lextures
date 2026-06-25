import { describe, expect, it } from 'vitest'
import { getTextareaCaretCoordinates, resolveTextareaPickerPosition } from '../workflow-prompt-caret-position'

describe('workflow-prompt-caret-position', () => {
  it('anchors the picker below the caret with a horizontal offset', () => {
    const textarea = document.createElement('textarea')
    textarea.value = 'hello'
    textarea.style.width = '320px'
    textarea.style.padding = '8px'
    document.body.append(textarea)

    const coords = getTextareaCaretCoordinates(textarea, 5)
    const picker = resolveTextareaPickerPosition(textarea, 5)

    expect(picker.top).toBeGreaterThan(coords.top)
    expect(picker.left).toBeGreaterThanOrEqual(0)
    expect(picker.maxWidth).toBeGreaterThanOrEqual(192)

    textarea.remove()
  })
})