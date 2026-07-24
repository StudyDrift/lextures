import { Editor } from '@tiptap/core'
import { Markdown } from '@tiptap/markdown'
import { TableKit } from '@tiptap/extension-table'
import StarterKit from '@tiptap/starter-kit'
import { afterEach, describe, expect, it } from 'vitest'

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
      TableKit.configure({ table: { resizable: false, renderWrapper: true } }),
      Markdown.configure({ markedOptions: { gfm: true } }),
    ],
    content,
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
    editor.commands.insertContent(sample, { contentType: 'markdown' })
    expect(editor.getJSON().content?.some((n) => n.type === 'table')).toBe(true)
    expect(editor.getMarkdown()).toContain('Traditional Software')
  })
})
