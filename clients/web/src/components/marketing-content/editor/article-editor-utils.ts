export type Directive = {
  id: string
  label: string
  markdown: string
}

export const directives: Directive[] = [
  ['key-takeaways', 'Key takeaways', ':::key-takeaways\n<!-- Add 3–5 concise bullets. -->\n- First takeaway\n- Second takeaway\n- Third takeaway\n:::\n'],
  ['answer', 'Direct answer', ':::answer\n<!-- Answer the primary question in 40–60 words. -->\nWrite the direct answer here.\n:::\n'],
  ['definition', 'Definition', ':::definition\n**Term:** Write a concise, self-contained definition.\n:::\n'],
  ['comparison-table', 'Comparison table', ':::comparison-table\n| Option | Best for | Considerations |\n| --- | --- | --- |\n| A | … | … |\n| B | … | … |\n:::\n'],
  ['steps', 'Steps', ':::steps\n1. First step\n2. Second step\n3. Third step\n:::\n'],
  ['faq', 'FAQ', ':::faq\n### Question one?\nAnswer one.\n\n### Question two?\nAnswer two.\n\n### Question three?\nAnswer three.\n:::\n'],
  ['callout', 'Callout', ':::callout{type="note"}\nImportant context belongs here.\n:::\n'],
  ['stat', 'Statistic', ':::stat\n**00%** — Explain the statistic and cite its source.\n:::\n'],
  ['sources', 'Sources', ':::sources\n- [Source title](https://example.com)\n:::\n'],
].map(([id, label, markdown]) => ({ id, label, markdown }))

export function slugify(value: string): string {
  return value.toLowerCase().trim().normalize('NFKD').replace(/[\u0300-\u036f]/g, '')
    .replace(/[^a-z0-9]+/g, '-').replace(/^-+|-+$/g, '').slice(0, 100)
}

export function commaList(value: string): string[] {
  return value.split(',').map((item) => item.trim()).filter(Boolean)
}

export function simpleLineDiff(before: string, after: string) {
  const oldLines = before.split('\n')
  const newLines = after.split('\n')
  const max = Math.max(oldLines.length, newLines.length)
  const lines: Array<{ type: 'same' | 'removed' | 'added'; text: string }> = []
  for (let index = 0; index < max; index += 1) {
    const oldLine = oldLines[index]
    const newLine = newLines[index]
    if (oldLine === newLine) lines.push({ type: 'same', text: oldLine ?? '' })
    else {
      if (oldLine !== undefined) lines.push({ type: 'removed', text: oldLine })
      if (newLine !== undefined) lines.push({ type: 'added', text: newLine })
    }
  }
  return lines
}
