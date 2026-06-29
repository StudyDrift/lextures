package com.lextures.android.core.accessibility

import com.lextures.android.core.design.primaryTextContrastMeetsAA
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class AccessibilitySupportTest {
    @Test
    fun chunkSentences_splitsOnPunctuation() {
        val sentences = AccessibilitySupport.chunkSentences("Hello world. How are you? Fine!")
        assertEquals(listOf("Hello world.", "How are you?", "Fine!"), sentences)
    }

    @Test
    fun plainTextFromMarkdown_stripsFormatting() {
        val plain = AccessibilitySupport.plainTextFromMarkdown(
            "# Title\n\n**Bold** text with [link](https://example.com).",
        )
        assertEquals("Title Bold text with link.", plain)
    }

    @Test
    fun contrastRatio_meetsWcagAAForBrandText() {
        assertTrue(primaryTextContrastMeetsAA)
        val ratio = AccessibilitySupport.contrastRatio(
            AccessibilitySupport.ColorComponents(0x1F2D2A),
            AccessibilitySupport.ColorComponents(0xFAF5EA),
        )
        assertTrue(AccessibilitySupport.meetsWcagAA(ratio))
    }

    @Test
    fun meetsWcagAA_requiresHigherRatioForBodyText() {
        assertTrue(AccessibilitySupport.meetsWcagAA(4.5))
        assertFalse(AccessibilitySupport.meetsWcagAA(4.0))
        assertTrue(AccessibilitySupport.meetsWcagAA(3.0, isLargeText = true))
    }
}
