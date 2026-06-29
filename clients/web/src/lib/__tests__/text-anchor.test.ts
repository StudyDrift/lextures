import { describe, expect, it, beforeEach, afterEach } from 'vitest'
import { findAnchorRange, getSelectionAnchor, parseTextAnchor, type TextAnchor } from '../text-anchor'

function mount(html: string): HTMLElement {
  const el = document.createElement('div')
  el.innerHTML = html
  document.body.appendChild(el)
  return el
}

describe('text-anchor', () => {
  let container: HTMLElement

  afterEach(() => {
    container?.remove()
    window.getSelection()?.removeAllRanges()
  })

  describe('getSelectionAnchor', () => {
    beforeEach(() => {
      container = mount('<p>The quick brown fox jumps over the lazy dog.</p>')
    })

    it('captures offsets, quote, and context from a selection', () => {
      const textNode = container.querySelector('p')!.firstChild as Text
      const range = document.createRange()
      range.setStart(textNode, 4) // "quick"
      range.setEnd(textNode, 15) // "quick brown"
      const sel = window.getSelection()!
      sel.removeAllRanges()
      sel.addRange(range)

      const anchor = getSelectionAnchor(container)
      expect(anchor).not.toBeNull()
      expect(anchor!.start).toBe(4)
      expect(anchor!.end).toBe(15)
      expect(anchor!.quote).toBe('quick brown')
      expect(anchor!.prefix).toBe('The ')
      expect(anchor!.suffix).toBe(' fox jumps over the lazy dog.')
    })

    it('returns null for a collapsed selection', () => {
      const textNode = container.querySelector('p')!.firstChild as Text
      const range = document.createRange()
      range.setStart(textNode, 4)
      range.setEnd(textNode, 4)
      const sel = window.getSelection()!
      sel.removeAllRanges()
      sel.addRange(range)
      expect(getSelectionAnchor(container)).toBeNull()
    })

    it('returns null when the selection is outside the container', () => {
      const outside = mount('<p>elsewhere</p>')
      const textNode = outside.querySelector('p')!.firstChild as Text
      const range = document.createRange()
      range.setStart(textNode, 0)
      range.setEnd(textNode, 4)
      const sel = window.getSelection()!
      sel.removeAllRanges()
      sel.addRange(range)
      expect(getSelectionAnchor(container)).toBeNull()
      outside.remove()
    })
  })

  describe('findAnchorRange', () => {
    it('resolves an anchor by exact offsets across multiple nodes', () => {
      container = mount('<p>Hello <strong>brave</strong> new world</p>')
      // Full text: "Hello brave new world"; "brave new" = [6, 15)
      const anchor: TextAnchor = { start: 6, end: 15, quote: 'brave new', prefix: 'Hello ', suffix: ' world' }
      const range = findAnchorRange(container, anchor)
      expect(range).not.toBeNull()
      expect(range!.toString()).toBe('brave new')
    })

    it('re-anchors by quote when offsets have drifted', () => {
      // Stored offsets assume earlier text that no longer exists; quote search recovers it.
      container = mount('<p>Intro paragraph added later. The target phrase is here.</p>')
      const anchor: TextAnchor = {
        start: 0,
        end: 12,
        quote: 'target phrase',
        prefix: 'The ',
        suffix: ' is here',
      }
      const range = findAnchorRange(container, anchor)
      expect(range).not.toBeNull()
      expect(range!.toString()).toBe('target phrase')
    })

    it('disambiguates repeated quotes using context', () => {
      container = mount('<p>set value then set value then done</p>')
      // "set value" appears twice; suffix " then done" points at the second one.
      const second = container.textContent!.lastIndexOf('set value')
      const anchor: TextAnchor = {
        start: second,
        end: second + 9,
        quote: 'set value',
        prefix: 'then ',
        suffix: ' then done',
      }
      const range = findAnchorRange(container, anchor)
      expect(range).not.toBeNull()
      expect(range!.startOffset).toBeGreaterThan(0)
      // Resolves to the occurrence followed by " then done".
      const after = container.textContent!.slice(range!.endOffset === undefined ? 0 : second + 9)
      expect(after.startsWith(' then done')).toBe(true)
    })

    it('returns null when the passage is gone', () => {
      container = mount('<p>completely different content</p>')
      const anchor: TextAnchor = { start: 0, end: 5, quote: 'zzzzz', prefix: '', suffix: '' }
      expect(findAnchorRange(container, anchor)).toBeNull()
    })
  })

  describe('parseTextAnchor', () => {
    it('parses a valid coords object', () => {
      expect(parseTextAnchor({ start: 1, end: 5, quote: 'q', prefix: 'p', suffix: 's' })).toEqual({
        start: 1,
        end: 5,
        quote: 'q',
        prefix: 'p',
        suffix: 's',
      })
    })

    it('rejects non-anchor coords', () => {
      expect(parseTextAnchor({ rects: [] })).toBeNull()
      expect(parseTextAnchor(null)).toBeNull()
      expect(parseTextAnchor({ x: 1, y: 2 })).toBeNull()
    })
  })
})
