import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'
import { normalizeMarkdownTables } from '../normalize-markdown-tables'

const fixturesPath = resolve(
  dirname(fileURLToPath(import.meta.url)),
  '../../../../../mobile/fixtures/markdown/corpus.json',
)

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

  it('heals the shared mobile fixture table-blank-lines (CT.M1 FR-3)', () => {
    const corpus = JSON.parse(readFileSync(fixturesPath, 'utf8')) as {
      fixtures: Array<{ id: string; markdown: string }>
    }
    const fixture = corpus.fixtures.find((f) => f.id === 'table-blank-lines')
    expect(fixture).toBeTruthy()
    const out = normalizeMarkdownTables(fixture!.markdown)
    expect(out).toContain('| A | B |\n| --- | --- |\n| 1 | 2 |\n| 3 | 4 |')
    expect(out).toContain('After table')
    expect(out).not.toMatch(/\|\s*\n\n\s*\|/)
  })
})
