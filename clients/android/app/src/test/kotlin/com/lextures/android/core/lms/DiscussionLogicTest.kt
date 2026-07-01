package com.lextures.android.core.lms

import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.put
import org.junit.Assert.assertEquals
import org.junit.Test

class DiscussionLogicTest {
    @Test
    fun nestPosts_ordersChildrenAfterParents() {
        val posts = listOf(
            post("root", null),
            post("c1", "root"),
            post("c2", "c1"),
            post("root2", null),
        )
        val nested = DiscussionLogic.nestPosts(posts)
        assertEquals(listOf("root", "c1", "c2", "root2"), nested.map { it.post.id })
        assertEquals(listOf(0, 1, 2, 0), nested.map { it.depth })
    }

    @Test
    fun sortThreads_pinsFirst() {
        val threads = listOf(
            thread("a", pinned = false, updatedAt = "2099-01-02T00:00:00Z"),
            thread("b", pinned = true, updatedAt = "2020-01-01T00:00:00Z"),
        )
        assertEquals(listOf("b", "a"), DiscussionLogic.sortThreads(threads).map { it.id })
    }

    @Test
    fun plainTextFromTipTapDoc() {
        val doc = DiscussionLogic.encodeBody("Hello\nWorld")
        assertEquals("Hello\nWorld", DiscussionLogic.plainText(doc))
    }

    @Test
    fun authorLabel_usesYouForViewer() {
        assertEquals("You", DiscussionLogic.authorLabel("u1", "u1", "You"))
        assertEquals("abcdefgh", DiscussionLogic.authorLabel("abcdefgh123", "other", "You"))
    }

    private fun post(id: String, parent: String?) = DiscussionPost(
        id = id,
        threadId = "t1",
        parentPostId = parent,
        authorId = "u1",
        body = buildJsonObject { put("type", JsonPrimitive("doc")) },
        createdAt = "2024-01-01T00:00:00Z",
        updatedAt = "2024-01-01T00:00:00Z",
    )

    private fun thread(id: String, pinned: Boolean, updatedAt: String) = DiscussionThreadSummary(
        id = id,
        forumId = "f1",
        authorId = "u1",
        title = id,
        isPinned = pinned,
        updatedAt = updatedAt,
        createdAt = updatedAt,
    )
}
