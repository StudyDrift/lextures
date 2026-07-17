import { describe, expect, it } from 'vitest'
import {
  applyReactionResult,
  boardPostReactionScore,
  renderBoardSurfacePng,
  videoEmbedFromUrl,
  type BoardPost,
} from '../boards-api'

describe('videoEmbedFromUrl', () => {
  it('parses YouTube watch URLs', () => {
    expect(videoEmbedFromUrl('https://www.youtube.com/watch?v=dQw4w9WgXcQ')).toEqual({
      provider: 'youtube',
      id: 'dQw4w9WgXcQ',
    })
  })

  it('parses youtu.be short links', () => {
    expect(videoEmbedFromUrl('https://youtu.be/dQw4w9WgXcQ')).toEqual({
      provider: 'youtube',
      id: 'dQw4w9WgXcQ',
    })
  })

  it('parses Vimeo URLs', () => {
    expect(videoEmbedFromUrl('https://vimeo.com/123456789')).toEqual({
      provider: 'vimeo',
      id: '123456789',
    })
  })

  it('returns null for plain articles', () => {
    expect(videoEmbedFromUrl('https://example.com/article')).toBeNull()
  })
})

describe('boardPostReactionScore', () => {
  it('uses reaction count for like/vote', () => {
    const post = { reactionCount: 7 } as BoardPost
    expect(boardPostReactionScore(post, 'like')).toBe(7)
    expect(boardPostReactionScore(post, 'vote')).toBe(7)
  })

  it('weights star average above count', () => {
    expect(
      boardPostReactionScore({ avgStars: 4, reactionCount: 1 } as BoardPost, 'star'),
    ).toBeGreaterThan(
      boardPostReactionScore({ avgStars: 3, reactionCount: 100 } as BoardPost, 'star'),
    )
  })
})

describe('renderBoardSurfacePng', () => {
  it('produces a PNG blob from card rows when canvas 2d is available', async () => {
    const canvas = document.createElement('canvas')
    if (!canvas.getContext('2d')) {
      // jsdom often lacks a real 2d context; skip rather than fail CI.
      return
    }
    const blob = await renderBoardSurfacePng('Demo', [
      { sectionTitle: 'Ideas', title: 'One', body: 'hello' },
    ])
    expect(blob.type).toBe('image/png')
    expect(blob.size).toBeGreaterThan(50)
  })
})

describe('applyReactionResult', () => {
  it('clears myReaction when toggle removes', () => {
    const post = {
      id: 'p1',
      reactionCount: 1,
      myReaction: { kind: 'like' },
    } as BoardPost
    const next = applyReactionResult(post, {
      active: false,
      removed: true,
      reactionCount: 0,
      commentCount: 2,
    })
    expect(next.myReaction).toBeNull()
    expect(next.reactionCount).toBe(0)
    expect(next.commentCount).toBe(2)
  })
})
