package com.lextures.android.features.boards

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.MoreVert
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.semantics.liveRegion
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.unit.dp
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.LifecycleEventObserver
import androidx.lifecycle.compose.LocalLifecycleOwner
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.lms.ArrangeBoardPostBody
import com.lextures.android.core.lms.Board
import com.lextures.android.core.lms.BoardLayout
import com.lextures.android.core.lms.BoardLayoutApi
import com.lextures.android.core.lms.BoardModerationApi
import com.lextures.android.core.lms.BoardPost
import com.lextures.android.core.lms.BoardPostsApi
import com.lextures.android.core.lms.BoardSection
import com.lextures.android.core.lms.BoardSortMode
import com.lextures.android.core.lms.BoardsApi
import com.lextures.android.core.lms.BoardsLogic
import com.lextures.android.core.lms.CourseSummary
import com.lextures.android.core.network.ApiError
import com.lextures.android.core.realtime.BoardSocket
import com.lextures.android.features.home.LmsErrorBanner
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import kotlin.math.max

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun BoardDetailScreen(
    session: AuthSession,
    course: CourseSummary,
    boardId: String,
    titleHint: String = "",
    canManage: Boolean = false,
    permissions: List<String> = emptyList(),
    currentUserId: String? = null,
    onBack: () -> Unit,
    onBoardChanged: () -> Unit = {},
    modifier: Modifier = Modifier,
) {
    val accessToken by session.accessToken.collectAsState()
    val scope = rememberCoroutineScope()
    val socket = remember(boardId) { BoardSocket() }
    val syncState by socket.connectionState.collectAsState()
    val socketRevision by socket.revision.collectAsState()
    val connectRevision by socket.connectRevision.collectAsState()
    val lockedNotice by socket.lockedOrFrozenNotice.collectAsState()
    val lastRefetchPlan by socket.lastRefetchPlan.collectAsState()
    val lifecycleOwner = LocalLifecycleOwner.current

    var board by remember { mutableStateOf<Board?>(null) }
    var posts by remember { mutableStateOf<List<BoardPost>>(emptyList()) }
    var sections by remember { mutableStateOf<List<BoardSection>>(emptyList()) }
    var sortMode by remember { mutableStateOf(BoardSortMode.Newest) }
    var loading by remember { mutableStateOf(true) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var unavailable by remember { mutableStateOf(false) }
    var menuOpen by remember { mutableStateOf(false) }
    var layoutMenuOpen by remember { mutableStateOf(false) }
    var sortMenuOpen by remember { mutableStateOf(false) }
    var showRename by remember { mutableStateOf(false) }
    var renameTitle by remember { mutableStateOf("") }
    var showArchive by remember { mutableStateOf(false) }
    var showComposer by remember { mutableStateOf(false) }
    var showShare by remember { mutableStateOf(false) }
    var showModerationQueue by remember { mutableStateOf(false) }
    var showModerationSettings by remember { mutableStateOf(false) }
    var editingPost by remember { mutableStateOf<BoardPost?>(null) }
    var editTitle by remember { mutableStateOf("") }
    var editBody by remember { mutableStateOf("") }
    var editLink by remember { mutableStateOf("") }
    var pendingLayout by remember { mutableStateOf<BoardLayout?>(null) }
    var announceMessage by remember { mutableStateOf("") }
    var knownPostIds by remember { mutableStateOf<Set<String>>(emptySet()) }

    val managesBoard = remember(board, course.courseCode, permissions, canManage) {
        BoardsLogic.canManageBoard(board, course.courseCode, permissions) || canManage
    }
    val canPost = remember(board, course.courseCode, permissions, managesBoard) {
        BoardsLogic.canPost(board, course.courseCode, permissions) &&
            BoardsLogic.canWritePosts(board, managesBoard)
    }
    val boardLocked = remember(board?.locked) { BoardsLogic.isBoardLocked(board) }
    val boardFrozen = remember(board?.frozenUntil) { BoardsLogic.isBoardFrozen(board) }
    val unfrozenAnnounce = L.text(R.string.mobile_boards_moderation_unfrozenAnnounce)
    val resolvedLayout = remember(board?.layout) { BoardsLogic.resolveLayout(board?.layout) }
    val forbiddenMessage = L.text(R.string.mobile_boards_post_forbidden)
    val movedMessage = L.text(R.string.mobile_boards_arrange_moved)
    val lockedAnnounce = L.text(R.string.mobile_boards_layout_lockedAnnounce)
    val unlockedAnnounce = L.text(R.string.mobile_boards_layout_unlockedAnnounce)
    val sectionCreatedMessage = L.text(R.string.mobile_boards_section_created)
    val sectionDeletedMessage = L.text(R.string.mobile_boards_section_deleted)
    val layoutLabels = BoardLayout.entries.associateWith { L.text(layoutStringRes(it)) }
    val layoutChangedTemplate = L.text(R.string.mobile_boards_layout_changed)
    val cardAddedMessage = L.text(R.string.mobile_boards_sync_cardAdded)
    val cardsAddedTemplate = L.text(R.string.mobile_boards_sync_cardsAdded)
    val boardUpdatedMessage = L.text(R.string.mobile_boards_sync_boardUpdated)
    val lockedNoticeMessage = L.text(R.string.mobile_boards_sync_lockedNotice)

    suspend fun load(quiet: Boolean = false, fromSocket: Boolean = false) {
        val token = accessToken ?: return
        if (!quiet) loading = true
        errorMessage = null
        unavailable = false
        val previousIds = knownPostIds
        try {
            board = BoardsApi.fetchBoard(course.courseCode, boardId, token)
            posts = BoardPostsApi.listPosts(course.courseCode, boardId, token)
            sections = BoardLayoutApi.listSections(course.courseCode, boardId, token)
            val nextIds = posts.map { it.id }.toSet()
            if (fromSocket && previousIds.isNotEmpty()) {
                val added = (nextIds - previousIds).size
                val announceCount = max(added, lastRefetchPlan.createdCount)
                announceMessage = when {
                    announceCount > 1 -> cardsAddedTemplate
                        .replace("%1\$d", announceCount.toString())
                        .replace("%d", announceCount.toString())
                    announceCount == 1 -> cardAddedMessage
                    lastRefetchPlan.full || lastRefetchPlan.postId != null -> boardUpdatedMessage
                    else -> announceMessage
                }
            }
            knownPostIds = nextIds
        } catch (e: ApiError.HttpStatus) {
            if (e.code == 404 || e.code == 403) {
                unavailable = true
                board = null
                posts = emptyList()
                sections = emptyList()
                knownPostIds = emptySet()
            } else if (!quiet) {
                errorMessage = session.mapError(e)
            }
        } catch (e: Exception) {
            if (!quiet) errorMessage = session.mapError(e)
        } finally {
            if (!quiet) loading = false
        }
    }

    suspend fun arrange(post: BoardPost, input: ArrangeBoardPostBody) {
        val token = accessToken ?: return
        if (!BoardsLogic.canArrangePost(post, board, currentUserId, managesBoard)) return
        val previous = posts
        posts = posts.map {
            if (it.id != post.id) {
                it
            } else {
                it.copy(
                    sectionId = input.sectionId ?: it.sectionId,
                    sortIndex = input.sortIndex ?: it.sortIndex,
                    position = input.position ?: it.position,
                    eventDate = when {
                        input.eventDate == null -> it.eventDate
                        input.eventDate.isEmpty() -> null
                        else -> input.eventDate
                    },
                    lat = input.lat ?: it.lat,
                    lng = input.lng ?: it.lng,
                )
            }
        }
        try {
            val updated = BoardLayoutApi.arrangePost(course.courseCode, boardId, post.id, input, token)
            posts = posts.map { if (it.id == updated.id) updated else it }
            announceMessage = movedMessage
        } catch (e: ApiError.HttpStatus) {
            posts = previous
            errorMessage = when {
                e.code == 403 && BoardsLogic.isLockOrFreezeMessage(e.message) -> lockedNoticeMessage
                e.code == 403 -> forbiddenMessage
                else -> session.mapError(e)
            }
        } catch (e: Exception) {
            posts = previous
            errorMessage = session.mapError(e)
        }
    }

    suspend fun hidePost(post: BoardPost) {
        val token = accessToken ?: return
        try {
            val updated = BoardModerationApi.hidePost(course.courseCode, boardId, post.id, accessToken = token)
            posts = if (managesBoard) {
                posts.map { if (it.id == updated.id) updated else it }
            } else {
                posts.filter { it.id != post.id }
            }
        } catch (e: Exception) {
            errorMessage = session.mapError(e)
        }
    }

    suspend fun removeModeratedPost(post: BoardPost) {
        val token = accessToken ?: return
        try {
            BoardModerationApi.removePost(course.courseCode, boardId, post.id, accessToken = token)
            posts = posts.filter { it.id != post.id }
        } catch (e: Exception) {
            errorMessage = session.mapError(e)
        }
    }

    suspend fun changeLayout(layout: BoardLayout) {
        val token = accessToken ?: return
        try {
            board = BoardsApi.patchBoard(
                course.courseCode,
                boardId,
                layout = layout.apiValue,
                accessToken = token,
            )
            val label = layoutLabels[layout].orEmpty()
            announceMessage = layoutChangedTemplate.replace("%1\$s", label).replace("%@", label)
            if (layout == BoardLayout.Columns) {
                sections = BoardLayoutApi.listSections(course.courseCode, boardId, token)
                posts = BoardPostsApi.listPosts(course.courseCode, boardId, token)
            }
        } catch (e: Exception) {
            errorMessage = session.mapError(e)
        }
    }

    DisposableEffect(boardId) {
        socket.connect(course.courseCode, boardId) { session.accessToken.value }
        onDispose { socket.disconnect() }
    }
    DisposableEffect(lifecycleOwner, boardId) {
        val observer = LifecycleEventObserver { _, event ->
            when (event) {
                Lifecycle.Event.ON_STOP -> socket.disconnect()
                Lifecycle.Event.ON_START -> {
                    socket.connect(course.courseCode, boardId) { session.accessToken.value }
                    scope.launch { load(quiet = true) }
                }
                else -> Unit
            }
        }
        lifecycleOwner.lifecycle.addObserver(observer)
        onDispose { lifecycleOwner.lifecycle.removeObserver(observer) }
    }
    LaunchedEffect(accessToken, boardId) { load() }
    LaunchedEffect(socketRevision) {
        if (socketRevision > 0) load(quiet = true, fromSocket = true)
    }
    LaunchedEffect(connectRevision) {
        if (connectRevision > 0) load(quiet = true)
    }
    LaunchedEffect(lockedNotice) {
        if (lockedNotice) {
            announceMessage = lockedNoticeMessage
            socket.clearLockedOrFrozenNotice()
            load(quiet = true)
        }
    }
    LaunchedEffect(board?.frozenUntil) {
        val until = BoardsLogic.parseBoardInstant(board?.frozenUntil) ?: return@LaunchedEffect
        val delayMs = until.toEpochMilli() - System.currentTimeMillis()
        if (delayMs <= 0L) {
            load(quiet = true)
            return@LaunchedEffect
        }
        delay(delayMs + 250L)
        load(quiet = true)
        if (!BoardsLogic.isBoardFrozen(board)) {
            announceMessage = unfrozenAnnounce
        }
    }

    val title = board?.title?.takeIf { it.isNotBlank() }
        ?: titleHint.takeIf { it.isNotBlank() }
        ?: L.text(R.string.mobile_boards_detailTitle)

    Column(modifier = modifier) {
        TopAppBar(
            title = { Text(title) },
            navigationIcon = {
                IconButton(onClick = onBack) {
                    Icon(
                        Icons.AutoMirrored.Filled.ArrowBack,
                        contentDescription = L.text(R.string.mobile_common_close),
                    )
                }
            },
            actions = {
                if (canPost && board != null && !unavailable) {
                    IconButton(onClick = { showComposer = true }) {
                        Icon(
                            Icons.Default.Add,
                            contentDescription = L.text(R.string.mobile_boards_compose_openAria),
                        )
                    }
                }
                if (managesBoard && board != null && !unavailable) {
                    IconButton(onClick = { menuOpen = true }) {
                        Icon(
                            Icons.Default.MoreVert,
                            contentDescription = L.text(R.string.mobile_boards_overflowMenu),
                        )
                    }
                    DropdownMenu(expanded = menuOpen, onDismissRequest = { menuOpen = false }) {
                        DropdownMenuItem(
                            text = { Text(L.text(R.string.mobile_boards_share_action)) },
                            onClick = {
                                menuOpen = false
                                showShare = true
                            },
                        )
                        DropdownMenuItem(
                            text = { Text(L.text(R.string.mobile_boards_moderation_queueAction)) },
                            onClick = {
                                menuOpen = false
                                showModerationQueue = true
                            },
                        )
                        DropdownMenuItem(
                            text = { Text(L.text(R.string.mobile_boards_moderation_settingsTitle)) },
                            onClick = {
                                menuOpen = false
                                showModerationSettings = true
                            },
                        )
                        DropdownMenuItem(
                            text = { Text(L.text(R.string.mobile_boards_layout_switcherAria)) },
                            onClick = {
                                menuOpen = false
                                layoutMenuOpen = true
                            },
                        )
                        DropdownMenuItem(
                            text = {
                                Text(
                                    if (board?.layoutLocked == true) {
                                        L.text(R.string.mobile_boards_layout_unlock)
                                    } else {
                                        L.text(R.string.mobile_boards_layout_lock)
                                    },
                                )
                            },
                            onClick = {
                                menuOpen = false
                                scope.launch {
                                    val token = accessToken ?: return@launch
                                    val current = board ?: return@launch
                                    try {
                                        board = BoardsApi.patchBoard(
                                            course.courseCode,
                                            boardId,
                                            layoutLocked = !current.layoutLocked,
                                            accessToken = token,
                                        )
                                        announceMessage = if (board?.layoutLocked == true) {
                                            lockedAnnounce
                                        } else {
                                            unlockedAnnounce
                                        }
                                    } catch (e: Exception) {
                                        errorMessage = session.mapError(e)
                                    }
                                }
                            },
                        )
                        DropdownMenuItem(
                            text = { Text(L.text(R.string.mobile_boards_rename)) },
                            onClick = {
                                menuOpen = false
                                renameTitle = board?.title.orEmpty()
                                showRename = true
                            },
                        )
                        if (board?.archived != true) {
                            DropdownMenuItem(
                                text = { Text(L.text(R.string.mobile_boards_archive)) },
                                onClick = {
                                    menuOpen = false
                                    showArchive = true
                                },
                            )
                        }
                    }
                    DropdownMenu(expanded = layoutMenuOpen, onDismissRequest = { layoutMenuOpen = false }) {
                        BoardLayout.entries.forEach { layout ->
                            DropdownMenuItem(
                                text = { Text(L.text(layoutStringRes(layout))) },
                                onClick = {
                                    layoutMenuOpen = false
                                    if (layout == resolvedLayout) return@DropdownMenuItem
                                    if (resolvedLayout == BoardLayout.Canvas && layout != BoardLayout.Canvas) {
                                        pendingLayout = layout
                                    } else {
                                        scope.launch { changeLayout(layout) }
                                    }
                                },
                            )
                        }
                    }
                }
            },
        )

        when {
            unavailable -> BoardsUnavailableScreen()
            loading && board == null -> {
                Column(
                    modifier = Modifier.fillMaxWidth().padding(24.dp),
                    horizontalAlignment = Alignment.CenterHorizontally,
                ) {
                    CircularProgressIndicator()
                }
            }
            else -> {
                Column(
                    modifier = Modifier
                        .padding(horizontal = 16.dp)
                        .verticalScroll(rememberScrollState()),
                    verticalArrangement = Arrangement.spacedBy(12.dp),
                ) {
                    errorMessage?.let { msg ->
                        Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                            LmsErrorBanner(message = msg)
                            TextButton(onClick = { scope.launch { load() } }) {
                                Text(L.text(R.string.mobile_common_retry))
                            }
                        }
                    }
                    if (board != null && !unavailable) {
                        BoardSyncStatusChip(state = syncState)
                    }
                    board?.description?.takeIf { it.isNotBlank() }?.let { desc ->
                        Text(desc, color = textSecondary())
                    }
                    if (boardLocked) {
                        Text(
                            L.text(R.string.mobile_boards_moderation_lockedBanner),
                            color = androidx.compose.ui.graphics.Color(0xFFB45309),
                            modifier = Modifier.semantics {
                                liveRegion = androidx.compose.ui.semantics.LiveRegionMode.Polite
                            },
                        )
                    } else if (boardFrozen) {
                        Text(
                            L.text(R.string.mobile_boards_moderation_frozenBanner),
                            color = androidx.compose.ui.graphics.Color(0xFFB45309),
                            modifier = Modifier.semantics {
                                liveRegion = androidx.compose.ui.semantics.LiveRegionMode.Polite
                            },
                        )
                    }
                    if (board?.layoutLocked == true) {
                        Text(L.text(R.string.mobile_boards_layout_lockedBadge), color = textSecondary())
                    }
                    if (!BoardsLogic.layoutHidesSortControls(resolvedLayout)) {
                        TextButton(onClick = { sortMenuOpen = true }) {
                            Text(L.text(R.string.mobile_boards_sort_label))
                        }
                        DropdownMenu(expanded = sortMenuOpen, onDismissRequest = { sortMenuOpen = false }) {
                            BoardSortMode.entries.forEach { mode ->
                                DropdownMenuItem(
                                    text = { Text(L.text(sortStringRes(mode))) },
                                    onClick = {
                                        sortMode = mode
                                        sortMenuOpen = false
                                    },
                                )
                            }
                        }
                    }
                    if (announceMessage.isNotBlank()) {
                        Text(
                            announceMessage,
                            color = textSecondary(),
                            modifier = Modifier.semantics { liveRegion = androidx.compose.ui.semantics.LiveRegionMode.Polite },
                        )
                    }
                    when {
                        loading && posts.isEmpty() -> {
                            Column(
                                modifier = Modifier.fillMaxWidth().padding(24.dp),
                                horizontalAlignment = Alignment.CenterHorizontally,
                            ) { CircularProgressIndicator() }
                        }
                        board != null -> {
                            CompositionLocalProvider(
                                LocalBoardEngagement provides BoardEngagementHandlers(
                                    courseCode = course.courseCode,
                                    accessToken = accessToken,
                                    onPostUpdate = { updated ->
                                        posts = posts.map { if (it.id == updated.id) updated else it }
                                    },
                                    onAnnounce = { announceMessage = it },
                                    onHidePost = { post -> scope.launch { hidePost(post) } },
                                    onRemovePost = { post -> scope.launch { removeModeratedPost(post) } },
                                ),
                            ) {
                            BoardSurface(
                                board = board!!,
                                posts = posts,
                                sections = sections,
                                sortMode = sortMode,
                                canManage = managesBoard,
                                currentUserId = currentUserId,
                                onEdit = { post ->
                                    editingPost = post
                                    editTitle = post.title
                                    editBody = BoardsLogic.bodyPlainText(post)
                                    editLink = post.linkUrl.orEmpty()
                                },
                                onDelete = { post ->
                                    scope.launch {
                                        val token = accessToken ?: return@launch
                                        val previous = posts
                                        posts = posts.filter { it.id != post.id }
                                        try {
                                            BoardPostsApi.deletePost(
                                                course.courseCode,
                                                boardId,
                                                post.id,
                                                token,
                                            )
                                        } catch (e: ApiError.HttpStatus) {
                                            posts = previous
                                            errorMessage = if (e.code == 403) {
                                                forbiddenMessage
                                            } else {
                                                session.mapError(e)
                                            }
                                        } catch (e: Exception) {
                                            posts = previous
                                            errorMessage = session.mapError(e)
                                        }
                                    }
                                },
                                onArrange = { post, input ->
                                    scope.launch { arrange(post, input) }
                                },
                                onCreateSection = if (managesBoard) {
                                    { title ->
                                        scope.launch {
                                            val token = accessToken ?: return@launch
                                            try {
                                                val created = BoardLayoutApi.createSection(
                                                    course.courseCode,
                                                    boardId,
                                                    title,
                                                    accessToken = token,
                                                )
                                                sections = sections + created
                                                announceMessage = sectionCreatedMessage
                                            } catch (e: Exception) {
                                                errorMessage = session.mapError(e)
                                            }
                                        }
                                    }
                                } else {
                                    null
                                },
                                onDeleteSection = if (managesBoard) {
                                    { sectionId ->
                                        scope.launch {
                                            val token = accessToken ?: return@launch
                                            try {
                                                BoardLayoutApi.deleteSection(
                                                    course.courseCode,
                                                    boardId,
                                                    sectionId,
                                                    token,
                                                )
                                                sections = sections.filter { it.id != sectionId }
                                                posts = BoardPostsApi.listPosts(course.courseCode, boardId, token)
                                                announceMessage = sectionDeletedMessage
                                            } catch (e: Exception) {
                                                errorMessage = session.mapError(e)
                                            }
                                        }
                                    }
                                } else {
                                    null
                                },
                            )
                            }
                        }
                    }
                    itemSpacer()
                }
            }
        }
    }

    if (showComposer) {
        val token = accessToken
        if (token != null) {
            BoardComposerSheet(
                session = session,
                courseCode = course.courseCode,
                boardId = boardId,
                accessToken = token,
                onDismiss = { showComposer = false },
                onCreated = { created ->
                    posts = listOf(created) + posts
                },
            )
        }
    }

    if (showShare) {
        board?.let { current ->
            BoardShareSheet(
                courseCode = course.courseCode,
                board = current,
                accessToken = accessToken,
                onBoardUpdated = {
                    board = it
                    onBoardChanged()
                },
                onDismiss = { showShare = false },
            )
        }
    }

    if (showModerationQueue) {
        ModerationQueueScreen(
            courseCode = course.courseCode,
            boardId = boardId,
            accessToken = accessToken,
            onDismiss = { showModerationQueue = false },
            onChanged = { scope.launch { load(quiet = true) } },
        )
    }

    if (showModerationSettings) {
        board?.let { current ->
            BoardModerationSettings(
                courseCode = course.courseCode,
                board = current,
                accessToken = accessToken,
                onDismiss = { showModerationSettings = false },
                onBoardUpdated = {
                    board = it
                    onBoardChanged()
                },
                onAnnounce = { announceMessage = it },
            )
        }
    }

    editingPost?.let { post ->
        AlertDialog(
            onDismissRequest = { editingPost = null },
            title = { Text(L.text(R.string.mobile_boards_post_edit)) },
            text = {
                Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    OutlinedTextField(
                        value = editTitle,
                        onValueChange = { editTitle = it },
                        label = { Text(L.text(R.string.mobile_boards_compose_titleLabel)) },
                        modifier = Modifier.fillMaxWidth(),
                        singleLine = true,
                    )
                    if (post.contentType == "text") {
                        OutlinedTextField(
                            value = editBody,
                            onValueChange = { editBody = it },
                            label = { Text(L.text(R.string.mobile_boards_compose_bodyLabel)) },
                            modifier = Modifier.fillMaxWidth(),
                        )
                    }
                    if (post.contentType == "link" || post.contentType == "video") {
                        OutlinedTextField(
                            value = editLink,
                            onValueChange = { editLink = it },
                            label = { Text(L.text(R.string.mobile_boards_compose_linkLabel)) },
                            modifier = Modifier.fillMaxWidth(),
                            singleLine = true,
                        )
                    }
                }
            },
            confirmButton = {
                TextButton(onClick = {
                    val target = editingPost ?: return@TextButton
                    editingPost = null
                    scope.launch {
                        val token = accessToken ?: return@launch
                        try {
                            val updated = BoardPostsApi.patchPost(
                                courseCode = course.courseCode,
                                boardId = boardId,
                                postId = target.id,
                                title = editTitle.trim(),
                                body = if (target.contentType == "text") {
                                    BoardsLogic.makeTextBody(editBody)
                                } else {
                                    null
                                },
                                linkUrl = if (target.contentType == "link" || target.contentType == "video") {
                                    editLink.trim()
                                } else {
                                    null
                                },
                                accessToken = token,
                            )
                            posts = posts.map { if (it.id == updated.id) updated else it }
                        } catch (e: ApiError.HttpStatus) {
                            errorMessage = if (e.code == 403) {
                                forbiddenMessage
                            } else {
                                session.mapError(e)
                            }
                        } catch (e: Exception) {
                            errorMessage = session.mapError(e)
                        }
                    }
                }) { Text(L.text(R.string.mobile_common_save)) }
            },
            dismissButton = {
                TextButton(onClick = { editingPost = null }) {
                    Text(L.text(R.string.mobile_common_cancel))
                }
            },
        )
    }

    if (showRename) {
        AlertDialog(
            onDismissRequest = { showRename = false },
            title = { Text(L.text(R.string.mobile_boards_rename)) },
            text = {
                OutlinedTextField(
                    value = renameTitle,
                    onValueChange = { renameTitle = it },
                    label = { Text(L.text(R.string.mobile_boards_titlePlaceholder)) },
                    modifier = Modifier.fillMaxWidth(),
                    singleLine = true,
                )
            },
            confirmButton = {
                TextButton(
                    onClick = {
                        val next = renameTitle.trim()
                        if (next.isEmpty()) return@TextButton
                        showRename = false
                        scope.launch {
                            val token = accessToken ?: return@launch
                            try {
                                board = BoardsApi.patchBoard(
                                    course.courseCode,
                                    boardId,
                                    title = next,
                                    accessToken = token,
                                )
                                onBoardChanged()
                            } catch (e: Exception) {
                                errorMessage = session.mapError(e)
                            }
                        }
                    },
                    enabled = renameTitle.trim().isNotEmpty(),
                ) { Text(L.text(R.string.mobile_common_save)) }
            },
            dismissButton = {
                TextButton(onClick = { showRename = false }) {
                    Text(L.text(R.string.mobile_common_cancel))
                }
            },
        )
    }

    if (showArchive) {
        AlertDialog(
            onDismissRequest = { showArchive = false },
            title = { Text(L.text(R.string.mobile_boards_archiveConfirmTitle)) },
            text = { Text(L.text(R.string.mobile_boards_archiveConfirmMessage)) },
            confirmButton = {
                TextButton(
                    onClick = {
                        showArchive = false
                        scope.launch {
                            val token = accessToken ?: return@launch
                            try {
                                BoardsApi.patchBoard(
                                    course.courseCode,
                                    boardId,
                                    archived = true,
                                    accessToken = token,
                                )
                                onBoardChanged()
                                onBack()
                            } catch (e: Exception) {
                                errorMessage = session.mapError(e)
                            }
                        }
                    },
                ) { Text(L.text(R.string.mobile_boards_archive)) }
            },
            dismissButton = {
                TextButton(onClick = { showArchive = false }) {
                    Text(L.text(R.string.mobile_common_cancel))
                }
            },
        )
    }

    pendingLayout?.let { layout ->
        AlertDialog(
            onDismissRequest = { pendingLayout = null },
            title = { Text(L.text(R.string.mobile_boards_layout_switchConfirmLabel)) },
            text = { Text(L.text(R.string.mobile_boards_layout_switchConfirm)) },
            confirmButton = {
                TextButton(onClick = {
                    pendingLayout = null
                    scope.launch { changeLayout(layout) }
                }) { Text(L.text(R.string.mobile_boards_layout_switchConfirmLabel)) }
            },
            dismissButton = {
                TextButton(onClick = { pendingLayout = null }) {
                    Text(L.text(R.string.mobile_common_cancel))
                }
            },
        )
    }
}

@Composable
private fun itemSpacer() {
    Spacer(modifier = Modifier.padding(8.dp))
}

private fun layoutStringRes(layout: BoardLayout): Int = when (layout) {
    BoardLayout.Wall -> R.string.mobile_boards_layout_wall
    BoardLayout.Stream -> R.string.mobile_boards_layout_stream
    BoardLayout.Grid -> R.string.mobile_boards_layout_grid
    BoardLayout.Columns -> R.string.mobile_boards_layout_columns
    BoardLayout.Canvas -> R.string.mobile_boards_layout_canvas
    BoardLayout.Timeline -> R.string.mobile_boards_layout_timeline
    BoardLayout.Map -> R.string.mobile_boards_layout_map
}

private fun sortStringRes(mode: BoardSortMode): Int = when (mode) {
    BoardSortMode.Newest -> R.string.mobile_boards_sort_newest
    BoardSortMode.Oldest -> R.string.mobile_boards_sort_oldest
    BoardSortMode.Author -> R.string.mobile_boards_sort_author
    BoardSortMode.MostReacted -> R.string.mobile_boards_sort_mostReacted
}
