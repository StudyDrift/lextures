import { describe, expect, it } from 'vitest'
import type { DiscussionPost } from '../../lib/discussions-api'
import { emptyTipTapDoc } from '../../lib/discussions-api'
import { isEmptyDiscussionDoc, resolveDiscussionOpeningPost } from './discussion-opening-post'

function textDoc(text: string) {
  return { type: 'doc', content: [{ type: 'paragraph', content: [{ type: 'text', text }] }] }
}

function post(over: Partial<DiscussionPost> & { id: string }): DiscussionPost {
  return {
    threadId: 't1',
    parentPostId: null,
    authorId: 'teacher',
    body: textDoc('hello'),
    upvoteCount: 0,
    viewerUpvoted: false,
    createdAt: '2026-10-09T10:00:00Z',
    updatedAt: '2026-10-09T10:00:00Z',
    ...over,
  }
}

describe('isEmptyDiscussionDoc', () => {
  it('treats empty docs and whitespace as empty', () => {
    expect(isEmptyDiscussionDoc(emptyTipTapDoc)).toBe(true)
    expect(isEmptyDiscussionDoc({ type: 'doc', content: [] })).toBe(true)
    expect(isEmptyDiscussionDoc(textDoc('   '))).toBe(true)
    expect(isEmptyDiscussionDoc(null)).toBe(true)
    expect(isEmptyDiscussionDoc('')).toBe(true)
  })

  it('detects text and non-text content', () => {
    expect(isEmptyDiscussionDoc(textDoc('Prompt'))).toBe(false)
    expect(isEmptyDiscussionDoc({ type: 'doc', content: [{ type: 'horizontalRule' }] })).toBe(false)
  })
})

describe('resolveDiscussionOpeningPost', () => {
  it('keeps the thread body when it has content', () => {
    const posts = [post({ id: 'p1' })]
    const r = resolveDiscussionOpeningPost({ authorId: 'teacher', body: textDoc('Prompt') }, posts)
    expect(r.promotedPost).toBeNull()
    expect(r.openingBody).toEqual(textDoc('Prompt'))
    expect(r.replies).toBe(posts)
  })

  it("promotes the author's first top-level reply when the body is blank", () => {
    const prompt = post({ id: 'p1', body: textDoc('Prompt: describe one thing') })
    const kid = post({ id: 'p2', authorId: 'kid', createdAt: '2026-10-09T11:00:00Z' })
    const nested = post({
      id: 'p3',
      authorId: 'kid',
      parentPostId: 'p1',
      createdAt: '2026-10-09T12:00:00Z',
    })
    const r = resolveDiscussionOpeningPost({ authorId: 'teacher', body: emptyTipTapDoc }, [
      kid,
      nested,
      prompt,
    ])
    expect(r.promotedPost?.id).toBe('p1')
    expect(r.openingBody).toEqual(textDoc('Prompt: describe one thing'))
    expect(r.replies.map((p) => p.id)).toEqual(['p2', 'p3'])
    expect(r.replies.find((p) => p.id === 'p3')?.parentPostId).toBeNull()
  })

  it('shows no opening body when the first reply is from someone else', () => {
    const r = resolveDiscussionOpeningPost({ authorId: 'teacher', body: emptyTipTapDoc }, [
      post({ id: 'p1', authorId: 'kid' }),
    ])
    expect(r.promotedPost).toBeNull()
    expect(r.openingBody).toBeNull()
    expect(r.replies).toHaveLength(1)
  })
})
