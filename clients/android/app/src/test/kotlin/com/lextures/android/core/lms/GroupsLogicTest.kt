package com.lextures.android.core.lms

import org.junit.Assert.assertEquals
import org.junit.Test

class GroupsLogicTest {
    @Test
    fun courseCollabDocsFiltersGroupScoped() {
        val docs = listOf(
            CollabDoc(id = "1", courseId = "c1", groupId = null, title = "Course"),
            CollabDoc(id = "2", courseId = "c1", groupId = "g1", title = "Group"),
        )
        assertEquals(listOf("1"), GroupsLogic.courseCollabDocs(docs).map { it.id })
        assertEquals(listOf("2"), GroupsLogic.groupCollabDocs(docs, "g1").map { it.id })
    }

    @Test
    fun memberRowsUsesRosterForAuthors() {
        val roster = listOf(
            FeedRosterPerson(userId = "u1", email = "a@school.edu", displayName = "Alex"),
            FeedRosterPerson(userId = "u2", email = "b@school.edu", displayName = "Blair"),
        )
        val rows = GroupsLogic.memberRows(roster, setOf("u2"))
        assertEquals(1, rows.size)
        assertEquals("Blair", rows.first().displayName)
    }

    @Test
    fun displayInitials() {
        assertEquals("AK", GroupsLogic.displayInitials("Alex Kim"))
        assertEquals("SO", GroupsLogic.displayInitials("solo"))
    }
}