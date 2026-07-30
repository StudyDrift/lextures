package com.lextures.android.core.notebook

import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.jsonPrimitive
import kotlinx.serialization.json.put
import java.util.UUID

/** A task parsed from a ```task fenced block (parity with web `notebook-task-markdown`). */
data class ParsedNotebookTask(
    val id: String,
    val text: String,
    val checked: Boolean,
    val dueAt: String?,
)

/** One slash-command / toolbar insert action (parity with web `markdown-body-slash`). */
data class NotebookSlashCommand(
    val id: String,
    val label: String,
    val detail: String,
    val keywords: List<String>,
)

/** Renderable markdown block for the notebook / course reading view (CT.M1). */
sealed interface NotebookBlock {
    data class Heading(val level: Int, val text: String) : NotebookBlock
    data class Paragraph(val text: String) : NotebookBlock
    data class BulletItem(val text: String, val depth: Int = 0) : NotebookBlock
    data class OrderedItem(val number: String, val text: String, val depth: Int = 0) : NotebookBlock
    /** GFM task-list item (`- [ ]` / `- [x]`), read-only in reader contexts. */
    data class TaskItem(val checked: Boolean, val text: String, val depth: Int = 0) : NotebookBlock
    data class Quote(val text: String) : NotebookBlock
    data class Code(val text: String, val language: String? = null) : NotebookBlock
    data class Math(val latex: String, val display: Boolean = true) : NotebookBlock
    data class Table(
        val align: List<MarkdownTableAlign>,
        val header: List<String>,
        val rows: List<List<String>>,
    ) : NotebookBlock
    /** ```lex-tool pointer; raw JSON must never be shown to learners (CT.M3 hosts it). */
    data class ToolFence(val instanceId: String, val toolId: String, val version: Int) : NotebookBlock
    data object Divider : NotebookBlock
    data class TaskBlock(val task: ParsedNotebookTask) : NotebookBlock
    data class Image(val alt: String, val url: String) : NotebookBlock

    /** [index] is the drawing's ordinal among all drawings on the page (for write-back). */
    data class Drawing(val index: Int, val elementsJson: String) : NotebookBlock
}

/**
 * One editable block in the WYSIWYG notebook editor (parity with the web block editor:
 * blocks stay rendered while editing; markdown is only the storage format).
 */
data class NotebookEditBlock(
    val id: String = UUID.randomUUID().toString(),
    val kind: Kind,
    val text: String = "",
) {
    sealed interface Kind {
        data object Paragraph : Kind
        data class Heading(val level: Int) : Kind
        data object Bullet : Kind
        data object Ordered : Kind
        data object Quote : Kind
        data object Code : Kind
        data object Divider : Kind
        data class Task(val taskId: String, val checked: Boolean, val dueAt: String?) : Kind
        data class Image(val alt: String, val url: String) : Kind
        data class Drawing(val elementsJson: String) : Kind
    }

    /** Whether the block carries user-editable text (false for divider / image / drawing). */
    val isTextual: Boolean
        get() = kind !is Kind.Divider && kind !is Kind.Image && kind !is Kind.Drawing
}

object NotebookMarkdown {
    private val taskBlockRegex = Regex("```task[ \\t]*\\n([\\s\\S]*?)```")
    private val orderedItemRegex = Regex("^(\\d+)[.)] (.*)$")
    private val taskListRegex = Regex("^[-*]\\s+\\[([ xX])]\\s+(.*)$")
    private val imageRegex = Regex("^!\\[([^\\]]*)]\\(([^)]+)\\)$")
    private val json = Json { ignoreUnknownKeys = true }

    fun newTaskId(): String = UUID.randomUUID().toString()

    fun taskMetaLine(id: String, checked: Boolean, dueAt: String?): String {
        val obj = buildJsonObject {
            put("id", id)
            put("checked", checked)
            if (dueAt != null) put("dueAt", dueAt) else put("dueAt", JsonNull)
        }
        return json.encodeToString(JsonObject.serializer(), obj)
    }

    private fun parseTaskMeta(line: String): Triple<String, Boolean, String?>? {
        val obj = runCatching { json.parseToJsonElement(line) as? JsonObject }.getOrNull() ?: return null
        val id = (obj["id"] as? JsonPrimitive)?.contentOrNull?.takeIf { it.isNotEmpty() } ?: return null
        val checked = (obj["checked"] as? JsonPrimitive)?.contentOrNull == "true"
        val dueAt = (obj["dueAt"] as? JsonPrimitive)?.takeIf { it.isString }?.contentOrNull
        return Triple(id, checked, dueAt)
    }

    private fun parseTaskInner(inner: String): ParsedNotebookTask? {
        val lines = inner.split("\n")
        val (id, checked, dueAt) = parseTaskMeta(lines.firstOrNull().orEmpty()) ?: return null
        val text = lines.drop(1).joinToString("\n").trim()
        return ParsedNotebookTask(id = id, text = text, checked = checked, dueAt = dueAt)
    }

    fun parseTasks(contentMd: String): List<ParsedNotebookTask> =
        taskBlockRegex.findAll(contentMd).mapNotNull { parseTaskInner(it.groupValues[1]) }.toList()

    private fun rewriteTask(
        contentMd: String,
        taskId: String,
        transform: (ParsedNotebookTask) -> Pair<Boolean, String?>,
    ): String = taskBlockRegex.replace(contentMd) { match ->
        val inner = match.groupValues[1]
        val task = parseTaskInner(inner)
        if (task == null || task.id != taskId) {
            match.value
        } else {
            val (checked, dueAt) = transform(task)
            val body = inner.split("\n").drop(1).joinToString("\n").trim()
            "```task\n${taskMetaLine(task.id, checked, dueAt)}\n$body\n```"
        }
    }

    fun setTaskChecked(contentMd: String, taskId: String, checked: Boolean): String =
        rewriteTask(contentMd, taskId) { checked to it.dueAt }

    fun setTaskDueAt(contentMd: String, taskId: String, dueAt: String?): String =
        rewriteTask(contentMd, taskId) { it.checked to dueAt }

    /** Kind labels used by the shared golden-fixture corpus (`clients/mobile/fixtures/markdown`). */
    fun fixtureKindName(block: NotebookBlock): String = when (block) {
        is NotebookBlock.Heading -> "heading"
        is NotebookBlock.Paragraph -> "paragraph"
        is NotebookBlock.BulletItem -> "bullet"
        is NotebookBlock.OrderedItem -> "ordered"
        is NotebookBlock.TaskItem -> "taskItem"
        is NotebookBlock.Quote -> "quote"
        is NotebookBlock.Code -> "code"
        is NotebookBlock.Math -> "math"
        is NotebookBlock.Table -> "table"
        is NotebookBlock.ToolFence -> "toolFence"
        NotebookBlock.Divider -> "divider"
        is NotebookBlock.TaskBlock -> "task"
        is NotebookBlock.Image -> "image"
        is NotebookBlock.Drawing -> "drawing"
    }

    // Block parsing (reading view)

    fun parseBlocks(contentMd: String): List<NotebookBlock> {
        val blocks = mutableListOf<NotebookBlock>()
        val paragraph = mutableListOf<String>()
        val quote = mutableListOf<String>()

        fun flushParagraph() {
            if (paragraph.isEmpty()) return
            val text = paragraph.joinToString("\n")
            paragraph.clear()
            parseStandaloneMath(text)?.let { blocks.add(it) } ?: blocks.add(NotebookBlock.Paragraph(text))
        }
        fun flushQuote() {
            if (quote.isNotEmpty()) {
                blocks.add(NotebookBlock.Quote(quote.joinToString("\n")))
                quote.clear()
            }
        }
        fun flushAll() {
            flushParagraph()
            flushQuote()
        }

        val lines = MarkdownTableLogic.normalizeMarkdownTables(contentMd.replace("\r\n", "\n")).split("\n")
        var i = 0
        var drawingIndex = 0
        while (i < lines.size) {
            val trimmed = lines[i].trim()
            val depth = listDepth(lines[i]).coerceAtMost(3)
            val table = MarkdownTableLogic.parseTable(lines, i)
            when {
                table != null -> {
                    flushAll()
                    blocks.add(NotebookBlock.Table(align = table.align, header = table.header, rows = table.rows))
                    i = table.end
                    continue
                }
                trimmed.startsWith("```drawing") -> {
                    flushAll()
                    val inner = mutableListOf<String>()
                    i++
                    while (i < lines.size && lines[i].trim() != "```") {
                        inner.add(lines[i])
                        i++
                    }
                    blocks.add(NotebookBlock.Drawing(index = drawingIndex, elementsJson = inner.joinToString("\n").trim()))
                    drawingIndex++
                }
                trimmed.startsWith("```task") -> {
                    flushAll()
                    val inner = mutableListOf<String>()
                    i++
                    while (i < lines.size && lines[i].trim() != "```") {
                        inner.add(lines[i])
                        i++
                    }
                    parseTaskInner(inner.joinToString("\n"))?.let { blocks.add(NotebookBlock.TaskBlock(it)) }
                }
                trimmed.startsWith("```") -> {
                    flushAll()
                    val language = fenceLanguage(trimmed)
                    val inner = mutableListOf<String>()
                    i++
                    while (i < lines.size && !lines[i].trim().startsWith("```")) {
                        inner.add(lines[i])
                        i++
                    }
                    val source = inner.joinToString("\n")
                    val tool = if (language == "lex-tool") parseLexToolFence(source) else null
                    if (tool != null) {
                        blocks.add(tool)
                    } else {
                        blocks.add(NotebookBlock.Code(text = source, language = language))
                    }
                }
                parseHeading(trimmed) != null -> {
                    flushAll()
                    blocks.add(parseHeading(trimmed)!!)
                }
                trimmed == "---" || trimmed == "***" || trimmed == "___" -> {
                    flushAll()
                    blocks.add(NotebookBlock.Divider)
                }
                imageRegex.matches(trimmed) -> {
                    flushAll()
                    val match = imageRegex.find(trimmed)!!
                    blocks.add(NotebookBlock.Image(alt = match.groupValues[1], url = match.groupValues[2]))
                }
                taskListRegex.matches(trimmed) -> {
                    flushAll()
                    val match = taskListRegex.find(trimmed)!!
                    blocks.add(
                        NotebookBlock.TaskItem(
                            checked = match.groupValues[1].equals("x", ignoreCase = true),
                            text = match.groupValues[2],
                            depth = depth,
                        ),
                    )
                }
                trimmed.startsWith("- ") || trimmed.startsWith("* ") -> {
                    flushAll()
                    blocks.add(NotebookBlock.BulletItem(text = trimmed.drop(2), depth = depth))
                }
                orderedItemRegex.matches(trimmed) -> {
                    flushAll()
                    val match = orderedItemRegex.find(trimmed)!!
                    blocks.add(
                        NotebookBlock.OrderedItem(
                            number = match.groupValues[1],
                            text = match.groupValues[2],
                            depth = depth,
                        ),
                    )
                }
                trimmed.startsWith(">") -> {
                    flushParagraph()
                    quote.add(trimmed.drop(1).trim())
                }
                trimmed.isEmpty() -> flushAll()
                else -> {
                    flushQuote()
                    paragraph.add(trimmed)
                }
            }
            i++
        }
        flushAll()
        return blocks
    }

    private fun fenceLanguage(opener: String): String? {
        val rest = opener.removePrefix("```").trim()
        if (rest.isEmpty()) return null
        return rest.split(Regex("\\s+")).firstOrNull()?.takeIf { it.isNotEmpty() }
    }

    private fun parseLexToolFence(source: String): NotebookBlock.ToolFence? {
        val obj = runCatching {
            json.parseToJsonElement(source.trim()) as? JsonObject
        }.getOrNull() ?: return null
        val instanceId = (obj["instanceId"] as? JsonPrimitive)?.contentOrNull?.trim().orEmpty()
        val toolId = (obj["toolId"] as? JsonPrimitive)?.contentOrNull?.trim().orEmpty()
        val version = when (val v = obj["v"]) {
            is JsonPrimitive -> v.contentOrNull?.toIntOrNull() ?: v.content.toIntOrNull()
            else -> null
        }
        if (instanceId.isEmpty() || toolId.isEmpty() || version != 1) return null
        return NotebookBlock.ToolFence(instanceId = instanceId, toolId = toolId, version = 1)
    }

    private fun parseStandaloneMath(text: String): NotebookBlock.Math? {
        val t = text.trim()
        if (!t.startsWith("$$") || !t.endsWith("$$") || t.length < 4) return null
        val inner = t.removePrefix("$$").removeSuffix("$$").trim()
        if (inner.isEmpty() || inner.contains("$$")) return null
        return NotebookBlock.Math(latex = inner, display = true)
    }

    private fun listDepth(line: String): Int {
        var spaces = 0
        for (ch in line) {
            when (ch) {
                ' ' -> spaces++
                '\t' -> spaces += 2
                else -> break
            }
        }
        return (spaces / 2).coerceAtMost(3)
    }

    private fun parseHeading(line: String): NotebookBlock.Heading? {
        if (!line.startsWith("#")) return null
        val hashes = line.takeWhile { it == '#' }
        if (hashes.length > 6) return null
        val rest = line.drop(hashes.length)
        if (!rest.startsWith(" ")) return null
        return NotebookBlock.Heading(level = hashes.length, text = rest.trim())
    }

    // Edit blocks (WYSIWYG editor, parity with web block editor)

    fun editBlocks(contentMd: String): List<NotebookEditBlock> {
        val out = mutableListOf<NotebookEditBlock>()
        for (block in parseBlocks(contentMd)) {
            when (block) {
                is NotebookBlock.Heading ->
                    out.add(NotebookEditBlock(kind = NotebookEditBlock.Kind.Heading(block.level), text = block.text))
                is NotebookBlock.Paragraph ->
                    block.text.split("\n").forEach {
                        out.add(NotebookEditBlock(kind = NotebookEditBlock.Kind.Paragraph, text = it))
                    }
                is NotebookBlock.BulletItem ->
                    out.add(NotebookEditBlock(kind = NotebookEditBlock.Kind.Bullet, text = block.text))
                is NotebookBlock.OrderedItem ->
                    out.add(NotebookEditBlock(kind = NotebookEditBlock.Kind.Ordered, text = block.text))
                is NotebookBlock.TaskItem -> {
                    val mark = if (block.checked) "x" else " "
                    out.add(NotebookEditBlock(kind = NotebookEditBlock.Kind.Bullet, text = "[$mark] ${block.text}"))
                }
                is NotebookBlock.Quote ->
                    block.text.split("\n").forEach {
                        out.add(NotebookEditBlock(kind = NotebookEditBlock.Kind.Quote, text = it))
                    }
                is NotebookBlock.Code ->
                    out.add(NotebookEditBlock(kind = NotebookEditBlock.Kind.Code, text = block.text))
                is NotebookBlock.Math ->
                    out.add(
                        NotebookEditBlock(
                            kind = NotebookEditBlock.Kind.Paragraph,
                            text = if (block.display) "$$${block.latex}$$" else "$${block.latex}$",
                        ),
                    )
                is NotebookBlock.Table ->
                    MarkdownTableLogic.serialize(block.align, block.header, block.rows)
                        .split("\n")
                        .forEach { out.add(NotebookEditBlock(kind = NotebookEditBlock.Kind.Paragraph, text = it)) }
                is NotebookBlock.ToolFence ->
                    out.add(
                        NotebookEditBlock(
                            kind = NotebookEditBlock.Kind.Code,
                            text = """{"instanceId":"${block.instanceId}","toolId":"${block.toolId}","v":${block.version}}""",
                        ),
                    )
                NotebookBlock.Divider ->
                    out.add(NotebookEditBlock(kind = NotebookEditBlock.Kind.Divider))
                is NotebookBlock.TaskBlock ->
                    out.add(
                        NotebookEditBlock(
                            kind = NotebookEditBlock.Kind.Task(block.task.id, block.task.checked, block.task.dueAt),
                            text = block.task.text,
                        ),
                    )
                is NotebookBlock.Image ->
                    out.add(NotebookEditBlock(kind = NotebookEditBlock.Kind.Image(block.alt, block.url)))
                is NotebookBlock.Drawing ->
                    out.add(NotebookEditBlock(kind = NotebookEditBlock.Kind.Drawing(block.elementsJson)))
            }
        }
        if (out.isEmpty()) out.add(NotebookEditBlock(kind = NotebookEditBlock.Kind.Paragraph))
        return out
    }

    /** Consecutive items of the same list/quote kind join with one newline, not a blank line. */
    private fun sameListRun(a: NotebookEditBlock.Kind, b: NotebookEditBlock.Kind?): Boolean = when {
        a is NotebookEditBlock.Kind.Bullet && b is NotebookEditBlock.Kind.Bullet -> true
        a is NotebookEditBlock.Kind.Ordered && b is NotebookEditBlock.Kind.Ordered -> true
        a is NotebookEditBlock.Kind.Quote && b is NotebookEditBlock.Kind.Quote -> true
        else -> false
    }

    fun markdownFromBlocks(blocks: List<NotebookEditBlock>): String {
        val out = StringBuilder()
        var previous: NotebookEditBlock.Kind? = null
        var orderedRun = 0

        for (block in blocks) {
            val chunk = when (val kind = block.kind) {
                NotebookEditBlock.Kind.Paragraph -> {
                    if (block.text.isBlank()) continue
                    block.text
                }
                is NotebookEditBlock.Kind.Heading -> "#".repeat(kind.level.coerceIn(1, 6)) + " " + block.text
                NotebookEditBlock.Kind.Bullet -> "- ${block.text}"
                NotebookEditBlock.Kind.Ordered -> {
                    orderedRun = if (previous is NotebookEditBlock.Kind.Ordered) orderedRun + 1 else 1
                    "$orderedRun. ${block.text}"
                }
                NotebookEditBlock.Kind.Quote -> "> ${block.text}"
                NotebookEditBlock.Kind.Code -> "```\n${block.text}\n```"
                NotebookEditBlock.Kind.Divider -> "---"
                is NotebookEditBlock.Kind.Task ->
                    "```task\n${taskMetaLine(kind.taskId, kind.checked, kind.dueAt)}\n${block.text}\n```"
                is NotebookEditBlock.Kind.Image -> "![${kind.alt}](${kind.url})"
                is NotebookEditBlock.Kind.Drawing -> "```drawing\n${kind.elementsJson}\n```"
            }
            when {
                out.isEmpty() -> out.append(chunk)
                sameListRun(block.kind, previous) -> out.append("\n").append(chunk)
                else -> out.append("\n\n").append(chunk)
            }
            previous = block.kind
        }
        return out.toString()
    }

    /** Replace the elements JSON of the page's Nth drawing fence (0-based, document order). */
    fun replaceDrawing(contentMd: String, index: Int, elementsJson: String): String {
        val out = mutableListOf<String>()
        var current = -1
        val lines = contentMd.replace("\r\n", "\n").split("\n")
        var i = 0
        while (i < lines.size) {
            val trimmed = lines[i].trim()
            if (trimmed.startsWith("```drawing")) {
                current++
                val inner = mutableListOf<String>()
                i++
                while (i < lines.size && lines[i].trim() != "```") {
                    inner.add(lines[i])
                    i++
                }
                i++
                val body = if (current == index) elementsJson else inner.joinToString("\n").trim()
                out.add("```drawing\n$body\n```")
            } else {
                out.add(lines[i])
                i++
            }
        }
        return out.joinToString("\n")
    }

    // Slash commands

    val slashCommands: List<NotebookSlashCommand> = listOf(
        NotebookSlashCommand("heading1", "Heading 1", "Large section heading", listOf("h1", "title", "heading")),
        NotebookSlashCommand("heading2", "Heading 2", "Medium section heading", listOf("h2", "heading")),
        NotebookSlashCommand("heading3", "Heading 3", "Small section heading", listOf("h3", "heading")),
        NotebookSlashCommand("task", "Task", "Checkbox task with optional due date", listOf("task", "todo", "checkbox", "checklist")),
        NotebookSlashCommand("drawing", "Drawing", "Insert a whiteboard to draw on", listOf("drawing", "whiteboard", "sketch", "draw", "canvas")),
        NotebookSlashCommand("bulletList", "Bullet list", "Unordered list", listOf("ul", "list", "bullets")),
        NotebookSlashCommand("orderedList", "Numbered list", "Ordered list", listOf("ol", "list", "numbers")),
        NotebookSlashCommand("blockquote", "Quote", "Indented quotation", listOf("quote", "blockquote")),
        NotebookSlashCommand("codeBlock", "Code", "Code block", listOf("code", "pre", "snippet")),
        NotebookSlashCommand("horizontalRule", "Divider", "Horizontal line", listOf("hr", "divider", "line", "rule")),
    )

    fun filterCommands(query: String): List<NotebookSlashCommand> {
        val q = query.trim().lowercase()
        if (q.isEmpty()) return slashCommands
        return slashCommands.filter { cmd ->
            val cmdId = cmd.id.lowercase()
            cmdId == q || cmdId.startsWith(q) || q.startsWith(cmdId) ||
                cmd.label.lowercase().contains(q) || cmd.detail.lowercase().contains(q) ||
                cmd.keywords.any { kw ->
                    kw == q || (kw.length >= 2 && q.length >= 2 && (kw.startsWith(q) || q.startsWith(kw)))
                }
        }
    }

    /** Human-readable preview: strips fences and task meta lines so cards never show raw JSON. */
    fun previewText(contentMd: String): String =
        parseBlocks(contentMd).mapNotNull { block ->
            when (block) {
                is NotebookBlock.Heading -> block.text
                is NotebookBlock.Paragraph -> block.text
                is NotebookBlock.BulletItem -> block.text
                is NotebookBlock.OrderedItem -> block.text
                is NotebookBlock.TaskItem -> block.text
                is NotebookBlock.Quote -> block.text
                is NotebookBlock.Code -> block.text
                is NotebookBlock.Math -> block.latex
                is NotebookBlock.Table ->
                    (listOf(block.header.joinToString(" ")) + block.rows.take(3).map { it.joinToString(" ") })
                        .joinToString(" ")
                is NotebookBlock.ToolFence -> block.toolId
                is NotebookBlock.TaskBlock -> block.task.text
                is NotebookBlock.Image -> block.alt
                is NotebookBlock.Drawing -> "Drawing"
                NotebookBlock.Divider -> null
            }.takeIf { !it.isNullOrBlank() }
        }.joinToString(" · ").trim()
}
