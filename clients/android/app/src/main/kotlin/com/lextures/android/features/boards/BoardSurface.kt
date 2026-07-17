package com.lextures.android.features.boards

import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import com.lextures.android.core.lms.ArrangeBoardPostBody
import com.lextures.android.core.lms.Board
import com.lextures.android.core.lms.BoardLayout
import com.lextures.android.core.lms.BoardPost
import com.lextures.android.core.lms.BoardReactionMode
import com.lextures.android.core.lms.BoardSection
import com.lextures.android.core.lms.BoardSortMode
import com.lextures.android.core.lms.BoardsLogic
import com.lextures.android.features.boards.layouts.CanvasLayout
import com.lextures.android.features.boards.layouts.ColumnsLayout
import com.lextures.android.features.boards.layouts.GridLayout
import com.lextures.android.features.boards.layouts.MapLayout
import com.lextures.android.features.boards.layouts.StreamLayout
import com.lextures.android.features.boards.layouts.TimelineLayout
import com.lextures.android.features.boards.layouts.WallLayout

/** Dispatches to a mobile layout renderer based on `board.layout` (VC.M3). */
@Composable
fun BoardSurface(
    board: Board,
    posts: List<BoardPost>,
    sections: List<BoardSection>,
    sortMode: BoardSortMode = BoardSortMode.Newest,
    canManage: Boolean,
    currentUserId: String?,
    onEdit: (BoardPost) -> Unit,
    onDelete: (BoardPost) -> Unit,
    onArrange: (BoardPost, ArrangeBoardPostBody) -> Unit,
    onCreateSection: ((String) -> Unit)? = null,
    onDeleteSection: ((String) -> Unit)? = null,
    modifier: Modifier = Modifier,
) {
    val reactionMode = BoardReactionMode.fromApi(board.reactionMode)
    when (BoardsLogic.resolveLayout(board.layout)) {
        BoardLayout.Wall -> WallLayout(
            posts = BoardsLogic.sortedPosts(posts, sortMode, reactionMode),
            sections = sections,
            board = board,
            canManage = canManage,
            currentUserId = currentUserId,
            onEdit = onEdit,
            onDelete = onDelete,
            onArrange = onArrange,
            modifier = modifier,
        )
        BoardLayout.Stream -> StreamLayout(
            posts = BoardsLogic.sortedPosts(posts, sortMode, reactionMode),
            sections = sections,
            board = board,
            canManage = canManage,
            currentUserId = currentUserId,
            onEdit = onEdit,
            onDelete = onDelete,
            onArrange = onArrange,
            modifier = modifier,
        )
        BoardLayout.Grid -> GridLayout(
            posts = BoardsLogic.sortedPosts(posts, sortMode, reactionMode),
            sections = sections,
            board = board,
            canManage = canManage,
            currentUserId = currentUserId,
            onEdit = onEdit,
            onDelete = onDelete,
            onArrange = onArrange,
            modifier = modifier,
        )
        BoardLayout.Columns -> ColumnsLayout(
            posts = posts,
            sections = sections,
            board = board,
            canManage = canManage,
            currentUserId = currentUserId,
            onEdit = onEdit,
            onDelete = onDelete,
            onArrange = onArrange,
            onCreateSection = onCreateSection,
            onDeleteSection = onDeleteSection,
            modifier = modifier,
        )
        BoardLayout.Canvas -> CanvasLayout(
            posts = posts,
            sections = sections,
            board = board,
            canManage = canManage,
            currentUserId = currentUserId,
            onEdit = onEdit,
            onDelete = onDelete,
            onArrange = onArrange,
            modifier = modifier,
        )
        BoardLayout.Timeline -> TimelineLayout(
            posts = posts,
            sections = sections,
            board = board,
            canManage = canManage,
            currentUserId = currentUserId,
            onEdit = onEdit,
            onDelete = onDelete,
            onArrange = onArrange,
            modifier = modifier,
        )
        BoardLayout.Map -> MapLayout(
            posts = posts,
            sections = sections,
            board = board,
            canManage = canManage,
            currentUserId = currentUserId,
            onEdit = onEdit,
            onDelete = onDelete,
            onArrange = onArrange,
            modifier = modifier,
        )
    }
}

@Composable
fun BoardPostCardSlot(
    post: BoardPost,
    siblings: List<BoardPost>,
    sections: List<BoardSection>,
    board: Board,
    canManage: Boolean,
    currentUserId: String?,
    onEdit: (BoardPost) -> Unit,
    onDelete: (BoardPost) -> Unit,
    onArrange: (BoardPost, ArrangeBoardPostBody) -> Unit,
    showTimeline: Boolean = false,
    showMap: Boolean = false,
    modifier: Modifier = Modifier,
) {
    BoardPostCard(
        post = post,
        canEdit = BoardsLogic.canEditOrDeletePost(post, currentUserId, canManage),
        onEdit = { onEdit(post) },
        onDelete = { onDelete(post) },
        modifier = modifier,
        canArrange = BoardsLogic.canArrangePost(post, board, currentUserId, canManage),
        canManageBoard = canManage,
        currentUserId = currentUserId,
        reactionMode = BoardReactionMode.fromApi(board.reactionMode),
        canInteract = BoardsLogic.canInteract(board) &&
            BoardsLogic.canWriteInteractions(board, canManage),
        assignmentLinked = BoardsLogic.assignmentLinked(board),
        sections = sections,
        siblings = siblings,
        showTimelineArrange = showTimeline,
        showMapArrange = showMap,
        onArrange = { onArrange(post, it) },
    )
}
