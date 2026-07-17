import SwiftUI

/// Dispatches to a mobile layout renderer based on `board.layout` (VC.M3).
struct BoardSurface: View {
    let board: Board
    let posts: [BoardPost]
    let sections: [BoardSection]
    var sortMode: BoardSortMode = .newest
    var canManage: Bool = false
    var currentUserId: String?
    var onEdit: (BoardPost) -> Void
    var onDelete: (BoardPost) -> Void
    var onArrange: (BoardPost, ArrangeBoardPostInput) -> Void
    var onCreateSection: ((String) -> Void)?
    var onDeleteSection: ((String) -> Void)?

    private var layout: BoardLayout {
        BoardsLogic.resolveLayout(board.layout)
    }

    private var reactionMode: BoardReactionMode {
        BoardReactionMode.fromAPI(board.reactionMode)
    }

    var body: some View {
        Group {
            switch layout {
            case .wall:
                WallLayoutView(
                    posts: BoardsLogic.sortedPosts(posts, mode: sortMode, reactionMode: reactionMode),
                    sections: sections,
                    board: board,
                    canManage: canManage,
                    currentUserId: currentUserId,
                    onEdit: onEdit,
                    onDelete: onDelete,
                    onArrange: onArrange
                )
            case .stream:
                StreamLayoutView(
                    posts: BoardsLogic.sortedPosts(posts, mode: sortMode, reactionMode: reactionMode),
                    sections: sections,
                    board: board,
                    canManage: canManage,
                    currentUserId: currentUserId,
                    onEdit: onEdit,
                    onDelete: onDelete,
                    onArrange: onArrange
                )
            case .grid:
                GridLayoutView(
                    posts: BoardsLogic.sortedPosts(posts, mode: sortMode, reactionMode: reactionMode),
                    sections: sections,
                    board: board,
                    canManage: canManage,
                    currentUserId: currentUserId,
                    onEdit: onEdit,
                    onDelete: onDelete,
                    onArrange: onArrange
                )
            case .columns:
                ColumnsLayoutView(
                    posts: posts,
                    sections: sections,
                    board: board,
                    canManage: canManage,
                    currentUserId: currentUserId,
                    onEdit: onEdit,
                    onDelete: onDelete,
                    onArrange: onArrange,
                    onCreateSection: onCreateSection,
                    onDeleteSection: onDeleteSection
                )
            case .canvas:
                CanvasLayoutView(
                    posts: posts,
                    sections: sections,
                    board: board,
                    canManage: canManage,
                    currentUserId: currentUserId,
                    onEdit: onEdit,
                    onDelete: onDelete,
                    onArrange: onArrange
                )
            case .timeline:
                TimelineLayoutView(
                    posts: posts,
                    sections: sections,
                    board: board,
                    canManage: canManage,
                    currentUserId: currentUserId,
                    onEdit: onEdit,
                    onDelete: onDelete,
                    onArrange: onArrange
                )
            case .map:
                MapLayoutView(
                    posts: posts,
                    sections: sections,
                    board: board,
                    canManage: canManage,
                    currentUserId: currentUserId,
                    onEdit: onEdit,
                    onDelete: onDelete,
                    onArrange: onArrange
                )
            }
        }
        .accessibilityLabel(L.text("mobile.boards.layout.\(layout.rawValue)"))
    }
}

struct BoardPostCardSlot: View {
    let post: BoardPost
    let siblings: [BoardPost]
    let sections: [BoardSection]
    let board: Board
    var canManage: Bool
    var currentUserId: String?
    var showTimeline: Bool = false
    var showMap: Bool = false
    var onEdit: (BoardPost) -> Void
    var onDelete: (BoardPost) -> Void
    var onArrange: (BoardPost, ArrangeBoardPostInput) -> Void

    private var canEdit: Bool {
        BoardsLogic.canEditOrDeletePost(post: post, currentUserId: currentUserId, canManage: canManage)
    }

    private var canArrange: Bool {
        BoardsLogic.canArrangePost(
            post: post,
            board: board,
            currentUserId: currentUserId,
            canManage: canManage
        )
    }

    var body: some View {
        BoardPostCard(
            post: post,
            canEdit: canEdit,
            canArrange: canArrange,
            canManageBoard: canManage,
            currentUserId: currentUserId,
            reactionMode: BoardReactionMode.fromAPI(board.reactionMode),
            canInteract: BoardsLogic.canInteract(board: board)
                && BoardsLogic.canWriteInteractions(board: board, canManage: canManage),
            assignmentLinked: BoardsLogic.assignmentLinked(board),
            sections: sections,
            siblings: siblings,
            showTimelineArrange: showTimeline,
            showMapArrange: showMap,
            onEdit: { onEdit(post) },
            onDelete: { onDelete(post) },
            onArrange: { onArrange(post, $0) }
        )
    }
}
