package com.lextures.android.core.lms

import com.lextures.android.core.config.AppConfiguration
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.decodeFromJsonElement
import java.net.URI
import java.time.Instant
import java.util.Collections
import java.util.Locale

/** Helpers for visual collaboration boards (VC.M1–VC.M7). */
object BoardsLogic {
    private val json = Json { ignoreUnknownKeys = true; coerceInputValues = true }

    /** Client-side debounce of repeat reports on the same target (server also rate-limits). */
    private val reportedTargets: MutableSet<String> = Collections.synchronizedSet(mutableSetOf())

    /** Unknown / future layouts fall back to stream (AC-7). */
    fun resolveLayout(raw: String?): BoardLayout =
        BoardLayout.fromApi(raw) ?: BoardLayout.Stream

    /** Fractional index between neighbors (client-side helper for drag reorder). */
    fun midpointSortIndex(before: Double?, after: Double?): Double = when {
        before == null && after == null -> 0.0
        before == null -> after!! - 1.0
        after == null -> before + 1.0
        after <= before -> before + 1.0
        else -> (before + after) / 2.0
    }

    fun canArrangePost(
        post: BoardPost,
        board: Board?,
        currentUserId: String?,
        canManage: Boolean,
    ): Boolean {
        if (canManage) return true
        if (isBoardLocked(board)) return false
        if (board?.capabilities?.canArrange == false) return false
        if (board?.canArrange == false) return false
        if (board?.layoutLocked == true) return false
        val author = post.authorId ?: return false
        return !currentUserId.isNullOrBlank() && author.equals(currentUserId, ignoreCase = true)
    }

    fun parseBoardInstant(raw: String?): Instant? {
        val value = raw?.trim().orEmpty()
        if (value.isEmpty()) return null
        return try {
            Instant.parse(value)
        } catch (_: Exception) {
            null
        }
    }

    fun isBoardLocked(board: Board?): Boolean = board?.locked == true

    fun isBoardFrozen(board: Board?, nowMillis: Long = System.currentTimeMillis()): Boolean {
        val until = parseBoardInstant(board?.frozenUntil) ?: return false
        return until.toEpochMilli() > nowMillis
    }

    /** Lock blocks all non-manager writes; freeze blocks post/comment only (matches server write-gate). */
    fun canWritePosts(board: Board?, canManage: Boolean, nowMillis: Long = System.currentTimeMillis()): Boolean {
        if (canManage) return true
        if (isBoardLocked(board)) return false
        if (isBoardFrozen(board, nowMillis)) return false
        return true
    }

    fun canWriteInteractions(board: Board?, canManage: Boolean): Boolean {
        if (canManage) return true
        return !isBoardLocked(board)
    }

    fun postSafetyState(post: BoardPost): BoardPostSafetyState {
        if (post.hidden || post.status.equals(BoardPostStatus.Rejected.apiValue, ignoreCase = true)) {
            return BoardPostSafetyState.Removed
        }
        if (post.status.equals(BoardPostStatus.Pending.apiValue, ignoreCase = true)) {
            return BoardPostSafetyState.PendingApproval
        }
        val scan = post.attachment?.scanStatus?.lowercase(Locale.US)
        if (scan == "blocked") return BoardPostSafetyState.FileBlocked
        if (scan == "pending") return BoardPostSafetyState.FileScanning
        return BoardPostSafetyState.Normal
    }

    fun reportTargetKey(postId: String?, commentId: String?): String {
        if (!commentId.isNullOrBlank()) return "c:$commentId"
        if (!postId.isNullOrBlank()) return "p:$postId"
        return ""
    }

    fun hasReported(postId: String? = null, commentId: String? = null): Boolean {
        val key = reportTargetKey(postId, commentId)
        if (key.isEmpty()) return false
        return reportedTargets.contains(key)
    }

    fun markReported(postId: String? = null, commentId: String? = null) {
        val key = reportTargetKey(postId, commentId)
        if (key.isEmpty()) return
        reportedTargets.add(key)
    }

    /** Test helper — clears in-memory report debounce state. */
    fun resetReportedTargetsForTests() {
        reportedTargets.clear()
    }

    fun isFilterBlockMessage(message: String?): Boolean {
        val msg = message.orEmpty().lowercase(Locale.US)
        return msg.contains("could not be posted") || msg.contains("couldn't post") || msg.contains("revise")
    }

    fun isLockOrFreezeMessage(message: String?): Boolean {
        val msg = message.orEmpty().lowercase(Locale.US)
        return msg.contains("locked") || msg.contains("frozen")
    }

    fun moderationMode(board: Board?): BoardModerationMode =
        BoardModerationMode.fromApi(board?.moderationMode)

    fun filterAction(board: Board?): BoardFilterAction =
        BoardFilterAction.fromApi(board?.filterAction)

    /** Org floor: managers cannot loosen below approval + block. */
    fun moderationControlsLockedByOrgFloor(board: Board?): Boolean =
        board?.minorModerationFloor == true

    fun sortedSections(sections: List<BoardSection>): List<BoardSection> =
        sections.sortedWith(compareBy({ it.sortIndex }, { it.createdAt }))

    fun postsInSection(posts: List<BoardPost>, sectionId: String?): List<BoardPost> =
        posts
            .filter { if (sectionId == null) it.sectionId == null else it.sectionId == sectionId }
            .sortedWith(compareBy({ it.sortIndex }, { it.createdAt }))

    fun datedPosts(posts: List<BoardPost>): List<BoardPost> =
        posts
            .filter { !it.eventDate.isNullOrBlank() }
            .sortedBy { it.eventDate.orEmpty() }

    fun undatedPosts(posts: List<BoardPost>): List<BoardPost> =
        posts.filter { it.eventDate.isNullOrBlank() }

    fun pinnedPosts(posts: List<BoardPost>): List<BoardPost> =
        posts.filter { it.lat != null && it.lng != null }

    fun unpinnedPosts(posts: List<BoardPost>): List<BoardPost> =
        posts.filter { it.lat == null || it.lng == null }

    data class MapPinCluster(
        val lat: Double,
        val lng: Double,
        val postIds: List<String>,
    )

    /** Simple grid clustering (parity with web map layout). */
    fun clusterPins(posts: List<BoardPost>, zoom: Double): List<MapPinCluster> {
        val pins = pinnedPosts(posts)
        val cell = maxOf(2.0, 40.0 / maxOf(zoom, 1.0))
        val buckets = linkedMapOf<String, MapPinCluster>()
        for (post in pins) {
            val lat = post.lat ?: continue
            val lng = post.lng ?: continue
            val key = "${kotlin.math.floor((lat + 90) / cell).toInt()}_${kotlin.math.floor((lng + 180) / cell).toInt()}"
            val existing = buckets[key]
            if (existing != null) {
                val ids = existing.postIds + post.id
                val n = ids.size.toDouble()
                buckets[key] = MapPinCluster(
                    lat = ((existing.lat * (n - 1)) + lat) / n,
                    lng = ((existing.lng * (n - 1)) + lng) / n,
                    postIds = ids,
                )
            } else {
                buckets[key] = MapPinCluster(lat = lat, lng = lng, postIds = listOf(post.id))
            }
        }
        return buckets.values.toList()
    }

    fun sortIndexMovingUp(post: BoardPost, siblings: List<BoardPost>): Double? {
        val ordered = siblings.sortedBy { it.sortIndex }
        val idx = ordered.indexOfFirst { it.id == post.id }
        if (idx <= 0) return null
        val before = if (idx >= 2) ordered[idx - 2].sortIndex else null
        val after = ordered[idx - 1].sortIndex
        return midpointSortIndex(before, after)
    }

    fun sortIndexMovingDown(post: BoardPost, siblings: List<BoardPost>): Double? {
        val ordered = siblings.sortedBy { it.sortIndex }
        val idx = ordered.indexOfFirst { it.id == post.id }
        if (idx < 0 || idx >= ordered.size - 1) return null
        val before = ordered[idx + 1].sortIndex
        val after = if (idx + 2 < ordered.size) ordered[idx + 2].sortIndex else null
        return midpointSortIndex(before, after)
    }

    fun layoutHidesSortControls(layout: BoardLayout): Boolean = when (layout) {
        BoardLayout.Canvas, BoardLayout.Timeline, BoardLayout.Map, BoardLayout.Columns -> true
        BoardLayout.Wall, BoardLayout.Stream, BoardLayout.Grid -> false
    }

    fun canCreateBoards(courseCode: String, permissions: List<String>): Boolean =
        CourseSettingsLogic.canManageCourse(courseCode, permissions)

    /** Prefer server capabilities.canManage; fall back to course create permission (VC.M6 FR-1). */
    fun canManageBoard(board: Board?, courseCode: String, permissions: List<String>): Boolean {
        board?.capabilities?.canManage?.let { return it }
        return canCreateBoards(courseCode, permissions)
    }

    /** Prefer server capabilities/canPost; fall back to create permission. */
    fun canPost(board: Board?, courseCode: String, permissions: List<String>): Boolean {
        board?.capabilities?.canPost?.let { return it }
        board?.canPost?.let { return it }
        return canCreateBoards(courseCode, permissions)
    }

    /** Prefer server capabilities/canInteract; default true when unset (VC.M5 FR-9 / VC.M6). */
    fun canInteract(board: Board?): Boolean {
        board?.capabilities?.canInteract?.let { return it }
        board?.canInteract?.let { return it }
        return true
    }

    fun canGrade(board: Board?, canManage: Boolean): Boolean =
        canManage && BoardReactionMode.fromApi(board?.reactionMode) == BoardReactionMode.Grade

    /** Authorship strictly from server payload — never invent an author (VC.M6 FR-2). */
    fun attributionLabel(authorId: String?, guestDisplayName: String?): String? {
        val guest = guestDisplayName?.trim().orEmpty()
        if (guest.isNotEmpty()) return guest
        val author = authorId?.trim().orEmpty()
        if (author.isNotEmpty()) return author
        return null
    }

    fun attributionLabel(post: BoardPost): String? =
        attributionLabel(post.authorId, post.guestDisplayName)

    /** External share links when the server allows (and not COPPA-floored). */
    fun externalSharingAllowed(board: Board?): Boolean {
        if (board?.minorModerationFloor == true) return false
        return board?.externalSharingAllowed == true
    }

    /** `public`/`link` visibility when external sharing is allowed (unlocked with VC.M7 moderation). */
    fun visibilityOptions(board: Board?): List<BoardVisibility> {
        val opts = BoardVisibility.inCourse.toMutableList()
        if (externalSharingAllowed(board)) {
            opts.add(BoardVisibility.Link)
            opts.add(BoardVisibility.Public)
        }
        return opts
    }

    fun shareUrl(share: BoardShare): String? {
        val raw = share.url?.trim().orEmpty()
        if (raw.isNotEmpty()) {
            return if (raw.startsWith("http://") || raw.startsWith("https://")) {
                raw
            } else {
                AppConfiguration.webUrl(if (raw.startsWith("/")) raw else "/$raw")
            }
        }
        val token = share.token?.trim().orEmpty()
        if (token.isNotEmpty()) return AppConfiguration.webUrl("/board-links/$token")
        return null
    }

    fun classifyBoardLinkError(status: Int, message: String?): BoardLinkAccessState {
        val msg = message.orEmpty().lowercase(Locale.US)
        if (status == 401 || msg.contains("password") || msg.contains("incorrect")) {
            return BoardLinkAccessState.NeedsPassword
        }
        return BoardLinkAccessState.Denied
    }

    fun assignmentLinked(board: Board?): Boolean =
        !board?.assignmentId?.trim().isNullOrEmpty()

    fun canEditOrDeletePost(post: BoardPost, currentUserId: String?, canManage: Boolean): Boolean {
        if (canManage) return true
        val author = post.authorId ?: return false
        return !currentUserId.isNullOrBlank() && author == currentUserId
    }

    /** FERPA-safe: render only the grade the server returned for this viewer (nil for peers). */
    fun visibleGrade(post: BoardPost): Double? = post.grade

    fun applyReactionResult(post: BoardPost, result: BoardReactionResult): BoardPost =
        post.copy(
            reactionCount = result.reactionCount ?: post.reactionCount,
            myReaction = if (result.active) (result.myReaction ?: post.myReaction) else null,
            avgStars = result.avgStars ?: if (result.active) post.avgStars else null,
            commentCount = result.commentCount ?: post.commentCount,
            grade = result.grade ?: if (result.active) post.grade else null,
        )

    /** Optimistic like/vote toggle before the server responds. */
    fun optimisticToggleReaction(post: BoardPost, kind: String): BoardPost {
        val pressed = post.myReaction != null
        return if (pressed) {
            post.copy(
                myReaction = null,
                reactionCount = maxOf(0, (post.reactionCount ?: 1) - 1),
            )
        } else {
            post.copy(
                myReaction = BoardMyReaction(kind = kind),
                reactionCount = (post.reactionCount ?: 0) + 1,
            )
        }
    }

    fun optimisticSetStars(post: BoardPost, value: Int): BoardPost {
        val had = post.myReaction != null
        return post.copy(
            myReaction = BoardMyReaction(kind = "star", value = value.toDouble()),
            reactionCount = if (had) post.reactionCount else (post.reactionCount ?: 0) + 1,
        )
    }

    fun boardPostReactionScore(post: BoardPost, mode: BoardReactionMode): Double = when (mode) {
        BoardReactionMode.Star -> (post.avgStars ?: 0.0) * 1000 + (post.reactionCount ?: 0).toDouble()
        BoardReactionMode.Grade -> post.grade ?: (post.reactionCount ?: 0).toDouble()
        BoardReactionMode.Like, BoardReactionMode.Vote -> (post.reactionCount ?: 0).toDouble()
        BoardReactionMode.None -> 0.0
    }

    data class NestedComment(
        val comment: BoardComment,
        val children: List<BoardComment>,
    )

    fun nestComments(comments: List<BoardComment>): List<NestedComment> {
        val byParent = linkedMapOf<String?, MutableList<BoardComment>>()
        for (c in comments) {
            byParent.getOrPut(c.parentId) { mutableListOf() }.add(c)
        }
        val roots = byParent[null].orEmpty()
        return roots.map { NestedComment(it, byParent[it.id].orEmpty()) }
    }

    fun visibleComments(comments: List<BoardComment>, canManageBoard: Boolean): List<BoardComment> =
        if (canManageBoard) comments else comments.filter { !it.hidden }

    fun commentPlainText(comment: BoardComment): String {
        val text = comment.body?.text?.trim().orEmpty()
        if (text.isNotEmpty()) return text
        val html = comment.body?.html.orEmpty()
        if (html.isNotEmpty()) return stripHtml(html)
        return ""
    }

    fun formatAvgStars(avg: Double): String = String.format(Locale.US, "%.1f", avg)

    fun formatGrade(value: Double): String =
        if (value == value.toLong().toDouble()) value.toLong().toString()
        else String.format(Locale.US, "%g", value)

    /** Sort by `updatedAt` descending. Archived boards stay unless filtered out. */
    fun sortedBoards(boards: List<Board>, includeArchived: Boolean): List<Board> {
        val filtered = if (includeArchived) boards else boards.filter { !it.archived }
        return filtered.sortedByDescending { LmsDates.parse(it.updatedAt)?.toEpochMilli() ?: 0L }
    }

    /** Sort posts for layouts that support sort controls (FR-9). Defaults to newest. */
    fun sortedPosts(
        posts: List<BoardPost>,
        mode: BoardSortMode = BoardSortMode.Newest,
        reactionMode: BoardReactionMode = BoardReactionMode.None,
    ): List<BoardPost> =
        when (mode) {
            BoardSortMode.Newest ->
                posts.sortedWith(
                    compareByDescending<BoardPost> { LmsDates.parse(it.createdAt)?.toEpochMilli() ?: 0L }
                        .thenByDescending { it.sortIndex },
                )
            BoardSortMode.Oldest ->
                posts.sortedWith(
                    compareBy<BoardPost> { LmsDates.parse(it.createdAt)?.toEpochMilli() ?: 0L }
                        .thenBy { it.sortIndex },
                )
            BoardSortMode.Author ->
                posts.sortedWith(
                    compareBy<BoardPost> { (it.authorId ?: "").lowercase(Locale.US) }
                        .thenBy { it.sortIndex },
                )
            BoardSortMode.MostReacted ->
                posts.sortedWith(
                    compareByDescending<BoardPost> { boardPostReactionScore(it, reactionMode) }
                        .thenByDescending { LmsDates.parse(it.createdAt)?.toEpochMilli() ?: 0L },
                )
        }

    fun relativeUpdatedLabel(board: Board): String {
        val relative = LmsDates.relative(board.updatedAt)
        if (relative.isBlank()) return ""
        return relative
    }

    fun isKnownContentType(raw: String): Boolean =
        BoardContentType.known.contains(raw.lowercase(Locale.US))

    fun bodyPlainText(post: BoardPost): String {
        val text = post.body?.text?.trim().orEmpty()
        if (text.isNotEmpty()) return text
        val html = post.body?.html.orEmpty()
        if (html.isNotEmpty()) return stripHtml(html)
        return ""
    }

    /** Never returns a URL when AV scan is pending/blocked (VC.M2 FR-8). */
    fun attachmentMediaUrl(attachment: BoardAttachment?): String? {
        if (attachment == null) return null
        if (!attachment.scanStatus.equals("clean", ignoreCase = true)) return null
        val raw = attachment.url?.trim().orEmpty()
        if (raw.isEmpty()) return null
        return absoluteUrl(raw)
    }

    fun absoluteUrl(pathOrUrl: String): String? {
        val trimmed = pathOrUrl.trim()
        if (trimmed.isEmpty()) return null
        if (trimmed.startsWith("http://") || trimmed.startsWith("https://")) return trimmed
        val path = if (trimmed.startsWith("/")) trimmed else "/$trimmed"
        return AppConfiguration.apiUrl(path).toString()
    }

    fun validateCompose(
        contentType: BoardContentType,
        text: String,
        linkUrl: String,
        hasFile: Boolean,
        altText: String,
        hasAudio: Boolean,
    ): BoardComposeValidation = when (contentType) {
        BoardContentType.Text ->
            if (text.isBlank()) BoardComposeValidation.MissingText else BoardComposeValidation.Ok
        BoardContentType.Link, BoardContentType.Video ->
            if (linkUrl.isBlank()) BoardComposeValidation.MissingLink else BoardComposeValidation.Ok
        BoardContentType.Image -> when {
            !hasFile -> BoardComposeValidation.MissingFile
            altText.isBlank() -> BoardComposeValidation.MissingAltText
            else -> BoardComposeValidation.Ok
        }
        BoardContentType.File ->
            if (hasFile) BoardComposeValidation.Ok else BoardComposeValidation.MissingFile
        BoardContentType.Audio ->
            if (hasAudio || hasFile) BoardComposeValidation.Ok else BoardComposeValidation.MissingAudio
        BoardContentType.Drawing -> BoardComposeValidation.MissingFile
    }

    fun videoEmbedFromUrl(urlString: String): BoardVideoEmbed? {
        val trimmed = urlString.trim()
        if (trimmed.isEmpty()) return null
        return try {
            val uri = URI(trimmed)
            val host = (uri.host ?: "").lowercase(Locale.US)
            when {
                host == "youtu.be" -> {
                    val id = uri.path.trim('/').substringBefore('/')
                    if (id.isNotEmpty()) BoardVideoEmbed("youtube", id) else null
                }
                host.contains("youtube.com") -> {
                    val query = uri.query.orEmpty()
                    val v = query.split('&').mapNotNull { part ->
                        val kv = part.split('=', limit = 2)
                        if (kv.size == 2 && kv[0] == "v") kv[1] else null
                    }.firstOrNull()
                    if (!v.isNullOrEmpty()) return BoardVideoEmbed("youtube", v)
                    val parts = uri.path.trim('/').split('/')
                    val idx = parts.indexOfFirst { it == "embed" || it == "shorts" || it == "v" }
                    if (idx >= 0 && idx + 1 < parts.size) BoardVideoEmbed("youtube", parts[idx + 1]) else null
                }
                host.contains("vimeo.com") -> {
                    val id = uri.path.trim('/').substringAfterLast('/')
                    if (id.isNotEmpty() && id.all { it.isDigit() }) BoardVideoEmbed("vimeo", id) else null
                }
                else -> null
            }
        } catch (_: Exception) {
            null
        }
    }

    fun embedUrl(embed: BoardVideoEmbed): String? = when (embed.provider) {
        "youtube" -> "https://www.youtube.com/embed/${embed.id}"
        "vimeo" -> "https://player.vimeo.com/video/${embed.id}"
        else -> null
    }

    fun formatFileSize(bytes: Long): String {
        if (bytes < 1024) return "$bytes B"
        val kb = bytes / 1024.0
        if (kb < 1024) return String.format(Locale.US, "%.1f KB", kb)
        val mb = kb / 1024.0
        if (mb < 1024) return String.format(Locale.US, "%.1f MB", mb)
        return String.format(Locale.US, "%.1f GB", mb / 1024.0)
    }

    fun parseDrawingElements(data: JsonElement?): List<WhiteboardElement> {
        if (data == null) return emptyList()
        return try {
            when (data) {
                is JsonArray -> json.decodeFromJsonElement(data)
                is JsonObject -> {
                    val elements = data["elements"]
                    if (elements != null) parseDrawingElements(elements) else emptyList()
                }
                else -> emptyList()
            }
        } catch (_: Exception) {
            emptyList()
        }
    }

    fun makeTextBody(text: String): BoardPostBody {
        val trimmed = text.trim()
        val escaped = trimmed
            .replace("&", "&amp;")
            .replace("<", "&lt;")
            .replace(">", "&gt;")
        return BoardPostBody(html = "<p>$escaped</p>", text = trimmed)
    }

    private fun stripHtml(html: String): String =
        html
            .replace(Regex("<[^>]+>"), "")
            .replace("&nbsp;", " ")
            .replace("&amp;", "&")
            .replace("&lt;", "<")
            .replace("&gt;", ">")
            .trim()
}
