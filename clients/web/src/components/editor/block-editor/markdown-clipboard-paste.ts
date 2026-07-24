/** Pipe row that looks like a GFM table line (at least one cell separator). */
function isPipeRow(line: string): boolean {
  const t = line.trim()
  if (!t.includes('|')) return false
  // Require a cell boundary, not a lone `|` decorative line.
  return /\|/.test(t.slice(1, -1)) || (t.startsWith('|') && t.endsWith('|'))
}

/** GFM table separator: `| --- | :---: | ---: |` (pipes optional at edges). */
function isSeparatorRow(line: string): boolean {
  const t = line.trim()
  if (!t.includes('-')) return false
  // Strip outer pipes, then every cell must be dashes with optional leading/trailing `:`.
  const inner = t.replace(/^\|/, '').replace(/\|$/, '')
  const cells = inner.split('|').map((c) => c.trim())
  if (cells.length < 1) return false
  return cells.every((c) => /^:?-{3,}:?$/.test(c))
}

/** True when plain text contains a GitHub-flavored markdown table. */
export function plainTextContainsMarkdownTable(text: string): boolean {
  const lines = text.split(/\r?\n/)
  for (let i = 0; i < lines.length - 1; i++) {
    const row = lines[i]
    const sep = lines[i + 1]
    if (row && sep && isPipeRow(row) && isSeparatorRow(sep)) return true
  }
  return false
}

function htmlContainsTable(html: string): boolean {
  return /<table[\s>]/i.test(html)
}

/**
 * Prefer Markdown paste when the clipboard plain text has a GFM table and the HTML
 * side does not already carry a real `<table>` (TipTap has no Markdown paste rules for tables).
 */
export function shouldPasteClipboardAsMarkdown(text: string, html: string): boolean {
  const plain = text.trim()
  if (!plain) return false
  if (!plainTextContainsMarkdownTable(plain)) return false
  if (!html.trim()) return true
  return !htmlContainsTable(html)
}
