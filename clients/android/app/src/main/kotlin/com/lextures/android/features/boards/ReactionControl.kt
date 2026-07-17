package com.lextures.android.features.boards

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Favorite
import androidx.compose.material.icons.filled.FavoriteBorder
import androidx.compose.material.icons.filled.School
import androidx.compose.material.icons.filled.Star
import androidx.compose.material.icons.filled.ThumbUp
import androidx.compose.material.icons.outlined.StarBorder
import androidx.compose.material.icons.outlined.ThumbUp
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.role
import androidx.compose.ui.semantics.selected
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.unit.dp
import com.lextures.android.R
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.lms.BoardEngagementApi
import com.lextures.android.core.lms.BoardPost
import com.lextures.android.core.lms.BoardReactionMode
import com.lextures.android.core.lms.BoardsLogic
import com.lextures.android.core.network.ApiError
import kotlinx.coroutines.launch

@Composable
fun ReactionControl(
    courseCode: String,
    boardId: String,
    post: BoardPost,
    reactionMode: BoardReactionMode,
    accessToken: String?,
    canInteract: Boolean,
    canGrade: Boolean,
    onPostUpdate: (BoardPost) -> Unit,
    onAnnounce: (String) -> Unit = {},
    onOpenGradeSheet: () -> Unit = {},
    modifier: Modifier = Modifier,
) {
    if (reactionMode == BoardReactionMode.None) return

    val scope = rememberCoroutineScope()
    var busy by remember { mutableStateOf(false) }
    val pressed = post.myReaction != null
    val count = post.reactionCount ?: 0
    val forbiddenMsg = L.text(R.string.mobile_boards_react_forbidden)
    val errorMsg = L.text(R.string.mobile_boards_react_error)
    val likeOn = L.text(R.string.mobile_boards_react_likeOn)
    val likeOff = L.text(R.string.mobile_boards_react_likeOff)
    val voteOn = L.text(R.string.mobile_boards_react_voteOn)
    val voteOff = L.text(R.string.mobile_boards_react_voteOff)
    val starSetTemplate = L.text(R.string.mobile_boards_react_starSet)
    val avgTemplate = L.text(R.string.mobile_boards_react_avgStars)
    val yourGradeTemplate = L.text(R.string.mobile_boards_react_yourGrade)
    val starLabel = L.text(R.string.mobile_boards_react_starLabel)
    val starNTemplate = L.text(R.string.mobile_boards_react_starN)

    when (reactionMode) {
        BoardReactionMode.Like -> ToggleReactionButton(
            pressed = pressed,
            count = count,
            showCountWhenZero = false,
            filledIcon = Icons.Filled.Favorite,
            outlinedIcon = Icons.Filled.FavoriteBorder,
            pressedTint = Color(0xFFE11D48),
            labelOn = L.text(R.string.mobile_boards_react_unlike),
            labelOff = L.text(R.string.mobile_boards_react_like),
            enabled = canInteract && !busy,
            onClick = {
                scope.launch {
                    runToggle(
                        courseCode, boardId, post, "like", accessToken, canInteract,
                        { busy }, { busy = it }, onPostUpdate, onAnnounce,
                        likeOn, likeOff, forbiddenMsg, errorMsg,
                    )
                }
            },
            modifier = modifier,
        )
        BoardReactionMode.Vote -> ToggleReactionButton(
            pressed = pressed,
            count = count,
            showCountWhenZero = true,
            filledIcon = Icons.Filled.ThumbUp,
            outlinedIcon = Icons.Outlined.ThumbUp,
            pressedTint = Color(0xFF4F46E5),
            labelOn = L.text(R.string.mobile_boards_react_unvote),
            labelOff = L.text(R.string.mobile_boards_react_vote),
            enabled = canInteract && !busy,
            onClick = {
                scope.launch {
                    runToggle(
                        courseCode, boardId, post, "vote", accessToken, canInteract,
                        { busy }, { busy = it }, onPostUpdate, onAnnounce,
                        voteOn, voteOff, forbiddenMsg, errorMsg,
                    )
                }
            },
            modifier = modifier,
        )
        BoardReactionMode.Star -> {
            val mine = post.myReaction?.value?.toInt() ?: 0
            Row(
                modifier = modifier.semantics { contentDescription = starLabel },
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(0.dp),
            ) {
                for (n in 1..5) {
                    val label = starNTemplate
                        .replace("%1\$d", n.toString())
                        .replace("%d", n.toString())
                    IconButton(
                        onClick = {
                            scope.launch {
                                runSetStars(
                                    courseCode, boardId, post, n, accessToken, canInteract,
                                    { busy }, { busy = it }, onPostUpdate, onAnnounce,
                                    starSetTemplate, forbiddenMsg, errorMsg,
                                )
                            }
                        },
                        enabled = canInteract && !busy,
                        modifier = Modifier
                            .size(40.dp)
                            .semantics {
                                contentDescription = label
                                selected = mine == n
                                role = Role.Button
                            },
                    ) {
                        Icon(
                            imageVector = if (mine >= n) Icons.Filled.Star else Icons.Outlined.StarBorder,
                            contentDescription = null,
                            tint = Color(0xFFF59E0B),
                        )
                    }
                }
                post.avgStars?.let { avg ->
                    Text(
                        avgTemplate
                            .replace("%1\$s", BoardsLogic.formatAvgStars(avg))
                            .replace("%2\$d", count.toString())
                            .replace("%@", BoardsLogic.formatAvgStars(avg))
                            .replace("%d", count.toString()),
                        color = textSecondary(),
                        modifier = Modifier.padding(start = 4.dp),
                    )
                }
            }
        }
        BoardReactionMode.Grade -> {
            if (canGrade) {
                TextButton(onClick = onOpenGradeSheet, modifier = modifier.heightIn(min = 36.dp)) {
                    Icon(Icons.Filled.School, contentDescription = null, modifier = Modifier.size(16.dp))
                    Text(
                        BoardsLogic.visibleGrade(post)?.let { BoardsLogic.formatGrade(it) }
                            ?: L.text(R.string.mobile_boards_react_grade),
                        color = textPrimary(),
                        modifier = Modifier.padding(start = 4.dp),
                    )
                }
            } else {
                BoardsLogic.visibleGrade(post)?.let { grade ->
                    Text(
                        yourGradeTemplate
                            .replace("%1\$s", BoardsLogic.formatGrade(grade))
                            .replace("%@", BoardsLogic.formatGrade(grade)),
                        color = textPrimary(),
                        modifier = modifier.heightIn(min = 36.dp).padding(horizontal = 8.dp),
                    )
                }
            }
        }
        BoardReactionMode.None -> Unit
    }
}

@Composable
private fun ToggleReactionButton(
    pressed: Boolean,
    count: Int,
    showCountWhenZero: Boolean,
    filledIcon: ImageVector,
    outlinedIcon: ImageVector,
    pressedTint: Color,
    labelOn: String,
    labelOff: String,
    enabled: Boolean,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
) {
    TextButton(
        onClick = onClick,
        enabled = enabled,
        modifier = modifier
            .heightIn(min = 36.dp)
            .semantics {
                contentDescription = if (pressed) labelOn else labelOff
                selected = pressed
                role = Role.Button
            },
    ) {
        Icon(
            imageVector = if (pressed) filledIcon else outlinedIcon,
            contentDescription = null,
            tint = if (pressed) pressedTint else textSecondary(),
            modifier = Modifier.size(18.dp),
        )
        if (count > 0 || showCountWhenZero) {
            Text("$count", color = textSecondary(), modifier = Modifier.padding(start = 4.dp))
        }
    }
}

private suspend fun runToggle(
    courseCode: String,
    boardId: String,
    post: BoardPost,
    kind: String,
    accessToken: String?,
    canInteract: Boolean,
    isBusy: () -> Boolean,
    setBusy: (Boolean) -> Unit,
    onPostUpdate: (BoardPost) -> Unit,
    onAnnounce: (String) -> Unit,
    announceOn: String,
    announceOff: String,
    forbiddenMsg: String,
    errorMsg: String,
) {
    val token = accessToken ?: return
    if (!canInteract || isBusy()) return
    setBusy(true)
    val previous = post
    onPostUpdate(BoardsLogic.optimisticToggleReaction(post, kind))
    try {
        val result = BoardEngagementApi.putReaction(
            courseCode, boardId, post.id, kind = kind, accessToken = token,
        )
        onPostUpdate(BoardsLogic.applyReactionResult(previous, result))
        onAnnounce(if (result.active) announceOn else announceOff)
    } catch (e: ApiError.HttpStatus) {
        onPostUpdate(previous)
        onAnnounce(if (e.code == 403) forbiddenMsg else errorMsg)
    } catch (_: Exception) {
        onPostUpdate(previous)
        onAnnounce(errorMsg)
    } finally {
        setBusy(false)
    }
}

private suspend fun runSetStars(
    courseCode: String,
    boardId: String,
    post: BoardPost,
    value: Int,
    accessToken: String?,
    canInteract: Boolean,
    isBusy: () -> Boolean,
    setBusy: (Boolean) -> Unit,
    onPostUpdate: (BoardPost) -> Unit,
    onAnnounce: (String) -> Unit,
    starSetTemplate: String,
    forbiddenMsg: String,
    errorMsg: String,
) {
    val token = accessToken ?: return
    if (!canInteract || isBusy()) return
    setBusy(true)
    val previous = post
    onPostUpdate(BoardsLogic.optimisticSetStars(post, value))
    try {
        val result = BoardEngagementApi.putReaction(
            courseCode, boardId, post.id, kind = "star", value = value.toDouble(), accessToken = token,
        )
        onPostUpdate(BoardsLogic.applyReactionResult(previous, result))
        onAnnounce(
            starSetTemplate
                .replace("%1\$d", value.toString())
                .replace("%d", value.toString()),
        )
    } catch (e: ApiError.HttpStatus) {
        onPostUpdate(previous)
        onAnnounce(if (e.code == 403) forbiddenMsg else errorMsg)
    } catch (_: Exception) {
        onPostUpdate(previous)
        onAnnounce(errorMsg)
    } finally {
        setBusy(false)
    }
}
