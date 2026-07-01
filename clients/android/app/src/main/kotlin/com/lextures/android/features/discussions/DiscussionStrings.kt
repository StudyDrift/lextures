package com.lextures.android.features.discussions

import androidx.compose.runtime.Composable
import com.lextures.android.R
import com.lextures.android.core.i18n.L

@Composable fun discussionsNewThread(): String = L.text(R.string.mobile_discussions_newThread)
@Composable fun discussionsEmptyTitle(): String = L.text(R.string.mobile_discussions_emptyTitle)
@Composable fun discussionsEmptyMessage(): String = L.text(R.string.mobile_discussions_emptyMessage)
@Composable fun discussionsPinned(): String = L.text(R.string.mobile_discussions_pinned)
@Composable fun discussionsLocked(): String = L.text(R.string.mobile_discussions_locked)
@Composable fun discussionsReplyCount(count: Int): String = L.format(R.string.mobile_discussions_replyCount, count)
@Composable fun discussionsBack(): String = L.text(R.string.mobile_discussions_back)
@Composable fun discussionsThread(): String = L.text(R.string.mobile_discussions_thread)
@Composable fun discussionsReply(): String = L.text(R.string.mobile_discussions_reply)
@Composable fun discussionsPostFirstHint(): String = L.text(R.string.mobile_discussions_postFirstHint)
@Composable fun discussionsUpvote(): String = L.text(R.string.mobile_discussions_upvote)
@Composable fun discussionsDelete(): String = L.text(R.string.mobile_discussions_delete)
@Composable fun discussionsThreadTitle(): String = L.text(R.string.mobile_discussions_threadTitle)
@Composable fun discussionsMessage(): String = L.text(R.string.mobile_discussions_message)
@Composable fun discussionsPost(): String = L.text(R.string.mobile_discussions_post)
@Composable fun discussionsCancel(): String = L.text(R.string.mobile_discussions_cancel)
@Composable fun discussionsAuthorYou(): String = L.text(R.string.mobile_discussions_authorYou)
