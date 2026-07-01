package com.lextures.android.core.lms

/** Pure helpers for the course feed (M7.6) kept separate from screens for testability. */
object FeedLogic {
    /**
     * Flattens root messages + their replies into a single chronological list, pinned
     * messages first (root-only; replies can't be pinned per the server contract).
     */
    fun orderedMessages(roots: List<FeedMessage>): List<FeedMessage> {
        val pinned = roots.filter { it.pinnedAt != null }.sortedBy { it.createdAt }
        val unpinned = roots.filter { it.pinnedAt == null }.sortedBy { it.createdAt }
        val flattened = unpinned.flatMap { root -> listOf(root) + root.replies.sortedBy { it.createdAt } }
        return pinned + flattened
    }

    fun canEdit(message: FeedMessage, viewerId: String?): Boolean =
        viewerId != null && message.authorUserId == viewerId

    fun canDelete(message: FeedMessage, viewerId: String?): Boolean = canEdit(message, viewerId)

    fun canPin(viewerIsStaff: Boolean, isReply: Boolean): Boolean = viewerIsStaff && !isReply

    private val imageMarkdown = Regex("""!\[[^\]]*\]\(([^)]+)\)""")

    /**
     * Markdown image syntax the web composer emits when a feed image is attached:
     * `![alt](path)`. Extracts the path, if any, so the bubble can render it.
     */
    fun extractImagePath(body: String): Pair<String, String?> {
        val match = imageMarkdown.find(body) ?: return body to null
        val path = match.groupValues[1]
        val remainder = body.removeRange(match.range).trim()
        return remainder to path
    }
}
