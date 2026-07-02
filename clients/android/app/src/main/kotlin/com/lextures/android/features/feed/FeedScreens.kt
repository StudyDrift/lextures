package com.lextures.android.features.feed

import android.graphics.BitmapFactory
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.Image
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.Photo
import androidx.compose.material.icons.filled.PushPin
import androidx.compose.material.icons.filled.Send
import androidx.compose.material.icons.filled.Forum
import androidx.compose.material.icons.filled.ThumbUp
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.ProgressIndicatorDefaults
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.produceState
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.asImageBitmap
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.OfflineBanner
import com.lextures.android.core.design.StalenessChip
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.lms.CourseSummary
import com.lextures.android.core.lms.FeedChannel
import com.lextures.android.core.lms.GroupFeedContext
import com.lextures.android.core.lms.FeedLogic
import com.lextures.android.core.lms.FeedMessage
import com.lextures.android.core.lms.FileDownloadManager
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.LmsDates
import com.lextures.android.core.lms.PostFeedMessageBody
import com.lextures.android.core.notebook.NotebookStore
import com.lextures.android.core.offline.OfflineCacheKey
import com.lextures.android.core.offline.OfflineService
import com.lextures.android.core.realtime.FeedSocket
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsEmptyState
import com.lextures.android.features.home.LmsErrorBanner
import com.lextures.android.features.home.LmsSkeletonList
import kotlinx.coroutines.launch
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import okhttp3.OkHttpClient

private val offlineJson = Json { ignoreUnknownKeys = true }
private val feedHttp = OkHttpClient()

@Composable
fun CourseFeedSection(session: AuthSession, course: CourseSummary, modifier: Modifier = Modifier) {
    FeedChannelsScreen(session = session, course = course, modifier = modifier)
}

@Composable
fun FeedChannelsScreen(
    session: AuthSession,
    course: CourseSummary,
    groupContext: GroupFeedContext? = null,
    modifier: Modifier = Modifier,
) {
    val accessToken by session.accessToken.collectAsState()
    val context = LocalContext.current
    val offline = remember { OfflineService.get(context) }
    val isOnline by offline.networkMonitor.isOnline.collectAsState()
    val socket = remember { FeedSocket() }
    val channelsRevision by socket.channelsRevision.collectAsState()

    var channels by remember { mutableStateOf<List<FeedChannel>>(emptyList()) }
    var cacheLabel by remember { mutableStateOf<String?>(null) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var loading by remember { mutableStateOf(true) }
    var showNewChannel by remember { mutableStateOf(false) }
    var openChannel by remember { mutableStateOf<FeedChannel?>(null) }

    suspend fun load() {
        val token = accessToken ?: return
        loading = true
        errorMessage = null
        try {
            val cacheKey = if (groupContext != null) {
                OfflineCacheKey.groupFeedChannels(course.courseCode, groupContext.groupId)
            } else {
                OfflineCacheKey.feedChannels(course.courseCode)
            }
            val result = offline.cachedFetch(
                key = cacheKey,
                accessToken = token,
                serializer = FeedChannel.serializer().let { kotlinx.serialization.builtins.ListSerializer(it) },
            ) {
                if (groupContext != null) {
                    LmsApi.fetchGroupFeedChannels(course.courseCode, groupContext.groupId, token)
                } else {
                    LmsApi.fetchFeedChannels(course.courseCode, token)
                }
            }
            channels = result.first
            cacheLabel = result.second?.takeIf { it.isStale(isOnline) }?.lastUpdatedLabel()
        } catch (e: Exception) {
            errorMessage = session.mapError(e)
        } finally {
            loading = false
        }
    }

    LaunchedEffect(accessToken) { load() }
    LaunchedEffect(Unit) {
        socket.connect(course.courseCode) { session.accessToken.value }
    }
    LaunchedEffect(channelsRevision) { if (channelsRevision > 0) load() }

    openChannel?.let { channel ->
        FeedChannelScreen(
            session = session,
            course = course,
            channel = channel,
            socket = socket,
            groupContext = groupContext,
            onBack = { openChannel = null },
        )
        return
    }

    Column(modifier = modifier, verticalArrangement = Arrangement.spacedBy(12.dp)) {
        if (!isOnline) OfflineBanner()
        cacheLabel?.let { StalenessChip(label = it) }
        errorMessage?.let { LmsErrorBanner(message = it) }

        if (course.viewerIsStaff && groupContext == null) {
            Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.End) {
                TextButton(onClick = { showNewChannel = true }) {
                    Icon(Icons.Default.Add, contentDescription = null)
                    Text(feedNewChannel())
                }
            }
        }

        when {
            loading && channels.isEmpty() -> LmsSkeletonList(count = 3)
            channels.isEmpty() -> LmsEmptyState(
                icon = Icons.Default.Forum,
                title = feedEmptyChannels(),
                message = "",
            )
            else -> {
                channels.sortedBy { it.sortOrder }.forEach { channel ->
                    LmsCard(modifier = Modifier.clickable { openChannel = channel }) {
                        Text(channel.name, fontWeight = FontWeight.SemiBold, color = textPrimary())
                    }
                }
            }
        }
    }

    if (showNewChannel) {
        var name by remember { mutableStateOf("") }
        val scope = rememberCoroutineScope()
        AlertDialog(
            onDismissRequest = { showNewChannel = false },
            title = { Text(feedNewChannel()) },
            text = {
                OutlinedTextField(
                    value = name,
                    onValueChange = { name = it },
                    label = { Text(feedChannelNamePlaceholder()) },
                    modifier = Modifier.fillMaxWidth(),
                )
            },
            confirmButton = {
                TextButton(
                    onClick = {
                        val trimmed = name.trim()
                        showNewChannel = false
                        if (trimmed.isEmpty()) return@TextButton
                        scope.launch {
                            val token = accessToken ?: return@launch
                            try {
                                LmsApi.createFeedChannel(course.courseCode, trimmed, token)
                                load()
                            } catch (e: Exception) {
                                errorMessage = session.mapError(e)
                            }
                        }
                    },
                    enabled = name.trim().isNotEmpty(),
                ) { Text(feedCreate()) }
            },
            dismissButton = {
                TextButton(onClick = { showNewChannel = false }) { Text(feedCreate()) }
            },
        )
    }
}

@Composable
fun FeedChannelScreen(
    session: AuthSession,
    course: CourseSummary,
    channel: FeedChannel,
    socket: FeedSocket,
    groupContext: GroupFeedContext? = null,
    onBack: () -> Unit,
) {
    val accessToken by session.accessToken.collectAsState()
    val context = LocalContext.current
    val offline = remember { OfflineService.get(context) }
    val isOnline by offline.networkMonitor.isOnline.collectAsState()
    val scope = rememberCoroutineScope()
    val messagesRevision by socket.messagesRevision.collectAsState()

    var roots by remember { mutableStateOf<List<FeedMessage>>(emptyList()) }
    var cacheLabel by remember { mutableStateOf<String?>(null) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var loading by remember { mutableStateOf(true) }
    var composerText by remember { mutableStateOf("") }
    var pendingImageBytes by remember { mutableStateOf<ByteArray?>(null) }
    var uploading by remember { mutableStateOf(false) }
    var pendingLikes by remember { mutableStateOf(setOf<String>()) }

    val viewerId = remember(accessToken) { NotebookStore.jwtSubject(accessToken) }
    val feedLabel = feedTitle()
    val orderedMessages = remember(roots) { FeedLogic.orderedMessages(roots) }

    suspend fun load() {
        val token = accessToken ?: return
        loading = true
        errorMessage = null
        try {
            val cacheKey = if (groupContext != null) {
                OfflineCacheKey.groupFeedMessages(course.courseCode, groupContext.groupId, channel.id)
            } else {
                OfflineCacheKey.feedMessages(course.courseCode, channel.id)
            }
            val result = offline.cachedFetch(
                key = cacheKey,
                accessToken = token,
                serializer = FeedMessage.serializer().let { kotlinx.serialization.builtins.ListSerializer(it) },
            ) {
                if (groupContext != null) {
                    LmsApi.fetchGroupFeedMessages(course.courseCode, groupContext.groupId, channel.id, token)
                } else {
                    LmsApi.fetchFeedMessages(course.courseCode, channel.id, token)
                }
            }
            roots = result.first
            cacheLabel = result.second?.takeIf { it.isStale(isOnline) }?.lastUpdatedLabel()
        } catch (e: Exception) {
            errorMessage = session.mapError(e)
        } finally {
            loading = false
        }
    }

    LaunchedEffect(accessToken, channel.id) { load() }
    LaunchedEffect(messagesRevision, channel.id) {
        if (socket.revision(channel.id) > 0) load()
    }

    val pickImageLauncher = rememberLauncherForActivityResult(ActivityResultContracts.GetContent()) { uri ->
        uri ?: return@rememberLauncherForActivityResult
        runCatching {
            context.contentResolver.openInputStream(uri)?.use { stream ->
                pendingImageBytes = stream.readBytes()
            }
        }
    }

    fun updateMessage(id: String, mutate: (FeedMessage) -> FeedMessage) {
        roots = roots.map { root ->
            when {
                root.id == id -> mutate(root)
                else -> root.copy(replies = root.replies.map { if (it.id == id) mutate(it) else it })
            }
        }
    }

    Column(modifier = Modifier.fillMaxWidth(), verticalArrangement = Arrangement.spacedBy(0.dp)) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            TextButton(onClick = onBack) { Text("←") }
            Text(channel.name, modifier = Modifier.weight(1f), fontWeight = FontWeight.SemiBold, color = textPrimary())
        }

        Column(
            modifier = Modifier
                .fillMaxWidth()
                .weight(1f, fill = false)
                .verticalScroll(rememberScrollState())
                .padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            if (!isOnline) OfflineBanner()
            cacheLabel?.let { StalenessChip(label = it) }
            errorMessage?.let { LmsErrorBanner(message = it) }

            when {
                loading && roots.isEmpty() -> LmsSkeletonList(count = 4)
                orderedMessages.isEmpty() -> LmsEmptyState(
                    icon = Icons.Default.Forum,
                    title = feedEmptyMessages(),
                    message = "",
                )
                else -> orderedMessages.forEach { message ->
                    FeedMessageBubble(
                        message = message,
                        course = course,
                        viewerId = viewerId,
                        accessToken = accessToken,
                        likePending = pendingLikes.contains(message.id),
                        onLike = {
                            scope.launch {
                                val token = accessToken ?: return@launch
                                pendingLikes = pendingLikes + message.id
                                val optimistic = !message.viewerHasLiked
                                updateMessage(message.id) {
                                    it.copy(
                                        viewerHasLiked = optimistic,
                                        likeCount = (it.likeCount + if (optimistic) 1 else -1).coerceAtLeast(0),
                                    )
                                }
                                try {
                                    if (optimistic) {
                                        LmsApi.likeFeedMessage(course.courseCode, message.id, token)
                                    } else {
                                        LmsApi.unlikeFeedMessage(course.courseCode, message.id, token)
                                    }
                                } catch (e: Exception) {
                                    updateMessage(message.id) { message }
                                    errorMessage = session.mapError(e)
                                } finally {
                                    pendingLikes = pendingLikes - message.id
                                }
                            }
                        },
                        onTogglePin = {
                            scope.launch {
                                val token = accessToken ?: return@launch
                                try {
                                    LmsApi.pinFeedMessage(course.courseCode, message.id, message.pinnedAt == null, token)
                                    load()
                                } catch (e: Exception) {
                                    errorMessage = session.mapError(e)
                                }
                            }
                        },
                        onDelete = {
                            scope.launch {
                                val token = accessToken ?: return@launch
                                try {
                                    LmsApi.deleteFeedMessage(course.courseCode, message.id, token)
                                    load()
                                } catch (e: Exception) {
                                    errorMessage = session.mapError(e)
                                }
                            }
                        },
                    )
                }
            }
        }

        Column(modifier = Modifier.fillMaxWidth().padding(12.dp), verticalArrangement = Arrangement.spacedBy(6.dp)) {
            if (pendingImageBytes != null) {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Icon(Icons.Default.Photo, contentDescription = feedAttachImage())
                    Text(feedAttachImage(), color = textSecondary())
                }
            }
            Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                IconButton(onClick = { pickImageLauncher.launch("image/*") }) {
                    Icon(Icons.Default.Photo, contentDescription = feedAttachImage())
                }
                OutlinedTextField(
                    value = composerText,
                    onValueChange = { composerText = it },
                    label = { Text(feedComposerPlaceholder()) },
                    modifier = Modifier.weight(1f),
                )
                if (uploading) {
                    CircularProgressIndicator(strokeWidth = ProgressIndicatorDefaults.CircularStrokeWidth)
                } else {
                    IconButton(
                        onClick = {
                            val text = composerText.trim()
                            val imageBytes = pendingImageBytes
                            if (text.isEmpty() && imageBytes == null) return@IconButton
                            composerText = ""
                            pendingImageBytes = null
                            scope.launch {
                                val token = accessToken ?: return@launch
                                try {
                                    var body = text
                                    if (imageBytes != null) {
                                        if (!isOnline) throw IllegalStateException("offline")
                                        uploading = true
                                        val upload = LmsApi.uploadFeedImage(
                                            course.courseCode,
                                            imageBytes,
                                            "photo.jpg",
                                            "image/jpeg",
                                            token,
                                        )
                                        uploading = false
                                        val markdown = "![image](${upload.content_path})"
                                        body = if (body.isEmpty()) markdown else "$body\n\n$markdown"
                                    }
                                    if (isOnline) {
                                        if (groupContext != null) {
                                            LmsApi.postGroupFeedMessage(
                                                course.courseCode,
                                                groupContext.groupId,
                                                channel.id,
                                                body,
                                                token,
                                            )
                                        } else {
                                            LmsApi.postFeedMessage(course.courseCode, channel.id, body, token)
                                        }
                                        load()
                                    } else {
                                        val path = if (groupContext != null) {
                                            "/api/v1/courses/${course.courseCode}/groups/${groupContext.groupId}" +
                                                "/feed/channels/${channel.id}/messages"
                                        } else {
                                            "/api/v1/courses/${course.courseCode}/feed/channels/${channel.id}/messages"
                                        }
                                        offline.enqueueMutation(
                                            method = "POST",
                                            path = path,
                                            bodyJson = offlineJson.encodeToString(PostFeedMessageBody(body)),
                                            label = feedLabel,
                                            accessToken = token,
                                            preferQueue = true,
                                        )
                                    }
                                } catch (e: Exception) {
                                    uploading = false
                                    composerText = text
                                    pendingImageBytes = imageBytes
                                    errorMessage = session.mapError(e)
                                }
                            }
                        },
                        enabled = composerText.trim().isNotEmpty() || pendingImageBytes != null,
                    ) {
                        Icon(Icons.Default.Send, contentDescription = feedSend())
                    }
                }
            }
        }
    }
}

@Composable
private fun FeedMessageBubble(
    message: FeedMessage,
    course: CourseSummary,
    viewerId: String?,
    accessToken: String?,
    likePending: Boolean,
    onLike: () -> Unit,
    onTogglePin: () -> Unit,
    onDelete: () -> Unit,
) {
    val (text, imagePath) = remember(message.body) { FeedLogic.extractImagePath(message.body) }
    LmsCard {
        Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text(message.authorLabel, fontWeight = FontWeight.SemiBold, color = textSecondary())
                if (message.pinnedAt != null) {
                    Icon(Icons.Default.PushPin, contentDescription = feedPinned())
                }
                Text(
                    LmsDates.relative(message.createdAt),
                    modifier = Modifier.weight(1f),
                    color = textSecondary(),
                )
            }
            if (text.isNotEmpty()) {
                Text(text, color = textPrimary())
            }
            imagePath?.let { path -> FeedImage(path = path, accessToken = accessToken) }
            Row(horizontalArrangement = Arrangement.spacedBy(12.dp), verticalAlignment = Alignment.CenterVertically) {
                TextButton(onClick = onLike, enabled = !likePending) {
                    Icon(Icons.Default.ThumbUp, contentDescription = feedLike())
                    Text(message.likeCount.toString())
                }
                if (FeedLogic.canPin(course.viewerIsStaff, message.parentMessageId != null)) {
                    TextButton(onClick = onTogglePin) { Text(if (message.pinnedAt != null) feedUnpin() else feedPin()) }
                }
                if (FeedLogic.canDelete(message, viewerId)) {
                    TextButton(onClick = onDelete) { Text(feedDelete()) }
                }
            }
        }
    }
}

/** Authenticated image bubble for feed attachments (server content paths require a bearer token). */
@Composable
private fun FeedImage(path: String, accessToken: String?) {
    val bitmap by produceState<android.graphics.Bitmap?>(initialValue = null, path, accessToken) {
        value = null
        val token = accessToken ?: return@produceState
        val url = com.lextures.android.core.config.AppConfiguration.apiUrl(path).toString()
        val request = FileDownloadManager.authorizedRequest(url, token)
        runCatching {
            feedHttp.newCall(request).execute().use { response ->
                if (!response.isSuccessful) return@produceState
                val bytes = response.body?.bytes() ?: return@produceState
                value = BitmapFactory.decodeByteArray(bytes, 0, bytes.size)
            }
        }
    }
    bitmap?.let {
        Image(
            bitmap = it.asImageBitmap(),
            contentDescription = null,
            modifier = Modifier.heightIn(max = 220.dp),
        )
    }
}
