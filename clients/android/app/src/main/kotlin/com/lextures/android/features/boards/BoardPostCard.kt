package com.lextures.android.features.boards

import android.content.Intent
import android.net.Uri
import android.webkit.WebView
import android.webkit.WebViewClient
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Description
import androidx.compose.material.icons.filled.MoreVert
import androidx.compose.material.icons.filled.OpenInNew
import androidx.compose.material.icons.filled.Warning
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.viewinterop.AndroidView
import androidx.compose.foundation.Canvas
import androidx.compose.ui.window.Dialog
import androidx.compose.ui.window.DialogProperties
import androidx.media3.common.MediaItem
import androidx.media3.common.util.UnstableApi
import androidx.media3.exoplayer.ExoPlayer
import androidx.media3.ui.PlayerView
import coil.compose.AsyncImage
import com.lextures.android.R
import com.lextures.android.core.design.isDarkTheme
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.lms.ArrangeBoardPostBody
import com.lextures.android.core.lms.BoardContentType
import com.lextures.android.core.lms.BoardPost
import com.lextures.android.core.lms.BoardPostSafetyState
import com.lextures.android.core.lms.BoardReactionMode
import com.lextures.android.core.lms.BoardSection
import com.lextures.android.core.lms.BoardsLogic
import com.lextures.android.core.lms.WhiteboardRenderer
import com.lextures.android.features.courses.MarkdownText
import com.lextures.android.features.home.LmsCard
import androidx.compose.material.icons.outlined.ChatBubbleOutline

@Composable
fun BoardPostCard(
    post: BoardPost,
    canEdit: Boolean,
    onEdit: () -> Unit,
    onDelete: () -> Unit,
    modifier: Modifier = Modifier,
    canArrange: Boolean = false,
    canManageBoard: Boolean = false,
    currentUserId: String? = null,
    reactionMode: BoardReactionMode = BoardReactionMode.None,
    canInteract: Boolean = true,
    assignmentLinked: Boolean = false,
    sections: List<BoardSection> = emptyList(),
    siblings: List<BoardPost> = emptyList(),
    showTimelineArrange: Boolean = false,
    showMapArrange: Boolean = false,
    onArrange: ((ArrangeBoardPostBody) -> Unit)? = null,
) {
    val context = LocalContext.current
    val engagement = LocalBoardEngagement.current
    var menuOpen by remember { mutableStateOf(false) }
    var confirmDelete by remember { mutableStateOf(false) }
    var fullImage by remember { mutableStateOf(false) }
    var showComments by remember { mutableStateOf(false) }
    var showGradeSheet by remember { mutableStateOf(false) }
    var showReport by remember { mutableStateOf(false) }
    val known = BoardContentType.fromApi(post.contentType)
    val typeLabel = typeLabel(post.contentType)
    val canGrade = canManageBoard && reactionMode == BoardReactionMode.Grade
    val safetyState = BoardsLogic.postSafetyState(post)
    val removedPlaceholder = L.text(R.string.mobile_boards_moderation_removedPlaceholder)
    val pendingBadge = L.text(R.string.mobile_boards_moderation_pendingBadge)

    LmsCard(modifier = modifier) {
        if (safetyState == BoardPostSafetyState.Removed) {
            Text(
                removedPlaceholder,
                color = textSecondary(),
                modifier = Modifier.semantics {
                    contentDescription = removedPlaceholder
                },
            )
            return@LmsCard
        }
        Row(modifier = Modifier.fillMaxWidth(), verticalAlignment = Alignment.Top) {
            Column(modifier = Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(2.dp)) {
                if (post.title.isNotBlank()) {
                    Text(post.title, fontWeight = FontWeight.SemiBold, color = textPrimary())
                }
                BoardsLogic.attributionLabel(post)?.let { author ->
                    Text(author, color = textSecondary())
                }
                Text(typeLabel, color = textSecondary())
                if (safetyState == BoardPostSafetyState.PendingApproval) {
                    Text(
                        pendingBadge,
                        color = androidx.compose.ui.graphics.Color(0xFFB45309),
                        fontWeight = FontWeight.SemiBold,
                    )
                }
            }
            if (canArrange && onArrange != null) {
                CardArrangeMenu(
                    post = post,
                    sections = sections,
                    siblings = siblings.ifEmpty { listOf(post) },
                    showTimeline = showTimelineArrange,
                    showMap = showMapArrange,
                    onMoveToSection = {
                        onArrange(ArrangeBoardPostBody(sectionId = it))
                    },
                    onReorder = {
                        onArrange(ArrangeBoardPostBody(sortIndex = it))
                    },
                    onSetEventDate = if (showTimelineArrange) {
                        { onArrange(ArrangeBoardPostBody(eventDate = it ?: "")) }
                    } else {
                        null
                    },
                    onSetCoords = if (showMapArrange) {
                        { lat, lng ->
                            onArrange(ArrangeBoardPostBody(lat = lat, lng = lng))
                        }
                    } else {
                        null
                    },
                )
            }
            if (canEdit || canManageBoard || engagement != null) {
                IconButton(onClick = { menuOpen = true }) {
                    Icon(
                        Icons.Default.MoreVert,
                        contentDescription = L.text(R.string.mobile_boards_post_actions),
                    )
                }
                DropdownMenu(expanded = menuOpen, onDismissRequest = { menuOpen = false }) {
                    if (canEdit && (known == BoardContentType.Text || known == BoardContentType.Link)) {
                        DropdownMenuItem(
                            text = { Text(L.text(R.string.mobile_boards_post_edit)) },
                            onClick = {
                                menuOpen = false
                                onEdit()
                            },
                        )
                    }
                    if (engagement != null) {
                        DropdownMenuItem(
                            text = { Text(L.text(R.string.mobile_boards_report_action)) },
                            onClick = {
                                menuOpen = false
                                showReport = true
                            },
                        )
                    }
                    if (canManageBoard) {
                        engagement?.onHidePost?.let { hide ->
                            DropdownMenuItem(
                                text = { Text(L.text(R.string.mobile_boards_moderation_hide)) },
                                onClick = {
                                    menuOpen = false
                                    hide(post)
                                },
                            )
                        }
                        engagement?.onRemovePost?.let { remove ->
                            DropdownMenuItem(
                                text = { Text(L.text(R.string.mobile_boards_moderation_remove)) },
                                onClick = {
                                    menuOpen = false
                                    remove(post)
                                },
                            )
                        }
                    }
                    if (canEdit) {
                        DropdownMenuItem(
                            text = { Text(L.text(R.string.mobile_boards_post_delete)) },
                            onClick = {
                                menuOpen = false
                                confirmDelete = true
                            },
                        )
                    }
                }
            }
        }

        when (known) {
            BoardContentType.Text -> {
                val plain = BoardsLogic.bodyPlainText(post)
                if (plain.isNotBlank()) MarkdownText(plain)
            }
            BoardContentType.Image -> MediaBlock(
                post = post,
                onOpenImage = { fullImage = true },
                onOpenFile = { url ->
                    context.startActivity(Intent(Intent.ACTION_VIEW, Uri.parse(url)))
                },
            )
            BoardContentType.File -> MediaBlock(
                post = post,
                onOpenImage = {},
                onOpenFile = { url ->
                    context.startActivity(Intent(Intent.ACTION_VIEW, Uri.parse(url)))
                },
            )
            BoardContentType.Audio -> MediaBlock(post = post, onOpenImage = {}, onOpenFile = {})
            BoardContentType.Video -> {
                val link = post.linkUrl
                val embed = link?.let { BoardsLogic.videoEmbedFromUrl(it) }
                val embedUrl = embed?.let { BoardsLogic.embedUrl(it) }
                if (embedUrl != null) {
                    BoardEmbedWebView(embedUrl)
                } else {
                    MediaBlock(post = post, onOpenImage = {}, onOpenFile = {})
                    link?.let { LinkBlock(post, it) }
                }
            }
            BoardContentType.Link -> {
                val link = post.linkUrl
                if (link != null) {
                    val embed = BoardsLogic.videoEmbedFromUrl(link)
                    val embedUrl = embed?.let { BoardsLogic.embedUrl(it) }
                    if (embedUrl != null) BoardEmbedWebView(embedUrl) else LinkBlock(post, link)
                }
            }
            BoardContentType.Drawing -> DrawingBlock(post)
            null -> Text(
                L.text(R.string.mobile_boards_post_unsupportedMessage),
                color = textSecondary(),
            )
        }

        if (engagement != null) {
            Row(
                modifier = Modifier.fillMaxWidth().padding(top = 4.dp),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically,
            ) {
                if (reactionMode != BoardReactionMode.None) {
                    ReactionControl(
                        courseCode = engagement.courseCode,
                        boardId = post.boardId,
                        post = post,
                        reactionMode = reactionMode,
                        accessToken = engagement.accessToken,
                        canInteract = canInteract,
                        canGrade = canGrade,
                        onPostUpdate = engagement.onPostUpdate,
                        onAnnounce = engagement.onAnnounce,
                        onOpenGradeSheet = { showGradeSheet = true },
                    )
                }
                TextButton(onClick = { showComments = true }) {
                    Icon(
                        Icons.Outlined.ChatBubbleOutline,
                        contentDescription = null,
                        modifier = Modifier.size(16.dp),
                        tint = textSecondary(),
                    )
                    val n = post.commentCount ?: 0
                    Text(
                        if (n > 0) "$n" else L.text(R.string.mobile_boards_comment_toggle),
                        color = textSecondary(),
                        modifier = Modifier.padding(start = 4.dp),
                    )
                }
            }
        }
    }

    if (showComments && engagement != null) {
        CommentSheet(
            courseCode = engagement.courseCode,
            boardId = post.boardId,
            postId = post.id,
            accessToken = engagement.accessToken,
            canInteract = canInteract,
            canManageBoard = canManageBoard,
            currentUserId = currentUserId,
            onCountChange = { delta ->
                engagement.onPostUpdate(
                    post.copy(commentCount = maxOf(0, (post.commentCount ?: 0) + delta)),
                )
            },
            onDismiss = { showComments = false },
        )
    }

    if (showGradeSheet && engagement != null) {
        GradeSheet(
            courseCode = engagement.courseCode,
            boardId = post.boardId,
            post = post,
            accessToken = engagement.accessToken,
            assignmentLinked = assignmentLinked,
            onPostUpdate = engagement.onPostUpdate,
            onAnnounce = engagement.onAnnounce,
            onDismiss = { showGradeSheet = false },
        )
    }

    if (showReport && engagement != null) {
        ReportDialog(
            courseCode = engagement.courseCode,
            boardId = post.boardId,
            accessToken = engagement.accessToken,
            postId = post.id,
            onDismiss = { showReport = false },
        )
    }

    if (confirmDelete) {
        AlertDialog(
            onDismissRequest = { confirmDelete = false },
            title = { Text(L.text(R.string.mobile_boards_post_deleteConfirm)) },
            confirmButton = {
                TextButton(onClick = {
                    confirmDelete = false
                    onDelete()
                }) { Text(L.text(R.string.mobile_boards_post_delete)) }
            },
            dismissButton = {
                TextButton(onClick = { confirmDelete = false }) {
                    Text(L.text(R.string.mobile_common_cancel))
                }
            },
        )
    }

    if (fullImage) {
        val url = BoardsLogic.attachmentMediaUrl(post.attachment)
        if (url != null) {
            Dialog(
                onDismissRequest = { fullImage = false },
                properties = DialogProperties(usePlatformDefaultWidth = false),
            ) {
                Box(modifier = Modifier.fillMaxSize()) {
                    AsyncImage(
                        model = url,
                        contentDescription = post.attachment?.altText
                            ?.ifBlank { L.text(R.string.mobile_boards_post_imageAltFallback) },
                        modifier = Modifier.fillMaxSize(),
                    )
                    TextButton(
                        onClick = { fullImage = false },
                        modifier = Modifier.align(Alignment.TopEnd).padding(8.dp),
                    ) { Text(L.text(R.string.mobile_common_close)) }
                }
            }
        }
    }
}

@Composable
private fun typeLabel(contentType: String): String {
    val key = when (contentType.lowercase()) {
        "text" -> R.string.mobile_boards_post_type_text
        "image" -> R.string.mobile_boards_post_type_image
        "file" -> R.string.mobile_boards_post_type_file
        "link" -> R.string.mobile_boards_post_type_link
        "video" -> R.string.mobile_boards_post_type_video
        "audio" -> R.string.mobile_boards_post_type_audio
        "drawing" -> R.string.mobile_boards_post_type_drawing
        else -> R.string.mobile_boards_post_type_unsupported
    }
    return L.text(key)
}

@Composable
private fun MediaBlock(
    post: BoardPost,
    onOpenImage: () -> Unit,
    onOpenFile: (String) -> Unit,
) {
    val att = post.attachment ?: return
    when (att.scanStatus.lowercase()) {
        "pending" -> Row(
            horizontalArrangement = Arrangement.spacedBy(8.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            CircularProgressIndicator(modifier = Modifier.size(18.dp), strokeWidth = 2.dp)
            Text(L.text(R.string.mobile_boards_post_scanning), color = textSecondary())
        }
        "blocked" -> Row(
            horizontalArrangement = Arrangement.spacedBy(8.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Icon(Icons.Default.Warning, contentDescription = null, tint = androidx.compose.ui.graphics.Color(0xFFE67E22))
            Text(L.text(R.string.mobile_boards_post_blocked), color = textPrimary())
        }
        else -> {
            val url = BoardsLogic.attachmentMediaUrl(att) ?: return
            when (BoardContentType.fromApi(post.contentType)) {
                BoardContentType.Image -> AsyncImage(
                    model = url,
                    contentDescription = att.altText.ifBlank {
                        L.text(R.string.mobile_boards_post_imageAltFallback)
                    },
                    modifier = Modifier
                        .fillMaxWidth()
                        .height(220.dp)
                        .clip(RoundedCornerShape(8.dp))
                        .clickable(onClick = onOpenImage),
                )
                BoardContentType.Audio, BoardContentType.Video -> BoardMediaPlayer(
                    url = url,
                    video = post.contentType.equals("video", ignoreCase = true),
                )
                BoardContentType.File -> Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .clickable { onOpenFile(url) },
                    horizontalArrangement = Arrangement.spacedBy(8.dp),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Icon(Icons.Default.Description, contentDescription = null, tint = textSecondary())
                    Column(modifier = Modifier.weight(1f)) {
                        Text(
                            att.fileName.ifBlank { L.text(R.string.mobile_boards_post_type_file) },
                            fontWeight = FontWeight.Medium,
                            color = textPrimary(),
                        )
                        if (att.sizeBytes > 0) {
                            Text(BoardsLogic.formatFileSize(att.sizeBytes), color = textSecondary())
                        }
                    }
                    Icon(Icons.Default.OpenInNew, contentDescription = null, tint = textSecondary())
                }
                else -> Unit
            }
        }
    }
}

@Composable
private fun LinkBlock(post: BoardPost, link: String) {
    val context = LocalContext.current
    val preview = post.linkPreview
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clickable {
                BoardsLogic.absoluteUrl(link)?.let {
                    context.startActivity(Intent(Intent.ACTION_VIEW, Uri.parse(it)))
                }
            },
        horizontalArrangement = Arrangement.spacedBy(10.dp),
    ) {
        val image = preview?.image?.let { BoardsLogic.absoluteUrl(it) }
        if (image != null) {
            AsyncImage(
                model = image,
                contentDescription = null,
                modifier = Modifier
                    .size(56.dp)
                    .clip(RoundedCornerShape(6.dp)),
            )
        }
        Column(modifier = Modifier.weight(1f)) {
            Text(
                preview?.title?.ifBlank { link } ?: link,
                fontWeight = FontWeight.SemiBold,
                color = textPrimary(),
                maxLines = 2,
                overflow = TextOverflow.Ellipsis,
            )
            preview?.description?.takeIf { it.isNotBlank() }?.let {
                Text(it, color = textSecondary(), maxLines = 2, overflow = TextOverflow.Ellipsis)
            }
        }
    }
}

@Composable
private fun DrawingBlock(post: BoardPost) {
    val elements = remember(post.drawingData) { BoardsLogic.parseDrawingElements(post.drawingData) }
    val dark = isDarkTheme()
    val drawingLabel = L.text(R.string.mobile_boards_post_type_drawing)
    if (elements.isEmpty()) {
        Text(L.text(R.string.mobile_boards_post_drawingEmpty), color = textSecondary())
        return
    }
    Canvas(
        modifier = Modifier
            .fillMaxWidth()
            .height(160.dp)
            .clip(RoundedCornerShape(8.dp))
            .semantics { contentDescription = drawingLabel },
    ) {
        WhiteboardRenderer.drawGrid(this, dark)
        elements.forEach { WhiteboardRenderer.drawElement(this, it) }
    }
}

@Composable
private fun BoardEmbedWebView(url: String) {
    val embedLabel = L.text(R.string.mobile_boards_post_videoEmbed)
    AndroidView(
        factory = { ctx ->
            WebView(ctx).apply {
                webViewClient = WebViewClient()
                settings.javaScriptEnabled = true
                settings.domStorageEnabled = true
                settings.mediaPlaybackRequiresUserGesture = false
                loadUrl(url)
            }
        },
        modifier = Modifier
            .fillMaxWidth()
            .aspectRatio(16f / 9f)
            .clip(RoundedCornerShape(8.dp))
            .semantics { contentDescription = embedLabel },
    )
}

@androidx.annotation.OptIn(UnstableApi::class)
@Composable
private fun BoardMediaPlayer(url: String, video: Boolean) {
    val context = LocalContext.current
    val a11y = if (video) {
        L.text(R.string.mobile_boards_post_type_video)
    } else {
        L.text(R.string.mobile_boards_post_type_audio)
    }
    val player = remember(url) {
        ExoPlayer.Builder(context).build().apply {
            setMediaItem(MediaItem.fromUri(url))
            prepare()
        }
    }
    DisposableEffect(player) {
        onDispose { player.release() }
    }
    AndroidView(
        factory = { ctx ->
            PlayerView(ctx).apply {
                this.player = player
                useController = true
            }
        },
        modifier = Modifier
            .fillMaxWidth()
            .then(if (video) Modifier.aspectRatio(16f / 9f) else Modifier.height(64.dp))
            .clip(RoundedCornerShape(8.dp))
            .semantics { contentDescription = a11y },
    )
}
