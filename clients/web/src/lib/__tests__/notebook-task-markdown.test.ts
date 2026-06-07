import { describe, expect, it } from 'vitest'
import { markTaskCheckedInMarkdown, parseNotebookTasksFromMarkdown } from '../notebook-task-markdown'

describe('parseNotebookTasksFromMarkdown', () => {
  it('parses task blocks from markdown', () => {
    const md = [
      '```task',
      '{"id":"abc-123","checked":false,"dueAt":null}',
      'Walk in the garden',
      '```',
    ].join('\n')
    expect(parseNotebookTasksFromMarkdown(md)).toEqual([
      { id: 'abc-123', text: 'Walk in the garden', checked: false, dueAt: null },
    ])
  })
})

describe('markTaskCheckedInMarkdown', () => {
  it('marks the matching task block checked', () => {
    const md = [
      'Intro',
      '',
      '```task',
      '{"id":"abc-123","checked":false,"dueAt":null}',
      'Finish reading',
      '```',
    ].join('\n')
    const next = markTaskCheckedInMarkdown(md, 'abc-123')
    expect(next).toContain('"checked":true')
    expect(next).toContain('Finish reading')
  })

  it('leaves other task blocks unchanged', () => {
    const md = [
      '```task',
      '{"id":"one","checked":false,"dueAt":null}',
      'A',
      '```',
      '',
      '```task',
      '{"id":"two","checked":false,"dueAt":null}',
      'B',
      '```',
    ].join('\n')
    const next = markTaskCheckedInMarkdown(md, 'one')
    expect(next).toContain('"id":"one","checked":true')
    expect(next).toContain('"id":"two","checked":false')
  })
})
