import { describe, expect, it } from 'vitest'
import {
  directives,
  formatQualityScore,
  isBlockingFinding,
  lintMetadata,
  scoreMeterPercent,
  simpleLineDiff,
  slugify,
} from '../article-editor-utils'

describe('article editor helpers', () => {
  it('derives a stable kebab-case slug', () => expect(slugify('  Finding Your Coursé! ')).toBe('finding-your-course'))
  it('provides a valid key-takeaways skeleton', () => expect(directives.find((v) => v.id === 'key-takeaways')?.markdown).toContain(':::key-takeaways'))
  it('keeps insert-block skeletons free of raw HTML', () => {
    for (const directive of directives) {
      expect(directive.markdown).not.toMatch(/<!--|<\/?[a-z]/i)
    }
    expect(directives.find((v) => v.id === 'callout')?.markdown).toMatch(/^:::callout\s+note/m)
  })
  it('summarises line additions and removals', () => {
    const diff = simpleLineDiff('one\ntwo', 'one\nthree')
    expect(diff.filter((v) => v.type === 'added')).toHaveLength(1)
    expect(diff.filter((v) => v.type === 'removed')).toHaveLength(1)
  })
  it('maps editor fields into lint metadata', () => {
    expect(lintMetadata({
      title: 'Title',
      description: 'Desc',
      authorSlug: 'chase',
      cluster: 'assessment',
      primaryQuestion: 'What works?',
      keywords: ['a'],
      locale: 'en',
      contentUpdatedAt: '2026-08-13T12:00:00Z',
    })).toEqual({
      title: 'Title',
      description: 'Desc',
      author: 'chase',
      authorSlug: 'chase',
      cluster: 'assessment',
      primaryQuestion: 'What works?',
      keywords: ['a'],
      locale: 'en',
      updated: '2026-08-13',
    })
  })
  it('treats extractability scores on the 0–10 scale', () => {
    expect(scoreMeterPercent(8)).toBe(80)
    expect(scoreMeterPercent(10)).toBe(100)
    expect(formatQualityScore(8.5)).toBe('8.5')
    expect(isBlockingFinding('error')).toBe(true)
    expect(isBlockingFinding('warn')).toBe(false)
  })
})
