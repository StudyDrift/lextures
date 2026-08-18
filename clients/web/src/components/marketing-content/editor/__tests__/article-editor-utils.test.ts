import { describe, expect, it } from 'vitest'
import type { MarketingArticle } from '../../../../lib/marketing-content-api'
import {
  directives,
  formatQualityScore,
  reconcileConcurrentSave,
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
  it('keeps edits made while an older save was in flight', () => {
    const current = {
      id: 'article-1', heroMediaId: 'new-image', path: '/blog/old-path', status: 'draft',
      revisionNo: 4, updatedAt: '2026-08-17T04:00:00Z',
    } as MarketingArticle
    const saved = {
      ...current, heroMediaId: null, path: '/blog/saved-path', revisionNo: 5,
      updatedAt: '2026-08-17T04:01:00Z',
    }

    expect(reconcileConcurrentSave(current, saved)).toMatchObject({
      heroMediaId: 'new-image',
      path: '/blog/saved-path',
      revisionNo: 5,
      updatedAt: '2026-08-17T04:01:00Z',
    })
  })
})
