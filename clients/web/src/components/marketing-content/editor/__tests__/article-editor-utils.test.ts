import { describe, expect, it } from 'vitest'
import { directives, simpleLineDiff, slugify } from '../article-editor-utils'

describe('article editor helpers', () => {
  it('derives a stable kebab-case slug', () => expect(slugify('  Finding Your Coursé! ')).toBe('finding-your-course'))
  it('provides a valid key-takeaways skeleton', () => expect(directives.find((v) => v.id === 'key-takeaways')?.markdown).toContain(':::key-takeaways'))
  it('summarises line additions and removals', () => {
    const diff = simpleLineDiff('one\ntwo', 'one\nthree')
    expect(diff.filter((v) => v.type === 'added')).toHaveLength(1)
    expect(diff.filter((v) => v.type === 'removed')).toHaveLength(1)
  })
})
