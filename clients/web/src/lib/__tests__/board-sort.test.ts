import { describe, expect, it } from 'vitest'
import { midpointSortIndex } from '../boards-api'
import { postsInSection, sortBoardPosts } from '../board-sort'
import type { BoardPost } from '../boards-api'

function post(partial: Partial<BoardPost> & { id: string }): BoardPost {
  return {
    boardId: 'b',
    authorId: null,
    contentType: 'text',
    title: '',
    sortIndex: 0,
    createdAt: '2026-01-01T00:00:00Z',
    updatedAt: '2026-01-01T00:00:00Z',
    ...partial,
  }
}

describe('midpointSortIndex', () => {
  it('inserts between neighbors', () => {
    expect(midpointSortIndex(1, 3)).toBe(2)
  })
  it('appends and prepends', () => {
    expect(midpointSortIndex(5, undefined)).toBe(6)
    expect(midpointSortIndex(undefined, 5)).toBe(4)
  })
})

describe('sortBoardPosts', () => {
  const posts = [
    post({ id: 'a', authorId: 'z', createdAt: '2026-01-03T00:00:00Z', sortIndex: 2 }),
    post({ id: 'b', authorId: 'a', createdAt: '2026-01-01T00:00:00Z', sortIndex: 0 }),
    post({ id: 'c', authorId: 'm', createdAt: '2026-01-02T00:00:00Z', sortIndex: 1 }),
  ]

  it('sorts newest first', () => {
    expect(sortBoardPosts(posts, 'newest').map((p) => p.id)).toEqual(['a', 'c', 'b'])
  })
  it('sorts oldest first', () => {
    expect(sortBoardPosts(posts, 'oldest').map((p) => p.id)).toEqual(['b', 'c', 'a'])
  })
  it('sorts by author', () => {
    expect(sortBoardPosts(posts, 'author').map((p) => p.id)).toEqual(['b', 'c', 'a'])
  })
  it('sorts by most reacted using reaction counts', () => {
    const reacted = [
      post({ id: 'a', reactionCount: 1, createdAt: '2026-01-03T00:00:00Z' }),
      post({ id: 'b', reactionCount: 5, createdAt: '2026-01-01T00:00:00Z' }),
      post({ id: 'c', reactionCount: 3, createdAt: '2026-01-02T00:00:00Z' }),
    ]
    expect(sortBoardPosts(reacted, 'mostReacted', 'like').map((p) => p.id)).toEqual([
      'b',
      'c',
      'a',
    ])
  })
  it('sorts by most reacted using star averages', () => {
    const rated = [
      post({ id: 'a', avgStars: 3, reactionCount: 10 }),
      post({ id: 'b', avgStars: 4.5, reactionCount: 2 }),
      post({ id: 'c', avgStars: 4.5, reactionCount: 8 }),
    ]
    expect(sortBoardPosts(rated, 'mostReacted', 'star').map((p) => p.id)).toEqual(['c', 'b', 'a'])
  })
})

describe('postsInSection', () => {
  it('filters and orders by sortIndex', () => {
    const posts = [
      post({ id: '1', sectionId: 's1', sortIndex: 2 }),
      post({ id: '2', sectionId: 's1', sortIndex: 1 }),
      post({ id: '3', sectionId: 's2', sortIndex: 0 }),
    ]
    expect(postsInSection(posts, 's1').map((p) => p.id)).toEqual(['2', '1'])
  })
})
