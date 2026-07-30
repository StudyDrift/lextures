package com.lextures.android.core.notebook

import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test
import java.io.File

class NotebookMarkdownTest {
    private val json = Json { ignoreUnknownKeys = true }

    @Test
    fun parsesCodeLanguageAndLexTool() {
        val code = NotebookMarkdown.parseBlocks("```python\nprint(1)\n```")
        val codeBlock = code.first() as NotebookBlock.Code
        assertEquals("python", codeBlock.language)
        assertEquals("print(1)", codeBlock.text)

        val tool = NotebookMarkdown.parseBlocks(
            """
            ```lex-tool
            {"instanceId":"abc-123","toolId":"inline_questions","v":1}
            ```
            """.trimIndent(),
        )
        val fence = tool.first() as NotebookBlock.ToolFence
        assertEquals("abc-123", fence.instanceId)
        assertEquals("inline_questions", fence.toolId)
        assertEquals(1, fence.version)
        assertTrue(tool.none { it is NotebookBlock.Code })
    }

    @Test
    fun parsesMathAndTaskList() {
        val math = NotebookMarkdown.parseBlocks("$$\\frac{a}{b}$$").first() as NotebookBlock.Math
        assertEquals("\\frac{a}{b}", math.latex)
        assertTrue(math.display)

        val tasks = NotebookMarkdown.parseBlocks("- [ ] Todo\n- [x] Done")
        assertEquals(2, tasks.size)
        assertEquals(NotebookBlock.TaskItem(checked = false, text = "Todo", depth = 0), tasks[0])
        assertEquals(NotebookBlock.TaskItem(checked = true, text = "Done", depth = 0), tasks[1])
    }

    @Test
    fun previewHidesLexToolJson() {
        val preview = NotebookMarkdown.previewText(
            """
            Hello

            ```lex-tool
            {"instanceId":"abc-123","toolId":"flashcards","v":1}
            ```
            """.trimIndent(),
        )
        assertFalse(preview.contains("abc-123"))
        assertFalse(preview.contains("instanceId"))
    }

    @Test
    fun sharedFixtureCorpusKindSequences() {
        val corpus = resolveCorpus()
        val root = json.decodeFromString(FixtureCorpus.serializer(), corpus.readText())
        assertTrue(root.fixtures.isNotEmpty())
        for (fixture in root.fixtures) {
            val actual = NotebookMarkdown.parseBlocks(fixture.markdown).map(NotebookMarkdown::fixtureKindName)
            assertEquals("fixture ${fixture.id}", fixture.kinds, actual)
        }
    }

    @Serializable
    private data class FixtureCorpus(val fixtures: List<Fixture>)

    @Serializable
    private data class Fixture(val id: String, val markdown: String, val kinds: List<String>)

    private fun resolveCorpus(): File {
        var dir: File? = File(System.getProperty("user.dir") ?: ".")
        repeat(8) {
            val current = dir ?: return@repeat
            val candidates = listOf(
                File(current, "clients/mobile/fixtures/markdown/corpus.json"),
                File(current, "../mobile/fixtures/markdown/corpus.json"),
                File(current, "../../mobile/fixtures/markdown/corpus.json"),
                File(current, "mobile/fixtures/markdown/corpus.json"),
            )
            candidates.firstOrNull { it.isFile }?.let { return it.canonicalFile }
            dir = current.parentFile
        }
        error("clients/mobile/fixtures/markdown/corpus.json not found from ${System.getProperty("user.dir")}")
    }
}
