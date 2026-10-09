import { describe, expect, it } from 'vitest'
import { discussionAuthorLabel } from './discussion-author-label'

describe('discussionAuthorLabel', () => {
  it('uses the display name', () => {
    expect(discussionAuthorLabel('QA Test Kid')).toBe('QA Test Kid')
  })

  it('falls back when the name is missing', () => {
    expect(discussionAuthorLabel(null)).toBe('Someone')
    expect(discussionAuthorLabel('  ')).toBe('Someone')
  })
})
