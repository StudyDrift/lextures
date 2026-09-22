import { describe, expect, it, vi } from 'vitest'
import {
  definitionLookupTerm,
  fetchWordDefinition,
  parseDictionaryPayload,
} from '../word-definition'

describe('definitionLookupTerm', () => {
  it('uses a single selected word and strips surrounding punctuation', () => {
    expect(definitionLookupTerm('  “Etymology,” ')).toBe('Etymology')
  })

  it('uses the first word of a longer selection', () => {
    expect(definitionLookupTerm('Lord consecrate')).toBe('Lord')
  })

  it('returns null when the selection has no word', () => {
    expect(definitionLookupTerm('— …')).toBeNull()
  })
})

describe('parseDictionaryPayload', () => {
  it('keeps the headword, phonetic, and the first two senses', () => {
    const parsed = parseDictionaryPayload(
      [
        {
          word: 'consecrate',
          phonetic: '/ˈkɒnsɪkreɪt/',
          meanings: [
            {
              partOfSpeech: 'verb',
              definitions: [
                { definition: 'Make or declare sacred.' },
                { definition: 'Dedicate formally to a purpose.' },
              ],
            },
            {
              partOfSpeech: 'adjective',
              definitions: [{ definition: 'Dedicated to a sacred purpose.' }],
            },
          ],
        },
      ],
      'consecrate',
    )
    expect(parsed).toEqual({
      word: 'consecrate',
      phonetic: '/ˈkɒnsɪkreɪt/',
      senses: [
        { partOfSpeech: 'verb', definition: 'Make or declare sacred.' },
        { partOfSpeech: 'verb', definition: 'Dedicate formally to a purpose.' },
      ],
    })
  })

  it('returns null when the payload has no definitions', () => {
    expect(parseDictionaryPayload([{ word: 'x', meanings: [] }], 'x')).toBeNull()
  })
})

describe('fetchWordDefinition', () => {
  it('retries a capitalized word in lowercase after a 404', async () => {
    const fetchImpl = vi.fn(async (input: RequestInfo | URL) => {
      const url = String(input)
      if (url.endsWith('/Lord')) {
        return new Response('{}', { status: 404 })
      }
      return new Response(
        JSON.stringify([
          {
            word: 'lord',
            meanings: [{ partOfSpeech: 'noun', definitions: [{ definition: 'A master or ruler.' }] }],
          },
        ]),
        { status: 200, headers: { 'Content-Type': 'application/json' } },
      )
    })
    const result = await fetchWordDefinition('Lord', fetchImpl as unknown as typeof fetch)
    expect(result.word).toBe('lord')
    expect(result.senses[0]?.definition).toBe('A master or ruler.')
    expect(fetchImpl).toHaveBeenCalledTimes(2)
  })
})
