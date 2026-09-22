export type WordSense = {
  partOfSpeech: string | null
  definition: string
}

export type WordDefinition = {
  word: string
  phonetic: string | null
  senses: WordSense[]
}

/** First dictionary-friendly word in a selection. */
export function definitionLookupTerm(text: string): string | null {
  const tokens = text.trim().split(/\s+/)
  for (const token of tokens) {
    const word = token.replace(/^[^\p{L}\p{N}]+|[^\p{L}\p{N}'’-]+$/gu, '')
    if (word.length >= 2 && /[\p{L}]/u.test(word)) return word
  }
  return null
}

type DictionaryPayload = {
  word?: unknown
  phonetic?: unknown
  phonetics?: { text?: unknown }[]
  meanings?: {
    partOfSpeech?: unknown
    definitions?: { definition?: unknown }[]
  }[]
}

function asString(value: unknown): string | null {
  return typeof value === 'string' && value.trim() ? value.trim() : null
}

/** Pull a short definition from a dictionaryapi.dev entry list. */
export function parseDictionaryPayload(payload: unknown, fallbackWord: string): WordDefinition | null {
  if (!Array.isArray(payload) || payload.length === 0) return null
  const entry = payload[0] as DictionaryPayload
  const senses: WordSense[] = []
  for (const meaning of entry.meanings ?? []) {
    const partOfSpeech = asString(meaning.partOfSpeech)
    for (const def of meaning.definitions ?? []) {
      const definition = asString(def.definition)
      if (!definition) continue
      senses.push({ partOfSpeech, definition })
      if (senses.length >= 2) break
    }
    if (senses.length >= 2) break
  }
  if (senses.length === 0) return null
  const phonetic =
    asString(entry.phonetic) ??
    (entry.phonetics ?? []).map((p) => asString(p.text)).find((t) => t != null) ??
    null
  return {
    word: asString(entry.word) ?? fallbackWord,
    phonetic,
    senses,
  }
}

async function requestEntry(
  term: string,
  fetchImpl: typeof fetch,
): Promise<WordDefinition | null> {
  const res = await fetchImpl(
    `https://api.dictionaryapi.dev/api/v2/entries/en/${encodeURIComponent(term)}`,
  )
  if (res.status === 404) return null
  if (!res.ok) throw new Error('Could not look up that word.')
  return parseDictionaryPayload(await res.json(), term)
}

export async function fetchWordDefinition(
  term: string,
  fetchImpl: typeof fetch = fetch,
): Promise<WordDefinition> {
  const direct = await requestEntry(term, fetchImpl)
  if (direct) return direct
  const lower = term.toLocaleLowerCase()
  if (lower !== term) {
    const folded = await requestEntry(lower, fetchImpl)
    if (folded) return folded
  }
  throw new Error(`No definition found for “${term}”.`)
}
