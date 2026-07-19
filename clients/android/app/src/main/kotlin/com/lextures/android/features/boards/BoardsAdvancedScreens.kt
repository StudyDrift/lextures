package com.lextures.android.features.boards

import android.content.Intent
import androidx.compose.foundation.clickable
import androidx.compose.foundation.gestures.detectHorizontalDragGestures
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.itemsIndexed
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FilterChip
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier.Modifier
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.unit.dp
import androidx.core.content.FileProvider
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.i18n.L
import com.lextures.android.core.lms.BoardAnalyticsApi
import com.lextures.android.core.lms.BoardAnalyticsSummary
import com.lextures.android.core.lms.BoardCopyMode
import com.lextures.android.core.lms.BoardExportApi
import com.lextures.android.core.lms.BoardExportFormat
import com.lextures.android.core.lms.BoardPost
import com.lextures.android.core.lms.BoardSection
import com.lextures.android.core.lms.BoardTemplate
import com.lextures.android.core.lms.BoardTemplateScope
import com.lextures.android.core.lms.BoardTemplatesApi
import com.lextures.android.core.lms.BoardsAdvancedLogic
import com.lextures.android.core.lms.BoardsAdvancedObservability
import com.lextures.android.core.lms.Board
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsErrorBanner
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import java.io.File

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun BoardTemplatePickerSheet(
    courseCode: String,
    session: AuthSession,
    onCreated: (Board) -> Unit,
    onDismiss: () -> Unit,
) {
    var templates by remember { mutableStateOf<List<BoardTemplate>>(emptyList()) }
    var scope by remember { mutableStateOf<BoardTemplateScope?>(null) }
    var query by remember { mutableStateOf("") }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var creatingId by remember { mutableStateOf<String?>(null) }
    val scopeIo = rememberCoroutineScope()

    val visible = remember(templates, scope, query) {
        BoardsAdvancedLogic.filterTemplates(templates, scope, query)
    }

    LaunchedEffect(courseCode) {
        loading = true
        error = null
        try {
            val token = session.accessToken.value ?: return@LaunchedEffect
            templates = BoardTemplatesApi.listTemplates(courseCode = courseCode, accessToken = token)
        } catch (_: Exception) {
            error = L.text(R.string.mobile_boards_templates_loadError)
        } finally {
            loading = false
        }
    }

    Column(Modifier = Modifier.fillMaxWidth().padding(16.dp)) {
        Text(L.text(R.string.mobile_boards_templates_title), style = MaterialTheme.typography.titleLarge)
        Spacer(Modifier = Modifier.height(8.dp))
        OutlinedTextField(
            value = query,
            onValueChange = { query = it },
            modifier = Modifier.fillMaxWidth(),
            label = { Text(L.text(R.string.mobile_boards_templates_search)) },
        )
        Row(horizontalArrangement = Arrangement.spacedBy(8.dp), modifier = Modifier.padding(vertical = 8.dp)) {
            FilterChip(
                selected = scope == null,
                onClick = { scope = null },
                label = { Text(L.text(R.string.mobile_boards_templates_scopeAll)) },
            )
            BoardTemplateScope.entries.forEach { s ->
                FilterChip(
                    selected = scope == s,
                    onClick = { scope = s },
                    label = {
                        Text(
                            when (s) {
                                BoardTemplateScope.Builtin -> L.text(R.string.mobile_boards_templates_scopeBuiltin)
                                BoardTemplateScope.Course -> L.text(R.string.mobile_boards_templates_scopeCourse)
                                BoardTemplateScope.Org -> L.text(R.string.mobile_boards_templates_scopeOrg)
                            },
                        )
                    },
                )
            }
        }
        error?.let { LmsErrorBanner(it) }
        when {
            loading && templates.isEmpty() -> CircularProgressIndicator(modifier = Modifier.align(Alignment.CenterHorizontally))
            visible.isEmpty() -> Text(L.text(R.string.mobile_boards_templates_emptyTitle), color = textSecondary())
            else -> LazyColumn(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                items(visible, key = { it.id }) { template ->
                    LmsCard {
                        Column(
                            Modifier
                                .fillMaxWidth()
                                .clickable(enabled = creatingId == null) {
                                    scopeIo.launch {
                                        val token = session.accessToken.value ?: return@launch
                                        creatingId = template.id
                                        try {
                                            val board = BoardTemplatesApi.createFromTemplate(
                                                courseCode = courseCode,
                                                templateId = template.id,
                                                title = template.title,
                                                accessToken = token,
                                            )
                                            BoardsAdvancedObservability.record(
                                                "board_template_used",
                                                mapOf("scope" to template.scope),
                                            )
                                            onCreated(board)
                                            onDismiss()
                                        } catch (_: Exception) {
                                            error = L.text(R.string.mobile_boards_templates_createError)
                                        } finally {
                                            creatingId = null
                                        }
                                    }
                                }
                                .padding(12.dp),
                        ) {
                            Text(template.title, color = textPrimary(), style = MaterialTheme.typography.titleMedium)
                            if (template.description.isNotBlank()) {
                                Text(template.description, color = textSecondary(), maxLines = 2)
                            }
                        }
                    }
                }
            }
        }
        TextButton(onClick = onDismiss, modifier = Modifier.align(Alignment.End)) {
            Text(L.text(R.string.mobile_common_cancel))
        }
    }
}

@Composable
fun BoardSaveAsTemplateSheet(
    courseCode: String,
    boardId: String,
    defaultTitle: String,
    session: AuthSession,
    onDismiss: () -> Unit,
) {
    var title by remember { mutableStateOf(defaultTitle) }
    var description by remember { mutableStateOf("") }
    var scope by remember { mutableStateOf("course") }
    var includePosts by remember { mutableStateOf(false) }
    var saving by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    val scopeIo = rememberCoroutineScope()

    Column(Modifier = Modifier.fillMaxWidth().padding(16.dp).verticalScroll(rememberScrollState())) {
        Text(L.text(R.string.mobile_boards_templates_saveAction), style = MaterialTheme.typography.titleLarge)
        OutlinedTextField(
            value = title,
            onValueChange = { title = it },
            modifier = Modifier.fillMaxWidth().padding(top = 8.dp),
            label = { Text(L.text(R.string.mobile_boards_templates_saveTitle)) },
        )
        OutlinedTextField(
            value = description,
            onValueChange = { description = it },
            modifier = Modifier.fillMaxWidth().padding(top = 8.dp),
            label = { Text(L.text(R.string.mobile_boards_templates_saveDescription)) },
        )
        Row(verticalAlignment = Alignment.CenterVertically, modifier = Modifier.padding(top = 8.dp)) {
            Text(L.text(R.string.mobile_boards_templates_includePosts), modifier = Modifier.weight(1f))
            Switch(checked = includePosts, onCheckedChange = { includePosts = it })
        }
        Row(horizontalArrangement = Arrangement.spacedBy(8.dp), modifier = Modifier.padding(vertical = 8.dp)) {
            FilterChip(selected = scope == "course", onClick = { scope = "course" }, label = {
                Text(L.text(R.string.mobile_boards_templates_scopeCourse))
            })
            FilterChip(selected = scope == "org", onClick = { scope = "org" }, label = {
                Text(L.text(R.string.mobile_boards_templates_scopeOrg))
            })
        }
        error?.let { LmsErrorBanner(it) }
        Row(horizontalArrangement = Arrangement.spacedBy(8.dp), modifier = Modifier.align(Alignment.End)) {
            TextButton(onClick = onDismiss) { Text(L.text(R.string.mobile_common_cancel)) }
            Button(
                enabled = title.isNotBlank() && !saving,
                onClick = {
                    scopeIo.launch {
                        val token = session.accessToken.value ?: return@launch
                        saving = true
                        try {
                            BoardTemplatesApi.saveAsTemplate(
                                courseCode = courseCode,
                                boardId = boardId,
                                scope = scope,
                                title = title.trim(),
                                description = description.trim(),
                                includePosts = includePosts,
                                accessToken = token,
                            )
                            BoardsAdvancedObservability.record(
                                "board_saved_as_template",
                                mapOf("scope" to scope, "include_posts" to if (includePosts) "1" else "0"),
                            )
                            onDismiss()
                        } catch (_: Exception) {
                            error = L.text(R.string.mobile_boards_templates_saveError)
                        } finally {
                            saving = false
                        }
                    }
                },
            ) { Text(L.text(R.string.mobile_common_save)) }
        }
    }
}

@Composable
fun BoardExportSheet(
    courseCode: String,
    boardId: String,
    boardTitle: String,
    session: AuthSession,
    onDismiss: () -> Unit,
) {
    val context = LocalContext.current
    var includeModeration by remember { mutableStateOf(false) }
    var busy by remember { mutableStateOf<BoardExportFormat?>(null) }
    var status by remember { mutableStateOf<String?>(null) }
    var error by remember { mutableStateOf<String?>(null) }
    val scopeIo = rememberCoroutineScope()

    Column(modifier = Modifier.fillMaxWidth().padding(16.dp)) {
        Text(L.text(R.string.mobile_boards_export_title), style = MaterialTheme.typography.titleLarge)
        Row(verticalAlignment = Alignment.CenterVertically, modifier = Modifier.padding(vertical = 8.dp)) {
            Text(L.text(R.string.mobile_boards_export_includeModeration), modifier = Modifier.weight(1f))
            Switch(checked = includeModeration, onCheckedChange = { includeModeration = it })
        }
        BoardExportFormat.entries.forEach { format ->
            val label = when (format) {
                BoardExportFormat.Pdf -> L.text(R.string.mobile_boards_export_formatPdf)
                BoardExportFormat.Csv -> L.text(R.string.mobile_boards_export_formatCsv)
                BoardExportFormat.Image -> L.text(R.string.mobile_boards_export_formatImage)
            }
            Button(
                enabled = busy == null,
                onClick = {
                    scopeIo.launch {
                        val token = session.accessToken.value ?: return@launch
                        busy = format
                        error = null
                        status = L.text(R.string.mobile_boards_export_queued)
                        try {
                            var job = BoardExportApi.createExport(
                                courseCode, boardId, format, includeModeration, token,
                            )
                            var attempt = 0
                            while (!BoardsAdvancedLogic.isExportTerminal(job.status)) {
                                status = L.text(R.string.mobile_boards_export_running)
                                delay((BoardsAdvancedLogic.pollDelaySeconds(attempt) * 1000).toLong())
                                job = BoardExportApi.fetchExportJob(courseCode, boardId, job.id, token)
                                attempt++
                                if (attempt > 20) break
                            }
                            if (!job.status.equals("done", ignoreCase = true)) {
                                throw IllegalStateException(job.error)
                            }
                            status = L.text(R.string.mobile_boards_export_ready)
                            val bytes = BoardExportApi.downloadExport(courseCode, boardId, job.id, token)
                            val ext = BoardsAdvancedLogic.exportFileExtension(format)
                            val mime = BoardsAdvancedLogic.exportMimeType(format)
                            val safe = (boardTitle.ifBlank { "board" }).replace("/", "-")
                            val file = File(context.cacheDir, "$safe.$ext")
                            file.writeBytes(bytes)
                            val uri = FileProvider.getUriForFile(
                                context,
                                "${context.packageName}.fileprovider",
                                file,
                            )
                            val share = Intent(Intent.ACTION_SEND).apply {
                                type = mime
                                putExtra(Intent.EXTRA_STREAM, uri)
                                addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION)
                            }
                            context.startActivity(Intent.createChooser(share, label))
                            BoardsAdvancedObservability.record("board_exported", mapOf("format" to format.apiValue))
                        } catch (_: Exception) {
                            error = L.text(R.string.mobile_boards_export_failed)
                            status = null
                        } finally {
                            busy = null
                        }
                    }
                },
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(vertical = 4.dp)
                    .semantics { contentDescription = label },
            ) {
                if (busy == format) CircularProgressIndicator(modifier = Modifier.height(18.dp))
                else Text(label)
            }
        }
        status?.let { Text(it, color = textSecondary(), modifier = Modifier.padding(top = 8.dp)) }
        error?.let { LmsErrorBanner(it) }
        TextButton(onClick = onDismiss, modifier = Modifier.align(Alignment.End)) {
            Text(L.text(R.string.mobile_common_close))
        }
    }
}

@Composable
fun BoardPresentModeScreen(
    boardTitle: String,
    posts: List<BoardPost>,
    sections: List<BoardSection>,
    onClose: () -> Unit,
) {
    val ordered = remember(posts, sections) {
        BoardsAdvancedLogic.orderedPostsForPresent(posts, sections)
    }
    var index by remember { mutableIntStateOf(0) }
    var overview by remember { mutableStateOf(false) }

    LaunchedEffect(Unit) {
        BoardsAdvancedObservability.record("board_presented")
    }

    Box(
        modifier = Modifier
            .fillMaxSize()
            .padding(0.dp),
    ) {
        Column(
            Modifier
                .fillMaxSize()
                .padding(16.dp)
                .pointerInput(ordered.size, index) {
                    detectHorizontalDragGestures { _, dragAmount ->
                        if (dragAmount < -40) index = (index + 1).coerceAtMost((ordered.size - 1).coerceAtLeast(0))
                        else if (dragAmount > 40) index = (index - 1).coerceAtLeast(0)
                    }
                },
        ) {
            Row(Modifier = Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
                Text(boardTitle, style = MaterialTheme.typography.titleMedium, modifier = Modifier.weight(1f))
                TextButton(onClick = { overview = !overview }) {
                    Text(
                        if (overview) L.text(R.string.mobile_boards_present_slideshow)
                        else L.text(R.string.mobile_boards_present_overview),
                    )
                }
                TextButton(onClick = onClose) { Text(L.text(R.string.mobile_boards_present_close)) }
            }
            Spacer(Modifier = Modifier.height(16.dp))
            if (overview) {
                LazyColumn(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    itemsIndexed(ordered, key = { _, p -> p.id }) { idx, post ->
                        LmsCard {
                            Text(
                                post.title.ifBlank { BoardsAdvancedLogic.postBodyText(post) },
                                modifier = Modifier
                                    .fillMaxWidth()
                                    .clickable {
                                        index = idx
                                        overview = false
                                    }
                                    .padding(12.dp),
                                color = textPrimary(),
                            )
                        }
                    }
                }
            } else if (ordered.isEmpty()) {
                Text(L.text(R.string.mobile_boards_present_empty), color = textSecondary())
            } else {
                val post = ordered[index.coerceIn(0, ordered.lastIndex)]
                Column(Modifier = Modifier.weight(1f), verticalArrangement = Arrangement.Center) {
                    if (post.title.isNotBlank()) {
                        Text(post.title, style = MaterialTheme.typography.headlineMedium, color = textPrimary())
                    }
                    Text(
                        BoardsAdvancedLogic.postBodyText(post),
                        style = MaterialTheme.typography.titleMedium,
                        color = textPrimary(),
                        modifier = Modifier.padding(top = 12.dp),
                    )
                }
                Row(modifier = Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
                    TextButton(onClick = { index = (index - 1).coerceAtLeast(0) }, enabled = index > 0) {
                        Text(L.text(R.string.mobile_boards_present_prev))
                    }
                    Spacer(modifier = Modifier.weight(1f))
                    Text("${index + 1} / ${ordered.size}", color = textSecondary())
                    Spacer(modifier = Modifier.weight(1f))
                    TextButton(
                        onClick = { index = (index + 1).coerceAtMost(ordered.lastIndex) },
                        enabled = index < ordered.lastIndex,
                    ) {
                        Text(L.text(R.string.mobile_boards_present_next))
                    }
                }
            }
        }
    }
}

@Composable
fun BoardAnalyticsSheet(
    courseCode: String,
    boardId: String,
    session: AuthSession,
    onDismiss: () -> Unit,
) {
    var summary by remember { mutableStateOf<BoardAnalyticsSummary?>(null) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }

    LaunchedEffect(boardId) {
        BoardsAdvancedObservability.record("board_admin_analytics_viewed")
        loading = true
        try {
            val token = session.accessToken.value ?: return@LaunchedEffect
            summary = BoardAnalyticsApi.fetchBoardAnalytics(courseCode, boardId, accessToken = token)
        } catch (_: Exception) {
            error = L.text(R.string.mobile_boards_analytics_loadError)
        } finally {
            loading = false
        }
    }

    Column(modifier = Modifier.fillMaxWidth().padding(16.dp).verticalScroll(rememberScrollState())) {
        Text(L.text(R.string.mobile_boards_analytics_title), style = MaterialTheme.typography.titleLarge)
        Text(L.text(R.string.mobile_boards_analytics_subtitle), color = textSecondary())
        Spacer(Modifier = Modifier.height(12.dp))
        when {
            loading && summary == null -> CircularProgressIndicator()
            error != null && summary == null -> LmsErrorBanner(error!!)
            summary != null -> {
                val s = summary!!
                Text("${L.text(R.string.mobile_boards_analytics_cards)}: ${s.cardCount}")
                Text("${L.text(R.string.mobile_boards_analytics_contributors)}: ${s.uniqueContributors}")
                Text("${L.text(R.string.mobile_boards_analytics_reactions)}: ${s.reactionCount}")
                Text("${L.text(R.string.mobile_boards_analytics_comments)}: ${s.commentCount}")
            }
            else -> Text(L.text(R.string.mobile_boards_analytics_empty), color = textSecondary())
        }
        TextButton(onClick = onDismiss, modifier = Modifier.align(Alignment.End)) {
            Text(L.text(R.string.mobile_common_close))
        }
    }
}

/** Convenience for list-screen duplicate flow. */
suspend fun duplicateBoardWithPolling(
    courseCode: String,
    sourceBoardId: String,
    mode: BoardCopyMode,
    title: String,
    accessToken: String,
): Board? {
    return when (
        val result = BoardTemplatesApi.duplicateBoard(
            targetCourseCode = courseCode,
            sourceBoardId = sourceBoardId,
            mode = mode,
            title = title,
            accessToken = accessToken,
        )
    ) {
        is com.lextures.android.core.lms.BoardCreateResult.BoardResult -> result.board
        is com.lextures.android.core.lms.BoardCreateResult.JobResult -> {
            var current = result.job
            var attempt = 0
            while (!BoardsAdvancedLogic.isCopyTerminal(current.status)) {
                delay((BoardsAdvancedLogic.pollDelaySeconds(attempt) * 1000).toLong())
                current = BoardTemplatesApi.fetchCopyJob(courseCode, current.id, accessToken)
                attempt++
                if (attempt > 30) break
            }
            val id = current.resultBoardId?.takeIf { it.isNotBlank() } ?: return null
            Board(id = id, courseId = "", title = current.title.ifBlank { title })
        }
    }
}
