package com.lextures.android.features.boards

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.unit.dp
import com.lextures.android.R
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.lms.BoardSyncState

/** Subtle Live / Reconnecting / Offline chip for the board header (VC.M4 FR-8). */
@Composable
fun BoardSyncStatusChip(
    state: BoardSyncState,
    modifier: Modifier = Modifier,
) {
    val label = when (state) {
        BoardSyncState.Connecting -> L.text(R.string.mobile_boards_sync_connecting)
        BoardSyncState.Live -> L.text(R.string.mobile_boards_sync_live)
        BoardSyncState.Reconnecting -> L.text(R.string.mobile_boards_sync_reconnecting)
        BoardSyncState.Offline -> L.text(R.string.mobile_boards_sync_offline)
    }
    val color = when (state) {
        BoardSyncState.Live -> Color(0xFF059669)
        BoardSyncState.Reconnecting -> Color(0xFFD97706)
        BoardSyncState.Connecting, BoardSyncState.Offline -> textSecondary()
    }
    Row(
        modifier = modifier.semantics { contentDescription = label },
        horizontalArrangement = Arrangement.spacedBy(6.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        if (state == BoardSyncState.Live) {
            Box(
                modifier = Modifier
                    .size(6.dp)
                    .clip(CircleShape)
                    .background(color),
            )
        }
        Text(
            text = label,
            style = MaterialTheme.typography.labelSmall,
            color = color,
        )
    }
}
