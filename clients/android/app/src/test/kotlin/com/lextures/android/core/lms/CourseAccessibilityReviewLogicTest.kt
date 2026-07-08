package com.lextures.android.core.lms

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class CourseAccessibilityReviewLogicTest {
    @Test
    fun coveragePercent_emptyTotalIs100() {
        assertEquals(100, CourseAccessibilityReviewLogic.coveragePercent(0, 0))
    }

    @Test
    fun coveragePercent_roundsCorrectly() {
        assertEquals(33, CourseAccessibilityReviewLogic.coveragePercent(1, 3))
        assertEquals(100, CourseAccessibilityReviewLogic.coveragePercent(2, 2))
    }

    @Test
    fun scanMarkdownImages_findsMissingAlt() {
        val markdown = "# Title\n![](/img/a.png)\n![ok](/img/b.png \"lex-decorative\")"
        val images = CourseAccessibilityReviewLogic.scanMarkdownImages(markdown)
        assertEquals(2, images.size)
        assertFalse(images[0].hasValidAlt)
        assertTrue(images[1].decorative)
        assertEquals(1, CourseAccessibilityReviewLogic.missingImages(markdown).size)
    }

    @Test
    fun applyAltTextUpdate_addsDescription() {
        val markdown = "![](/files/chart.png)"
        val updated = CourseAccessibilityReviewLogic.applyAltTextUpdate(
            markdown,
            0,
            "Bar chart of enrollment",
            false,
        )
        assertEquals("![Bar chart of enrollment](/files/chart.png)", updated)
    }

    @Test
    fun applyAltTextUpdate_marksDecorative() {
        val markdown = "![](/files/icon.png)"
        val updated = CourseAccessibilityReviewLogic.applyAltTextUpdate(
            markdown,
            0,
            "",
            true,
        )
        assertEquals("![](/files/icon.png \"lex-decorative\")", updated)
    }

    @Test
    fun supportsInlineEdit_onlyForMarkdownItems() {
        assertTrue(CourseAccessibilityReviewLogic.supportsInlineEdit("content_page"))
        assertTrue(CourseAccessibilityReviewLogic.supportsInlineEdit("assignment"))
        assertFalse(CourseAccessibilityReviewLogic.supportsInlineEdit("quiz"))
    }

    @Test
    fun paginatedUncoveredItems_pagesAtTwenty() {
        val items = (0 until 25).map { index ->
            UncoveredAccessibilityItem(
                itemId = "item-$index",
                title = "Item $index",
                kind = "content_page",
                missing = 1,
                total = 1,
            )
        }
        assertEquals(20, CourseAccessibilityReviewLogic.paginatedUncoveredItems(items, 0).size)
        assertTrue(CourseAccessibilityReviewLogic.hasMorePages(items, 0))
        assertFalse(CourseAccessibilityReviewLogic.hasMorePages(items, 1))
    }
}
