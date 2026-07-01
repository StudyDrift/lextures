package com.lextures.android.core.lms

import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class LibraryResourceLogicTest {
    @Test
    fun libraryItemsFilter() {
        val items = listOf(
            CourseStructureItem(id = "1", kind = "library_resource", title = "Reading", sortOrder = 0),
            CourseStructureItem(id = "2", kind = "quiz", title = "Quiz", sortOrder = 1),
        )
        assertEquals(1, LibraryResourceLogic.libraryItems(items).size)
        assertTrue(LibraryResourceLogic.hasLibraryResources(items))
    }

    @Test
    fun resolveAccessUsesEzproxyUrl() {
        val payload = LibraryResourcePayload(
            itemId = "abc",
            resourceType = "catalog_item",
            ezproxyUrl = "https://ezproxy.example.edu/login?url=https://publisher.example/book",
        )
        val state = LibraryResourceLogic.resolveAccess(payload)
        assertTrue(state is LibraryAccessState.Ready)
        assertTrue((state as LibraryAccessState.Ready).url.contains("ezproxy.example.edu"))
    }

    @Test
    fun resolveAccessLegantoGatedWithoutUrl() {
        val payload = LibraryResourcePayload(
            itemId = "abc",
            resourceType = "leganto_list",
            metadata = LibraryResourceMeta(legantoListId = "list-1"),
        )
        val state = LibraryResourceLogic.resolveAccess(payload)
        assertTrue(state is LibraryAccessState.Gated)
        assertEquals("mobile.library.legantoGated", (state as LibraryAccessState.Gated).messageKey)
    }

    @Test
    fun defaultOerProviderPrefersCommons() {
        assertEquals(
            "oer_commons",
            LibraryResourceLogic.defaultOerProvider(listOf("merlot", "oer_commons")),
        )
    }
}