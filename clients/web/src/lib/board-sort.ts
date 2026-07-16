import {
  boardPostReactionScore,
  type BoardPost,
  type BoardReactionMode,
  type BoardSortMode,
} from './boards-api'

/** Sort posts for layouts that support sort controls (FR-9). */
export function sortBoardPosts(
  posts: BoardPost[],
  mode: BoardSortMode,
  reactionMode: BoardReactionMode = 'none',
): BoardPost[] {
  const copy = [...posts]
  switch (mode) {
    case 'newest':
      return copy.sort((a, b) => b.createdAt.localeCompare(a.createdAt))
    case 'oldest':
      return copy.sort((a, b) => a.createdAt.localeCompare(b.createdAt))
    case 'author':
      return copy.sort((a, b) => {
        const aa = (a.authorId ?? '').toLowerCase()
        const bb = (b.authorId ?? '').toLowerCase()
        if (aa !== bb) return aa.localeCompare(bb)
        return a.sortIndex - b.sortIndex
      })
    case 'mostReacted':
      return copy.sort((a, b) => {
        const sa = boardPostReactionScore(a, reactionMode)
        const sb = boardPostReactionScore(b, reactionMode)
        if (sb !== sa) return sb - sa
        return b.createdAt.localeCompare(a.createdAt)
      })
    default: {
      const _exhaustive: never = mode
      return _exhaustive
    }
  }
}

export function postsInSection(posts: BoardPost[], sectionId: string | null): BoardPost[] {
  return posts
    .filter((p) => (sectionId == null ? !p.sectionId : p.sectionId === sectionId))
    .sort((a, b) => a.sortIndex - b.sortIndex || a.createdAt.localeCompare(b.createdAt))
}
