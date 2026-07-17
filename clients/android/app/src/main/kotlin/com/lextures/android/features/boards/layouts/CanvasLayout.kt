package com.lextures.android.features.boards.layouts

import androidx.compose.foundation.background
import androidx.compose.foundation.gestures.detectTransformGestures
import androidx.compose.foundation.gestures.detectDragGesturesAfterLongPress
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableFloatStateOf
import androidx.compose.runtime.mutableStateMapOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.unit.IntOffset
import androidx.compose.ui.unit.dp
import com.lextures.android.R
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.lms.ArrangeBoardPostBody
import com.lextures.android.core.lms.Board
import com.lextures.android.core.lms.BoardPost
import com.lextures.android.core.lms.BoardPostPosition
import com.lextures.android.core.lms.BoardSection
import com.lextures.android.core.lms.BoardsLogic
import com.lextures.android.features.boards.BoardPostCardSlot

@Composable
fun CanvasLayout(
    posts: List<BoardPost>,
    sections: List<BoardSection>,
    board: Board,
    canManage: Boolean,
    currentUserId: String?,
    onEdit: (BoardPost) -> Unit,
    onDelete: (BoardPost) -> Unit,
    onArrange: (BoardPost, ArrangeBoardPostBody) -> Unit,
    modifier: Modifier = Modifier,
) {
    if (posts.isEmpty()) {
        BoardPostsEmpty(modifier)
        return
    }

    var scale by remember { mutableFloatStateOf(1f) }
    var pan by remember { mutableStateOf(Offset.Zero) }
    val dragOffsets = remember { mutableStateMapOf<String, Offset>() }
    val density = LocalDensity.current
    val defaultW = 220f
    val defaultH = 160f

    fun positionFor(post: BoardPost, index: Int): BoardPostPosition {
        val p = post.position
        if (p != null) return p
        val col = index % 3
        val row = index / 3
        return BoardPostPosition(
            x = 40.0 + col * (defaultW + 24),
            y = 40.0 + row * (defaultH + 24),
            w = defaultW.toDouble(),
            h = defaultH.toDouble(),
        )
    }

    Column(modifier = modifier) {
        Text(
            L.text(R.string.mobile_boards_layout_canvasHint),
            color = textSecondary(),
            modifier = Modifier.padding(bottom = 8.dp),
        )
        Box(
            modifier = Modifier
                .fillMaxWidth()
                .height(420.dp)
                .clip(RoundedCornerShape(12.dp))
                .background(textSecondary().copy(alpha = 0.08f))
                .pointerInput(Unit) {
                    detectTransformGestures { _, panChange, zoomChange, _ ->
                        scale = (scale * zoomChange).coerceIn(0.5f, 2.5f)
                        pan += panChange
                    }
                },
        ) {
            // Virtualize roughly off-screen cards for larger boards (viewport ~420dp tall).
            val visible = if (posts.size > 40) {
                posts.mapIndexedNotNull { index, post ->
                    val pos = positionFor(post, index)
                    val x = ((pos.x ?: 0.0).toFloat() * scale) + pan.x
                    val y = ((pos.y ?: 0.0).toFloat() * scale) + pan.y
                    if (x > -400f && y > -400f && x < 1200f && y < 1000f) {
                        post to index
                    } else {
                        null
                    }
                }
            } else {
                posts.mapIndexed { index, post -> post to index }
            }

            visible.forEach { (post, index) ->
                val pos = positionFor(post, index)
                val live = dragOffsets[post.id] ?: Offset.Zero
                val canArrange = BoardsLogic.canArrangePost(post, board, currentUserId, canManage)
                val xDp = with(density) {
                    ((((pos.x ?: 40.0).toFloat() + live.x) * scale) + pan.x).toDp()
                }
                val yDp = with(density) {
                    ((((pos.y ?: 40.0).toFloat() + live.y) * scale) + pan.y).toDp()
                }
                val wDp = with(density) { ((pos.w ?: defaultW.toDouble()).toFloat() * scale).toDp() }
                Box(
                    modifier = Modifier
                        .offset { IntOffset(xDp.roundToPx(), yDp.roundToPx()) }
                        .width(wDp)
                        .graphicsLayer { this.scaleX = 1f; this.scaleY = 1f }
                        .then(
                            if (canArrange) {
                                Modifier.pointerInput(post.id, scale) {
                                    detectDragGesturesAfterLongPress(
                                        onDrag = { change, dragAmount ->
                                            change.consume()
                                            val cur = dragOffsets[post.id] ?: Offset.Zero
                                            dragOffsets[post.id] = cur + (dragAmount / scale)
                                        },
                                        onDragEnd = {
                                            val delta = dragOffsets.remove(post.id) ?: Offset.Zero
                                            onArrange(
                                                post,
                                                ArrangeBoardPostBody(
                                                    position = BoardPostPosition(
                                                        x = (pos.x ?: 40.0) + delta.x,
                                                        y = (pos.y ?: 40.0) + delta.y,
                                                        w = pos.w ?: defaultW.toDouble(),
                                                        h = pos.h ?: defaultH.toDouble(),
                                                    ),
                                                ),
                                            )
                                        },
                                        onDragCancel = { dragOffsets.remove(post.id) },
                                    )
                                }
                            } else {
                                Modifier
                            },
                        ),
                ) {
                    BoardPostCardSlot(
                        post = post,
                        siblings = posts,
                        sections = sections,
                        board = board,
                        canManage = canManage,
                        currentUserId = currentUserId,
                        onEdit = onEdit,
                        onDelete = onDelete,
                        onArrange = onArrange,
                    )
                }
            }
        }
    }
}
