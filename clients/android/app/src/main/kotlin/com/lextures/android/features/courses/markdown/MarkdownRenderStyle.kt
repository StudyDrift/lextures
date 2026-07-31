package com.lextures.android.features.courses.markdown

import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import com.lextures.android.core.accessibility.AccessibilitySupport

/** Spacing and typography tokens for the shared markdown renderer (CT.M2 FR-11 / FR-13). */
object MarkdownRenderStyle {
    fun blockSpacing(compact: Boolean): Dp = if (compact) 6.dp else 12.dp

    fun headingSize(level: Int, compact: Boolean): Int =
        when {
            compact && level == 1 -> 18
            compact && level == 2 -> 16
            compact -> 15
            level == 1 -> 24
            level == 2 -> 19
            else -> 16
        }

    fun headingTopPadding(level: Int, compact: Boolean): Dp =
        when {
            compact && level == 1 -> 2.dp
            compact -> 0.dp
            level == 1 -> 6.dp
            else -> 2.dp
        }

    fun allowsAffordances(suppressAffordances: Boolean): Boolean = !suppressAffordances

    fun plainLabel(text: String): String = AccessibilitySupport.plainTextFromMarkdown(text)
}
