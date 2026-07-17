package com.lextures.android.features.boards.layouts

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.GridView
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.lextures.android.R
import com.lextures.android.core.i18n.L
import com.lextures.android.core.lms.ArrangeBoardPostBody
import com.lextures.android.core.lms.Board
import com.lextures.android.core.lms.BoardPost
import com.lextures.android.core.lms.BoardSection
import com.lextures.android.features.boards.BoardPostCardSlot
import com.lextures.android.features.home.LmsEmptyState

@Composable
fun WallLayout(
    posts: List<BoardPost>,
    sections: List<BoardSection>,
    board: Board,
    canManage: Boolean,
    currentUserId: String?,
    onEdit: (BoardPost) -> Unit,
    onDelete: (BoardPost) -> Unit,
    onArrange: (BoardPost, ArrangeBoardPostBody) -> Unit,
    modifier: Modifier = Modifier,
) {
    if (posts.isEmpty()) {
        BoardPostsEmpty(modifier)
        return
    }
    LazyVerticalGrid(
        columns = GridCells.Fixed(2),
        modifier = modifier.fillMaxWidth().heightIn(max = 4000.dp),
        contentPadding = PaddingValues(bottom = 8.dp),
        horizontalArrangement = Arrangement.spacedBy(12.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp),
        userScrollEnabled = false,
    ) {
        items(posts, key = { it.id }) { post ->
            BoardPostCardSlot(
                post = post,
                siblings = posts,
                sections = sections,
                board = board,
                canManage = canManage,
                currentUserId = currentUserId,
                onEdit = onEdit,
                onDelete = onDelete,
                onArrange = onArrange,
            )
        }
    }
}

@Composable
fun StreamLayout(
    posts: List<BoardPost>,
    sections: List<BoardSection>,
    board: Board,
    canManage: Boolean,
    currentUserId: String?,
    onEdit: (BoardPost) -> Unit,
    onDelete: (BoardPost) -> Unit,
    onArrange: (BoardPost, ArrangeBoardPostBody) -> Unit,
    modifier: Modifier = Modifier,
) {
    if (posts.isEmpty()) {
        BoardPostsEmpty(modifier)
        return
    }
    Column(modifier = modifier, verticalArrangement = Arrangement.spacedBy(12.dp)) {
        posts.forEach { post ->
            BoardPostCardSlot(
                post = post,
                siblings = posts,
                sections = sections,
                board = board,
                canManage = canManage,
                currentUserId = currentUserId,
                onEdit = onEdit,
                onDelete = onDelete,
                onArrange = onArrange,
            )
        }
    }
}

@Composable
fun GridLayout(
    posts: List<BoardPost>,
    sections: List<BoardSection>,
    board: Board,
    canManage: Boolean,
    currentUserId: String?,
    onEdit: (BoardPost) -> Unit,
    onDelete: (BoardPost) -> Unit,
    onArrange: (BoardPost, ArrangeBoardPostBody) -> Unit,
    modifier: Modifier = Modifier,
) {
    WallLayout(
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

@Composable
fun BoardPostsEmpty(modifier: Modifier = Modifier) {
    LmsEmptyState(
        icon = Icons.Default.GridView,
        title = L.text(R.string.mobile_boards_postsEmptyTitle),
        message = L.text(R.string.mobile_boards_postsEmptyMessage),
        modifier = modifier,
    )
}
