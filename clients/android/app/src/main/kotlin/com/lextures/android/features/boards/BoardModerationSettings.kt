package com.lextures.android.features.boards

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.ExposedDropdownMenuBox
import androidx.compose.material3.ExposedDropdownMenuDefaults
import androidx.compose.material3.MenuAnchorType
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.lextures.android.R
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.lms.Board
import com.lextures.android.core.lms.BoardFilterAction
import com.lextures.android.core.lms.BoardModerationApi
import com.lextures.android.core.lms.BoardModerationMode
import com.lextures.android.core.lms.BoardsLogic
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun BoardModerationSettings(
    courseCode: String,
    board: Board,
    accessToken: String?,
    onDismiss: () -> Unit,
    onBoardUpdated: (Board) -> Unit,
    onAnnounce: (String) -> Unit,
) {
    val sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)
    val scope = rememberCoroutineScope()
    var moderationMode by remember(board.id, board.moderationMode) {
        mutableStateOf(BoardsLogic.moderationMode(board))
    }
    var filterAction by remember(board.id, board.filterAction) {
        mutableStateOf(BoardsLogic.filterAction(board))
    }
    var modeMenuOpen by remember { mutableStateOf(false) }
    var filterMenuOpen by remember { mutableStateOf(false) }
    var saving by remember { mutableStateOf(false) }
    var errorMessage by remember { mutableStateOf<String?>(null) }

    val floorLocked = BoardsLogic.moderationControlsLockedByOrgFloor(board)
    val frozen = BoardsLogic.isBoardFrozen(board)
    val settingsError = L.text(R.string.mobile_boards_moderation_settingsError)
    val lockedAnnounce = L.text(R.string.mobile_boards_moderation_lockedAnnounce)
    val unlockedAnnounce = L.text(R.string.mobile_boards_moderation_unlockedAnnounce)
    val frozenAnnounceTemplate = L.text(R.string.mobile_boards_moderation_frozenAnnounce)
    val unfrozenAnnounce = L.text(R.string.mobile_boards_moderation_unfrozenAnnounce)
    val modeOpen = L.text(R.string.mobile_boards_moderation_mode_open)
    val modeApproval = L.text(R.string.mobile_boards_moderation_mode_approval)
    val filterFlag = L.text(R.string.mobile_boards_moderation_filter_flag)
    val filterBlock = L.text(R.string.mobile_boards_moderation_filter_block)

    fun modeLabel(mode: BoardModerationMode): String = when (mode) {
        BoardModerationMode.Open -> modeOpen
        BoardModerationMode.Approval -> modeApproval
    }

    fun filterLabel(action: BoardFilterAction): String = when (action) {
        BoardFilterAction.Flag -> filterFlag
        BoardFilterAction.Block -> filterBlock
    }

    fun patch(
        moderationModeValue: String? = null,
        filterActionValue: String? = null,
        locked: Boolean? = null,
        frozenUntil: String? = null,
        freezeMinutes: Int? = null,
    ) {
        val token = accessToken ?: return
        scope.launch {
            saving = true
            errorMessage = null
            try {
                val updated = BoardModerationApi.patchBoardModeration(
                    courseCode = courseCode,
                    boardId = board.id,
                    moderationMode = moderationModeValue,
                    filterAction = filterActionValue,
                    locked = locked,
                    frozenUntil = frozenUntil,
                    freezeMinutes = freezeMinutes,
                    accessToken = token,
                )
                onBoardUpdated(updated)
                when {
                    locked != null -> onAnnounce(if (locked) lockedAnnounce else unlockedAnnounce)
                    freezeMinutes != null -> onAnnounce(
                        frozenAnnounceTemplate
                            .replace("%d", freezeMinutes.toString())
                            .replace("%1\$d", freezeMinutes.toString()),
                    )
                    frozenUntil == "" -> onAnnounce(unfrozenAnnounce)
                }
            } catch (_: Exception) {
                errorMessage = settingsError
                moderationMode = BoardsLogic.moderationMode(board)
                filterAction = BoardsLogic.filterAction(board)
            } finally {
                saving = false
            }
        }
    }

    ModalBottomSheet(onDismissRequest = onDismiss, sheetState = sheetState) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 16.dp, vertical = 8.dp)
                .verticalScroll(rememberScrollState()),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            Text(
                L.text(R.string.mobile_boards_moderation_settingsTitle),
                fontWeight = FontWeight.SemiBold,
                color = textPrimary(),
            )
            errorMessage?.let { Text(it, color = Color.Red) }
            if (floorLocked) {
                Text(L.text(R.string.mobile_boards_moderation_minorsFloor), color = Color(0xFFB45309))
            }

            ExposedDropdownMenuBox(
                expanded = modeMenuOpen && !floorLocked,
                onExpandedChange = { if (!floorLocked && !saving) modeMenuOpen = it },
            ) {
                OutlinedTextField(
                    value = modeLabel(moderationMode),
                    onValueChange = {},
                    readOnly = true,
                    enabled = !floorLocked && !saving,
                    label = { Text(L.text(R.string.mobile_boards_moderation_modeLabel)) },
                    trailingIcon = { ExposedDropdownMenuDefaults.TrailingIcon(expanded = modeMenuOpen) },
                    modifier = Modifier
                        .menuAnchor(MenuAnchorType.PrimaryNotEditable)
                        .fillMaxWidth(),
                )
                ExposedDropdownMenu(
                    expanded = modeMenuOpen && !floorLocked,
                    onDismissRequest = { modeMenuOpen = false },
                ) {
                    BoardModerationMode.entries.forEach { mode ->
                        DropdownMenuItem(
                            text = { Text(modeLabel(mode)) },
                            onClick = {
                                moderationMode = mode
                                modeMenuOpen = false
                                patch(moderationModeValue = mode.apiValue)
                            },
                        )
                    }
                }
            }

            ExposedDropdownMenuBox(
                expanded = filterMenuOpen && !floorLocked,
                onExpandedChange = { if (!floorLocked && !saving) filterMenuOpen = it },
            ) {
                OutlinedTextField(
                    value = filterLabel(filterAction),
                    onValueChange = {},
                    readOnly = true,
                    enabled = !floorLocked && !saving,
                    label = { Text(L.text(R.string.mobile_boards_moderation_filterLabel)) },
                    trailingIcon = { ExposedDropdownMenuDefaults.TrailingIcon(expanded = filterMenuOpen) },
                    modifier = Modifier
                        .menuAnchor(MenuAnchorType.PrimaryNotEditable)
                        .fillMaxWidth(),
                )
                ExposedDropdownMenu(
                    expanded = filterMenuOpen && !floorLocked,
                    onDismissRequest = { filterMenuOpen = false },
                ) {
                    BoardFilterAction.entries.forEach { action ->
                        DropdownMenuItem(
                            text = { Text(filterLabel(action)) },
                            onClick = {
                                filterAction = action
                                filterMenuOpen = false
                                patch(filterActionValue = action.apiValue)
                            },
                        )
                    }
                }
            }

            TextButton(enabled = !saving, onClick = { patch(locked = !board.locked) }) {
                Text(
                    if (board.locked) {
                        L.text(R.string.mobile_boards_moderation_unlock)
                    } else {
                        L.text(R.string.mobile_boards_moderation_lock)
                    },
                )
            }
            TextButton(enabled = !saving, onClick = { patch(freezeMinutes = 5) }) {
                Text(L.text(R.string.mobile_boards_moderation_freeze5))
            }
            if (frozen) {
                TextButton(enabled = !saving, onClick = { patch(frozenUntil = "") }) {
                    Text(L.text(R.string.mobile_boards_moderation_unfreeze))
                }
            }
            if (board.locked) {
                Text(L.text(R.string.mobile_boards_moderation_lockedBanner), color = textSecondary())
            } else if (frozen) {
                Text(L.text(R.string.mobile_boards_moderation_frozenBanner), color = textSecondary())
            }
            TextButton(onClick = onDismiss, modifier = Modifier.fillMaxWidth()) {
                Text(L.text(R.string.mobile_common_close))
            }
        }
    }
}
