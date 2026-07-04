package com.lextures.android.features.reader

import com.lextures.android.core.lms.CaptionRecord
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class ReaderLogicTest {
    @Test
    fun parseVtt_extractsCueText() {
        val raw = """
            WEBVTT

            00:00:01.000 --> 00:00:03.000
            Hello <c>world</c>.

            00:00:04.000 --> 00:00:06.500
            Second line.
        """.trimIndent()
        val cues = ReaderLogic.parseVtt(raw)
        assertEquals(2, cues.size)
        assertEquals("Hello world.", cues[0].text)
        assertEquals(4.0, cues[1].start, 0.01)
    }

    @Test
    fun activeCue_selectsCurrentWindow() {
        val cues = listOf(
            ReaderLogic.VttCue(0.0, 2.0, "A"),
            ReaderLogic.VttCue(2.0, 5.0, "B"),
        )
        assertEquals("A", ReaderLogic.activeCue(1.5, cues)?.text)
        assertEquals("B", ReaderLogic.activeCue(3.0, cues)?.text)
        assertNull(ReaderLogic.activeCue(6.0, cues))
    }

    @Test
    fun storageObjectId_parsesApiFilePath() {
        val id = ReaderLogic.storageObjectId("https://api.example.com/api/v1/files/550e8400-e29b-41d4-a716-446655440000/content")
        assertEquals("550e8400-e29b-41d4-a716-446655440000", id)
    }

    @Test
    fun readyCaptions_filtersNonReady() {
        val records = listOf(
            CaptionRecord(id = "1", lang = "en", status = "ready"),
            CaptionRecord(id = "2", lang = "es", status = "processing"),
        )
        assertEquals(1, ReaderLogic.readyCaptions(records).size)
    }

    @Test
    fun videoUrl_detectsDirectMp4() {
        val url = ReaderLogic.videoUrl("https://api.example.com/api/v1/files/abc/content.mp4")
        assertEquals("https://api.example.com/api/v1/files/abc/content.mp4", url)
    }

    @Test
    fun videoUrl_ignoresPlainText() {
        assertNull(ReaderLogic.videoUrl("This is a normal paragraph."))
    }

    @Test
    fun dyslexiaFontFaceMapping() {
        assertTrue(ReaderLogic.dyslexiaFromFontFace("open-dyslexic"))
        assertEquals("open-dyslexic", ReaderLogic.fontFaceFromDyslexia(true, "default"))
        assertEquals("default", ReaderLogic.fontFaceFromDyslexia(false, "open-dyslexic"))
    }
}