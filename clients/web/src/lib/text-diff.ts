/** Minimal line-level diff for base-vs-variant previews (AC.5). */

export type DiffLine =
  | { type: 'same'; text: string }
  | { type: 'add'; text: string }
  | { type: 'remove'; text: string }

/**
 * LCS-based line diff. Fine for content pages (hundreds of lines); not for multi-MB docs.
 */
export function diffLines(base: string, variant: string): DiffLine[] {
  const split = (s: string) => {
    if (s === '') return [] as string[]
    return s.replace(/\r\n/g, '\n').split('\n')
  }
  const a = split(base)
  const b = split(variant)
  const n = a.length
  const m = b.length
  const dp: number[][] = Array.from({ length: n + 1 }, () => Array(m + 1).fill(0))
  for (let i = n - 1; i >= 0; i--) {
    for (let j = m - 1; j >= 0; j--) {
      if (a[i] === b[j]) dp[i][j] = dp[i + 1][j + 1] + 1
      else dp[i][j] = Math.max(dp[i + 1][j], dp[i][j + 1])
    }
  }
  const out: DiffLine[] = []
  let i = 0
  let j = 0
  while (i < n && j < m) {
    if (a[i] === b[j]) {
      out.push({ type: 'same', text: a[i] })
      i++
      j++
    } else if (dp[i + 1][j] >= dp[i][j + 1]) {
      out.push({ type: 'remove', text: a[i] })
      i++
    } else {
      out.push({ type: 'add', text: b[j] })
      j++
    }
  }
  while (i < n) {
    out.push({ type: 'remove', text: a[i] })
    i++
  }
  while (j < m) {
    out.push({ type: 'add', text: b[j] })
    j++
  }
  return out
}
