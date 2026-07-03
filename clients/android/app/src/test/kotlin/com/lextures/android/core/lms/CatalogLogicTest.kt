package com.lextures.android.core.lms

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class CatalogLogicTest {
    @Test
    fun isPaidAndFree() {
        assertTrue(CatalogLogic.isPaid(1999))
        assertFalse(CatalogLogic.isPaid(0))
        assertTrue(CatalogLogic.isFree(0))
        assertFalse(CatalogLogic.isFree(500))
    }

    @Test
    fun isEnrolledMatchesCourseCodeCaseInsensitively() {
        val courses = listOf(
            CourseSummary(id = "1", courseCode = "SPAN101", title = "Spanish", description = ""),
        )
        assertTrue(CatalogLogic.isEnrolled("span101", courses))
        assertFalse(CatalogLogic.isEnrolled("FRENCH101", courses))
    }

    @Test
    fun previewParagraphsTrimsAndLimits() {
        val text = "Learn greetings.\n\nPractice daily.\n\n\nSpeak with confidence.\nExtra."
        assertEquals(
            listOf("Learn greetings.", "Practice daily."),
            CatalogLogic.previewParagraphs(text, limit = 2),
        )
    }

    @Test
    fun cacheKeyIncludesFilters() {
        val key = CatalogLogic.cacheKey(
            query = "spanish",
            category = "Languages",
            level = CatalogLevelFilter.Beginner,
            price = CatalogPriceFilter.Free,
            sort = CatalogSortMode.Relevance,
        )
        assertTrue(key.contains("spanish"))
        assertTrue(key.contains("Beginner"))
        assertTrue(key.contains("Free"))
    }

    @Test
    fun catalogWebPath() {
        assertEquals("/explore/spanish-a1", CatalogLogic.catalogWebPath("spanish-a1"))
    }
}