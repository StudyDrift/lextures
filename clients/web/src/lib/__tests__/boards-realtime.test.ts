import { describe, expect, it } from 'vitest'
import {
  applyArrangementToPost,
  arrangementFromPost,
  boardWsUrl,
  mergePostsWithCrdt,
  type BoardArrangement,
} from '../boards-realtime'
import type { BoardPost } from '../boards-api'

function stubPost(overrides: Partial<BoardPost> = {}): BoardPost {
  return {
    id: 'p1',
    boardId: 'b1',
    authorId: 'u1',
    contentType: 'text',
    title: 'Hello',
    body: { text: 'hi' },
    sortIndex: 1,
    position: { x: 10, y: 20, w: 100, h: 80 },
    createdAt: '2026-01-01T00:00:00Z',
    updatedAt: '2026-01-01T00:00:00Z',
    ...overrides,
  }
}

describe('boardWsUrl', () => {
  it('builds a boards websocket base path', () => {
    const url = boardWsUrl('CS101', 'board-uuid')
    expect(url).toContain('/api/v1/courses/CS101/boards/board-uuid')
    expect(url.startsWith('ws')).toBe(true)
  })
})

describe('mergePostsWithCrdt', () => {
  it('applies arrangement fields and removes deleted posts', () => {
    const posts = [
      stubPost({ id: 'a', sortIndex: 1 }),
      stubPost({ id: 'b', sortIndex: 2 }),
    ]
    const arrangements = new Map<string, BoardArrangement>([
      ['a', { id: 'a', sortIndex: 9, position: { x: 1, y: 2, w: 3, h: 4 } }],
      ['b', { id: 'b', deleted: true }],
    ])
    const merged = mergePostsWithCrdt(posts, arrangements)
    expect(merged).toHaveLength(1)
    expect(merged[0].id).toBe('a')
    expect(merged[0].sortIndex).toBe(9)
    expect(merged[0].position).toEqual({ x: 1, y: 2, w: 3, h: 4 })
  })
})

describe('arrangement helpers', () => {
  it('round-trips arrangement from a post', () => {
    const post = stubPost({ sectionId: 's1', lat: 1, lng: 2 })
    const arr = arrangementFromPost(post)
    const next = applyArrangementToPost(stubPost({ id: 'p1', sortIndex: 0 }), arr)
    expect(next.sectionId).toBe('s1')
    expect(next.lat).toBe(1)
    expect(next.lng).toBe(2)
    expect(next.position).toEqual(post.position)
  })
})
