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
import androidx.compose.material3.ScrollableTabRow
import androidx.compose.material3.Tab
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.lextures.android.R
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.lms.BoardModerationApi
import com.lextures.android.core.lms.BoardModerationQueue
import com.lextures.android.core.lms.BoardPost
import com.lextures.android.core.lms.BoardReport
import com.lextures.android.core.lms.BoardsLogic
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsErrorBanner
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ModerationQueueScreen(
    courseCode: String,
    boardId: String,
    accessToken: String?,
    onDismiss: () -> Unit,
    onChanged: () -> Unit,
) {
    val sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)
    val scope = rememberCoroutineScope()
    var tab by remember { mutableIntStateOf(0) }
    var queue by remember { mutableStateOf<BoardModerationQueue?>(null) }
    var loading by remember { mutableStateOf(true) }
    var busyId by remember { mutableStateOf<String?>(null) }
    var errorMessage by remember { mutableStateOf<String?>(null) }

    val title = L.text(R.string.mobile_boards_moderation_title)
    val loadError = L.text(R.string.mobile_boards_moderation_loadError)
    val actionError = L.text(R.string.mobile_boards_moderation_actionError)
    val tabs = listOf(
        L.text(R.string.mobile_boards_moderation_tabPending) to (queue?.pending?.size ?: 0),
        L.text(R.string.mobile_boards_moderation_tabReports) to (queue?.reports?.size ?: 0),
        L.text(R.string.mobile_boards_moderation_tabFlagged) to (queue?.flagged?.size ?: 0),
    )

    suspend fun load() {
        val token = accessToken ?: return
        loading = true
        errorMessage = null
        try {
            queue = BoardModerationApi.fetchModerationQueue(courseCode, boardId, token)
        } catch (_: Exception) {
            errorMessage = loadError
        } finally {
            loading = false
        }
    }

    LaunchedEffect(courseCode, boardId, accessToken) { load() }

    ModalBottomSheet(onDismissRequest = onDismiss, sheetState = sheetState) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 16.dp, vertical = 8.dp)
                .semantics { contentDescription = title },
            verticalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            Text(title, fontWeight = FontWeight.SemiBold, color = textPrimary())
            ScrollableTabRow(selectedTabIndex = tab, edgePadding = 0.dp) {
                tabs.forEachIndexed { index, (label, count) ->
                    Tab(
                        selected = tab == index,
                        onClick = { tab = index },
                        text = { Text("$label ($count)") },
                    )
                }
            }
            errorMessage?.let { LmsErrorBanner(it) }
            when {
                loading && queue == null -> CircularProgressIndicator(modifier = Modifier.padding(16.dp))
                tab == 0 -> PendingList(
                    posts = queue?.pending.orEmpty(),
                    busyId = busyId,
                    onApprove = { id ->
                        scope.launch {
                            val token = accessToken ?: return@launch
                            busyId = id
                            try {
                                BoardModerationApi.approvePost(courseCode, boardId, id, accessToken = token)
                                load()
                                onChanged()
                            } catch (_: Exception) {
                                errorMessage = actionError
                            } finally {
                                busyId = null
                            }
                        }
                    },
                    onReject = { id ->
                        scope.launch {
                            val token = accessToken ?: return@launch
                            busyId = id
                            try {
                                BoardModerationApi.rejectPost(courseCode, boardId, id, accessToken = token)
                                load()
                                onChanged()
                            } catch (_: Exception) {
                                errorMessage = actionError
                            } finally {
                                busyId = null
                            }
                        }
                    },
                )
                else -> ReportList(
                    reports = if (tab == 1) queue?.reports.orEmpty() else queue?.flagged.orEmpty(),
                    busyId = busyId,
                    onAction = { id, action ->
                        scope.launch {
                            val token = accessToken ?: return@launch
                            busyId = id
                            try {
                                BoardModerationApi.resolveReport(
                                    courseCode, boardId, id, action, accessToken = token,
                                )
                                load()
                                onChanged()
                            } catch (_: Exception) {
                                errorMessage = actionError
                            } finally {
                                busyId = null
                            }
                        }
                    },
                )
            }
            TextButton(onClick = onDismiss, modifier = Modifier.fillMaxWidth()) {
                Text(L.text(R.string.mobile_common_close))
            }
        }
    }
}

@Composable
private fun PendingList(
    posts: List<BoardPost>,
    busyId: String?,
    onApprove: (String) -> Unit,
    onReject: (String) -> Unit,
) {
    if (posts.isEmpty()) {
        Text(L.text(R.string.mobile_boards_moderation_emptyPending), color = textSecondary())
        return
    }
    LazyColumn(verticalArrangement = Arrangement.spacedBy(8.dp)) {
        items(posts, key = { it.id }) { post ->
            LmsCard {
                Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    Text(
                        post.title.ifBlank { L.text(R.string.mobile_boards_moderation_untitled) },
                        fontWeight = FontWeight.SemiBold,
                        color = textPrimary(),
                    )
                    val preview = BoardsLogic.bodyPlainText(post)
                    if (preview.isNotBlank()) {
                        Text(preview, color = textSecondary(), maxLines = 3)
                    }
                    Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        TextButton(enabled = busyId != post.id, onClick = { onApprove(post.id) }) {
                            Text(L.text(R.string.mobile_boards_moderation_approve))
                        }
                        TextButton(enabled = busyId != post.id, onClick = { onReject(post.id) }) {
                            Text(L.text(R.string.mobile_boards_moderation_reject))
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun ReportList(
    reports: List<BoardReport>,
    busyId: String?,
    onAction: (String, String) -> Unit,
) {
    if (reports.isEmpty()) {
        Text(L.text(R.string.mobile_boards_moderation_emptyReports), color = textSecondary())
        return
    }
    LazyColumn(verticalArrangement = Arrangement.spacedBy(8.dp)) {
        items(reports, key = { it.id }) { report ->
            LmsCard {
                Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    Text(kindLabel(report.kind), fontWeight = FontWeight.SemiBold, color = textPrimary())
                    if (report.reason.isNotBlank()) {
                        Text(report.reason, color = textSecondary())
                    }
                    Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        TextButton(enabled = busyId != report.id, onClick = { onAction(report.id, "dismiss") }) {
                            Text(L.text(R.string.mobile_boards_moderation_dismiss))
                        }
                        TextButton(enabled = busyId != report.id, onClick = { onAction(report.id, "hide") }) {
                            Text(L.text(R.string.mobile_boards_moderation_hide))
                        }
                        TextButton(enabled = busyId != report.id, onClick = { onAction(report.id, "remove") }) {
                            Text(L.text(R.string.mobile_boards_moderation_remove))
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun kindLabel(kind: String): String = when (kind.lowercase()) {
    "filter" -> L.text(R.string.mobile_boards_moderation_kind_filter)
    "av_blocked" -> L.text(R.string.mobile_boards_moderation_kind_av_blocked)
    else -> L.text(R.string.mobile_boards_moderation_kind_user)
}
