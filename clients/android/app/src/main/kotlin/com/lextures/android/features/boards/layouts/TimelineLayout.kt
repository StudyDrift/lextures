package com.lextures.android.features.boards.layouts

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.lextures.android.R
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.lms.ArrangeBoardPostBody
import com.lextures.android.core.lms.Board
import com.lextures.android.core.lms.BoardPost
import com.lextures.android.core.lms.BoardSection
import com.lextures.android.core.lms.BoardsLogic
import com.lextures.android.features.boards.BoardPostCardSlot
import java.time.Instant
import java.time.ZoneId
import java.time.format.DateTimeFormatter
import java.time.format.FormatStyle

@Composable
fun TimelineLayout(
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
    val dated = remember(posts) { BoardsLogic.datedPosts(posts) }
    val undated = remember(posts) { BoardsLogic.undatedPosts(posts) }
    val dateFmt = remember {
        DateTimeFormatter.ofLocalizedDate(FormatStyle.MEDIUM).withZone(ZoneId.systemDefault())
    }

    Column(modifier = modifier, verticalArrangement = Arrangement.spacedBy(16.dp)) {
        dated.forEach { post ->
            Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                Box(
                    modifier = Modifier
                        .padding(top = 6.dp)
                        .size(10.dp)
                        .background(MaterialTheme.colorScheme.primary, CircleShape),
                )
                Column(modifier = Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(6.dp)) {
                    Text(formatEventDate(post.eventDate, dateFmt), color = textSecondary())
                    BoardPostCardSlot(
                        post = post,
                        siblings = dated,
                        sections = sections,
                        board = board,
                        canManage = canManage,
                        currentUserId = currentUserId,
                        onEdit = onEdit,
                        onDelete = onDelete,
                        onArrange = onArrange,
                        showTimeline = true,
                    )
                }
            }
        }

        Column(
            modifier = Modifier
                .fillMaxWidth()
                .border(1.dp, textSecondary().copy(alpha = 0.4f), RoundedCornerShape(12.dp))
                .padding(12.dp),
            verticalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            Text(L.text(R.string.mobile_boards_layout_undatedTray), color = textPrimary())
            if (undated.isEmpty()) {
                Text(L.text(R.string.mobile_boards_layout_undatedEmpty), color = textSecondary())
            } else {
                undated.forEach { post ->
                    BoardPostCardSlot(
                        post = post,
                        siblings = undated,
                        sections = sections,
                        board = board,
                        canManage = canManage,
                        currentUserId = currentUserId,
                        onEdit = onEdit,
                        onDelete = onDelete,
                        onArrange = onArrange,
                        showTimeline = true,
                    )
                }
            }
        }
    }
}

private fun formatEventDate(raw: String?, fmt: DateTimeFormatter): String {
    if (raw.isNullOrBlank()) return ""
    return runCatching {
        val instant = Instant.parse(raw)
        fmt.format(instant)
    }.getOrElse {
        runCatching {
            fmt.format(java.time.LocalDate.parse(raw).atStartOfDay(ZoneId.systemDefault()).toInstant())
        }.getOrDefault(raw)
    }
}
