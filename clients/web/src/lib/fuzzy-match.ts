/**
 * Lightweight fuzzy match for UI filters (command-palette style).
 * Returns a score when `query` matches `text` as a subsequence or substring;
 * higher is better. Returns null when there is no match.
 */
export function fuzzyMatchScore(query: string, text: string): number | null {
  const q = query.trim().toLowerCase()
  const t = text.toLowerCase()
  if (!q) return 0
  if (!t) return null

  const substringAt = t.indexOf(q)
  if (substringAt !== -1) {
    let score = 200 - substringAt
    if (substringAt === 0) score += 40
    else if (t[substringAt - 1] === ' ' || t[substringAt - 1] === '-' || t[substringAt - 1] === '/') {
      score += 20
    }
    return score
  }

  let ti = 0
  let score = 0
  let prevMatch = -2
  for (let qi = 0; qi < q.length; qi++) {
    const ch = q[qi]!
    if (ch === ' ') continue
    const idx = t.indexOf(ch, ti)
    if (idx === -1) return null
    score += 1
    if (idx === prevMatch + 1) score += 4
    if (idx === 0 || t[idx - 1] === ' ' || t[idx - 1] === '-' || t[idx - 1] === '/') {
      score += 3
    }
    prevMatch = idx
    ti = idx + 1
  }
  return score
}

/** True when every whitespace-separated query token fuzzy-matches `text`. */
export function fuzzyMatches(query: string, text: string): boolean {
  const q = query.trim().toLowerCase()
  if (!q) return true
  const tokens = q.split(/\s+/).filter(Boolean)
  return tokens.every((token) => fuzzyMatchScore(token, text) != null)
}
