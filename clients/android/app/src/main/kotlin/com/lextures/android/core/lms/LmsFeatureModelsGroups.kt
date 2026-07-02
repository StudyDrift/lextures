package com.lextures.android.core.lms

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
data class GroupPublic(
    val id: String,
    val groupSetId: String,
    val name: String,
    val sortOrder: Int = 0,
    val createdAt: String = "",
    val memberCount: Int = 0,
)

@Serializable
data class GroupsListResponse(
    val groups: List<GroupPublic> = emptyList(),
)

@Serializable
enum class CollabDocType {
    @SerialName("rich_text") RichText,
    @SerialName("whiteboard") Whiteboard,
}

@Serializable
data class CollabDoc(
    val id: String,
    val courseId: String,
    val groupId: String? = null,
    val title: String,
    val docType: CollabDocType = CollabDocType.RichText,
    val createdBy: String = "",
    val createdAt: String = "",
    val updatedAt: String = "",
)

@Serializable
data class CollabDocsListResponse(
    val docs: List<CollabDoc> = emptyList(),
)

data class GroupFeedContext(
    val groupId: String,
    val groupName: String,
)

data class GroupMemberRow(
    val id: String,
    val displayName: String,
    val email: String,
)

enum class GroupSpaceTab {
    Members,
    Discussion,
    Files,
    Docs,
}