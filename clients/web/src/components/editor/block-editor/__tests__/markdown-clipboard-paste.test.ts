import { describe, expect, it } from 'vitest'
import {
  plainTextContainsMarkdownTable,
  shouldPasteClipboardAsMarkdown,
} from '../markdown-clipboard-paste'

const sample = `Traditional software works like a detailed recipe.

| Feature | Traditional Software | AI Systems |
|----------------------|---------------------------------------|-----------------------------------------|
| How it works | Follows fixed rules written by people | Learns patterns from data |

**Quick check (ungraded):**
In one sentence, explain the biggest difference.`

describe('plainTextContainsMarkdownTable', () => {
  it('detects GFM tables in pasted assignment content', () => {
    expect(plainTextContainsMarkdownTable(sample)).toBe(true)
  })

  it('returns false for plain prose', () => {
    expect(plainTextContainsMarkdownTable('Hello\n\nWorld')).toBe(false)
  })

  it('returns false for a lone pipe line without a separator', () => {
    expect(plainTextContainsMarkdownTable('| a | b |\n| c | d |')).toBe(false)
  })
})

describe('shouldPasteClipboardAsMarkdown', () => {
  it('prefers markdown when plain text has a table and HTML does not', () => {
    expect(
      shouldPasteClipboardAsMarkdown(sample, '<p style="color:white">Traditional software</p>'),
    ).toBe(true)
  })

  it('prefers markdown when there is no HTML', () => {
    expect(shouldPasteClipboardAsMarkdown(sample, '')).toBe(true)
  })

  it('defers to HTML paste when clipboard already has a table element', () => {
    expect(
      shouldPasteClipboardAsMarkdown(sample, '<table><tr><td>Feature</td></tr></table>'),
    ).toBe(false)
  })

  it('does not force markdown for ordinary text', () => {
    expect(shouldPasteClipboardAsMarkdown('hello world', '<p>hello world</p>')).toBe(false)
  })
})
