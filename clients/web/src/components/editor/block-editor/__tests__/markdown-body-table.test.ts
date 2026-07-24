import { Editor } from '@tiptap/core'
import { Markdown } from '@tiptap/markdown'
import { TableKit } from '@tiptap/extension-table'
import StarterKit from '@tiptap/starter-kit'
import { afterEach, describe, expect, it } from 'vitest'
import { adjustSelectedColumnWidth, insertDefaultTable } from '../markdown-table-commands'
import { normalizeMarkdownTables } from '../../../syllabus/normalize-markdown-tables'

const sample = `Traditional software works like a detailed recipe.

| Feature | Traditional Software | AI Systems |
|----------------------|---------------------------------------|-----------------------------------------|
| How it works | Follows fixed rules written by people | Learns patterns from data |
| Handling new situations | Only works if the situation was programmed | Can handle new situations that resemble training data |

**Quick check (ungraded):**
In one sentence, explain the biggest difference between traditional software and AI.
`

function createEditor(content: string) {
  return new Editor({
    extensions: [
      StarterKit,
      TableKit.configure({ table: { resizable: true, renderWrapper: true } }),
      Markdown.configure({ markedOptions: { gfm: true } }),
    ],
    content: normalizeMarkdownTables(content),
    contentType: 'markdown',
  })
}

describe('MarkdownBodyEditor table support', () => {
  let editor: Editor | null = null

  afterEach(() => {
    editor?.destroy()
    editor = null
  })

  it('parses GFM tables from markdown content', () => {
    editor = createEditor(sample)
    const json = editor.getJSON()
    const hasTable = json.content?.some((n) => n.type === 'table')
    expect(hasTable).toBe(true)
    const md = editor.getMarkdown()
    expect(md).toContain('| Feature')
    expect(md).toContain('How it works')
    expect(md).toContain('**Quick check (ungraded):**')
  })

  it('inserts pasted markdown tables as table nodes', () => {
    editor = createEditor('Before')
    editor.commands.insertContent(normalizeMarkdownTables(sample), { contentType: 'markdown' })
    expect(editor.getJSON().content?.some((n) => n.type === 'table')).toBe(true)
    expect(editor.getMarkdown()).toContain('Traditional Software')
  })

  it('heals blank-line-broken pipe tables into real table nodes', () => {
    const broken = `Intro

| Feature | A | B |

|---|---|---|

| row | 1 | 2 |
`
    editor = createEditor(broken)
    expect(editor.getJSON().content?.some((n) => n.type === 'table')).toBe(true)
    expect(editor.getMarkdown()).not.toMatch(/\|\s*\n\n\s*\|/)
  })

  it('insertDefaultTable creates a table node (not pipe text)', () => {
    editor = createEditor('Hello')
    expect(insertDefaultTable(editor)).toBe(true)
    expect(editor.getJSON().content?.some((n) => n.type === 'table')).toBe(true)
    expect(editor.isActive('table')).toBe(true)
  })

  it('addColumnAfter grows the table and widen adjusts colwidth', () => {
    editor = createEditor('| A | B |\n| --- | --- |\n| 1 | 2 |')
    // Place caret in first body cell
    editor.commands.setTextSelection(editor.state.doc.content.size - 4)
    expect(editor.isActive('table')).toBe(true)
    const colsBefore = editor.getMarkdown().split('\n').find((l) => l.startsWith('| A'))
    expect(colsBefore).toBeTruthy()
    editor.chain().focus().addColumnAfter().run()
    expect(editor.getMarkdown()).toMatch(/\| A\s+\| B\s+\|/)
    expect(adjustSelectedColumnWidth(editor, 40)).toBe(true)
  })
})
