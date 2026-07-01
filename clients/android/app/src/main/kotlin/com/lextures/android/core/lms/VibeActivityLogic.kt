package com.lextures.android.core.lms

import java.net.URLEncoder

sealed interface VibeActivityBlockKind {
    data class Heading(val level: Int, val text: String) : VibeActivityBlockKind
    data class Paragraph(val text: String) : VibeActivityBlockKind
    data class BulletList(val items: List<String>) : VibeActivityBlockKind
    data class OrderedList(val items: List<String>) : VibeActivityBlockKind
    data class Reveal(val trigger: String, val body: String) : VibeActivityBlockKind
    data class CheckButton(val label: String, val feedback: String?) : VibeActivityBlockKind
    data class FreeResponse(val prompt: String, val placeholder: String?) : VibeActivityBlockKind
    data class Unsupported(val message: String) : VibeActivityBlockKind
    data object Divider : VibeActivityBlockKind
}

data class VibeActivityBlock(
    val id: Int,
    val kind: VibeActivityBlockKind,
)

data class VibeActivityDocument(
    val blocks: List<VibeActivityBlock>,
    val requiresWebFallback: Boolean,
)

object VibeActivityLogic {
    private val detailsRegex = Regex("""<details[^>]*>([\s\S]*?)</details>""", RegexOption.IGNORE_CASE)
    private val summaryRegex = Regex("""<summary[^>]*>([\s\S]*?)</summary>""", RegexOption.IGNORE_CASE)
    private val buttonRevealRegex = Regex(
        """<button[^>]*onclick\s*=\s*["'][^"']*getElementById\(['"]([^'"]+)['"]\)[^"']*["'][^>]*>([\s\S]*?)</button>""",
        RegexOption.IGNORE_CASE,
    )
    private val blockTagRegex = Regex(
        """(?i)<(h[1-6]|p|ul|ol|hr|textarea|input|button|iframe|canvas|video|div)([^>]*)>([\s\S]*?)</\1>|<hr\s*/?>""",
    )
    private val listItemRegex = Regex("""<li[^>]*>([\s\S]*?)</li>""", RegexOption.IGNORE_CASE)
    private val bodyRegex = Regex("""<body[^>]*>([\s\S]*?)</body>""", RegexOption.IGNORE_CASE)
    private val hardUnsupportedRegex = Regex("""(?i)<(iframe|canvas|object|embed|applet)[^>]*>""")

    fun webPath(courseCode: String, itemId: String): String {
        val code = URLEncoder.encode(courseCode, "UTF-8").replace("+", "%20")
        val id = URLEncoder.encode(itemId, "UTF-8").replace("+", "%20")
        return "/courses/$code/modules/vibe-activity/$id"
    }

    fun parse(html: String?): VibeActivityDocument {
        val trimmed = html?.trim().orEmpty()
        if (trimmed.isEmpty()) {
            return VibeActivityDocument(
                blocks = listOf(VibeActivityBlock(0, VibeActivityBlockKind.Paragraph("Empty activity. The instructor has not added content yet."))),
                requiresWebFallback = false,
            )
        }

        val body = extractBody(trimmed)
        val stripped = stripNoise(body)
        if (hardUnsupportedRegex.containsMatchIn(stripped) && !hasReadableContent(stripped)) {
            return VibeActivityDocument(
                blocks = listOf(
                    VibeActivityBlock(
                        0,
                        VibeActivityBlockKind.Unsupported("This activity uses features that need a larger screen."),
                    ),
                ),
                requiresWebFallback = true,
            )
        }

        val blocks = mutableListOf<VibeActivityBlock>()
        var cursor = 0
        var working = stripped

        while (true) {
            val match = detailsRegex.find(working) ?: break
            blocks.addAll(parseSimpleBlocks(working.substring(0, match.range.first), cursor))
            cursor = blocks.size

            val inner = match.groupValues.getOrElse(1) { "" }
            val summary = summaryRegex.find(inner)
            val trigger = htmlToPlainText(summary?.groupValues?.getOrElse(1) { "" }.orEmpty().ifEmpty { "Reveal" })
            val bodyHtml = inner.replace(summaryRegex, "")
            val bodyText = htmlToPlainText(bodyHtml)
            if (trigger.isNotEmpty() || bodyText.isNotEmpty()) {
                blocks.add(VibeActivityBlock(cursor, VibeActivityBlockKind.Reveal(trigger, bodyText)))
                cursor += 1
            }
            working = working.substring(match.range.last + 1)
        }

        while (true) {
            val buttonMatch = buttonRevealRegex.find(working) ?: break
            blocks.addAll(parseSimpleBlocks(working.substring(0, buttonMatch.range.first), cursor))
            cursor = blocks.size

            val targetId = buttonMatch.groupValues.getOrElse(1) { "" }
            val label = htmlToPlainText(buttonMatch.groupValues.getOrElse(2) { "" }.ifEmpty { "Reveal" })
            val targetRegex = Regex("""<([a-z]+)[^>]*id=["']${Regex.escape(targetId)}["'][^>]*>([\s\S]*?)</\1>""", RegexOption.IGNORE_CASE)
            val targetMatch = targetRegex.find(working)
            val hiddenBody = htmlToPlainText(targetMatch?.groupValues?.getOrElse(2) { "" }.orEmpty())
            blocks.add(VibeActivityBlock(cursor, VibeActivityBlockKind.Reveal(label, hiddenBody)))
            cursor += 1

            working = buildString {
                append(working.substring(0, buttonMatch.range.first))
                append(working.substring(buttonMatch.range.last + 1))
            }.let { next ->
                targetMatch?.let { m ->
                    next.substring(0, m.range.first) + next.substring(m.range.last + 1)
                } ?: next
            }
        }

        blocks.addAll(parseSimpleBlocks(working, cursor))

        val requiresFallback = blocks.any { it.kind is VibeActivityBlockKind.Unsupported }
        if (blocks.isEmpty()) {
            return VibeActivityDocument(
                blocks = listOf(
                    VibeActivityBlock(
                        0,
                        VibeActivityBlockKind.Unsupported("This activity uses features that need a larger screen."),
                    ),
                ),
                requiresWebFallback = true,
            )
        }
        return VibeActivityDocument(blocks = blocks, requiresWebFallback = requiresFallback)
    }

    private fun parseSimpleBlocks(html: String, startId: Int): List<VibeActivityBlock> {
        if (html.trim().isEmpty()) return emptyList()
        val blocks = mutableListOf<VibeActivityBlock>()
        var id = startId
        var index = 0

        fun flushText(text: String) {
            val plain = htmlToPlainText(text)
            if (plain.isNotEmpty()) {
                blocks.add(VibeActivityBlock(id, VibeActivityBlockKind.Paragraph(plain)))
                id += 1
            }
        }

        for (match in blockTagRegex.findAll(html)) {
            if (match.range.first > index) {
                flushText(html.substring(index, match.range.first))
            }

            val tag = match.groupValues.getOrElse(1) { "" }.lowercase()
            val attrs = match.groupValues.getOrElse(2) { "" }
            val inner = match.groupValues.getOrElse(3) { "" }

            when (tag) {
                "h1", "h2", "h3", "h4", "h5", "h6" -> {
                    val level = tag.drop(1).toIntOrNull() ?: 2
                    blocks.add(VibeActivityBlock(id, VibeActivityBlockKind.Heading(level, htmlToPlainText(inner))))
                    id += 1
                }
                "p" -> {
                    blocks.add(VibeActivityBlock(id, VibeActivityBlockKind.Paragraph(htmlToPlainText(inner))))
                    id += 1
                }
                "ul" -> {
                    val items = listItems(inner)
                    if (items.isEmpty()) flushText(inner) else {
                        blocks.add(VibeActivityBlock(id, VibeActivityBlockKind.BulletList(items)))
                        id += 1
                    }
                }
                "ol" -> {
                    val items = listItems(inner)
                    if (items.isEmpty()) flushText(inner) else {
                        blocks.add(VibeActivityBlock(id, VibeActivityBlockKind.OrderedList(items)))
                        id += 1
                    }
                }
                "hr", "" -> {
                    blocks.add(VibeActivityBlock(id, VibeActivityBlockKind.Divider))
                    id += 1
                }
                "textarea" -> {
                    blocks.add(
                        VibeActivityBlock(
                            id,
                            VibeActivityBlockKind.FreeResponse(
                                htmlToPlainText(inner),
                                attributeValue("placeholder", attrs),
                            ),
                        ),
                    )
                    id += 1
                }
                "input" -> {
                    val type = attributeValue("type", attrs)?.lowercase().orEmpty().ifEmpty { "text" }
                    if (type == "text") {
                        blocks.add(
                            VibeActivityBlock(
                                id,
                                VibeActivityBlockKind.FreeResponse("", attributeValue("placeholder", attrs)),
                            ),
                        )
                        id += 1
                    } else {
                        blocks.add(VibeActivityBlock(id, VibeActivityBlockKind.Unsupported("This input type works best on the web.")))
                        id += 1
                    }
                }
                "button" -> {
                    val label = htmlToPlainText(inner)
                    if (label.isNotEmpty()) {
                        blocks.add(VibeActivityBlock(id, VibeActivityBlockKind.CheckButton(label, null)))
                        id += 1
                    }
                }
                "iframe", "canvas", "video" -> {
                    blocks.add(VibeActivityBlock(id, VibeActivityBlockKind.Unsupported("Embedded media works best on the web.")))
                    id += 1
                }
                "div" -> {
                    val childBlocks = parseSimpleBlocks(inner, id)
                    if (childBlocks.isEmpty()) {
                        val plain = htmlToPlainText(inner)
                        if (plain.isNotEmpty()) {
                            blocks.add(VibeActivityBlock(id, VibeActivityBlockKind.Paragraph(plain)))
                            id += 1
                        }
                    } else {
                        blocks.addAll(childBlocks)
                        id = (blocks.lastOrNull()?.id ?: id) + 1
                    }
                }
                else -> flushText(inner)
            }
            index = match.range.last + 1
        }

        if (index < html.length) {
            flushText(html.substring(index))
        }
        if (blocks.isEmpty()) {
            val plain = htmlToPlainText(html)
            if (plain.isNotEmpty()) {
                blocks.add(VibeActivityBlock(startId, VibeActivityBlockKind.Paragraph(plain)))
            }
        }
        return blocks
    }

    private fun listItems(html: String): List<String> =
        listItemRegex.findAll(html).mapNotNull { match ->
            htmlToPlainText(match.groupValues.getOrElse(1) { "" }).takeIf { it.isNotEmpty() }
        }.toList()

    private fun extractBody(html: String): String =
        bodyRegex.find(html)?.groupValues?.getOrElse(1) { html } ?: html

    private fun stripNoise(html: String): String {
        var out = html
        for (pattern in listOf("""<script[\s\S]*?</script>""", """<style[\s\S]*?</style>""", """<link[^>]*>""")) {
            out = out.replace(Regex(pattern, RegexOption.IGNORE_CASE), "")
        }
        return out
    }

    private fun hasReadableContent(html: String): Boolean = htmlToPlainText(html).isNotEmpty()

    fun htmlToPlainText(html: String): String {
        var text = html
        text = text.replace(Regex("""<br\s*/?>""", RegexOption.IGNORE_CASE), "\n")
        text = text.replace(Regex("""<[^>]+>"""), " ")
        text = decodeEntities(text)
        text = text.replace(Regex("""\s+"""), " ")
        return text.trim()
    }

    private fun decodeEntities(value: String): String =
        value
            .replace("&nbsp;", " ")
            .replace("&amp;", "&")
            .replace("&lt;", "<")
            .replace("&gt;", ">")
            .replace("&quot;", "\"")
            .replace("&#39;", "'")

    private fun attributeValue(name: String, attrs: String): String? {
        val regex = Regex("""$name\s*=\s*["']([^"']*)["']""", RegexOption.IGNORE_CASE)
        return regex.find(attrs)?.groupValues?.getOrElse(1) { null }
    }
}