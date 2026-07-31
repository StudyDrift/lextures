package com.lextures.android.core.accessibility

import com.lextures.android.core.notebook.NotebookBlock
import com.lextures.android.core.notebook.NotebookMarkdown
import kotlin.math.pow

/** Pure helpers shared by read-aloud, contrast checks, and tests. */
object AccessibilitySupport {
    const val MINIMUM_TAP_TARGET_DP = 48

    data class ColorComponents(val red: Double, val green: Double, val blue: Double) {
        constructor(hex: Long) : this(
            red = ((hex shr 16) and 0xFF) / 255.0,
            green = ((hex shr 8) and 0xFF) / 255.0,
            blue = (hex and 0xFF) / 255.0,
        )
    }

    fun contrastRatio(foreground: ColorComponents, background: ColorComponents): Double {
        val l1 = relativeLuminance(foreground) + 0.05
        val l2 = relativeLuminance(background) + 0.05
        return if (l1 > l2) l1 / l2 else l2 / l1
    }

    fun meetsWcagAA(ratio: Double, isLargeText: Boolean = false): Boolean =
        if (isLargeText) ratio >= 3.0 else ratio >= 4.5

    fun chunkSentences(text: String): List<String> {
        val trimmed = text.replace(Regex("\\s+"), " ").trim()
        if (trimmed.isEmpty()) return emptyList()

        val sentences = mutableListOf<String>()
        var current = StringBuilder()
        for (character in trimmed) {
            current.append(character)
            if (character in ".!?…") {
                val chunk = current.toString().trim()
                if (chunk.isNotEmpty()) sentences.add(chunk)
                current = StringBuilder()
            }
        }
        val tail = current.toString().trim()
        if (tail.isNotEmpty()) sentences.add(tail)
        return sentences
    }

    /**
     * Plain-text projection for read-aloud (CT.M2 FR-12 / AC-10).
     * Uses the shared block parser so tables/code/math linearise without pipe/fence markup.
     */
    fun plainTextFromMarkdown(markdown: String): String {
        val parts = mutableListOf<String>()
        for (block in NotebookMarkdown.parseBlocks(markdown)) {
            when (block) {
                is NotebookBlock.Heading -> appendStripped(block.text, parts)
                is NotebookBlock.Paragraph -> appendStripped(block.text, parts)
                is NotebookBlock.Quote -> appendStripped(block.text, parts)
                is NotebookBlock.BulletItem -> appendStripped(block.text, parts)
                is NotebookBlock.OrderedItem -> appendStripped(block.text, parts)
                is NotebookBlock.TaskItem -> appendStripped(block.text, parts)
                is NotebookBlock.TaskBlock -> appendStripped(block.task.text, parts)
                is NotebookBlock.Code -> {
                    val trimmed = block.text.trim()
                    if (trimmed.isNotEmpty()) parts.add(trimmed)
                }
                is NotebookBlock.Math -> {
                    val trimmed = block.latex.trim()
                    if (trimmed.isNotEmpty()) parts.add(trimmed)
                }
                is NotebookBlock.Table -> {
                    val headerLine = block.header.map(::stripInlineMarkdown).filter { it.isNotEmpty() }.joinToString(", ")
                    if (headerLine.isNotEmpty()) parts.add(headerLine)
                    for (row in block.rows) {
                        val rowLine = row.map(::stripInlineMarkdown).filter { it.isNotEmpty() }.joinToString(", ")
                        if (rowLine.isNotEmpty()) parts.add(rowLine)
                    }
                }
                is NotebookBlock.ToolFence -> if (block.toolId.isNotEmpty()) parts.add(block.toolId)
                is NotebookBlock.Image -> if (block.alt.isNotEmpty()) parts.add(block.alt)
                is NotebookBlock.Drawing -> parts.add("Drawing")
                NotebookBlock.Divider -> Unit
            }
        }
        return parts.joinToString(" ").replace(Regex("\\s+"), " ").trim()
    }

    fun stripInlineMarkdown(text: String): String {
        var line = text
        val replacements = listOf(
            Regex("!\\[[^\\]]*]\\([^)]*\\)") to "",
            Regex("\\[([^\\]]+)]\\([^)]*\\)") to "$1",
            Regex("`([^`]+)`") to "$1",
            Regex("\\*\\*([^*]+)\\*\\*") to "$1",
            Regex("\\*([^*]+)\\*") to "$1",
            Regex("__([^_]+)__") to "$1",
            Regex("_([^_]+)_") to "$1",
            Regex("~~([^~]+)~~") to "$1",
            Regex("\\$\\$([^$]+)\\$\\$") to "$1",
            Regex("\\$([^$]+)\\$") to "$1",
        )
        for ((pattern, replacement) in replacements) {
            line = pattern.replace(line, replacement)
        }
        return line.trim()
    }

    private fun appendStripped(text: String, parts: MutableList<String>) {
        val stripped = stripInlineMarkdown(text)
        if (stripped.isNotEmpty()) parts.add(stripped)
    }

    private fun relativeLuminance(color: ColorComponents): Double {
        fun channel(value: Double): Double =
            if (value <= 0.03928) value / 12.92 else ((value + 0.055) / 1.055).pow(2.4)
        val red = channel(color.red)
        val green = channel(color.green)
        val blue = channel(color.blue)
        return 0.2126 * red + 0.7152 * green + 0.0722 * blue
    }
}
