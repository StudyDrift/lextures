package com.lextures.android.core.lms

import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.put
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class CourseImportExportLogicTest {
    @Test
    fun exportFileName() {
        assertEquals("C-ABC-course-export.json", CourseImportExportLogic.exportFileName("C-ABC"))
    }

    @Test
    fun webImportExportPath() {
        assertEquals("/courses/C-1/settings/import-export", CourseImportExportLogic.webImportExportPath("C-1"))
    }

    @Test
    fun parseValidImportFile() {
        val parsed = CourseImportExportLogic.parseImportFileText(
            """{"formatVersion":1,"courseCode":"C-1"}""",
        )
        assertEquals("C-1", parsed["courseCode"].toString().trim('"'))
    }

    @Test
    fun parseInvalidJson() {
        val error = runCatching { CourseImportExportLogic.parseImportFileText("{not valid") }.exceptionOrNull()
        assertTrue(error is CourseImportExportLogic.ImportExportError.InvalidJson)
    }

    @Test
    fun parseNonObjectJsonRejected() {
        val error = runCatching { CourseImportExportLogic.parseImportFileText("\"hello\"") }.exceptionOrNull()
        assertTrue(error is CourseImportExportLogic.ImportExportError.InvalidObject)
    }

    @Test
    fun parseEmptyObjectRejected() {
        val error = runCatching { CourseImportExportLogic.parseImportFileText("{}") }.exceptionOrNull()
        assertTrue(error is CourseImportExportLogic.ImportExportError.InvalidObject)
    }

    @Test
    fun fileTooLargeRejected() {
        val huge = buildJsonObject {
            put("payload", "x".repeat(CourseImportExportLogic.MAX_IMPORT_BYTES + 1))
        }.toString()
        val error = runCatching { CourseImportExportLogic.parseImportFileText(huge) }.exceptionOrNull()
        assertTrue(error is CourseImportExportLogic.ImportExportError.FileTooLarge)
    }
}
