import type { Awareness } from 'y-protocols/awareness'
import type {
  Board,
  BoardLayout,
  BoardPost,
  BoardSection,
  BoardSortMode,
  ArrangeBoardPostInput,
} from '../../../lib/boards-api'

export type BoardSurfaceProps = {
  courseCode: string
  board: Board
  posts: BoardPost[]
  sections: BoardSection[]
  sortMode: BoardSortMode
  canManageBoard: boolean
  canArrangePost: (post: BoardPost) => boolean
  canManagePost: (post: BoardPost) => boolean
  onDeletePost: (postId: string) => void
  onPostUpdate: (post: BoardPost) => void
  onArrange: (postId: string, input: ArrangeBoardPostInput) => Promise<void>
  onSectionsChange: (sections: BoardSection[]) => void
  onCreateSection: (title: string) => Promise<BoardSection>
  onDeleteSection: (sectionId: string) => Promise<void>
  onAnnounce: (message: string) => void
  /** VC.4: awareness for live cursors on spatial layouts. */
  awareness?: Awareness | null
  onCursorMove?: (cursor: { x: number; y: number } | null) => void
}

export type LayoutRendererProps = BoardSurfaceProps

export function isBoardLayout(v: string): v is BoardLayout {
  return (
    v === 'wall' ||
    v === 'stream' ||
    v === 'grid' ||
    v === 'columns' ||
    v === 'canvas' ||
    v === 'timeline' ||
    v === 'map'
  )
}

/** Shared PostCard engagement props from a board surface. */
export function postCardEngagementProps(props: BoardSurfaceProps, post: BoardPost) {
  return {
    courseCode: props.courseCode,
    boardId: props.board.id,
    reactionMode: props.board.reactionMode,
    canManage: props.canManagePost(post),
    canManageBoard: props.canManageBoard,
    canInteract: !props.board.archived,
    assignmentLinked: !!props.board.assignmentId,
    onDelete: props.onDeletePost,
    onPostUpdate: props.onPostUpdate,
    onAnnounce: props.onAnnounce,
  }
}
