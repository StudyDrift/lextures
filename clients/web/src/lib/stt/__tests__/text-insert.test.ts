import { describe, expect, it } from 'vitest'
import { commitTextAtSelection, insertTextAtSelection } from '../text-insert'

describe('insertTextAtSelection', () => {
  it('inserts text at the caret', () => {
    const el = document.createElement('textarea')
    el.value = 'Hello '
    el.selectionStart = 6
    el.selectionEnd = 6
    const result = insertTextAtSelection(el, 'world', null, 0)
    expect(result.value).toBe('Hello world')
    expect(result.caret).toBe(11)
  })

  it('replaces interim text on subsequent interim updates', () => {
    const el = document.createElement('textarea')
    el.value = 'Start '
    el.selectionStart = 6
    el.selectionEnd = 6
    const first = insertTextAtSelection(el, 'partial', null, 0)
    el.value = first.value
    el.selectionStart = first.caret
    el.selectionEnd = first.caret
    const second = insertTextAtSelection(el, 'done', first.interimStart, first.interimLength)
    expect(second.value).toBe('Start done')
  })
})

describe('commitTextAtSelection', () => {
  it('commits final text replacing interim segment', () => {
    const el = document.createElement('textarea')
    el.value = 'Hello partial there'
    const result = commitTextAtSelection(el, 'Hello world', 0, 13)
    expect(result.value).toBe('Hello world there')
  })
})
