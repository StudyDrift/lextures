import { describe, expect, it } from 'vitest'
import { chunkSentences, extractReadableContent } from './extractReadableContent'

describe('chunkSentences', () => {
  it('splits on sentence boundaries', () => {
    expect(chunkSentences('Hello world. How are you? Fine!')).toEqual([
      'Hello world.',
      'How are you?',
      'Fine!',
    ])
  })

  it('returns empty for blank input', () => {
    expect(chunkSentences('   ')).toEqual([])
  })
})

describe('extractReadableContent', () => {
  it('reads paragraphs inside data-content-reader', () => {
    document.body.innerHTML = `
      <main><p>Shell chrome</p></main>
      <div data-content-reader>
        <p>First sentence here. Second sentence follows.</p>
      </div>
    `
    const sentences = extractReadableContent(document)
    expect(sentences.length).toBeGreaterThanOrEqual(+2)
    expect(sentences.some((s) => s.text.includes('First sentence'))).toBe(true)
    expect(sentences.every((s) => s.element.closest('[data-content-reader]'))).toBe(true)
  })

  it('announces code blocks without reading source', () => {
    document.body.innerHTML = `
      <article data-content-reader>
        <pre>secret code</pre>
      </article>
    `
    const sentences = extractReadableContent(document)
    expect(sentences.some((s) => s.text === 'Code block.')).toBe(true)
    expect(sentences.some((s) => s.text.includes('secret code'))).toBe(false)
  })
})
