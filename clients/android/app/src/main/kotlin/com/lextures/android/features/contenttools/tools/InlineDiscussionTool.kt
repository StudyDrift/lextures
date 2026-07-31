package com.lextures.android.features.contenttools.tools

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontStyle
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.compose.LocalLifecycleOwner
import androidx.lifecycle.repeatOnLifecycle
import com.lextures.android.R
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalLocalePreferences
import com.lextures.android.core.lms.ContentToolHostLogic
import com.lextures.android.core.lms.ContentToolPack2Logic
import com.lextures.android.core.offline.OfflineService
import com.lextures.android.features.contenttools.ContentToolDraftStore
import com.lextures.android.features.contenttools.ContentToolRendererProps
import com.lextures.android.features.contenttools.ToolComposer
import com.lextures.android.features.notebooks.NotebookContentView
import kotlinx.coroutines.launch
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.jsonObject

private data class DiscussionPost(
    val id: String,
    val parentPostId: String?,
    val text: String,
    val authorDisplay: String?,
    val upvoteCount: Int,
    val endorsed: Boolean,
    val removed: Boolean,
    val tombstone: Boolean,
    val isOwn: Boolean,
    val canEdit: Boolean,
    val canDelete: Boolean,
    val createdAt: String?,
)

@Composable
fun InlineDiscussionTool(props: ContentToolRendererProps) {
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    val scope = rememberCoroutineScope()
    val offline = remember { OfflineService.get(context.applicationContext) }
    val online by offline.networkMonitor.isOnline.collectAsState()
    val draftStore = remember { ContentToolDraftStore.create(context) }
    val lifecycleOwner = LocalLifecycleOwner.current

    var draft by remember(props.instanceId) { mutableStateOf("") }
    var busy by remember { mutableStateOf(false) }
    var errorText by remember { mutableStateOf<String?>(null) }
    var posts by remember { mutableStateOf<List<DiscussionPost>>(emptyList()) }
    var locked by remember { mutableStateOf(false) }
    var page by remember { mutableIntStateOf(1) }
    var pageSize by remember { mutableIntStateOf(ContentToolPack2Logic.DEFAULT_PAGE_SIZE) }
    var total by remember { mutableStateOf<Int?>(null) }
    var anonymity by remember { mutableStateOf("named") }
    var replyTo by remember { mutableStateOf<String?>(null) }
    var editingId by remember { mutableStateOf<String?>(null) }
    var reportTarget by remember { mutableStateOf<String?>(null) }
    var deleteTarget by remember { mutableStateOf<String?>(null) }
    var myPosts by remember { mutableIntStateOf(0) }
    var myReplies by remember { mutableIntStateOf(0) }
    var viewerCanEndorse by remember { mutableStateOf(false) }
    var viewerCanModerate by remember { mutableStateOf(false) }

    val draftKey = ContentToolPack2Logic.draftStorageKey(
        props.instanceId,
        editingId ?: replyTo ?: "composer",
    )
    val allowReplies = ContentToolPack2Logic.boolField(props.config, "allowReplies") != false
    val prompt = ContentToolHostLogic.stringField(props.config, "prompt").orEmpty()

    val emptyLabel = L.text(R.string.mobile_contentTools_tools_inline_discussion_empty)
    val lockedHint = L.text(R.string.mobile_contentTools_tools_inline_discussion_lockedHint)
    val label = L.text(R.string.mobile_contentTools_tools_inline_discussion_label)
    val loadMore = L.text(R.string.mobile_contentTools_tools_inline_discussion_loadMore)
    val cancelLabel = L.text(R.string.mobile_contentTools_runtime_cancel)
    val composerLabel = L.text(R.string.mobile_contentTools_tools_inline_discussion_composerLabel)
    val postLabel = L.text(R.string.mobile_contentTools_tools_inline_discussion_submitPost)
    val replyLabel = L.text(R.string.mobile_contentTools_tools_inline_discussion_submitReply)
    val saveEdit = L.text(R.string.mobile_contentTools_tools_inline_discussion_saveEdit)
    val offlineLabel = L.text(R.string.mobile_contentTools_runtime_offlineComposer)
    val postAnnounced = L.text(R.string.mobile_contentTools_tools_inline_discussion_postAnnounced)
    val reportThanks = L.text(R.string.mobile_contentTools_tools_inline_discussion_reportThanks)
    val tombstoneLabel = L.text(R.string.mobile_contentTools_tools_inline_discussion_tombstone)
    val classmate = L.text(R.string.mobile_contentTools_tools_inline_discussion_classmate)
    val loadError = L.text(R.string.mobile_contentTools_tools_inline_discussion_loadError)
    val postError = L.text(R.string.mobile_contentTools_tools_inline_discussion_postError)
    val reportError = L.text(R.string.mobile_contentTools_tools_inline_discussion_reportError)
    val deleteError = L.text(R.string.mobile_contentTools_tools_inline_discussion_deleteError)
    val endorseError = L.text(R.string.mobile_contentTools_tools_inline_discussion_endorseError)
    val forbiddenError = L.text(R.string.mobile_contentTools_tools_inline_discussion_error_forbidden)
    val retryLabel = L.text(R.string.mobile_contentTools_runtime_retry)
    val reportLabel = L.text(R.string.mobile_contentTools_tools_inline_discussion_report)
    val deleteLabel = L.text(R.string.mobile_contentTools_tools_inline_discussion_delete)

    suspend fun loadThread(targetPage: Int, append: Boolean) {
        busy = true
        try {
            val raw = props.runAction("thread", buildJsonObject { put("page", JsonPrimitive(targetPage)) })
            val result = ContentToolPack2Logic.objectMap(raw)
            locked = ContentToolPack2Logic.boolField(raw, "locked") == true
            anonymity = (result["anonymity"] as? JsonPrimitive)?.contentOrNull ?: anonymity
            pageSize = ContentToolPack2Logic.numberField(raw, "pageSize")?.toInt() ?: pageSize
            total = ContentToolPack2Logic.numberField(raw, "total")?.toInt()
            page = targetPage
            val parsed = ContentToolPack2Logic.arrayField(raw, "posts").mapNotNull { parsePost(it) }
            posts = if (append) posts + parsed else parsed
            result["requirements"]?.jsonObject?.let { req ->
                myPosts = ContentToolPack2Logic.numberField(
                    kotlinx.serialization.json.JsonObject(req),
                    "myPosts",
                )?.toInt() ?: 0
                myReplies = ContentToolPack2Logic.numberField(
                    kotlinx.serialization.json.JsonObject(req),
                    "myReplies",
                )?.toInt() ?: 0
            }
        } catch (_: Exception) {
            errorText = loadError
        } finally {
            busy = false
        }
    }

    LaunchedEffect(props.instanceId) {
        if (draft.isEmpty()) {
            draft = ContentToolHostLogic.stringField(props.state, "draft")
                ?: draftStore.load(draftKey)
        }
        loadThread(1, append = false)
    }

    LaunchedEffect(lifecycleOwner) {
        lifecycleOwner.lifecycle.repeatOnLifecycle(Lifecycle.State.STARTED) {
            loadThread(1, append = false)
        }
    }

    val roots = posts.filter { it.parentPostId.isNullOrBlank() }

    Column(
        modifier = Modifier.fillMaxWidth(),
        verticalArrangement = Arrangement.spacedBy(10.dp),
    ) {
        if (prompt.isNotBlank()) {
            NotebookContentView(markdown = prompt, compact = true)
        }
        if (anonymity == "anonymous_to_peers") {
            Text(
                L.text(R.string.mobile_contentTools_tools_inline_discussion_anonymityNote),
                fontSize = 11.sp,
                color = textSecondary(),
            )
        }
        Text(
            L.text(R.string.mobile_contentTools_tools_inline_discussion_moderationNote),
            fontSize = 11.sp,
            color = textSecondary(),
        )
        Text(
            L.format(R.string.mobile_contentTools_tools_inline_discussion_progress, myPosts, myReplies),
            fontSize = 11.sp,
        )

        when {
            locked -> Text(lockedHint, fontSize = 12.sp, color = textSecondary())
            posts.isEmpty() -> Text(emptyLabel, fontSize = 12.sp, color = textSecondary())
            else -> {
                LazyColumn(
                    modifier = Modifier
                        .fillMaxWidth()
                        .heightIn(max = 360.dp)
                        .semantics { contentDescription = label },
                ) {
                    items(roots, key = { it.id }) { post ->
                        PostRow(
                            post = post,
                            depth = 0,
                            anonymity = anonymity,
                            classmate = classmate,
                            tombstoneLabel = tombstoneLabel,
                            allowReplies = allowReplies,
                            readOnly = props.readOnly,
                            viewerCanEndorse = viewerCanEndorse,
                            viewerCanModerate = viewerCanModerate,
                            busy = busy,
                            onReply = { replyTo = it; editingId = null },
                            onEdit = {
                                editingId = it.id
                                replyTo = null
                                draft = it.text
                            },
                            onDelete = { deleteTarget = it },
                            onReport = { reportTarget = it },
                            onUpvote = {
                                scope.launch {
                                    busy = true
                                    runCatching {
                                        props.runAction("upvote", buildJsonObject { put("postId", JsonPrimitive(it)) })
                                        loadThread(1, false)
                                    }.onFailure {
                                        errorText = retryLabel
                                    }
                                    busy = false
                                }
                            },
                            onEndorse = {
                                scope.launch {
                                    busy = true
                                    runCatching {
                                        props.runAction("endorse", buildJsonObject { put("postId", JsonPrimitive(it)) })
                                        viewerCanEndorse = true
                                        loadThread(1, false)
                                    }.onFailure {
                                        viewerCanEndorse = false
                                        errorText = endorseError
                                    }
                                    busy = false
                                }
                            },
                            onModerate = {
                                scope.launch {
                                    busy = true
                                    runCatching {
                                        props.runAction(
                                            "moderate",
                                            buildJsonObject {
                                                put("postId", JsonPrimitive(it))
                                                put("action", JsonPrimitive("removed"))
                                            },
                                        )
                                        viewerCanModerate = true
                                        loadThread(1, false)
                                    }.onFailure {
                                        viewerCanModerate = false
                                        errorText = forbiddenError
                                    }
                                    busy = false
                                }
                            },
                        )
                        posts.filter { it.parentPostId == post.id }.forEach { reply ->
                            PostRow(
                                post = reply,
                                depth = 1,
                                anonymity = anonymity,
                                classmate = classmate,
                                tombstoneLabel = tombstoneLabel,
                                allowReplies = false,
                                readOnly = props.readOnly,
                                viewerCanEndorse = viewerCanEndorse,
                                viewerCanModerate = viewerCanModerate,
                                busy = busy,
                                onReply = {},
                                onEdit = {
                                    editingId = it.id
                                    replyTo = null
                                    draft = it.text
                                },
                                onDelete = { deleteTarget = it },
                                onReport = { reportTarget = it },
                                onUpvote = {
                                    scope.launch {
                                        busy = true
                                        runCatching {
                                            props.runAction("upvote", buildJsonObject { put("postId", JsonPrimitive(it)) })
                                            loadThread(1, false)
                                        }
                                        busy = false
                                    }
                                },
                                onEndorse = {},
                                onModerate = {},
                            )
                        }
                    }
                }
                ContentToolPack2Logic.nextPage(page, pageSize, total)?.let { next ->
                    TextButton(onClick = { scope.launch { loadThread(next, true) } }, enabled = !busy) {
                        Text(loadMore)
                    }
                }
            }
        }

        errorText?.let { Text(it, color = LexturesColors.Coral, fontSize = 12.sp) }

        if (!props.readOnly) {
            if (replyTo != null) {
                Text(L.text(R.string.mobile_contentTools_tools_inline_discussion_replyLabel), fontWeight = FontWeight.SemiBold, fontSize = 12.sp)
                TextButton(onClick = { replyTo = null; editingId = null }) { Text(cancelLabel) }
            } else if (editingId != null) {
                Text(L.text(R.string.mobile_contentTools_tools_inline_discussion_editLabel), fontWeight = FontWeight.SemiBold, fontSize = 12.sp)
            }
            ToolComposer(
                placeholder = composerLabel,
                sendLabel = when {
                    editingId != null -> saveEdit
                    replyTo != null -> replyLabel
                    else -> postLabel
                },
                text = draft,
                onTextChange = { draft = it },
                draftKey = draftKey,
                enabled = true,
                online = online,
                busy = busy,
                onSend = {
                    scope.launch {
                        val text = draft.trim()
                        if (text.isEmpty() || !online || busy || props.readOnly) {
                            if (!online) errorText = offlineLabel
                            return@launch
                        }
                        busy = true
                        errorText = null
                        try {
                            val raw = if (editingId != null) {
                                props.runAction(
                                    "edit",
                                    buildJsonObject {
                                        put("postId", JsonPrimitive(editingId!!))
                                        put("text", JsonPrimitive(text))
                                    },
                                )
                            } else {
                                props.runAction(
                                    "post",
                                    buildJsonObject {
                                        put("text", JsonPrimitive(text))
                                        put("idempotencyKey", JsonPrimitive(ContentToolHostLogic.newIdempotencyKey()))
                                        replyTo?.let { put("parentPostId", JsonPrimitive(it)) }
                                    },
                                )
                            }
                            val code = (ContentToolPack2Logic.objectMap(raw)["error"] as? JsonPrimitive)?.contentOrNull
                            if (code != null) {
                                errorText = L.text(context, localePrefs, pack2ErrorRes(code))
                            } else {
                                draft = ""
                                replyTo = null
                                editingId = null
                                draftStore.clear(draftKey)
                                props.save(mapOf("draft" to JsonPrimitive("")))
                                props.announce(postAnnounced, false)
                                loadThread(1, false)
                            }
                        } catch (_: Exception) {
                            errorText = postError
                        } finally {
                            busy = false
                        }
                    }
                },
            )
        }
    }

    reportTarget?.let { postId ->
        AlertDialog(
            onDismissRequest = { reportTarget = null },
            title = { Text(reportLabel) },
            confirmButton = {
                TextButton(onClick = {
                    val id = postId
                    reportTarget = null
                    scope.launch {
                        busy = true
                        runCatching {
                            props.runAction(
                                "report",
                                buildJsonObject {
                                    put("postId", JsonPrimitive(id))
                                    put("category", JsonPrimitive("inappropriate"))
                                },
                            )
                            props.announce(reportThanks, false)
                        }.onFailure {
                            errorText = reportError
                        }
                        busy = false
                    }
                }) { Text(reportLabel) }
            },
            dismissButton = {
                TextButton(onClick = { reportTarget = null }) { Text(cancelLabel) }
            },
        )
    }

    deleteTarget?.let { postId ->
        AlertDialog(
            onDismissRequest = { deleteTarget = null },
            title = { Text(deleteLabel) },
            confirmButton = {
                TextButton(onClick = {
                    val id = postId
                    deleteTarget = null
                    scope.launch {
                        busy = true
                        runCatching {
                            props.runAction("delete", buildJsonObject { put("postId", JsonPrimitive(id)) })
                            loadThread(1, false)
                        }.onFailure {
                            errorText = deleteError
                        }
                        busy = false
                    }
                }) { Text(deleteLabel) }
            },
            dismissButton = {
                TextButton(onClick = { deleteTarget = null }) { Text(cancelLabel) }
            },
        )
    }
}

@Composable
private fun PostRow(
    post: DiscussionPost,
    depth: Int,
    anonymity: String,
    classmate: String,
    tombstoneLabel: String,
    allowReplies: Boolean,
    readOnly: Boolean,
    viewerCanEndorse: Boolean,
    viewerCanModerate: Boolean,
    busy: Boolean,
    onReply: (String) -> Unit,
    onEdit: (DiscussionPost) -> Unit,
    onDelete: (String) -> Unit,
    onReport: (String) -> Unit,
    onUpvote: (String) -> Unit,
    onEndorse: (String) -> Unit,
    onModerate: (String) -> Unit,
) {
    val controls = ContentToolPack2Logic.discussionControls(
        isOwn = post.isOwn,
        canEditFlag = post.canEdit,
        canDeleteFlag = post.canDelete,
        allowReplies = allowReplies,
        viewerCanEndorse = viewerCanEndorse,
        viewerCanModerate = viewerCanModerate,
        readOnly = readOnly,
        removed = post.removed || post.tombstone,
    )
    val author = ContentToolPack2Logic.authorDisplay(post.authorDisplay, anonymity, post.isOwn) ?: classmate
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .padding(start = (depth * 16).dp, bottom = 8.dp)
            .semantics {
                contentDescription = "$author ${post.createdAt.orEmpty()}"
            },
    ) {
        if (ContentToolPack2Logic.shouldRenderTombstone(post.removed, post.tombstone, null)) {
            Text(tombstoneLabel, fontSize = 12.sp, fontStyle = FontStyle.Italic, color = textSecondary())
        } else {
            Text(author, fontWeight = FontWeight.SemiBold, fontSize = 11.sp, color = textSecondary())
            NotebookContentView(markdown = post.text, compact = true)
            if (post.endorsed) {
                Text(
                    L.text(R.string.mobile_contentTools_tools_inline_discussion_endorsedBadge),
                    fontSize = 11.sp,
                    color = LexturesColors.Primary,
                )
            }
        }
        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            if (controls.canUpvote) {
                TextButton(onClick = { onUpvote(post.id) }, enabled = !busy) {
                    Text(L.format(R.string.mobile_contentTools_tools_inline_discussion_upvote, post.upvoteCount))
                }
            }
            if (controls.canReply) {
                TextButton(onClick = { onReply(post.id) }) {
                    Text(L.text(R.string.mobile_contentTools_tools_inline_discussion_reply))
                }
            }
            if (controls.canEdit) {
                TextButton(onClick = { onEdit(post) }) {
                    Text(L.text(R.string.mobile_contentTools_tools_inline_discussion_edit))
                }
            }
            if (controls.canDelete) {
                TextButton(onClick = { onDelete(post.id) }) {
                    Text(L.text(R.string.mobile_contentTools_tools_inline_discussion_delete))
                }
            }
            if (controls.canReport) {
                TextButton(onClick = { onReport(post.id) }) {
                    Text(L.text(R.string.mobile_contentTools_tools_inline_discussion_report))
                }
            }
            if (controls.canEndorse) {
                TextButton(onClick = { onEndorse(post.id) }) {
                    Text(L.text(R.string.mobile_contentTools_tools_inline_discussion_endorse))
                }
            }
            if (controls.canModerate) {
                TextButton(onClick = { onModerate(post.id) }) {
                    Text(L.text(R.string.mobile_contentTools_tools_inline_discussion_moderate))
                }
            }
        }
    }
}

private fun parsePost(raw: kotlinx.serialization.json.JsonElement): DiscussionPost? {
    val o = ContentToolPack2Logic.objectMap(raw)
    val id = (o["id"] as? JsonPrimitive)?.contentOrNull ?: return null
    return DiscussionPost(
        id = id,
        parentPostId = (o["parentPostId"] as? JsonPrimitive)?.contentOrNull,
        text = (o["text"] as? JsonPrimitive)?.contentOrNull.orEmpty(),
        authorDisplay = (o["authorDisplay"] as? JsonPrimitive)?.contentOrNull,
        upvoteCount = ContentToolPack2Logic.numberField(raw, "upvoteCount")?.toInt() ?: 0,
        endorsed = ContentToolPack2Logic.boolField(raw, "endorsed") == true,
        removed = ContentToolPack2Logic.boolField(raw, "removed") == true,
        tombstone = ContentToolPack2Logic.boolField(raw, "tombstone") == true,
        isOwn = ContentToolPack2Logic.boolField(raw, "isOwn") == true,
        canEdit = ContentToolPack2Logic.boolField(raw, "canEdit") == true,
        canDelete = ContentToolPack2Logic.boolField(raw, "canDelete") == true,
        createdAt = (o["createdAt"] as? JsonPrimitive)?.contentOrNull,
    )
}
