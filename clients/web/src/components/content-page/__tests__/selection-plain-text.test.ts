import { afterEach, describe, expect, it } from 'vitest'
import { plainTextFromRange } from '../selection-plain-text'

describe('plainTextFromRange', () => {
  afterEach(() => {
    document.body.innerHTML = ''
  })

  it('returns empty string for a collapsed range', () => {
    document.body.innerHTML = '<p>Only one</p>'
    const p = document.querySelector('p')!
    const range = document.createRange()
    range.setStart(p.firstChild!, 0)
    range.collapse(true)
    expect(plainTextFromRange(range)).toBe('')
  })

  it('keeps both paragraphs when serializing a multi-paragraph range', () => {
    document.body.innerHTML = '<div id="root"><p>First paragraph here.</p><p>Second paragraph here.</p></div>'
    const p1 = document.querySelectorAll('p')[0]!
    const p2 = document.querySelectorAll('p')[1]!
    const range = document.createRange()
    range.setStart(p1.firstChild!, 0)
    range.setEnd(p2.firstChild!, p2.firstChild!.textContent!.length)

    const text = plainTextFromRange(range)
    expect(text).toContain('First paragraph here.')
    expect(text).toContain('Second paragraph here.')
    expect(text.indexOf('First')).toBeLessThan(text.indexOf('Second'))
  })

  it('still has full text after DOM remount would shrink Range.toString()', () => {
    document.body.innerHTML = '<div id="root"><p>Alpha block text.</p><p>Beta block text.</p></div>'
    const root = document.getElementById('root')!
    const p1 = root.querySelectorAll('p')[0]!
    const p2 = root.querySelectorAll('p')[1]!
    const range = document.createRange()
    range.setStart(p1.firstChild!, 0)
    range.setEnd(p2.firstChild!, p2.firstChild!.textContent!.length)

    const captured = plainTextFromRange(range)
    expect(captured).toContain('Alpha block text.')
    expect(captured).toContain('Beta block text.')

    // Simulate React replacing paragraph nodes (content page reader re-render).
    const np1 = document.createElement('p')
    np1.textContent = 'Alpha block text.'
    const np2 = document.createElement('p')
    np2.textContent = 'Beta block text.'
    p1.replaceWith(np1)
    p2.replaceWith(np2)

    // Live Range.toString() often collapses to the first block after remount.
    const live = range.toString()
    expect(captured.length).toBeGreaterThanOrEqual(live.length)
    expect(captured).toContain('Beta block text.')
  })
})
