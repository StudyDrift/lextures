import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  applyArrangementToPost,
  arrangementFromPost,
  boardWsUrl,
  mergePostsWithCrdt,
  parseBoardChangedEvent,
  shouldApplyCrdtArrangement,
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
  afterEach(() => {
    vi.unstubAllEnvs()
  })

  it('builds a boards websocket base path via the page origin in local dev', () => {
    vi.stubEnv('VITE_API_URL', 'http://localhost:8080')
    const url = boardWsUrl('CS101', 'board-uuid')
    expect(url).toContain('/api/v1/courses/CS101/boards/board-uuid')
    expect(url.startsWith('ws')).toBe(true)
  })
})

describe('parseBoardChangedEvent', () => {
  it('parses board.changed text frames', () => {
    expect(parseBoardChangedEvent('{"type":"board.changed","reason":"post.created","postId":"p1"}')).toEqual({
      type: 'board.changed',
      reason: 'post.created',
      postId: 'p1',
    })
    expect(parseBoardChangedEvent('{"type":"other"}')).toBeNull()
    expect(parseBoardChangedEvent('not-json')).toBeNull()
  })
})

describe('mergePostsWithCrdt', () => {
  it('applies arrangement fields and removes deleted posts', () => {
    const posts = [
      stubPost({ id: 'a', sortIndex: 1 }),
      stubPost({ id: 'b', sortIndex: 2 }),
    ]
    const arrangements = new Map<string, BoardArrangement>([
      ['a', { id: 'a', sortIndex: 9, position: { x: 1, y: 2, w: 3, h: 4 }, updatedAt: '2026-01-02T00:00:00Z' }],
      ['b', { id: 'b', deleted: true }],
    ])
    const merged = mergePostsWithCrdt(posts, arrangements)
    expect(merged).toHaveLength(1)
    expect(merged[0].id).toBe('a')
    expect(merged[0].sortIndex).toBe(9)
    expect(merged[0].position).toEqual({ x: 1, y: 2, w: 3, h: 4 })
  })

  it('does not let a stale CRDT arrangement overwrite newer REST location/order', () => {
    const posts = [
      stubPost({
        id: 'a',
        sortIndex: 5,
        position: { x: 100, y: 200, w: 100, h: 80 },
        updatedAt: '2026-01-03T00:00:00Z',
      }),
    ]
    const arrangements = new Map<string, BoardArrangement>([
      [
        'a',
        {
          id: 'a',
          sortIndex: 1,
          position: { x: 1, y: 2, w: 3, h: 4 },
          updatedAt: '2026-01-01T00:00:00Z',
        },
      ],
    ])
    const merged = mergePostsWithCrdt(posts, arrangements)
    expect(merged[0].sortIndex).toBe(5)
    expect(merged[0].position).toEqual({ x: 100, y: 200, w: 100, h: 80 })
  })
})

describe('shouldApplyCrdtArrangement', () => {
  it('applies CRDT when newer or unversioned', () => {
    const post = stubPost({ updatedAt: '2026-01-01T00:00:00Z' })
    expect(shouldApplyCrdtArrangement(post, { id: 'p1', updatedAt: '2026-01-02T00:00:00Z' })).toBe(true)
    expect(shouldApplyCrdtArrangement(post, { id: 'p1' })).toBe(true)
    expect(shouldApplyCrdtArrangement(post, { id: 'p1', updatedAt: '2025-01-01T00:00:00Z' })).toBe(false)
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
