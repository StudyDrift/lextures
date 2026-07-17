package com.lextures.android.features.boards

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.heading
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.lextures.android.R
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.lms.BoardComment
import com.lextures.android.core.lms.BoardEngagementApi
import com.lextures.android.core.lms.BoardsLogic
import com.lextures.android.core.network.ApiError
import com.lextures.android.features.home.LmsErrorBanner
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun CommentSheet(
    courseCode: String,
    boardId: String,
    postId: String,
    accessToken: String?,
    canInteract: Boolean,
    canManageBoard: Boolean,
    currentUserId: String?,
    onCountChange: (Int) -> Unit,
    onDismiss: () -> Unit,
) {
    val sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)
    val scope = rememberCoroutineScope()
    var comments by remember { mutableStateOf<List<BoardComment>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var draft by remember { mutableStateOf("") }
    var replyTo by remember { mutableStateOf<String?>(null) }
    var busy by remember { mutableStateOf(false) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var reportCommentId by remember { mutableStateOf<String?>(null) }

    val errorGeneric = L.text(R.string.mobile_boards_comment_error)
    val forbiddenMsg = L.text(R.string.mobile_boards_react_forbidden)
    val threadAria = L.text(R.string.mobile_boards_comment_threadAria)

    suspend fun load() {
        val token = accessToken ?: return
        loading = true
        errorMessage = null
        try {
            comments = BoardEngagementApi.listComments(courseCode, boardId, postId, token)
        } catch (_: Exception) {
            errorMessage = errorGeneric
        } finally {
            loading = false
        }
    }

    LaunchedEffect(courseCode, boardId, postId, accessToken) {
        load()
    }

    ModalBottomSheet(onDismissRequest = onDismiss, sheetState = sheetState) {
        Column(modifier = Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 8.dp)) {
            Text(
                L.text(R.string.mobile_boards_comment_threadHeading),
                fontWeight = FontWeight.SemiBold,
                color = textSecondary(),
                modifier = Modifier
                    .padding(bottom = 8.dp)
                    .semantics { heading() },
            )

            when {
                loading -> CircularProgressIndicator(modifier = Modifier.padding(16.dp))
                errorMessage != null && comments.isEmpty() -> {
                    LmsErrorBanner(errorMessage!!)
                    TextButton(onClick = { scope.launch { load() } }) {
                        Text(L.text(R.string.mobile_common_retry))
                    }
                }
                else -> {
                    val nested = BoardsLogic.nestComments(
                        BoardsLogic.visibleComments(comments, canManageBoard),
                    )
                    if (nested.isEmpty()) {
                        Text(L.text(R.string.mobile_boards_comment_empty), color = textSecondary())
                    } else {
                        LazyColumn(
                            modifier = Modifier
                                .fillMaxWidth()
                                .semantics { contentDescription = threadAria },
                            verticalArrangement = Arrangement.spacedBy(8.dp),
                        ) {
                            items(nested, key = { it.comment.id }) { row ->
                                CommentRow(
                                    comment = row.comment,
                                    currentUserId = currentUserId,
                                    canInteract = canInteract,
                                    canManageBoard = canManageBoard,
                                    onReply = { replyTo = row.comment.id },
                                    onReport = { reportCommentId = row.comment.id },
                                    onHide = {
                                        scope.launch {
                                            val token = accessToken ?: return@launch
                                            if (!canManageBoard || busy) return@launch
                                            busy = true
                                            try {
                                                val updated = BoardEngagementApi.patchComment(
                                                    courseCode, boardId, postId, row.comment.id,
                                                    hidden = true, accessToken = token,
                                                )
                                                comments = comments.map {
                                                    if (it.id == updated.id) updated else it
                                                }
                                                onCountChange(-1)
                                            } catch (_: Exception) {
                                                errorMessage = errorGeneric
                                            } finally {
                                                busy = false
                                            }
                                        }
                                    },
                                    onDelete = {
                                        scope.launch {
                                            val token = accessToken ?: return@launch
                                            if (busy) return@launch
                                            busy = true
                                            try {
                                                BoardEngagementApi.deleteComment(
                                                    courseCode, boardId, postId, row.comment.id, token,
                                                )
                                                comments = comments.map {
                                                    if (it.id == row.comment.id) it.copy(hidden = true) else it
                                                }
                                                onCountChange(-1)
                                            } catch (_: Exception) {
                                                errorMessage = errorGeneric
                                            } finally {
                                                busy = false
                                            }
                                        }
                                    },
                                )
                                row.children.forEach { child ->
                                    CommentRow(
                                        comment = child,
                                        currentUserId = currentUserId,
                                        canInteract = canInteract,
                                        canManageBoard = canManageBoard,
                                        indented = true,
                                        onReply = { replyTo = child.id },
                                        onReport = { reportCommentId = child.id },
                                        onHide = {
                                            scope.launch {
                                                val token = accessToken ?: return@launch
                                                if (!canManageBoard || busy) return@launch
                                                busy = true
                                                try {
                                                    val updated = BoardEngagementApi.patchComment(
                                                        courseCode, boardId, postId, child.id,
                                                        hidden = true, accessToken = token,
                                                    )
                                                    comments = comments.map {
                                                        if (it.id == updated.id) updated else it
                                                    }
                                                    onCountChange(-1)
                                                } catch (_: Exception) {
                                                    errorMessage = errorGeneric
                                                } finally {
                                                    busy = false
                                                }
                                            }
                                        },
                                        onDelete = {
                                            scope.launch {
                                                val token = accessToken ?: return@launch
                                                if (busy) return@launch
                                                busy = true
                                                try {
                                                    BoardEngagementApi.deleteComment(
                                                        courseCode, boardId, postId, child.id, token,
                                                    )
                                                    comments = comments.map {
                                                        if (it.id == child.id) it.copy(hidden = true) else it
                                                    }
                                                    onCountChange(-1)
                                                } catch (_: Exception) {
                                                    errorMessage = errorGeneric
                                                } finally {
                                                    busy = false
                                                }
                                            }
                                        },
                                    )
                                }
                            }
                        }
                    }
                }
            }

            if (canInteract) {
                if (replyTo != null) {
                    Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        Text(L.text(R.string.mobile_boards_comment_replying), color = textSecondary())
                        TextButton(onClick = { replyTo = null }) {
                            Text(L.text(R.string.mobile_boards_comment_cancelReply))
                        }
                    }
                }
                errorMessage?.takeIf { comments.isNotEmpty() }?.let {
                    Text(it, color = androidx.compose.ui.graphics.Color.Red)
                }
                OutlinedTextField(
                    value = draft,
                    onValueChange = { if (it.length <= 4000) draft = it },
                    modifier = Modifier.fillMaxWidth().padding(top = 8.dp),
                    placeholder = { Text(L.text(R.string.mobile_boards_comment_placeholder)) },
                    label = { Text(L.text(R.string.mobile_boards_comment_add)) },
                    minLines = 2,
                    maxLines = 5,
                )
                TextButton(
                    onClick = {
                        scope.launch {
                            val token = accessToken ?: return@launch
                            val text = draft.trim()
                            if (!canInteract || text.isEmpty() || busy) return@launch
                            busy = true
                            errorMessage = null
                            try {
                                val created = BoardEngagementApi.createComment(
                                    courseCode, boardId, postId,
                                    body = BoardsLogic.makeTextBody(text),
                                    parentId = replyTo,
                                    accessToken = token,
                                )
                                comments = comments + created
                                draft = ""
                                replyTo = null
                                onCountChange(1)
                            } catch (e: ApiError.HttpStatus) {
                                errorMessage = if (e.code == 403) forbiddenMsg else errorGeneric
                            } catch (_: Exception) {
                                errorMessage = errorGeneric
                            } finally {
                                busy = false
                            }
                        }
                    },
                    enabled = !busy && draft.isNotBlank(),
                    modifier = Modifier.padding(vertical = 8.dp),
                ) {
                    Text(L.text(R.string.mobile_boards_comment_submit))
                }
            }
        }
    }

    reportCommentId?.let { commentId ->
        ReportDialog(
            courseCode = courseCode,
            boardId = boardId,
            accessToken = accessToken,
            commentId = commentId,
            onDismiss = { reportCommentId = null },
        )
    }
}

@Composable
private fun CommentRow(
    comment: BoardComment,
    currentUserId: String?,
    canInteract: Boolean,
    canManageBoard: Boolean,
    indented: Boolean = false,
    onReply: () -> Unit,
    onReport: () -> Unit,
    onHide: () -> Unit,
    onDelete: () -> Unit,
) {
    val isAuthor = !currentUserId.isNullOrBlank() &&
        !comment.authorId.isNullOrBlank() &&
        currentUserId.equals(comment.authorId, ignoreCase = true)

    if (comment.hidden && canManageBoard) {
        Text(
            L.text(R.string.mobile_boards_comment_hiddenPlaceholder),
            color = textSecondary(),
            fontStyle = androidx.compose.ui.text.font.FontStyle.Italic,
            modifier = Modifier
                .fillMaxWidth()
                .padding(start = if (indented) 16.dp else 0.dp, top = 4.dp, bottom = 4.dp),
        )
        return
    }

    Column(
        modifier = Modifier
            .fillMaxWidth()
            .padding(start = if (indented) 16.dp else 0.dp, top = 4.dp, bottom = 4.dp),
        verticalArrangement = Arrangement.spacedBy(4.dp),
    ) {
        Text(BoardsLogic.commentPlainText(comment), color = textPrimary())
        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            if (canInteract) {
                TextButton(onClick = onReply) { Text(L.text(R.string.mobile_boards_comment_reply)) }
            }
            TextButton(onClick = onReport) { Text(L.text(R.string.mobile_boards_report_action)) }
            if (isAuthor) {
                TextButton(onClick = onDelete) { Text(L.text(R.string.mobile_boards_comment_delete)) }
            }
            if (canManageBoard && !comment.hidden) {
                TextButton(onClick = onHide) { Text(L.text(R.string.mobile_boards_comment_hide)) }
            }
        }
    }
}
