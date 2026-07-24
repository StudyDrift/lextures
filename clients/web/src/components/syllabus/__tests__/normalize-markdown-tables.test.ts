import { describe, expect, it } from 'vitest'
import { normalizeMarkdownTables } from '../normalize-markdown-tables'

const sample = `Traditional software works like a detailed recipe.

| Feature | Traditional Software | AI Systems |
|-----------------|---------------------------------|---------------------------------|
| How it works | Follows fixed rules written by people | Learns patterns from data |
| Handling new situations | Only works if the situation was programmed | Can handle new situations that resemble training data |

**Quick check:**
In one sentence.
`

describe('normalizeMarkdownTables', () => {
  it('leaves valid GFM tables unchanged', () => {
    expect(normalizeMarkdownTables(sample)).toBe(sample)
  })

  it('collapses blank lines between table rows so GFM can parse them', () => {
    const broken = `Intro

| Feature | Traditional Software | AI Systems |

|-----------------|---------------------------------|---------------------------------|

| How it works | Follows fixed rules | Learns patterns |

| Example | Calculator | Spam filter |

**Quick check:**
Done.
`
    const out = normalizeMarkdownTables(broken)
    expect(out).toContain(
      [
        '| Feature | Traditional Software | AI Systems |',
        '|-----------------|---------------------------------|---------------------------------|',
        '| How it works | Follows fixed rules | Learns patterns |',
        '| Example | Calculator | Spam filter |',
      ].join('\n'),
    )
    expect(out).toContain('**Quick check:**')
    expect(out).not.toMatch(/\|\s*\n\n\s*\|/)
  })

  it('does not merge unrelated pipe-ish paragraphs', () => {
    const md = `Use | as OR in regex.

Then write more.`
    expect(normalizeMarkdownTables(md)).toBe(md)
  })
})
