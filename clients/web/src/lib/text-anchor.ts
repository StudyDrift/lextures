// Text-anchor annotations for reflowable previews (DOCX/PPTX/XLSX/Markdown/text/code).
//
// Pixel-rect overlays don't survive reflow, so for these formats a highlight is stored as a
// character range into the container's text content — `{ start, end }` — plus the quoted text
// and a little surrounding context (`prefix`/`suffix`). On read-back we resolve the range from
// the offsets first, and fall back to searching `quote` (disambiguated by context) when the
// offsets have drifted (e.g. the renderer changed slightly). This mirrors the W3C
// TextQuoteSelector / TextPositionSelector approach used by Hypothesis.

export type TextAnchor = {
  start: number
  end: number
  quote: string
  prefix: string
  suffix: string
}

/** Characters of surrounding context captured on each side for re-anchoring. */
const CONTEXT_LEN = 32

/** Concatenated text content of a container, matching the units used for offsets. */
function containerText(container: HTMLElement): string {
  const r = container.ownerDocument.createRange()
  r.selectNodeContents(container)
  return r.toString()
}

/** Character offset of a (node, nodeOffset) boundary within the container's text content. */
function offsetWithin(container: HTMLElement, node: Node, nodeOffset: number): number {
  const r = container.ownerDocument.createRange()
  r.selectNodeContents(container)
  try {
    r.setEnd(node, nodeOffset)
  } catch {
    return containerText(container).length
  }
  return r.toString().length
}

/**
 * Build a {@link TextAnchor} from the current selection, if it lies inside `container`.
 * Returns null when there is no usable (non-collapsed, in-container) selection.
 */
export function getSelectionAnchor(container: HTMLElement): TextAnchor | null {
  const sel = container.ownerDocument.defaultView?.getSelection()
  if (!sel || sel.rangeCount === 0 || sel.isCollapsed) return null
  const range = sel.getRangeAt(0)
  if (!container.contains(range.startContainer) || !container.contains(range.endContainer)) {
    return null
  }
  const quote = range.toString()
  if (!quote.trim()) return null

  const start = offsetWithin(container, range.startContainer, range.startOffset)
  const end = offsetWithin(container, range.endContainer, range.endOffset)
  if (end <= start) return null

  const full = containerText(container)
  return {
    start,
    end,
    quote,
    prefix: full.slice(Math.max(0, start - CONTEXT_LEN), start),
    suffix: full.slice(end, end + CONTEXT_LEN),
  }
}

/** Map a [from, to) character range onto a DOM Range by walking the container's text nodes. */
function rangeFromOffsets(container: HTMLElement, from: number, to: number): Range | null {
  if (from < 0 || to < from) return null
  const walker = container.ownerDocument.createTreeWalker(container, NodeFilter.SHOW_TEXT)
  let acc = 0
  let startNode: Node | null = null
  let startOff = 0
  let endNode: Node | null = null
  let endOff = 0
  for (let n = walker.nextNode(); n; n = walker.nextNode()) {
    const len = n.nodeValue?.length ?? 0
    if (startNode === null && from <= acc + len) {
      startNode = n
      startOff = from - acc
    }
    if (startNode !== null && to <= acc + len) {
      endNode = n
      endOff = to - acc
      break
    }
    acc += len
  }
  if (startNode === null || endNode === null) return null
  const range = container.ownerDocument.createRange()
  try {
    range.setStart(startNode, startOff)
    range.setEnd(endNode, endOff)
  } catch {
    return null
  }
  return range
}

/** Score a candidate quote occurrence by how well its surroundings match the stored context. */
function contextScore(full: string, idx: number, anchor: TextAnchor): number {
  let score = 0
  if (anchor.prefix) {
    const before = full.slice(Math.max(0, idx - anchor.prefix.length), idx)
    if (before.endsWith(anchor.prefix)) score += 2
    else if (before && anchor.prefix.endsWith(before.slice(-Math.min(8, before.length)))) score += 1
  }
  if (anchor.suffix) {
    const after = full.slice(idx + anchor.quote.length, idx + anchor.quote.length + anchor.suffix.length)
    if (after.startsWith(anchor.suffix)) score += 2
    else if (after && anchor.suffix.startsWith(after.slice(0, Math.min(8, after.length)))) score += 1
  }
  return score
}

/** Best occurrence of `quote`, preferring matching context, then proximity to the stored start. */
function bestQuoteIndex(full: string, anchor: TextAnchor): number {
  if (!anchor.quote) return -1
  let best = -1
  let bestScore = -Infinity
  let from = full.indexOf(anchor.quote)
  while (from !== -1) {
    const score = contextScore(full, from, anchor) - Math.abs(from - anchor.start) / 1e6
    if (score > bestScore) {
      bestScore = score
      best = from
    }
    from = full.indexOf(anchor.quote, from + 1)
  }
  return best
}

/**
 * Resolve a stored {@link TextAnchor} back to a live DOM Range inside `container`.
 * Tries exact offsets, then a context-disambiguated quote search, then the raw offsets.
 * Returns null when the passage can no longer be located.
 */
export function findAnchorRange(container: HTMLElement, anchor: TextAnchor): Range | null {
  const full = containerText(container)

  // Exact: offsets still point at the same text.
  if (anchor.end <= full.length && full.slice(anchor.start, anchor.end) === anchor.quote) {
    const r = rangeFromOffsets(container, anchor.start, anchor.end)
    if (r) return r
  }

  // Drifted: relocate by quoted text.
  const idx = bestQuoteIndex(full, anchor)
  if (idx >= 0) {
    const r = rangeFromOffsets(container, idx, idx + anchor.quote.length)
    if (r) return r
  }

  // No quote to search by (shouldn't normally happen): honour the offsets directly.
  if (!anchor.quote && anchor.start < anchor.end && anchor.end <= full.length) {
    return rangeFromOffsets(container, anchor.start, anchor.end)
  }
  return null
}

/** Narrow unknown `coordsJson` to a TextAnchor, or null when it isn't one. */
export function parseTextAnchor(coords: unknown): TextAnchor | null {
  if (!coords || typeof coords !== 'object') return null
  const o = coords as Record<string, unknown>
  const start = typeof o.start === 'number' ? o.start : Number(o.start)
  const end = typeof o.end === 'number' ? o.end : Number(o.end)
  if (!Number.isFinite(start) || !Number.isFinite(end)) return null
  return {
    start,
    end,
    quote: typeof o.quote === 'string' ? o.quote : '',
    prefix: typeof o.prefix === 'string' ? o.prefix : '',
    suffix: typeof o.suffix === 'string' ? o.suffix : '',
  }
}
