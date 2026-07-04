package com.lextures.android.features.discussions

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.Forum
import androidx.compose.material.icons.filled.Lock
import androidx.compose.material.icons.filled.PushPin
import androidx.compose.material.icons.filled.ThumbUp
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.OfflineBanner
import com.lextures.android.core.design.StalenessChip
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.lms.CourseSummary
import com.lextures.android.core.lms.CreateDiscussionPostBody
import com.lextures.android.core.lms.CreateDiscussionThreadBody
import com.lextures.android.core.lms.DiscussionForum
import com.lextures.android.core.lms.DiscussionLogic
import com.lextures.android.core.lms.DiscussionPost
import com.lextures.android.core.lms.DiscussionPostsResponse
import com.lextures.android.core.lms.DiscussionThreadDetail
import com.lextures.android.core.lms.DiscussionThreadSummary
import com.lextures.android.features.reader.ImmersiveReaderPreferencesSheet
import com.lextures.android.features.reader.ReaderToolbarOrLegacy
import com.lextures.android.features.reader.UgcTranslationTarget
import com.lextures.android.features.reader.rememberImmersiveReaderState
import java.util.Locale
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.LmsDates
import com.lextures.android.core.notebook.NotebookStore
import com.lextures.android.core.offline.OfflineCacheKey
import com.lextures.android.core.offline.OfflineService
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsEmptyState
import com.lextures.android.features.home.LmsErrorBanner
import com.lextures.android.features.home.LmsSegmentedChips
import com.lextures.android.features.home.LmsSkeletonList
import kotlinx.coroutines.launch
import kotlinx.serialization.builtins.ListSerializer
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json

private val offlineJson = Json { ignoreUnknownKeys = true }

@Composable
fun CourseDiscussionsSection(
    session: AuthSession,
    course: CourseSummary,
    initialThreadId: String? = null,
    modifier: Modifier = Modifier,
) {
    val accessToken by session.accessToken.collectAsState()
    val context = LocalContext.current
    val offline = remember { OfflineService.get(context) }
    val isOnline by offline.networkMonitor.isOnline.collectAsState()
    val scope = rememberCoroutineScope()

    val newThreadLabel = discussionsNewThread()
    val emptyTitle = discussionsEmptyTitle()
    val emptyMessage = discussionsEmptyMessage()
    val pinnedLabel = discussionsPinned()
    val lockedLabel = discussionsLocked()

    var forums by remember { mutableStateOf<List<DiscussionForum>>(emptyList()) }
    var selectedForumId by remember { mutableStateOf<String?>(null) }
    var threads by remember { mutableStateOf<List<DiscussionThreadSummary>>(emptyList()) }
    var cacheLabel by remember { mutableStateOf<String?>(null) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var loading by remember { mutableStateOf(true) }
    var openThreadId by remember { mutableStateOf<String?>(null) }
    var showNewThread by remember { mutableStateOf(false) }
    var consumedInitial by remember { mutableStateOf(false) }

    LaunchedEffect(accessToken) {
        val token = accessToken ?: return@LaunchedEffect
        loading = true
        errorMessage = null
        try {
            val result = offline.cachedFetch(
                key = OfflineCacheKey.discussionForums(course.courseCode),
                accessToken = token,
                serializer = ListSerializer(DiscussionForum.serializer()),
            ) { LmsApi.fetchDiscussionForums(course.courseCode, token) }
            forums = result.first.sortedBy { it.position }
            if (selectedForumId == null) selectedForumId = forums.firstOrNull()?.id
            cacheLabel = result.second?.takeIf { it.isStale(isOnline) }?.lastUpdatedLabel()
        } catch (e: Exception) {
            errorMessage = session.mapError(e)
        } finally {
            loading = false
        }
    }

    LaunchedEffect(accessToken, selectedForumId) {
        val token = accessToken ?: return@LaunchedEffect
        val forumId = selectedForumId ?: return@LaunchedEffect
        loading = true
        try {
            val result = offline.cachedFetch(
                key = OfflineCacheKey.discussionThreads(course.courseCode, forumId),
                accessToken = token,
                serializer = ListSerializer(DiscussionThreadSummary.serializer()),
            ) { LmsApi.fetchDiscussionThreads(course.courseCode, forumId, token) }
            threads = result.first
        } catch (e: Exception) {
            errorMessage = session.mapError(e)
        } finally {
            loading = false
        }
    }

    LaunchedEffect(threads, initialThreadId) {
        if (!consumedInitial && initialThreadId != null && threads.any { it.id == initialThreadId }) {
            consumedInitial = true
            openThreadId = initialThreadId
        }
    }

    openThreadId?.let { threadId ->
        DiscussionThreadScreen(
            session = session,
            course = course,
            threadId = threadId,
            onBack = { openThreadId = null },
        )
        return
    }

    Column(modifier = modifier, verticalArrangement = Arrangement.spacedBy(12.dp)) {
        if (!isOnline) OfflineBanner()
        cacheLabel?.let { StalenessChip(label = it) }
        errorMessage?.let { LmsErrorBanner(message = it) }

        if (forums.size > 1 && selectedForumId != null) {
            LmsSegmentedChips(
                options = forums.map { it.id to it.name },
                selectedId = selectedForumId ?: forums.first().id,
                onSelect = { selectedForumId = it },
            )
        }

        Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.End) {
            TextButton(onClick = { showNewThread = true }, enabled = selectedForumId != null) {
                Icon(Icons.Default.Add, contentDescription = null)
                Text(newThreadLabel)
            }
        }

        when {
            loading && threads.isEmpty() -> LmsSkeletonList(count = 4)
            DiscussionLogic.sortThreads(threads).isEmpty() -> LmsEmptyState(
                icon = Icons.Default.Forum,
                title = emptyTitle,
                message = emptyMessage,
            )
            else -> {
                DiscussionLogic.sortThreads(threads).forEach { thread ->
                    LmsCard(modifier = Modifier.clickable { openThreadId = thread.id }) {
                        Column(verticalArrangement = Arrangement.spacedBy(6.dp)) {
                            Row(verticalAlignment = Alignment.Top) {
                                Text(
                                    thread.title,
                                    modifier = Modifier.weight(1f),
                                    fontWeight = FontWeight.SemiBold,
                                    color = textPrimary(),
                                )
                                if (thread.isPinned) {
                                    Icon(Icons.Default.PushPin, contentDescription = pinnedLabel)
                                }
                                if (thread.isLocked) {
                                    Icon(Icons.Default.Lock, contentDescription = lockedLabel)
                                }
                            }
                            Text(
                                "${discussionsReplyCount(thread.replyCount)} · ${LmsDates.relative(thread.updatedAt)}",
                                color = textSecondary(),
                            )
                        }
                    }
                }
            }
        }
    }

    if (showNewThread && selectedForumId != null) {
        PostComposerDialog(
            title = newThreadLabel,
            showSubject = true,
            onDismiss = { showNewThread = false },
            onPost = { subject, body ->
                scope.launch {
                    val token = accessToken ?: return@launch
                    val forumId = selectedForumId ?: return@launch
                    try {
                        val doc = DiscussionLogic.encodeBody(body)
                        if (!isOnline) {
                            val payload = CreateDiscussionThreadBody(title = subject, body = doc)
                            offline.enqueueMutation(
                                method = "POST",
                                path = "/api/v1/courses/${course.courseCode}/forums/$forumId/threads",
                                bodyJson = offlineJson.encodeToString(payload),
                                label = context.getString(R.string.mobile_discussions_newThread),
                                accessToken = token,
                                preferQueue = true,
                            )
                        } else {
                            val thread = LmsApi.createDiscussionThread(
                                course.courseCode,
                                forumId,
                                subject,
                                doc,
                                token,
                            )
                            openThreadId = thread.id
                        }
                        showNewThread = false
                        val forumIdReload = selectedForumId ?: return@launch
                        threads = offline.cachedFetch(
                            key = OfflineCacheKey.discussionThreads(course.courseCode, forumIdReload),
                            accessToken = token,
                            serializer = ListSerializer(DiscussionThreadSummary.serializer()),
                        ) { LmsApi.fetchDiscussionThreads(course.courseCode, forumIdReload, token) }.first
                    } catch (e: Exception) {
                        errorMessage = session.mapError(e)
                    }
                }
            },
        )
    }
}

@Composable
fun DiscussionThreadScreen(
    session: AuthSession,
    course: CourseSummary,
    threadId: String,
    onBack: () -> Unit,
) {
    val accessToken by session.accessToken.collectAsState()
    val context = LocalContext.current
    val offline = remember { OfflineService.get(context) }
    val isOnline by offline.networkMonitor.isOnline.collectAsState()
    val scope = rememberCoroutineScope()
    val readerState = rememberImmersiveReaderState(accessToken)
    val targetLang = remember { Locale.getDefault().language }

    val backLabel = discussionsBack()
    val threadLabel = discussionsThread()
    val replyLabel = discussionsReply()
    val postFirstHint = discussionsPostFirstHint()
    val upvoteLabel = discussionsUpvote()
    val deleteLabel = discussionsDelete()
    val authorYou = discussionsAuthorYou()

    var thread by remember { mutableStateOf<DiscussionThreadDetail?>(null) }
    var posts by remember { mutableStateOf<List<DiscussionPost>>(emptyList()) }
    var hiddenUntilFirstPost by remember { mutableStateOf(false) }
    var cacheLabel by remember { mutableStateOf<String?>(null) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var loading by remember { mutableStateOf(true) }
    var replyParentId by remember { mutableStateOf<String?>(null) }
    var showComposer by remember { mutableStateOf(false) }

    val viewerId = remember(accessToken) { NotebookStore.jwtSubject(accessToken) }

    suspend fun reload() {
        val token = accessToken ?: return
        loading = true
        try {
            val threadResult = offline.cachedFetch(
                key = OfflineCacheKey.discussionThread(course.courseCode, threadId),
                accessToken = token,
                serializer = DiscussionThreadDetail.serializer(),
            ) { LmsApi.fetchDiscussionThread(course.courseCode, threadId, token) }
            val postsResult = offline.cachedFetch(
                key = OfflineCacheKey.discussionPosts(course.courseCode, threadId),
                accessToken = token,
                serializer = DiscussionPostsResponse.serializer(),
            ) { LmsApi.fetchDiscussionPosts(course.courseCode, threadId, token) }
            thread = threadResult.first
            posts = postsResult.first.posts.orEmpty()
            hiddenUntilFirstPost = postsResult.first.hiddenUntilFirstPost
            cacheLabel = threadResult.second?.takeIf { it.isStale(isOnline) }?.lastUpdatedLabel()
                ?: postsResult.second?.takeIf { it.isStale(isOnline) }?.lastUpdatedLabel()
        } catch (e: Exception) {
            errorMessage = session.mapError(e)
        } finally {
            loading = false
        }
    }

    LaunchedEffect(accessToken, threadId) { reload() }
    ImmersiveReaderPreferencesSheet(readerState, accessToken)

    Column(
        modifier = Modifier
            .fillMaxWidth()
            .verticalScroll(rememberScrollState())
            .padding(16.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            TextButton(onClick = onBack) { Text(backLabel) }
            Text(
                thread?.title ?: threadLabel,
                modifier = Modifier.weight(1f),
                fontWeight = FontWeight.SemiBold,
                color = textPrimary(),
            )
            if (thread != null && DiscussionLogic.canReply(thread!!, course.viewerIsStaff)) {
                IconButton(onClick = { replyParentId = null; showComposer = true }) {
                    Icon(Icons.Default.Forum, contentDescription = replyLabel)
                }
            }
        }

        if (!isOnline) OfflineBanner()
        cacheLabel?.let { StalenessChip(label = it) }
        errorMessage?.let { LmsErrorBanner(message = it) }

        if (loading && thread == null) {
            LmsSkeletonList(count = 3)
        } else {
            thread?.let { detail ->
                LmsCard {
                    Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                        Text(
                            DiscussionLogic.authorLabel(detail.authorId, viewerId, authorYou),
                            color = textSecondary(),
                            fontWeight = FontWeight.SemiBold,
                        )
                        if (detail.bodyPlainText.isNotBlank()) {
                            ReaderToolbarOrLegacy(
                                text = detail.bodyPlainText,
                                accessToken = accessToken,
                                capabilities = readerState.capabilities,
                                ugcTranslation = UgcTranslationTarget(
                                    contentType = "discussion_post",
                                    contentId = detail.id,
                                    text = detail.bodyPlainText,
                                    targetLang = targetLang,
                                ),
                                onOpenPreferences = readerState.onShowPreferences,
                                ttsSpeed = readerState.store.row.ttsSpeed.toFloat(),
                            )
                            Text(detail.bodyPlainText, color = textPrimary())
                        }
                    }
                }
            }

            if (hiddenUntilFirstPost) {
                LmsCard {
                    Text(postFirstHint, color = textSecondary())
                }
            } else {
                DiscussionLogic.nestPosts(posts).forEach { nested ->
                    val post = nested.post
                    LmsCard {
                        Column(
                            modifier = Modifier.padding(start = (nested.depth.coerceAtMost(6) * 12).dp),
                            verticalArrangement = Arrangement.spacedBy(8.dp),
                        ) {
                            Row {
                                Text(
                                    DiscussionLogic.authorLabel(post.authorId, viewerId, authorYou),
                                    modifier = Modifier.weight(1f),
                                    color = textSecondary(),
                                    fontWeight = FontWeight.SemiBold,
                                )
                                Text(LmsDates.shortDateTime(post.createdAt), color = textSecondary())
                            }
                            ReaderToolbarOrLegacy(
                                text = post.bodyPlainText,
                                accessToken = accessToken,
                                capabilities = readerState.capabilities,
                                ugcTranslation = UgcTranslationTarget(
                                    contentType = "discussion_post",
                                    contentId = post.id,
                                    text = post.bodyPlainText,
                                    targetLang = targetLang,
                                ),
                                onOpenPreferences = readerState.onShowPreferences,
                                ttsSpeed = readerState.store.row.ttsSpeed.toFloat(),
                            )
                            Text(post.bodyPlainText, color = textPrimary())
                            Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                                TextButton(onClick = {
                                    scope.launch {
                                        val token = accessToken ?: return@launch
                                        val optimistic = !post.viewerUpvoted
                                        posts = posts.map {
                                            if (it.id != post.id) it else it.copy(
                                                viewerUpvoted = optimistic,
                                                upvoteCount = (it.upvoteCount + if (optimistic) 1 else -1).coerceAtLeast(0),
                                            )
                                        }
                                        try {
                                            val response = LmsApi.upvoteDiscussionPost(
                                                course.courseCode,
                                                post.id,
                                                token,
                                            )
                                            posts = posts.map {
                                                if (it.id != post.id) it else it.copy(
                                                    viewerUpvoted = response.wasAdded,
                                                    upvoteCount = response.upvoteCount,
                                                )
                                            }
                                        } catch (e: Exception) {
                                            posts = posts.map {
                                                if (it.id != post.id) it else post
                                            }
                                            errorMessage = session.mapError(e)
                                        }
                                    }
                                }) {
                                    Icon(Icons.Default.ThumbUp, contentDescription = upvoteLabel)
                                    Text(post.upvoteCount.toString())
                                }
                                if (thread != null && DiscussionLogic.canReply(thread!!, course.viewerIsStaff)) {
                                    TextButton(onClick = {
                                        replyParentId = post.id
                                        showComposer = true
                                    }) { Text(replyLabel) }
                                }
                                if (DiscussionLogic.canDeletePost(post, viewerId)) {
                                    TextButton(onClick = {
                                        scope.launch {
                                            val token = accessToken ?: return@launch
                                            try {
                                                LmsApi.deleteDiscussionPost(course.courseCode, post.id, token)
                                                posts = posts.filter { it.id != post.id }
                                            } catch (e: Exception) {
                                                errorMessage = session.mapError(e)
                                            }
                                        }
                                    }) { Text(deleteLabel) }
                                }
                            }
                        }
                    }
                }
            }
        }
    }

    if (showComposer) {
        PostComposerDialog(
            title = replyLabel,
            showSubject = false,
            onDismiss = { showComposer = false },
            onPost = { _, body ->
                scope.launch {
                    val token = accessToken ?: return@launch
                    val doc = DiscussionLogic.encodeBody(body)
                    try {
                        if (!isOnline) {
                            val payload = CreateDiscussionPostBody(parentPostId = replyParentId, body = doc)
                            offline.enqueueMutation(
                                method = "POST",
                                path = "/api/v1/courses/${course.courseCode}/discussion-threads/$threadId/posts",
                                bodyJson = offlineJson.encodeToString(payload),
                                label = context.getString(R.string.mobile_discussions_reply),
                                accessToken = token,
                                preferQueue = true,
                            )
                        } else {
                            LmsApi.createDiscussionPost(
                                course.courseCode,
                                threadId,
                                replyParentId,
                                doc,
                                token,
                            )
                        }
                        showComposer = false
                        reload()
                    } catch (e: Exception) {
                        errorMessage = session.mapError(e)
                    }
                }
            },
        )
    }
}

@Composable
private fun PostComposerDialog(
    title: String,
    showSubject: Boolean,
    onDismiss: () -> Unit,
    onPost: (subject: String, body: String) -> Unit,
) {
    var subject by remember { mutableStateOf("") }
    var body by remember { mutableStateOf("") }
    val threadTitleLabel = discussionsThreadTitle()
    val messageLabel = discussionsMessage()
    val postLabel = discussionsPost()
    val cancelLabel = discussionsCancel()

    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text(title) },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                if (showSubject) {
                    OutlinedTextField(
                        value = subject,
                        onValueChange = { subject = it },
                        label = { Text(threadTitleLabel) },
                        modifier = Modifier.fillMaxWidth(),
                    )
                }
                OutlinedTextField(
                    value = body,
                    onValueChange = { body = it },
                    label = { Text(messageLabel) },
                    modifier = Modifier.fillMaxWidth(),
                    minLines = 4,
                )
            }
        },
        confirmButton = {
            TextButton(
                onClick = { onPost(subject.trim(), body) },
                enabled = !DiscussionLogic.isBodyEmpty(body) && (!showSubject || subject.trim().isNotEmpty()),
            ) { Text(postLabel) }
        },
        dismissButton = {
            TextButton(onClick = onDismiss) { Text(cancelLabel) }
        },
    )
}
