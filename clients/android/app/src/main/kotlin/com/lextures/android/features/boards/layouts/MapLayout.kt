package com.lextures.android.features.boards.layouts

import androidx.compose.foundation.Canvas
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableFloatStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.unit.dp
import com.lextures.android.R
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.lms.ArrangeBoardPostBody
import com.lextures.android.core.lms.Board
import com.lextures.android.core.lms.BoardPost
import com.lextures.android.core.lms.BoardSection
import com.lextures.android.core.lms.BoardsLogic
import com.lextures.android.features.boards.BoardPostCardSlot

/**
 * CSP-safe custom map with clustering + list fallback (parity with web; no Maps API key).
 */
@Composable
fun MapLayout(
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
    val pinned = remember(posts) { BoardsLogic.pinnedPosts(posts) }
    val unpinned = remember(posts) { BoardsLogic.unpinnedPosts(posts) }
    var zoom by remember { mutableFloatStateOf(1f) }
    var selectedId by remember { mutableStateOf<String?>(null) }
    var showList by remember { mutableStateOf(false) }
    val clusters = remember(pinned, zoom) { BoardsLogic.clusterPins(pinned, zoom.toDouble()) }
    val selected = selectedId?.let { id -> posts.firstOrNull { it.id == id } }
    val mapLabel = L.text(R.string.mobile_boards_layout_map)
    val mapPinLabel = L.text(R.string.mobile_boards_layout_mapPin)
    val mapEmptyLabel = L.text(R.string.mobile_boards_layout_mapEmpty)
    val showMapLabel = L.text(R.string.mobile_boards_layout_mapShowMap)
    val listFallbackLabel = L.text(R.string.mobile_boards_layout_mapListFallback)
    val unpinnedTrayLabel = L.text(R.string.mobile_boards_layout_unpinnedTray)
    val zoomLabel = L.format(R.string.mobile_boards_layout_mapZoom, zoom.toInt())

    Column(modifier = modifier, verticalArrangement = Arrangement.spacedBy(12.dp)) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Text(mapLabel, color = textPrimary())
            TextButton(onClick = { zoom = (zoom + 1f).coerceAtMost(8f) }) { Text("+") }
            TextButton(onClick = { zoom = (zoom - 1f).coerceAtLeast(1f) }) { Text("−") }
            Text(zoomLabel, color = textSecondary())
            TextButton(onClick = { showList = !showList }) {
                Text(if (showList) showMapLabel else listFallbackLabel)
            }
        }

        when {
            showList -> {
                Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    if (pinned.isEmpty()) {
                        Text(mapEmptyLabel, color = textSecondary())
                    } else {
                        pinned.forEach { post ->
                            val pinTitle = post.title.ifBlank { mapPinLabel }
                            Row(
                                modifier = Modifier
                                    .fillMaxWidth()
                                    .clickable { selectedId = post.id }
                                    .background(textSecondary().copy(alpha = 0.08f), RoundedCornerShape(10.dp))
                                    .padding(10.dp)
                                    .semantics { contentDescription = pinTitle },
                            ) {
                                Column {
                                    Text(pinTitle, color = textPrimary())
                                    Text(
                                        String.format("%.4f, %.4f", post.lat, post.lng),
                                        color = textSecondary(),
                                    )
                                }
                            }
                        }
                    }
                }
            }
            pinned.isEmpty() -> {
                Box(
                    modifier = Modifier
                        .fillMaxWidth()
                        .height(280.dp)
                        .background(Color(0xFFDCEFE8), RoundedCornerShape(12.dp))
                        .semantics { contentDescription = mapLabel },
                    contentAlignment = Alignment.Center,
                ) {
                    Text(mapEmptyLabel, color = textSecondary())
                }
            }
            else -> {
                Canvas(
                    modifier = Modifier
                        .fillMaxWidth()
                        .height(280.dp)
                        .background(Color(0xFFDCEFE8), RoundedCornerShape(12.dp))
                        .semantics { contentDescription = mapLabel },
                ) {
                    val w = size.width
                    val h = size.height
                    for (i in 0..6) {
                        val y = h / 6f * i
                        drawLine(Color.Gray.copy(alpha = 0.25f), Offset(0f, y), Offset(w, y))
                    }
                    for (i in 0..12) {
                        val x = w / 12f * i
                        drawLine(Color.Gray.copy(alpha = 0.25f), Offset(x, 0f), Offset(x, h))
                    }
                    clusters.forEach { cluster ->
                        val x = (((cluster.lng + 180) / 360.0) * w).toFloat()
                        val y = (((90 - cluster.lat) / 180.0) * h).toFloat()
                        val radius = if (cluster.postIds.size > 1) 18f else 10f
                        drawCircle(Color(0xFF0F766E), radius = radius, center = Offset(x, y))
                    }
                }
            }
        }

        selected?.let { post ->
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
                showMap = true,
            )
        }

        if (unpinned.isNotEmpty()) {
            Column(
                modifier = Modifier
                    .fillMaxWidth()
                    .border(1.dp, textSecondary().copy(alpha = 0.4f), RoundedCornerShape(12.dp))
                    .padding(12.dp),
                verticalArrangement = Arrangement.spacedBy(8.dp),
            ) {
                Text(unpinnedTrayLabel, color = textPrimary())
                unpinned.forEach { post ->
                    BoardPostCardSlot(
                        post = post,
                        siblings = unpinned,
                        sections = sections,
                        board = board,
                        canManage = canManage,
                        currentUserId = currentUserId,
                        onEdit = onEdit,
                        onDelete = onDelete,
                        onArrange = onArrange,
                        showMap = true,
                    )
                }
            }
        }

        if (posts.isEmpty()) {
            BoardPostsEmpty()
        }
    }
}
