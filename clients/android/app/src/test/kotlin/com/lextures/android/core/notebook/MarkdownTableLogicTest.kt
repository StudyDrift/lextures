package com.lextures.android.core.notebook

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class MarkdownTableLogicTest {
    @Test
    fun healsBlankLinesInsideTable() {
        val input = """
            | A | B |

            | --- | --- |

            | 1 | 2 |

            After
        """.trimIndent()
        val healed = MarkdownTableLogic.normalizeMarkdownTables(input)
        assertFalse(healed.contains("|\n\n|"))
        assertTrue(healed.contains("| A | B |"))
        assertTrue(healed.contains("| --- | --- |"))
        assertTrue(healed.contains("| 1 | 2 |"))
        assertTrue(healed.contains("After"))
    }

    @Test
    fun leavesPipeProseAlone() {
        val input = "Use | as OR in boolean logic."
        assertEquals(input, MarkdownTableLogic.normalizeMarkdownTables(input))
    }

    @Test
    fun parsesAlignment() {
        val lines = listOf(
            "| Left | Center | Right |",
            "| :--- | :---: | ---: |",
            "| a | b | c |",
        )
        val table = MarkdownTableLogic.parseTable(lines, 0)!!
        assertEquals(listOf("Left", "Center", "Right"), table.header)
        assertEquals(
            listOf(MarkdownTableAlign.Left, MarkdownTableAlign.Center, MarkdownTableAlign.Right),
            table.align,
        )
        assertEquals(listOf(listOf("a", "b", "c")), table.rows)
    }
}
