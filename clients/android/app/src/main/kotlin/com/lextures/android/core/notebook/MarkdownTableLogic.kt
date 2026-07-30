package com.lextures.android.core.notebook

/** Column alignment for a GFM pipe table. */
enum class MarkdownTableAlign {
    Default,
    Left,
    Center,
    Right,
}

/**
 * Pure table detection / blank-line healing / cell split
 * (parity with web `normalize-markdown-tables`).
 */
object MarkdownTableLogic {
    private val bareSeparator = Regex("""^\|?[\s:|-]+$""")
    private val sepA = Regex("""^\|?(\s*:?-{3,}:?\s*\|)+\s*:?-{3,}:?\s*\|?$""")
    private val sepB = Regex("""^\|?(\s*:?-{3,}:?\s*\|?)+\s*$""")

    /** Collapse blank lines inside GitHub-flavored markdown tables (web FR-3 parity). */
    fun normalizeMarkdownTables(markdown: String): String {
        if (!markdown.contains("|")) return markdown
        val lines = markdown.replace("\r\n", "\n").split("\n")
        val out = mutableListOf<String>()
        var i = 0
        while (i < lines.size) {
            val line = lines[i]
            val sepIdx = nextNonEmptyIndex(lines, i + 1)
            val looksLikeHeader =
                isTableRowLine(line) &&
                    sepIdx < lines.size &&
                    isTableSeparatorLine(lines[sepIdx])
            if (!looksLikeHeader) {
                out.add(line)
                i++
                continue
            }
            val tableLines = mutableListOf(line)
            i++
            while (i < lines.size) {
                val L = lines[i]
                if (L.trim().isEmpty()) {
                    val next = nextNonEmptyIndex(lines, i + 1)
                    if (next < lines.size && isTableRelatedLine(lines[next])) {
                        i = next
                        continue
                    }
                    break
                }
                if (!isTableRelatedLine(L)) break
                tableLines.add(L)
                i++
            }
            if (tableLines.size >= 2 && isTableSeparatorLine(tableLines[1])) {
                out.addAll(tableLines)
            } else {
                out.add(line)
                if (tableLines.size > 1) out.addAll(tableLines.drop(1))
            }
        }
        return out.joinToString("\n")
    }

    /**
     * Try to parse a GFM table starting at [start].
     * Returns null when the candidate is not a valid table.
     */
    fun parseTable(
        lines: List<String>,
        start: Int,
    ): ParsedMarkdownTable? {
        if (start >= lines.size || !isTableRowLine(lines[start])) return null
        val sepIdx = nextNonEmptyIndex(lines, start + 1)
        if (sepIdx >= lines.size || !isTableSeparatorLine(lines[sepIdx])) return null

        val header = splitCells(lines[start])
        val align = parseAlignments(lines[sepIdx], header.size)
        val rows = mutableListOf<List<String>>()
        var i = sepIdx + 1
        while (i < lines.size) {
            val trimmed = lines[i].trim()
            if (trimmed.isEmpty()) break
            if (!isTableRowLine(lines[i])) break
            if (isTableSeparatorLine(lines[i])) break
            rows.add(padCells(splitCells(lines[i]), header.size))
            i++
        }
        return ParsedMarkdownTable(align = align, header = header, rows = rows, end = i)
    }

    fun isTableRowLine(line: String): Boolean {
        val t = line.trim()
        if (!t.contains("|")) return false
        if (bareSeparator.matches(t) && t.contains("-")) return false
        if (t.length >= 2 && t.substring(1, t.length - 1).contains("|")) return true
        return t.startsWith("|") && t.endsWith("|")
    }

    fun isTableSeparatorLine(line: String): Boolean {
        val t = line.trim()
        if (!t.contains("-")) return false
        return sepA.matches(t) || sepB.matches(t)
    }

    fun isTableRelatedLine(line: String): Boolean =
        isTableRowLine(line) || isTableSeparatorLine(line)

    fun splitCells(line: String): List<String> {
        var t = line.trim()
        if (t.startsWith("|")) t = t.drop(1)
        if (t.endsWith("|")) t = t.dropLast(1)
        return t.split("|").map { it.trim() }
    }

    fun parseAlignments(separator: String, columnCount: Int): List<MarkdownTableAlign> {
        val cells = splitCells(separator)
        val parsed = cells.map { cell ->
            val c = cell.trim()
            val left = c.startsWith(":")
            val right = c.endsWith(":")
            when {
                left && right -> MarkdownTableAlign.Center
                right -> MarkdownTableAlign.Right
                left -> MarkdownTableAlign.Left
                else -> MarkdownTableAlign.Default
            }
        }
        return if (parsed.size >= columnCount) {
            parsed.take(columnCount)
        } else {
            parsed + List(columnCount - parsed.size) { MarkdownTableAlign.Default }
        }
    }

    fun padCells(cells: List<String>, count: Int): List<String> =
        if (cells.size >= count) cells.take(count)
        else cells + List(count - cells.size) { "" }

    fun nextNonEmptyIndex(lines: List<String>, from: Int): Int {
        var j = from
        while (j < lines.size && lines[j].trim().isEmpty()) j++
        return j
    }

    fun serialize(align: List<MarkdownTableAlign>, header: List<String>, rows: List<List<String>>): String {
        fun row(cells: List<String>) = "| " + cells.joinToString(" | ") + " |"
        val sep = align.map {
            when (it) {
                MarkdownTableAlign.Left -> ":---"
                MarkdownTableAlign.Center -> ":---:"
                MarkdownTableAlign.Right -> "---:"
                MarkdownTableAlign.Default -> "---"
            }
        }
        return buildList {
            add(row(header))
            add("| " + sep.joinToString(" | ") + " |")
            addAll(rows.map(::row))
        }.joinToString("\n")
    }
}

data class ParsedMarkdownTable(
    val align: List<MarkdownTableAlign>,
    val header: List<String>,
    val rows: List<List<String>>,
    val end: Int,
)
