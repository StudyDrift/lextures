package com.lextures.android.features.reader

import com.lextures.android.core.lms.CaptionRecord
import com.lextures.android.core.lms.ReadingPreferencesRow
import java.net.URI
import java.util.Locale

object ReaderLogic {
    data class VttCue(val start: Double, val end: Double, val text: String)

    val commonLocales = listOf(
        "en" to "English",
        "es" to "Spanish",
        "fr" to "French",
        "de" to "German",
        "ar" to "Arabic",
        "zh" to "Chinese",
        "ja" to "Japanese",
        "ko" to "Korean",
    )

    fun parseVtt(raw: String): List<VttCue> {
        val lines = raw.replace("\r\n", "\n").split("\n")
        val cues = mutableListOf<VttCue>()
        var index = 0
        while (index < lines.size) {
            val line = lines[index].trim()
            if (line.isEmpty() || line == "WEBVTT" || line.startsWith("NOTE")) {
                index++
                continue
            }
            if ("-->" in line) {
                val parts = line.split("-->")
                if (parts.size == 2) {
                    val start = parseVttTimestamp(parts[0].trim())
                    val end = parseVttTimestamp(parts[1].trim().substringBefore(" "))
                    if (start != null && end != null) {
                        index++
                        val textLines = mutableListOf<String>()
                        while (index < lines.size && lines[index].trim().isNotEmpty()) {
                            textLines += stripVttTags(lines[index])
                            index++
                        }
                        val text = textLines.joinToString(" ").trim()
                        if (text.isNotEmpty()) cues += VttCue(start, end, text)
                        continue
                    }
                }
            }
            index++
        }
        return cues
    }

    fun activeCue(time: Double, cues: List<VttCue>): VttCue? =
        cues.firstOrNull { time >= it.start && time < it.end }

    fun videoUrl(text: String): String? {
        val trimmed = text.trim()
        if (!trimmed.startsWith("http", ignoreCase = true)) return null
        val uri = runCatching { URI(trimmed) }.getOrNull() ?: return null
        val host = uri.host?.lowercase(Locale.US).orEmpty()
        if (host.contains("youtube.com") || host.contains("youtu.be") || host.contains("vimeo.com")) {
            return trimmed
        }
        val path = uri.path.lowercase(Locale.US)
        if (listOf(".mp4", ".mov", ".m3u8", ".webm").any { path.endsWith(it) }) {
            return trimmed
        }
        return null
    }

    fun storageObjectId(url: String): String? {
        val patterns = listOf(
            Regex("""/api/v1/files/([0-9a-fA-F-]{36})"""),
            Regex("""/files/([0-9a-fA-F-]{36})"""),
        )
        for (pattern in patterns) {
            pattern.find(URI(url).path)?.groupValues?.getOrNull(1)?.let { return it }
        }
        return null
    }

    fun localeLabel(code: String): String =
        commonLocales.firstOrNull { it.first == code.lowercase(Locale.US) }?.second ?: code.uppercase(Locale.US)

    fun readyCaptions(records: List<CaptionRecord>): List<CaptionRecord> =
        records.filter { it.status.equals("ready", ignoreCase = true) }

    fun defaultReadingPreferences(): ReadingPreferencesRow = ReadingPreferencesRow()

    fun dyslexiaFromFontFace(fontFace: String): Boolean = fontFace == "open-dyslexic"

    fun fontFaceFromDyslexia(enabled: Boolean, current: String): String =
        if (enabled) "open-dyslexic" else if (current == "open-dyslexic") "default" else current

    private fun parseVttTimestamp(value: String): Double? {
        val chunks = value.trim().split(":")
        if (chunks.size < 2) return null
        var seconds = 0.0
        if (chunks.size == 3) seconds += chunks[0].toDoubleOrNull()?.times(3600) ?: return null
        val minuteIndex = if (chunks.size == 3) 1 else 0
        val secondIndex = if (chunks.size == 3) 2 else 1
        val minutes = chunks[minuteIndex].toDoubleOrNull() ?: return null
        val secParts = chunks[secondIndex].split(".")
        val secs = secParts[0].toDoubleOrNull() ?: return null
        seconds += minutes * 60 + secs
        if (secParts.size > 1) {
            seconds += (secParts[1].take(3).toDoubleOrNull() ?: 0.0) / 1000.0
        }
        return seconds
    }

    private fun stripVttTags(line: String): String = line.replace(Regex("<[^>]+>"), "")
}