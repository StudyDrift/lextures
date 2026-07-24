/** Pipe row that looks like a GFM table line (at least one cell separator). */
function isTableRowLine(line: string): boolean {
  const t = line.trim()
  if (!t.includes('|')) return false
  // Reject bare separator lines here; they are handled separately.
  if (/^\|?[\s:|-]+$/.test(t) && /-/.test(t)) return false
  return /\|/.test(t.slice(1, -1)) || (t.startsWith('|') && t.endsWith('|'))
}

/** GFM table separator: `| --- | :---: | ---: |` (pipes optional at edges). */
function isTableSeparatorLine(line: string): boolean {
  const t = line.trim()
  if (!t.includes('-')) return false
  return /^\|?(\s*:?-{3,}:?\s*\|)+\s*:?-{3,}:?\s*\|?$/.test(t) || /^\|?(\s*:?-{3,}:?\s*\|?)+\s*$/.test(t)
}

function isTableRelatedLine(line: string): boolean {
  return isTableRowLine(line) || isTableSeparatorLine(line)
}

function nextNonEmptyIndex(lines: string[], from: number): number {
  let j = from
  while (j < lines.length && lines[j]!.trim() === '') j++
  return j
}

/**
 * Collapse blank lines inside GitHub-flavored markdown tables.
 *
 * TipTap / CommonMark treat blank lines as paragraph breaks, so a table typed or
 * AI-generated with empty lines between rows becomes plain pipe text. Healing those
 * gaps restores a real `<table>` in both the editor and ReactMarkdown reader.
 */
export function normalizeMarkdownTables(markdown: string): string {
  if (!markdown.includes('|')) return markdown

  const lines = markdown.split('\n')
  const out: string[] = []
  let i = 0

  while (i < lines.length) {
    const line = lines[i]!
    const sepIdx = nextNonEmptyIndex(lines, i + 1)
    const looksLikeHeader =
      isTableRowLine(line) && sepIdx < lines.length && isTableSeparatorLine(lines[sepIdx]!)

    if (!looksLikeHeader) {
      out.push(line)
      i++
      continue
    }

    const tableLines: string[] = [line]
    i++
    while (i < lines.length) {
      const L = lines[i]!
      if (L.trim() === '') {
        const next = nextNonEmptyIndex(lines, i + 1)
        if (next < lines.length && isTableRelatedLine(lines[next]!)) {
          i = next
          continue
        }
        break
      }
      if (!isTableRelatedLine(L)) break
      tableLines.push(L)
      i++
    }

    if (tableLines.length >= 2 && isTableSeparatorLine(tableLines[1]!)) {
      out.push(...tableLines)
    } else {
      // Not a valid table after all — emit original lines without collapsing.
      out.push(line)
      // Rewind was not kept; emit collected non-header lines as-is if any.
      for (let k = 1; k < tableLines.length; k++) out.push(tableLines[k]!)
    }
  }

  return out.join('\n')
}
