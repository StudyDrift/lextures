package com.lextures.android.features.boards

import androidx.compose.runtime.compositionLocalOf
import com.lextures.android.core.lms.BoardPost

/** Shared handlers for card reactions/comments/moderation (VC.M5 / VC.M7). */
data class BoardEngagementHandlers(
    val courseCode: String,
    val accessToken: String?,
    val onPostUpdate: (BoardPost) -> Unit,
    val onAnnounce: (String) -> Unit = {},
    val onHidePost: ((BoardPost) -> Unit)? = null,
    val onRemovePost: ((BoardPost) -> Unit)? = null,
)

val LocalBoardEngagement = compositionLocalOf<BoardEngagementHandlers?> { null }
