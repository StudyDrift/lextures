package com.lextures.android.core.lms

import kotlinx.serialization.Serializable
import kotlinx.serialization.json.JsonElement

/** Visual collaboration board (VC.M1–VC.M7). Subset of web `Board`; unknown fields ignored. */
@Serializable
data class Board(
    val id: String,
    val courseId: String,
    val title: String,
    val description: String = "",
    val slug: String = "",
    val archived: Boolean = false,
    val layout: String = "wall",
    val layoutLocked: Boolean = false,
    val reactionMode: String = "none",
    val assignmentId: String? = null,
    val visibility: String = "course",
    val visibilityTarget: String? = null,
    val attribution: String = "named",
    val canPost: Boolean? = null,
    val canInteract: Boolean? = null,
    val canArrange: Boolean? = null,
    val moderationMode: String = "open",
    val filterAction: String = "flag",
    val locked: Boolean = false,
    val frozenUntil: String? = null,
    val capabilities: BoardCapabilities? = null,
    val externalSharingAllowed: Boolean? = null,
    val minorModerationFloor: Boolean? = null,
    val createdBy: String? = null,
    val createdAt: String = "",
    val updatedAt: String = "",
)

enum class BoardVisibility(val apiValue: String) {
    Course("course"),
    Section("section"),
    Group("group"),
    Invite("invite"),
    Link("link"),
    Public("public"),
    ;

    companion object {
        /** In-course scopes (always offered). link/public added when external sharing is allowed (VC.M7). */
        val inCourse: List<BoardVisibility> = listOf(Course, Section, Group, Invite)

        fun fromApi(raw: String?): BoardVisibility =
            entries.firstOrNull { it.apiValue.equals(raw, ignoreCase = true) } ?: Course
    }
}

enum class BoardModerationMode(val apiValue: String) {
    Open("open"),
    Approval("approval"),
    ;

    companion object {
        fun fromApi(raw: String?): BoardModerationMode =
            entries.firstOrNull { it.apiValue.equals(raw, ignoreCase = true) } ?: Open
    }
}

enum class BoardFilterAction(val apiValue: String) {
    Flag("flag"),
    Block("block"),
    ;

    companion object {
        fun fromApi(raw: String?): BoardFilterAction =
            entries.firstOrNull { it.apiValue.equals(raw, ignoreCase = true) } ?: Flag
    }
}

enum class BoardPostStatus(val apiValue: String) {
    Approved("approved"),
    Pending("pending"),
    Rejected("rejected"),
    ;

    companion object {
        fun fromApi(raw: String?): BoardPostStatus =
            entries.firstOrNull { it.apiValue.equals(raw, ignoreCase = true) } ?: Approved
    }
}

enum class BoardReportKind(val apiValue: String) {
    User("user"),
    Filter("filter"),
    AvBlocked("av_blocked"),
}

enum class BoardPostSafetyState {
    Normal,
    PendingApproval,
    Removed,
    FileScanning,
    FileBlocked,
}

enum class BoardAttribution(val apiValue: String) {
    Named("named"),
    AnonToPeers("anon_to_peers"),
    Anonymous("anonymous"),
    ;

    companion object {
        fun fromApi(raw: String?): BoardAttribution =
            entries.firstOrNull { it.apiValue.equals(raw, ignoreCase = true) } ?: Named
    }
}

enum class BoardMemberRole(val apiValue: String) {
    Owner("owner"),
    Editor("editor"),
    Contributor("contributor"),
    Viewer("viewer"),
}

enum class BoardShareCapability(val apiValue: String) {
    View("view"),
    Contribute("contribute"),
    ;

    companion object {
        fun fromApi(raw: String?): BoardShareCapability =
            entries.firstOrNull { it.apiValue.equals(raw, ignoreCase = true) } ?: View
    }
}

enum class BoardReactionMode(val apiValue: String) {
    None("none"),
    Like("like"),
    Vote("vote"),
    Star("star"),
    Grade("grade"),
    ;

    companion object {
        fun fromApi(raw: String?): BoardReactionMode =
            entries.firstOrNull { it.apiValue.equals(raw, ignoreCase = true) } ?: None
    }
}

@Serializable
data class BoardMyReaction(
    val kind: String = "",
    val value: Double? = null,
)

@Serializable
data class BoardComment(
    val id: String,
    val postId: String,
    val parentId: String? = null,
    val authorId: String? = null,
    val body: BoardPostBody? = null,
    val hidden: Boolean = false,
    val createdAt: String = "",
    val updatedAt: String = "",
)

@Serializable
data class BoardCommentsListResponse(
    val comments: List<BoardComment> = emptyList(),
)

@Serializable
data class PutBoardReactionBody(
    val kind: String? = null,
    val value: Double? = null,
)

@Serializable
data class CreateBoardCommentBody(
    val body: BoardPostBody,
    val parentId: String? = null,
)

@Serializable
data class PatchBoardCommentBody(
    val body: BoardPostBody? = null,
    val hidden: Boolean? = null,
)

@Serializable
data class BoardReactionResult(
    val active: Boolean = false,
    val removed: Boolean? = null,
    val reactionCount: Int? = null,
    val myReaction: BoardMyReaction? = null,
    val avgStars: Double? = null,
    val commentCount: Int? = null,
    val grade: Double? = null,
)

@Serializable
data class BoardGradeSyncResult(
    val synced: Boolean = false,
    val pointsEarned: Double = 0.0,
)

enum class BoardLayout(val apiValue: String) {
    Wall("wall"),
    Stream("stream"),
    Grid("grid"),
    Columns("columns"),
    Canvas("canvas"),
    Timeline("timeline"),
    Map("map"),
    ;

    companion object {
        val allApiValues: Set<String> = entries.map { it.apiValue }.toSet()

        fun fromApi(raw: String?): BoardLayout? =
            entries.firstOrNull { it.apiValue.equals(raw, ignoreCase = true) }
    }
}

enum class BoardSortMode {
    Newest,
    Oldest,
    Author,
    MostReacted,
}

@Serializable
data class BoardSection(
    val id: String,
    val boardId: String,
    val title: String = "",
    val sortIndex: Double = 0.0,
    val createdAt: String = "",
)

@Serializable
data class BoardSectionsListResponse(
    val sections: List<BoardSection> = emptyList(),
)

@Serializable
data class CreateBoardSectionBody(
    val title: String,
    val sortIndex: Double? = null,
)

@Serializable
data class PatchBoardSectionBody(
    val title: String? = null,
    val sortIndex: Double? = null,
)

@Serializable
data class ArrangeBoardPostBody(
    val sectionId: String? = null,
    val sortIndex: Double? = null,
    val position: BoardPostPosition? = null,
    val eventDate: String? = null,
    val lat: Double? = null,
    val lng: Double? = null,
    val clearGeo: Boolean? = null,
)

@Serializable
data class BoardCapabilities(
    val canView: Boolean? = null,
    val canPost: Boolean? = null,
    val canInteract: Boolean? = null,
    val canArrange: Boolean? = null,
    val canManage: Boolean? = null,
)

@Serializable
data class BoardsListResponse(
    val boards: List<Board> = emptyList(),
)

@Serializable
data class CreateBoardBody(
    val title: String,
    val description: String = "",
)

@Serializable
data class PatchBoardBody(
    val title: String? = null,
    val description: String? = null,
    val archived: Boolean? = null,
    val layout: String? = null,
    val layoutLocked: Boolean? = null,
    val visibility: String? = null,
    val visibilityTarget: String? = null,
    val attribution: String? = null,
    val canPost: Boolean? = null,
    val canInteract: Boolean? = null,
    val canArrange: Boolean? = null,
    val moderationMode: String? = null,
    val filterAction: String? = null,
    val locked: Boolean? = null,
    val frozenUntil: String? = null,
    val freezeMinutes: Int? = null,
)

@Serializable
data class BoardReport(
    val id: String,
    val boardId: String = "",
    val postId: String? = null,
    val commentId: String? = null,
    val reporterId: String? = null,
    val reason: String = "",
    val kind: String = "user",
    val status: String = "open",
    val createdAt: String = "",
    val resolvedAt: String? = null,
    val resolvedBy: String? = null,
)

@Serializable
data class BoardModerationQueue(
    val pending: List<BoardPost> = emptyList(),
    val reports: List<BoardReport> = emptyList(),
    val flagged: List<BoardReport> = emptyList(),
    val minorsFloor: Boolean = false,
)

@Serializable
data class CreateBoardReportBody(
    val postId: String? = null,
    val commentId: String? = null,
    val reason: String? = null,
)

@Serializable
data class ResolveBoardReportBody(
    val action: String,
    val reason: String? = null,
)

@Serializable
data class BoardModerationActionBody(
    val reason: String? = null,
)

@Serializable
data class BoardMember(
    val boardId: String = "",
    val userId: String,
    val role: String = "contributor",
    val createdAt: String = "",
)

@Serializable
data class BoardMembersListResponse(
    val members: List<BoardMember> = emptyList(),
)

@Serializable
data class UpsertBoardMemberBody(
    val userId: String,
    val role: String,
)

@Serializable
data class BoardShare(
    val id: String,
    val boardId: String = "",
    val capability: String = "view",
    val hasPassword: Boolean = false,
    val expiresAt: String? = null,
    val revokedAt: String? = null,
    val createdBy: String? = null,
    val createdAt: String = "",
    val token: String? = null,
    val url: String? = null,
)

@Serializable
data class BoardSharesListResponse(
    val shares: List<BoardShare> = emptyList(),
)

@Serializable
data class CreateBoardShareBody(
    val capability: String,
    val password: String? = null,
    val expiresAt: String? = null,
)

@Serializable
data class BoardLinkResolve(
    val board: Board,
    val capability: String = "view",
    val requiresPassword: Boolean? = null,
    val posts: List<BoardPost> = emptyList(),
)

@Serializable
data class CreateBoardLinkPostBody(
    val displayName: String,
    val contentType: String,
    val title: String? = null,
    val body: BoardPostBody? = null,
    val linkUrl: String? = null,
)

enum class BoardLinkAccessState {
    Loading,
    NeedsPassword,
    Denied,
    Ready,
}

// region Posts (VC.M2)

enum class BoardContentType(val apiValue: String) {
    Text("text"),
    Image("image"),
    File("file"),
    Link("link"),
    Video("video"),
    Audio("audio"),
    Drawing("drawing"),
    ;

    companion object {
        val known: Set<String> = entries.map { it.apiValue }.toSet()

        fun fromApi(raw: String): BoardContentType? =
            entries.firstOrNull { it.apiValue.equals(raw, ignoreCase = true) }
    }
}

@Serializable
data class BoardAttachment(
    val id: String,
    val url: String? = null,
    val fileName: String = "",
    val mimeType: String = "",
    val sizeBytes: Long = 0,
    val altText: String = "",
    val scanStatus: String = "pending",
)

@Serializable
data class BoardLinkPreview(
    val title: String? = null,
    val description: String? = null,
    val image: String? = null,
    val siteName: String? = null,
    val fetchedAt: String? = null,
    val url: String? = null,
    val provider: String? = null,
    val embedId: String? = null,
)

@Serializable
data class BoardPostBody(
    val html: String? = null,
    val text: String? = null,
)

@Serializable
data class BoardPostPosition(
    val x: Double? = null,
    val y: Double? = null,
    val w: Double? = null,
    val h: Double? = null,
)

@Serializable
data class BoardPost(
    val id: String,
    val boardId: String,
    val authorId: String? = null,
    val guestDisplayName: String? = null,
    val contentType: String,
    val title: String = "",
    val body: BoardPostBody? = null,
    val linkUrl: String? = null,
    val linkPreview: BoardLinkPreview? = null,
    val drawingData: JsonElement? = null,
    val attachment: BoardAttachment? = null,
    val sectionId: String? = null,
    val sortIndex: Double = 0.0,
    val position: BoardPostPosition? = null,
    val eventDate: String? = null,
    val lat: Double? = null,
    val lng: Double? = null,
    val status: String = "approved",
    val hidden: Boolean = false,
    val reactionCount: Int? = null,
    val myReaction: BoardMyReaction? = null,
    val avgStars: Double? = null,
    val commentCount: Int? = null,
    val grade: Double? = null,
    val createdAt: String = "",
    val updatedAt: String = "",
)

@Serializable
data class BoardPostsListResponse(
    val posts: List<BoardPost> = emptyList(),
)

@Serializable
data class CreateBoardPostBody(
    val contentType: String,
    val title: String? = null,
    val body: BoardPostBody? = null,
    val linkUrl: String? = null,
    val attachmentId: String? = null,
)

@Serializable
data class PatchBoardPostBody(
    val title: String? = null,
    val body: BoardPostBody? = null,
    val linkUrl: String? = null,
)

@Serializable
data class BoardLinkPreviewBody(
    val url: String,
)

data class BoardVideoEmbed(
    val provider: String,
    val id: String,
)

enum class BoardComposeValidation {
    Ok,
    MissingText,
    MissingLink,
    MissingFile,
    MissingAltText,
    MissingAudio,
}

// endregion
