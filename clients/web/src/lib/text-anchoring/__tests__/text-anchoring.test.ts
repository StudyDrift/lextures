import { describe, expect, it } from 'vitest'
import {
  buildQuoteAnchor,
  plainPassageFromMarkdown,
  reanchorAnnotations,
  resolveQuoteAnchor,
  segmentPassage,
} from '..'

describe('text-anchoring', () => {
  const passage =
    'The author claims that energy is conserved. However, no evidence is given. The conclusion follows anyway.'

  it('buildQuoteAnchor captures quote and context', () => {
    const start = passage.indexOf('energy is conserved')
    const end = start + 'energy is conserved'.length
    const built = buildQuoteAnchor(passage, start, end, 0)
    expect(built).not.toBeNull()
    expect(built!.quote).toBe('energy is conserved')
    expect(built!.anchor.prefix).toContain('claims that ')
    expect(built!.anchor.suffix).toContain('. However')
    expect(built!.anchor.approxOffset).toBe(start)
    expect(built!.anchor.unitIndex).toBe(0)
  })

  it('resolves by exact offset then by quote after insert-before', () => {
    const start = passage.indexOf('no evidence is given')
    const end = start + 'no evidence is given'.length
    const built = buildQuoteAnchor(passage, start, end)!
    expect(resolveQuoteAnchor(passage, built.quote, built.anchor)).toEqual({ start, end })

    const edited = 'Preface. ' + passage
    const resolved = resolveQuoteAnchor(edited, built.quote, built.anchor)
    expect(resolved).not.toBeNull()
    expect(edited.slice(resolved!.start, resolved!.end)).toBe(built.quote)
  })

  it('returns null (orphan) when quote is deleted', () => {
    const start = passage.indexOf('no evidence is given')
    const end = start + 'no evidence is given'.length
    const built = buildQuoteAnchor(passage, start, end)!
    const edited = passage.replace('no evidence is given', 'support is missing')
    expect(resolveQuoteAnchor(edited, built.quote, built.anchor)).toBeNull()
  })

  it('reanchorAnnotations marks orphans without deleting', () => {
    const start = passage.indexOf('The conclusion follows anyway')
    const end = start + 'The conclusion follows anyway'.length
    const built = buildQuoteAnchor(passage, start, end)!
    const anns = reanchorAnnotations(passage.replace('The conclusion follows anyway', 'x'), [
      { id: '1', quote: built.quote, anchor: built.anchor, orphaned: false },
    ])
    expect(anns).toHaveLength(1)
    expect(anns[0]!.orphaned).toBe(true)
    expect(anns[0]!.quote).toBe(built.quote)
  })

  it('segments sentences', () => {
    const units = segmentPassage(passage, 'sentence')
    expect(units.length).toBeGreaterThanOrEqual(3)
    expect(units[0].text).toContain('energy is conserved')
  })

  it('segments paragraphs and lines', () => {
    const multi = 'First paragraph.\n\nSecond paragraph.'
    expect(segmentPassage(multi, 'paragraph').length).toBe(2)
    expect(segmentPassage('a\nb\nc', 'line').length).toBe(3)
  })

  it('plainPassageFromMarkdown strips common markup', () => {
    expect(plainPassageFromMarkdown('## Hello **world**')).toBe('Hello world')
  })
})
