package com.lextures.android.features.boards

import android.content.Intent
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Checkbox
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
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
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.text.input.VisualTransformation
import androidx.compose.ui.unit.dp
import com.lextures.android.R
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.lms.Board
import com.lextures.android.core.lms.BoardAccessApi
import com.lextures.android.core.lms.BoardAttribution
import com.lextures.android.core.lms.BoardMember
import com.lextures.android.core.lms.BoardMemberRole
import com.lextures.android.core.lms.BoardShare
import com.lextures.android.core.lms.BoardShareCapability
import com.lextures.android.core.lms.BoardVisibility
import com.lextures.android.core.lms.BoardsLogic
import com.lextures.android.core.network.ApiError
import kotlinx.coroutines.launch
import java.time.Instant

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun BoardShareSheet(
    courseCode: String,
    board: Board,
    accessToken: String?,
    onBoardUpdated: (Board) -> Unit,
    onDismiss: () -> Unit,
) {
    val sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)
    val scope = rememberCoroutineScope()
    val context = LocalContext.current

    var visibility by remember { mutableStateOf(BoardVisibility.fromApi(board.visibility)) }
    var visibilityTarget by remember { mutableStateOf(board.visibilityTarget.orEmpty()) }
    var attribution by remember { mutableStateOf(BoardAttribution.fromApi(board.attribution)) }
    var canPost by remember { mutableStateOf(board.canPost ?: true) }
    var canInteract by remember { mutableStateOf(board.canInteract ?: true) }
    var canArrange by remember { mutableStateOf(board.canArrange ?: true) }
    var members by remember { mutableStateOf<List<BoardMember>>(emptyList()) }
    var shares by remember { mutableStateOf<List<BoardShare>>(emptyList()) }
    var memberUserId by remember { mutableStateOf("") }
    var shareCap by remember { mutableStateOf(BoardShareCapability.View) }
    var sharePassword by remember { mutableStateOf("") }
    var showPassword by remember { mutableStateOf(false) }
    var saving by remember { mutableStateOf(false) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var externalBlockedReason by remember { mutableStateOf<String?>(null) }
    var visibilityMenuOpen by remember { mutableStateOf(false) }
    var attributionMenuOpen by remember { mutableStateOf(false) }
    var capMenuOpen by remember { mutableStateOf(false) }

    val externalAllowed = BoardsLogic.externalSharingAllowed(board)
    val visibilityOptions = BoardsLogic.visibilityOptions(board)
    val saveError = L.text(R.string.mobile_boards_share_saveError)
    val createLinkLabel = L.text(R.string.mobile_boards_share_createLink)

    LaunchedEffect(board.id, accessToken) {
        val token = accessToken ?: return@LaunchedEffect
        members = runCatching { BoardAccessApi.listMembers(courseCode, board.id, token) }.getOrDefault(emptyList())
        if (externalAllowed) {
            try {
                shares = BoardAccessApi.listShares(courseCode, board.id, token)
            } catch (e: ApiError.HttpStatus) {
                if (e.code == 403) externalBlockedReason = "disabled"
                shares = emptyList()
            } catch (_: Exception) {
                shares = emptyList()
            }
        }
    }

    ModalBottomSheet(onDismissRequest = onDismiss, sheetState = sheetState) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .verticalScroll(rememberScrollState())
                .padding(horizontal = 16.dp, vertical = 12.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            Text(
                L.text(R.string.mobile_boards_share_title),
                fontWeight = FontWeight.SemiBold,
                color = textPrimary(),
            )
            Text(L.text(R.string.mobile_boards_share_subtitle), color = textSecondary())

            errorMessage?.let { Text(it, color = androidx.compose.ui.graphics.Color.Red) }

            Text(
                L.text(R.string.mobile_boards_access_visibility),
                fontWeight = FontWeight.Medium,
                color = textPrimary(),
            )
            Box {
                TextButton(onClick = { visibilityMenuOpen = true }) {
                    Text(visibilityLabel(visibility))
                }
                DropdownMenu(expanded = visibilityMenuOpen, onDismissRequest = { visibilityMenuOpen = false }) {
                    visibilityOptions.forEach { option ->
                        DropdownMenuItem(
                            text = { Text(visibilityLabel(option)) },
                            onClick = {
                                visibility = option
                                visibilityMenuOpen = false
                            },
                        )
                    }
                }
            }
            if (visibility == BoardVisibility.Section || visibility == BoardVisibility.Group) {
                OutlinedTextField(
                    value = visibilityTarget,
                    onValueChange = { visibilityTarget = it },
                    modifier = Modifier.fillMaxWidth(),
                    label = { Text(L.text(R.string.mobile_boards_access_visibilityTarget)) },
                    placeholder = { Text(L.text(R.string.mobile_boards_access_visibilityTargetPlaceholder)) },
                )
            }
            if (board.externalSharingAllowed != true) {
                Text(L.text(R.string.mobile_boards_share_externalDisabled), color = textSecondary())
            }
            if (board.minorModerationFloor == true || externalBlockedReason == "minors") {
                Text(L.text(R.string.mobile_boards_share_minorsBlocked), color = textSecondary())
            }

            Text(
                L.text(R.string.mobile_boards_access_attribution),
                fontWeight = FontWeight.Medium,
                color = textPrimary(),
            )
            Box {
                TextButton(onClick = { attributionMenuOpen = true }) {
                    Text(attributionLabel(attribution))
                }
                DropdownMenu(expanded = attributionMenuOpen, onDismissRequest = { attributionMenuOpen = false }) {
                    BoardAttribution.entries.forEach { option ->
                        DropdownMenuItem(
                            text = { Text(attributionLabel(option)) },
                            onClick = {
                                attribution = option
                                attributionMenuOpen = false
                            },
                        )
                    }
                }
            }

            Text(
                L.text(R.string.mobile_boards_access_contributorPolicy),
                fontWeight = FontWeight.Medium,
                color = textPrimary(),
            )
            Row(verticalAlignment = Alignment.CenterVertically) {
                Checkbox(checked = canPost, onCheckedChange = { canPost = it })
                Text(L.text(R.string.mobile_boards_access_canPost))
            }
            Row(verticalAlignment = Alignment.CenterVertically) {
                Checkbox(checked = canInteract, onCheckedChange = { canInteract = it })
                Text(L.text(R.string.mobile_boards_access_canInteract))
            }
            Row(verticalAlignment = Alignment.CenterVertically) {
                Checkbox(checked = canArrange, onCheckedChange = { canArrange = it })
                Text(L.text(R.string.mobile_boards_access_canArrange))
            }

            TextButton(
                onClick = {
                    val token = accessToken ?: return@TextButton
                    scope.launch {
                        saving = true
                        errorMessage = null
                        try {
                            val updated = BoardAccessApi.patchBoardAccess(
                                courseCode = courseCode,
                                boardId = board.id,
                                visibility = visibility.apiValue,
                                visibilityTarget = if (
                                    visibility == BoardVisibility.Section || visibility == BoardVisibility.Group
                                ) {
                                    visibilityTarget.ifBlank { null }
                                } else {
                                    ""
                                },
                                attribution = attribution.apiValue,
                                canPost = canPost,
                                canInteract = canInteract,
                                canArrange = canArrange,
                                accessToken = token,
                            )
                            onBoardUpdated(updated)
                        } catch (e: Exception) {
                            errorMessage = e.message ?: saveError
                        } finally {
                            saving = false
                        }
                    }
                },
                enabled = !saving,
            ) {
                Text(L.text(R.string.mobile_boards_share_saveAccess))
            }

            if (visibility == BoardVisibility.Invite) {
                Text(
                    L.text(R.string.mobile_boards_share_members),
                    fontWeight = FontWeight.Medium,
                    color = textPrimary(),
                )
                Row(
                    horizontalArrangement = Arrangement.spacedBy(8.dp),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    OutlinedTextField(
                        value = memberUserId,
                        onValueChange = { memberUserId = it },
                        modifier = Modifier.weight(1f),
                        label = { Text(L.text(R.string.mobile_boards_share_memberUserId)) },
                        singleLine = true,
                    )
                    TextButton(
                        onClick = {
                            val token = accessToken ?: return@TextButton
                            val uid = memberUserId.trim()
                            if (uid.isEmpty()) return@TextButton
                            scope.launch {
                                try {
                                    val m = BoardAccessApi.upsertMember(
                                        courseCode,
                                        board.id,
                                        uid,
                                        BoardMemberRole.Contributor.apiValue,
                                        token,
                                    )
                                    members = members.filterNot { it.userId == m.userId } + m
                                    memberUserId = ""
                                } catch (e: Exception) {
                                    errorMessage = e.message ?: saveError
                                }
                            }
                        },
                    ) {
                        Text(L.text(R.string.mobile_boards_share_addMember))
                    }
                }
                members.forEach { member ->
                    val memberRole = roleLabel(member.role)
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.SpaceBetween,
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Text("${member.userId.take(8)}… · $memberRole")
                        TextButton(
                            onClick = {
                                val token = accessToken ?: return@TextButton
                                scope.launch {
                                    try {
                                        BoardAccessApi.removeMember(courseCode, board.id, member.userId, token)
                                        members = members.filterNot { it.userId == member.userId }
                                    } catch (e: Exception) {
                                        errorMessage = e.message ?: saveError
                                    }
                                }
                            },
                        ) {
                            Text(L.text(R.string.mobile_boards_share_removeMember))
                        }
                    }
                }
            }

            if (externalAllowed) {
                Text(
                    L.text(R.string.mobile_boards_share_links),
                    fontWeight = FontWeight.Medium,
                    color = textPrimary(),
                )
                Box {
                    TextButton(onClick = { capMenuOpen = true }) {
                        Text(capabilityLabel(shareCap))
                    }
                    DropdownMenu(expanded = capMenuOpen, onDismissRequest = { capMenuOpen = false }) {
                        BoardShareCapability.entries.forEach { option ->
                            DropdownMenuItem(
                                text = { Text(capabilityLabel(option)) },
                                onClick = {
                                    shareCap = option
                                    capMenuOpen = false
                                },
                            )
                        }
                    }
                }
                OutlinedTextField(
                    value = sharePassword,
                    onValueChange = { sharePassword = it },
                    modifier = Modifier.fillMaxWidth(),
                    label = { Text(L.text(R.string.mobile_boards_share_passwordOptional)) },
                    visualTransformation = if (showPassword) {
                        VisualTransformation.None
                    } else {
                        PasswordVisualTransformation()
                    },
                    keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Password),
                    trailingIcon = {
                        TextButton(onClick = { showPassword = !showPassword }) {
                            Text(
                                if (showPassword) L.text(R.string.mobile_boards_share_hidePassword)
                                else L.text(R.string.mobile_boards_share_showPassword),
                            )
                        }
                    },
                    singleLine = true,
                )
                TextButton(
                    onClick = {
                        val token = accessToken ?: return@TextButton
                        scope.launch {
                            try {
                                val share = BoardAccessApi.createShare(
                                    courseCode,
                                    board.id,
                                    shareCap.apiValue,
                                    sharePassword,
                                    accessToken = token,
                                )
                                shares = listOf(share) + shares
                                sharePassword = ""
                                BoardsLogic.shareUrl(share)?.let { url ->
                                    context.startActivity(
                                        Intent.createChooser(
                                            Intent(Intent.ACTION_SEND).apply {
                                                type = "text/plain"
                                                putExtra(Intent.EXTRA_TEXT, url)
                                            },
                                            createLinkLabel,
                                        ),
                                    )
                                }
                            } catch (e: ApiError.HttpStatus) {
                                val msg = e.message.orEmpty()
                                externalBlockedReason = when {
                                    msg.contains("minors", ignoreCase = true) -> "minors"
                                    e.code == 403 -> "disabled"
                                    else -> externalBlockedReason
                                }
                                errorMessage = msg.ifBlank { saveError }
                            } catch (e: Exception) {
                                errorMessage = e.message ?: saveError
                            }
                        }
                    },
                ) {
                    Text(createLinkLabel)
                }
                shares.forEach { share ->
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.SpaceBetween,
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Column(modifier = Modifier.weight(1f)) {
                            Text(capabilityLabel(BoardShareCapability.fromApi(share.capability)))
                            if (share.hasPassword) {
                                Text(
                                    L.text(R.string.mobile_boards_share_passwordProtected),
                                    color = textSecondary(),
                                )
                            }
                            if (share.revokedAt != null) {
                                Text(L.text(R.string.mobile_boards_share_revoked), color = textSecondary())
                            }
                        }
                        if (share.revokedAt == null) {
                            TextButton(
                                onClick = {
                                    val token = accessToken ?: return@TextButton
                                    scope.launch {
                                        try {
                                            BoardAccessApi.revokeShare(courseCode, board.id, share.id, token)
                                            shares = shares.map {
                                                if (it.id == share.id) {
                                                    it.copy(revokedAt = Instant.now().toString())
                                                } else {
                                                    it
                                                }
                                            }
                                        } catch (e: Exception) {
                                            errorMessage = e.message ?: saveError
                                        }
                                    }
                                },
                            ) {
                                Text(L.text(R.string.mobile_boards_share_revoke))
                            }
                        }
                    }
                }
            }

            TextButton(onClick = onDismiss, modifier = Modifier.align(Alignment.End)) {
                Text(L.text(R.string.mobile_common_close))
            }
        }
    }
}

@Composable
private fun visibilityLabel(v: BoardVisibility): String = when (v) {
    BoardVisibility.Course -> L.text(R.string.mobile_boards_access_visibility_course)
    BoardVisibility.Section -> L.text(R.string.mobile_boards_access_visibility_section)
    BoardVisibility.Group -> L.text(R.string.mobile_boards_access_visibility_group)
    BoardVisibility.Invite -> L.text(R.string.mobile_boards_access_visibility_invite)
    BoardVisibility.Link -> L.text(R.string.mobile_boards_access_visibility_link)
    BoardVisibility.Public -> L.text(R.string.mobile_boards_access_visibility_public)
}

@Composable
private fun attributionLabel(a: BoardAttribution): String = when (a) {
    BoardAttribution.Named -> L.text(R.string.mobile_boards_access_attribution_named)
    BoardAttribution.AnonToPeers -> L.text(R.string.mobile_boards_access_attribution_anon_to_peers)
    BoardAttribution.Anonymous -> L.text(R.string.mobile_boards_access_attribution_anonymous)
}

@Composable
private fun capabilityLabel(c: BoardShareCapability): String = when (c) {
    BoardShareCapability.View -> L.text(R.string.mobile_boards_share_capability_view)
    BoardShareCapability.Contribute -> L.text(R.string.mobile_boards_share_capability_contribute)
}

@Composable
private fun roleLabel(role: String): String = when (role.lowercase()) {
    "owner" -> L.text(R.string.mobile_boards_share_role_owner)
    "editor" -> L.text(R.string.mobile_boards_share_role_editor)
    "viewer" -> L.text(R.string.mobile_boards_share_role_viewer)
    else -> L.text(R.string.mobile_boards_share_role_contributor)
}
