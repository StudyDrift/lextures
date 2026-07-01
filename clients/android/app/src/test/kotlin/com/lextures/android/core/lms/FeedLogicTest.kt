package com.lextures.android.core.lms

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class FeedLogicTest {
    @Test
    fun orderedMessages_putsPinnedFirstThenFlattensReplies() {
        val reply = message(id = "r1", createdAt = "2024-01-01T00:01:00Z")
        val root = message(id = "root", createdAt = "2024-01-01T00:00:00Z", replies = listOf(reply))
        val pinned = message(id = "pinned", createdAt = "2024-01-01T00:02:00Z", pinnedAt = "2024-01-01T00:02:30Z")
        val ordered = FeedLogic.orderedMessages(listOf(root, pinned))
        assertEquals(listOf("pinned", "root", "r1"), ordered.map { it.id })
    }

    @Test
    fun canEditAndDelete_requireAuthorMatch() {
        val msg = message(id = "m1", authorUserId = "u1")
        assertTrue(FeedLogic.canEdit(msg, "u1"))
        assertFalse(FeedLogic.canEdit(msg, "u2"))
        assertFalse(FeedLogic.canEdit(msg, null))
        assertTrue(FeedLogic.canDelete(msg, "u1"))
    }

    @Test
    fun canPin_requiresStaffAndRootMessage() {
        assertTrue(FeedLogic.canPin(viewerIsStaff = true, isReply = false))
        assertFalse(FeedLogic.canPin(viewerIsStaff = true, isReply = true))
        assertFalse(FeedLogic.canPin(viewerIsStaff = false, isReply = false))
    }

    @Test
    fun extractImagePath_splitsMarkdownFromText() {
        val (text, path) = FeedLogic.extractImagePath("hello\n\n![image](/api/v1/x/content)")
        assertEquals("hello", text)
        assertEquals("/api/v1/x/content", path)
    }

    @Test
    fun extractImagePath_returnsNullWhenNoMarkdown() {
        val (text, path) = FeedLogic.extractImagePath("just text")
        assertEquals("just text", text)
        assertNull(path)
    }

    private fun message(
        id: String,
        authorUserId: String = "u1",
        createdAt: String = "2024-01-01T00:00:00Z",
        pinnedAt: String? = null,
        replies: List<FeedMessage> = emptyList(),
    ) = FeedMessage(
        id = id,
        channelId = "c1",
        authorUserId = authorUserId,
        authorEmail = "user@example.com",
        body = "body",
        pinnedAt = pinnedAt,
        createdAt = createdAt,
        replies = replies,
    )
}
