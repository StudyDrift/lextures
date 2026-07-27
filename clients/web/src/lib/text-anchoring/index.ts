/**
 * Quote-plus-context text anchoring for Content Tools annotation surfaces (CT.13).
 *
 * Stores `{ quote, prefix, suffix, approxOffset, unitIndex? }` and re-resolves against
 * the live passage. When the quote cannot be found, the annotation is marked orphaned
 * rather than deleted. Reusable by future annotation tools.
 */

export type QuoteAnchor = {
  prefix: string
  suffix: string
  approxOffset: number
  unitIndex?: number
}

export type ResolvedRange = {
  start: number
  end: number
}

const CONTEXT_LEN = 32

/** Characters of surrounding context captured on each side. */
export function buildQuoteAnchor(
  passage: string,
  start: number,
  end: number,
  unitIndex?: number,
): { quote: string; anchor: QuoteAnchor } | null {
  if (start < 0 || end <= start || end > passage.length) return null
  const quote = passage.slice(start, end)
  if (!quote.trim()) return null
  const anchor: QuoteAnchor = {
    prefix: passage.slice(Math.max(0, start - CONTEXT_LEN), start),
    suffix: passage.slice(end, end + CONTEXT_LEN),
    approxOffset: start,
  }
  if (unitIndex != null && Number.isFinite(unitIndex) && unitIndex >= 0) {
    anchor.unitIndex = unitIndex
  }
  return { quote, anchor }
}

function contextScore(full: string, idx: number, quote: string, anchor: QuoteAnchor): number {
  let score = 0
  if (anchor.prefix) {
    const before = full.slice(Math.max(0, idx - anchor.prefix.length), idx)
    if (before.endsWith(anchor.prefix)) score += 2
    else if (before && anchor.prefix.endsWith(before.slice(-Math.min(8, before.length)))) score += 1
  }
  if (anchor.suffix) {
    const after = full.slice(idx + quote.length, idx + quote.length + anchor.suffix.length)
    if (after.startsWith(anchor.suffix)) score += 2
    else if (after && anchor.suffix.startsWith(after.slice(0, Math.min(8, after.length)))) score += 1
  }
  score -= Math.abs(idx - anchor.approxOffset) / 1e6
  return score
}

function bestQuoteIndex(full: string, quote: string, anchor: QuoteAnchor): number {
  if (!quote) return -1
  let best = -1
  let bestScore = -Infinity
  let from = full.indexOf(quote)
  while (from !== -1) {
    const score = contextScore(full, from, quote, anchor)
    if (score > bestScore) {
      bestScore = score
      best = from
    }
    from = full.indexOf(quote, from + 1)
  }
  return best
}

/**
 * Resolve a stored quote+anchor against the current passage text.
 * Returns null when the quote can no longer be located (orphaned).
 */
export function resolveQuoteAnchor(
  passage: string,
  quote: string,
  anchor: QuoteAnchor,
): ResolvedRange | null {
  if (!quote) return null

  // Exact: approxOffset still points at the same text.
  const end = anchor.approxOffset + quote.length
  if (
    anchor.approxOffset >= 0 &&
    end <= passage.length &&
    passage.slice(anchor.approxOffset, end) === quote
  ) {
    return { start: anchor.approxOffset, end }
  }

  const idx = bestQuoteIndex(passage, quote, anchor)
  if (idx >= 0) {
    return { start: idx, end: idx + quote.length }
  }
  return null
}

export type UnitGranularity = 'sentence' | 'paragraph' | 'line'

export type PassageUnit = {
  index: number
  text: string
  start: number
  end: number
}

/** Locale-aware-ish segmentation; sentence split uses Unicode sentence terminators. */
export function segmentPassage(
  passage: string,
  granularity: UnitGranularity = 'sentence',
): PassageUnit[] {
  const text = passage ?? ''
  if (!text) return []

  switch (granularity) {
    case 'paragraph': {
      const units: PassageUnit[] = []
      const re = /[^\n]+/g
      let m: RegExpExecArray | null
      let idx = 0
      while ((m = re.exec(text)) !== null) {
        const chunk = m[0]
        if (!chunk.trim()) continue
        units.push({ index: idx++, text: chunk, start: m.index, end: m.index + chunk.length })
      }
      return units.length ? units : [{ index: 0, text, start: 0, end: text.length }]
    }
    case 'line': {
      const units: PassageUnit[] = []
      let start = 0
      let idx = 0
      for (let i = 0; i <= text.length; i++) {
        if (i === text.length || text[i] === '\n') {
          const chunk = text.slice(start, i)
          if (chunk.trim()) {
            units.push({ index: idx++, text: chunk, start, end: i })
          }
          start = i + 1
        }
      }
      return units.length ? units : [{ index: 0, text, start: 0, end: text.length }]
    }
    case 'sentence': {
      const units: PassageUnit[] = []
      // Split on sentence terminators followed by whitespace or end; keep delimiter with unit.
      const re = /[^.!?…。！？]+(?:[.!?…。！？]+|$)/gu
      let m: RegExpExecArray | null
      let idx = 0
      while ((m = re.exec(text)) !== null) {
        const raw = m[0]
        const leading = raw.match(/^\s*/)?.[0].length ?? 0
        const trailing = raw.match(/\s*$/)?.[0].length ?? 0
        const start = m.index + leading
        const end = m.index + raw.length - trailing
        if (end <= start) continue
        const chunk = text.slice(start, end)
        if (!chunk.trim()) continue
        units.push({ index: idx++, text: chunk, start, end })
      }
      return units.length ? units : [{ index: 0, text, start: 0, end: text.length }]
    }
    default: {
      const _exhaustive: never = granularity
      return _exhaustive
    }
  }
}

export type AnchoredAnnotation = {
  id: string
  quote: string
  anchor: QuoteAnchor
  orphaned?: boolean
}

/** Re-resolve annotations; mark orphaned when the quote cannot be found. Never deletes. */
export function reanchorAnnotations<T extends AnchoredAnnotation>(
  passage: string,
  annotations: T[],
): T[] {
  return annotations.map((a) => {
    const resolved = resolveQuoteAnchor(passage, a.quote, a.anchor)
    if (resolved) {
      return {
        ...a,
        orphaned: false,
        anchor: {
          ...a.anchor,
          approxOffset: resolved.start,
        },
      }
    }
    return { ...a, orphaned: true }
  })
}

/** Strip markdown-ish syntax to plain passage text for anchoring/segmentation. */
export function plainPassageFromMarkdown(md: string): string {
  return (md ?? '')
    .replace(/```[\s\S]*?```/g, ' ')
    .replace(/`[^`]+`/g, (m) => m.slice(1, -1))
    .replace(/!\[[^\]]*]\([^)]*\)/g, ' ')
    .replace(/\[([^\]]*)]\([^)]*\)/g, '$1')
    .replace(/^#{1,6}\s+/gm, '')
    .replace(/[*_~]+/g, '')
    .replace(/\r\n/g, '\n')
    .trim()
}
