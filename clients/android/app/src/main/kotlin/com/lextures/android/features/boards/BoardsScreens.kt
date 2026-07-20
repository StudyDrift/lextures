package com.lextures.android.features.boards

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.ChevronRight
import androidx.compose.material.icons.filled.GridView
import androidx.compose.material.icons.filled.MoreVert
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Checkbox
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.rememberModalBottomSheetState
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
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.OfflineBanner
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.lms.Board
import com.lextures.android.core.lms.BoardCopyMode
import com.lextures.android.core.lms.BoardsAdvancedLogic
import com.lextures.android.core.lms.BoardsApi
import com.lextures.android.core.lms.BoardsLogic
import com.lextures.android.core.lms.CourseSummary
import com.lextures.android.core.navigation.MobilePlatformFeatures
import com.lextures.android.core.network.ApiError
import com.lextures.android.core.offline.OfflineService
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsEmptyState
import com.lextures.android.features.home.LmsErrorBanner
import com.lextures.android.features.home.LmsSkeletonList
import kotlinx.coroutines.launch

@Composable
fun BoardsUnavailableScreen(modifier: Modifier = Modifier) {
    LmsEmptyState(
        icon = Icons.Default.GridView,
        title = L.text(R.string.mobile_boards_unavailableTitle),
        message = L.text(R.string.mobile_boards_unavailableMessage),
        modifier = modifier.padding(vertical = 24.dp),
    )
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun BoardsListScreen(
    session: AuthSession,
    course: CourseSummary,
    permissions: List<String>,
    currentUserId: String? = null,
    initialBoardId: String? = null,
    platformFeatures: MobilePlatformFeatures = MobilePlatformFeatures(),
    modifier: Modifier = Modifier,
) {
    val accessToken by session.accessToken.collectAsState()
    val context = LocalContext.current
    val offline = remember { OfflineService.get(context) }
    val isOnline by offline.networkMonitor.isOnline.collectAsState()
    val scope = rememberCoroutineScope()

    var boards by remember { mutableStateOf<List<Board>>(emptyList()) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var loading by remember { mutableStateOf(true) }
    var includeArchived by remember { mutableStateOf(false) }
    var featureUnavailable by remember { mutableStateOf(false) }
    var showNewBoard by remember { mutableStateOf(false) }
    var showCreateMenu by remember { mutableStateOf(false) }
    var showTemplatePicker by remember { mutableStateOf(false) }
    var newTitle by remember { mutableStateOf("") }
    var newDescription by remember { mutableStateOf("") }
    var openBoard by remember { mutableStateOf<Board?>(null) }
    var pendingOpenId by remember { mutableStateOf(initialBoardId) }
    var renameTarget by remember { mutableStateOf<Board?>(null) }
    var renameTitle by remember { mutableStateOf("") }
    var archiveTarget by remember { mutableStateOf<Board?>(null) }
    var duplicateTarget by remember { mutableStateOf<Board?>(null) }
    var overflowBoardId by remember { mutableStateOf<String?>(null) }
    val duplicateError = L.text(R.string.mobile_boards_templates_duplicateError)

    val canCreate = remember(course.courseCode, permissions) {
        BoardsLogic.canCreateBoards(course.courseCode, permissions)
    }
    val canUseTemplates = remember(course.isVisualBoardsEnabled, platformFeatures, canCreate) {
        BoardsAdvancedLogic.canUseTemplates(course.isVisualBoardsEnabled, platformFeatures, canCreate)
    }
    val advancedEnabled = remember(course.isVisualBoardsEnabled, platformFeatures) {
        BoardsAdvancedLogic.isAdvancedEnabled(course.isVisualBoardsEnabled, platformFeatures)
    }
    val visibleBoards = remember(boards, includeArchived) {
        BoardsLogic.sortedBoards(boards, includeArchived)
    }

    suspend fun load() {
        val token = accessToken ?: return
        loading = true
        errorMessage = null
        featureUnavailable = false
        try {
            boards = BoardsApi.listBoards(course.courseCode, includeArchived, token)
        } catch (e: ApiError.HttpStatus) {
            if (e.code == 404) {
                featureUnavailable = true
                boards = emptyList()
            } else {
                errorMessage = session.mapError(e)
            }
        } catch (e: Exception) {
            errorMessage = session.mapError(e)
        } finally {
            loading = false
        }
    }

    LaunchedEffect(accessToken, includeArchived) { load() }
    LaunchedEffect(boards, pendingOpenId, featureUnavailable) {
        val id = pendingOpenId ?: return@LaunchedEffect
        if (featureUnavailable || !course.isVisualBoardsEnabled) return@LaunchedEffect
        if (loading) return@LaunchedEffect
        val match = boards.firstOrNull { it.id == id }
        openBoard = match ?: Board(id = id, courseId = course.id, title = "")
        pendingOpenId = null
    }

    openBoard?.let { board ->
        BoardDetailScreen(
            session = session,
            course = course,
            boardId = board.id,
            titleHint = board.title,
            canManage = canCreate,
            permissions = permissions,
            currentUserId = currentUserId,
            platformFeatures = platformFeatures,
            onBack = { openBoard = null },
            onBoardChanged = { scope.launch { load() } },
        )
        return
    }

    if (featureUnavailable || !course.isVisualBoardsEnabled) {
        BoardsUnavailableScreen(modifier = modifier)
        return
    }

    Column(modifier = modifier, verticalArrangement = Arrangement.spacedBy(12.dp)) {
        if (!isOnline) OfflineBanner()
        errorMessage?.let { msg ->
            Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                LmsErrorBanner(message = msg)
                TextButton(onClick = { scope.launch { load() } }) {
                    Text(L.text(R.string.mobile_common_retry))
                }
            }
        }

        Row(
            modifier = Modifier.fillMaxWidth(),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Row(
                modifier = Modifier
                    .weight(1f)
                    .clickable { includeArchived = !includeArchived },
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Checkbox(checked = includeArchived, onCheckedChange = { includeArchived = it })
                Text(L.text(R.string.mobile_boards_showArchived), color = textPrimary())
            }
            if (canCreate) {
                TextButton(onClick = { showCreateMenu = true }) {
                    Icon(Icons.Default.Add, contentDescription = null)
                    Text(L.text(R.string.mobile_boards_newBoard))
                }
                DropdownMenu(
                    expanded = showCreateMenu,
                    onDismissRequest = { showCreateMenu = false },
                ) {
                    DropdownMenuItem(
                        text = { Text(L.text(R.string.mobile_boards_newBoard)) },
                        onClick = {
                            showCreateMenu = false
                            showNewBoard = true
                        },
                    )
                    if (canUseTemplates) {
                        DropdownMenuItem(
                            text = { Text(L.text(R.string.mobile_boards_templates_fromTemplate)) },
                            onClick = {
                                showCreateMenu = false
                                showTemplatePicker = true
                            },
                        )
                    }
                }
            }
        }

        when {
            loading && visibleBoards.isEmpty() -> LmsSkeletonList(count = 3)
            visibleBoards.isEmpty() -> LmsEmptyState(
                icon = Icons.Default.GridView,
                title = L.text(R.string.mobile_boards_emptyTitle),
                message = L.text(R.string.mobile_boards_emptyMessage),
            )
            else -> visibleBoards.forEach { board ->
                val a11y = buildString {
                    append(board.title)
                    if (board.description.isNotBlank()) {
                        append(", ")
                        append(board.description)
                    }
                    val updated = BoardsLogic.relativeUpdatedLabel(board)
                    if (updated.isNotBlank()) {
                        append(", ")
                        append(updated)
                    }
                }
                LmsCard(
                    modifier = Modifier
                        .semantics { contentDescription = a11y }
                        .clickable { openBoard = board },
                ) {
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Column(modifier = Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(4.dp)) {
                            Text(
                                board.title,
                                fontWeight = FontWeight.SemiBold,
                                color = textPrimary(),
                            )
                            if (board.description.isNotBlank()) {
                                Text(
                                    board.description,
                                    color = textSecondary(),
                                    maxLines = 2,
                                    overflow = TextOverflow.Ellipsis,
                                )
                            }
                            val updated = BoardsLogic.relativeUpdatedLabel(board)
                            if (updated.isNotBlank()) {
                                Text(
                                    L.format(R.string.mobile_boards_updatedRelative, updated),
                                    color = textSecondary(),
                                )
                            }
                            if (board.archived) {
                                Text(L.text(R.string.mobile_boards_archivedBadge), color = textSecondary())
                            }
                        }
                        if (canCreate) {
                            IconButton(onClick = { overflowBoardId = board.id }) {
                                Icon(
                                    Icons.Default.MoreVert,
                                    contentDescription = L.text(R.string.mobile_boards_overflowMenu),
                                )
                            }
                            DropdownMenu(
                                expanded = overflowBoardId == board.id,
                                onDismissRequest = { overflowBoardId = null },
                            ) {
                                DropdownMenuItem(
                                    text = { Text(L.text(R.string.mobile_boards_rename)) },
                                    onClick = {
                                        overflowBoardId = null
                                        renameTarget = board
                                        renameTitle = board.title
                                    },
                                )
                                if (advancedEnabled) {
                                    DropdownMenuItem(
                                        text = { Text(L.text(R.string.mobile_boards_templates_duplicate)) },
                                        onClick = {
                                            overflowBoardId = null
                                            duplicateTarget = board
                                        },
                                    )
                                }
                                if (!board.archived) {
                                    DropdownMenuItem(
                                        text = { Text(L.text(R.string.mobile_boards_archive)) },
                                        onClick = {
                                            overflowBoardId = null
                                            archiveTarget = board
                                        },
                                    )
                                }
                            }
                        }
                        Icon(Icons.Default.ChevronRight, contentDescription = null, tint = textSecondary())
                    }
                }
            }
        }
    }

    if (showNewBoard) {
        AlertDialog(
            onDismissRequest = { showNewBoard = false },
            title = { Text(L.text(R.string.mobile_boards_newBoard)) },
            text = {
                Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    OutlinedTextField(
                        value = newTitle,
                        onValueChange = { newTitle = it },
                        label = { Text(L.text(R.string.mobile_boards_titlePlaceholder)) },
                        modifier = Modifier.fillMaxWidth(),
                        singleLine = true,
                    )
                    OutlinedTextField(
                        value = newDescription,
                        onValueChange = { newDescription = it },
                        label = { Text(L.text(R.string.mobile_boards_descriptionPlaceholder)) },
                        modifier = Modifier.fillMaxWidth(),
                    )
                }
            },
            confirmButton = {
                TextButton(
                    onClick = {
                        val title = newTitle.trim()
                        if (title.isEmpty()) return@TextButton
                        val description = newDescription.trim()
                        showNewBoard = false
                        newTitle = ""
                        newDescription = ""
                        scope.launch {
                            val token = accessToken ?: return@launch
                            try {
                                val created = BoardsApi.createBoard(course.courseCode, title, description, token)
                                load()
                                openBoard = created
                            } catch (e: Exception) {
                                errorMessage = session.mapError(e)
                            }
                        }
                    },
                    enabled = newTitle.trim().isNotEmpty(),
                ) { Text(L.text(R.string.mobile_boards_create)) }
            },
            dismissButton = {
                TextButton(onClick = { showNewBoard = false }) {
                    Text(L.text(R.string.mobile_common_cancel))
                }
            },
        )
    }

    renameTarget?.let { target ->
        AlertDialog(
            onDismissRequest = { renameTarget = null },
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
                        val title = renameTitle.trim()
                        if (title.isEmpty()) return@TextButton
                        renameTarget = null
                        scope.launch {
                            val token = accessToken ?: return@launch
                            try {
                                BoardsApi.patchBoard(course.courseCode, target.id, title = title, accessToken = token)
                                load()
                            } catch (e: Exception) {
                                errorMessage = session.mapError(e)
                            }
                        }
                    },
                    enabled = renameTitle.trim().isNotEmpty(),
                ) { Text(L.text(R.string.mobile_common_save)) }
            },
            dismissButton = {
                TextButton(onClick = { renameTarget = null }) {
                    Text(L.text(R.string.mobile_common_cancel))
                }
            },
        )
    }

    archiveTarget?.let { target ->
        AlertDialog(
            onDismissRequest = { archiveTarget = null },
            title = { Text(L.text(R.string.mobile_boards_archiveConfirmTitle)) },
            text = { Text(L.text(R.string.mobile_boards_archiveConfirmMessage)) },
            confirmButton = {
                TextButton(
                    onClick = {
                        archiveTarget = null
                        scope.launch {
                            val token = accessToken ?: return@launch
                            try {
                                BoardsApi.patchBoard(
                                    course.courseCode,
                                    target.id,
                                    archived = true,
                                    accessToken = token,
                                )
                                load()
                            } catch (e: Exception) {
                                errorMessage = session.mapError(e)
                            }
                        }
                    },
                ) { Text(L.text(R.string.mobile_boards_archive)) }
            },
            dismissButton = {
                TextButton(onClick = { archiveTarget = null }) {
                    Text(L.text(R.string.mobile_common_cancel))
                }
            },
        )
    }

    duplicateTarget?.let { target ->
        AlertDialog(
            onDismissRequest = { duplicateTarget = null },
            title = { Text(L.text(R.string.mobile_boards_templates_duplicateTitle)) },
            text = { Text(L.text(R.string.mobile_boards_templates_duplicateMessage)) },
            confirmButton = {
                TextButton(
                    onClick = {
                        duplicateTarget = null
                        scope.launch {
                            val token = accessToken ?: return@launch
                            try {
                                val board = duplicateBoardWithPolling(
                                    courseCode = course.courseCode,
                                    sourceBoardId = target.id,
                                    mode = BoardCopyMode.Structure,
                                    title = target.title,
                                    accessToken = token,
                                )
                                load()
                                if (board != null) openBoard = board
                            } catch (e: Exception) {
                                errorMessage = duplicateError
                            }
                        }
                    },
                ) { Text(L.text(R.string.mobile_boards_templates_duplicateStructure)) }
            },
            dismissButton = {
                TextButton(
                    onClick = {
                        duplicateTarget = null
                        scope.launch {
                            val token = accessToken ?: return@launch
                            try {
                                val board = duplicateBoardWithPolling(
                                    courseCode = course.courseCode,
                                    sourceBoardId = target.id,
                                    mode = BoardCopyMode.Full,
                                    title = target.title,
                                    accessToken = token,
                                )
                                load()
                                if (board != null) openBoard = board
                            } catch (_: Exception) {
                                errorMessage = duplicateError
                            }
                        }
                    },
                ) { Text(L.text(R.string.mobile_boards_templates_duplicateFull)) }
            },
        )
    }

    if (showTemplatePicker) {
        ModalBottomSheet(
            onDismissRequest = { showTemplatePicker = false },
            sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true),
        ) {
            BoardTemplatePickerSheet(
                courseCode = course.courseCode,
                session = session,
                onCreated = { board ->
                    scope.launch { load() }
                    openBoard = board
                },
                onDismiss = { showTemplatePicker = false },
            )
        }
    }
}
