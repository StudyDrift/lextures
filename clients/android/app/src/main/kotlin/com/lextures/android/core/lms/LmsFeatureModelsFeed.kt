package com.lextures.android.core.lms

import kotlinx.serialization.Serializable

// region Course feed & channels (M7.6)

@Serializable
data class FeedChannel(
    val id: String = "",
    val name: String = "",
    val sortOrder: Int = 0,
    val createdAt: String = "",
)

@Serializable
data class FeedMessage(
    val id: String = "",
    val channelId: String = "",
    val authorUserId: String = "",
    val authorEmail: String = "",
    val authorDisplayName: String? = null,
    val parentMessageId: String? = null,
    val body: String = "",
    val mentionsEveryone: Boolean = false,
    val mentionUserIds: List<String> = emptyList(),
    val pinnedAt: String? = null,
    val createdAt: String = "",
    val editedAt: String? = null,
    val likeCount: Int = 0,
    val viewerHasLiked: Boolean = false,
    val replies: List<FeedMessage> = emptyList(),
) {
    val authorLabel: String get() = authorDisplayName?.takeIf { it.isNotBlank() } ?: authorEmail
}

@Serializable
data class FeedRosterPerson(
    val userId: String = "",
    val email: String = "",
    val displayName: String? = null,
    val avatarUrl: String? = null,
) {
    val label: String get() = displayName?.takeIf { it.isNotBlank() } ?: email
}

@Serializable
data class FeedImageUpload(
    val id: String = "",
    val content_path: String = "",
    val mime_type: String = "",
    val byte_size: Long = 0,
)

@Serializable
data class FeedChannelsResponse(val channels: List<FeedChannel> = emptyList())

@Serializable
data class FeedMessagesResponse(val messages: List<FeedMessage> = emptyList())

@Serializable
data class FeedRosterResponse(val people: List<FeedRosterPerson> = emptyList())

@Serializable
data class CreateFeedChannelBody(val name: String)

@Serializable
data class PostFeedMessageBody(
    val body: String,
    val parentMessageId: String? = null,
    val mentionUserIds: List<String> = emptyList(),
    val mentionsEveryone: Boolean = false,
)

@Serializable
data class PatchFeedMessageBody(val body: String)

@Serializable
data class PinFeedMessageBody(val pinned: Boolean)

@Serializable
data class PostFeedMessageResponse(val id: String = "")

// endregion
