import { describe, expect, it } from 'vitest'
import {
  EDITOR_SECTION_BOUNDARY,
  markdownToSectionsForEditor,
  sectionsToMarkdown,
  stripEditorSectionBoundaries,
} from '../syllabus-section-markdown'

describe('sectionsToMarkdown', () => {
  it('joins sections with ## headings and double newlines', () => {
    const md = sectionsToMarkdown([
      { id: '1', heading: 'A', markdown: 'Body one.' },
      { id: '2', heading: 'B', markdown: 'Body two.' },
    ])
    expect(md).toBe('## A\n\nBody one.\n\n## B\n\nBody two.')
  })

  it('trims trailing body whitespace and keeps untitled sections separate', () => {
    const md = sectionsToMarkdown([
      { id: '1', heading: '', markdown: '  solo  \n' },
      { id: '2', heading: '  ', markdown: 'x' },
    ])
    expect(md).toBe(`  solo\n\n${EDITOR_SECTION_BOUNDARY}\n\nx`)
  })

  it('does not insert a boundary before the first untitled section', () => {
    const md = sectionsToMarkdown([{ id: '1', heading: '', markdown: 'only' }])
    expect(md).toBe('only')
  })

  it('separates a titled section from a following untitled one', () => {
    const md = sectionsToMarkdown([
      { id: '1', heading: 'Intro', markdown: 'Hello' },
      { id: '2', heading: '', markdown: 'More body' },
    ])
    expect(md).toBe(`## Intro\n\nHello\n\n${EDITOR_SECTION_BOUNDARY}\n\nMore body`)
  })

  it('skips fully empty sections', () => {
    const md = sectionsToMarkdown([
      { id: '1', heading: '', markdown: '' },
      { id: '2', heading: 'Keep', markdown: 'yes' },
    ])
    expect(md).toBe('## Keep\n\nyes')
  })
})

describe('markdownToSectionsForEditor', () => {
  it('returns one empty section for blank input', () => {
    const sections = markdownToSectionsForEditor('', () => 'only-id')
    expect(sections).toHaveLength(1)
    expect(sections[0]).toEqual({ id: 'only-id', heading: '', markdown: '' })
  })

  it('splits on ## headings into multiple sections', () => {
    const ids = ['x', 'y', 'z']
    let i = 0
    const sections = markdownToSectionsForEditor(
      '## Intro\n\nHello\n\n## Outro\n\nBye',
      () => ids[i++]!,
    )
    expect(sections).toHaveLength(2)
    expect(sections[0]).toMatchObject({ heading: 'Intro', markdown: 'Hello' })
    expect(sections[1]).toMatchObject({ heading: 'Outro', markdown: 'Bye' })
  })

  it('parses first chunk without leading ## as heading-only body', () => {
    const sections = markdownToSectionsForEditor('Plain intro line', () => 'id1')
    expect(sections).toHaveLength(1)
    expect(sections[0]).toEqual({ id: 'id1', heading: '', markdown: 'Plain intro line' })
  })

  it('splits on editor section boundaries for untitled sections', () => {
    const ids = ['a', 'b']
    let i = 0
    const sections = markdownToSectionsForEditor(
      `first body\n\n${EDITOR_SECTION_BOUNDARY}\n\nsecond body`,
      () => ids[i++]!,
    )
    expect(sections).toHaveLength(2)
    expect(sections[0]).toMatchObject({ heading: '', markdown: 'first body' })
    expect(sections[1]).toMatchObject({ heading: '', markdown: 'second body' })
  })

  it('round-trips multiple untitled sections', () => {
    const original = [
      { id: '1', heading: '', markdown: 'Alpha' },
      { id: '2', heading: '', markdown: 'Beta' },
      { id: '3', heading: 'Titled', markdown: 'Gamma' },
      { id: '4', heading: '', markdown: 'Delta' },
    ]
    const md = sectionsToMarkdown(original)
    const ids = ['w', 'x', 'y', 'z']
    let i = 0
    const back = markdownToSectionsForEditor(md, () => ids[i++]!)
    expect(back).toHaveLength(4)
    expect(back.map((s) => ({ heading: s.heading, markdown: s.markdown }))).toEqual([
      { heading: '', markdown: 'Alpha' },
      { heading: '', markdown: 'Beta' },
      { heading: 'Titled', markdown: 'Gamma' },
      { heading: '', markdown: 'Delta' },
    ])
  })
})

describe('stripEditorSectionBoundaries', () => {
  it('removes boundary markers for reader display', () => {
    const md = `Alpha\n\n${EDITOR_SECTION_BOUNDARY}\n\nBeta`
    expect(stripEditorSectionBoundaries(md)).toBe('Alpha\n\nBeta')
  })

  it('leaves normal content unchanged', () => {
    expect(stripEditorSectionBoundaries('## Intro\n\nHello')).toBe('## Intro\n\nHello')
  })
})
