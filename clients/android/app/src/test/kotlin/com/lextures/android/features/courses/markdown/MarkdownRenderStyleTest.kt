package com.lextures.android.features.courses.markdown

import com.lextures.android.features.notebooks.inlineMarkdown
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class MarkdownRenderStyleTest {
    @Test
    fun compactSpacingIsTighterThanDefault() {
        assertTrue(MarkdownRenderStyle.blockSpacing(true) < MarkdownRenderStyle.blockSpacing(false))
        assertTrue(MarkdownRenderStyle.headingSize(1, true) < MarkdownRenderStyle.headingSize(1, false))
    }

    @Test
    fun lockdownSuppressesAffordances() {
        assertTrue(MarkdownRenderStyle.allowsAffordances(false))
        assertFalse(MarkdownRenderStyle.allowsAffordances(true))
    }

    @Test
    fun inlineMarkdownKeepsFormattingAndCanSuppressLinks() {
        val withLink = inlineMarkdown("See [docs](https://example.com) and **bold**")
        assertTrue(withLink.text.contains("docs"))
        assertTrue(withLink.text.contains("bold"))

        val suppressed = inlineMarkdown("See [docs](https://example.com)", suppressLinks = true)
        assertEquals("See docs", suppressed.text)
    }

    @Test
    fun plainLabelMergesChoiceText() {
        val label = MarkdownRenderStyle.plainLabel("Use `print()` and **bold**")
        assertEquals("Use print() and bold", label)
        assertFalse(label.contains("`"))
        assertFalse(label.contains("**"))
    }
}
