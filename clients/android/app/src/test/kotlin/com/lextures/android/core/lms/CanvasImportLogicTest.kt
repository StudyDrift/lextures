package com.lextures.android.core.lms

import com.lextures.android.core.navigation.MobilePlatformFeatures
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class CanvasImportLogicTest {
    @Test
    fun validateCredentials() {
        assertEquals(
            "mobile.canvasImport.error.urlRequired",
            CanvasImportLogic.validateCredentials("", "tok"),
        )
        assertEquals(
            "mobile.canvasImport.error.urlInvalid",
            CanvasImportLogic.validateCredentials("canvas.example.edu", "tok"),
        )
        assertEquals(
            "mobile.canvasImport.error.tokenRequired",
            CanvasImportLogic.validateCredentials("https://canvas.example.edu", "  "),
        )
        assertNull(
            CanvasImportLogic.validateCredentials("https://canvas.example.edu/", "secret-token"),
        )
    }

    @Test
    fun normalizeBaseUrl() {
        assertEquals(
            "https://canvas.example.edu",
            CanvasImportLogic.normalizeBaseUrl(" https://canvas.example.edu/ "),
        )
    }

    @Test
    fun includeDefaultsAndToggle() {
        val include = CanvasImportLogic.Include.ALL
        CanvasImportLogic.IncludeCategory.entries.forEach {
            assertTrue(include.value(it))
        }
        val gradesOff = include.set(CanvasImportLogic.IncludeCategory.Grades, false)
        assertFalse(gradesOff.grades)
        assertTrue(gradesOff.modules)
        assertEquals(0, gradesOff.enabledCategoryCounts()["grades"])
        assertEquals(1, gradesOff.enabledCategoryCounts()["modules"])
    }

    @Test
    fun parseWSMessage() {
        val parsed = CanvasImportLogic.parseWSMessage("""{"type":"progress","message":"Importing modules"}""")
        assertEquals(CanvasImportLogic.WSMessageType.Progress, parsed?.type)
        assertEquals("Importing modules", parsed?.message)
        assertTrue(CanvasImportLogic.isTerminal(CanvasImportLogic.WSMessageType.Complete))
        assertTrue(CanvasImportLogic.isTerminal(CanvasImportLogic.WSMessageType.Error))
        assertFalse(CanvasImportLogic.isTerminal(CanvasImportLogic.WSMessageType.Progress))
    }

    @Test
    fun entryGateRequiresFlagPermissionAndOnline() {
        val perms = listOf(CanvasImportLogic.COURSE_CREATE_PERMISSION)
        assertFalse(
            CanvasImportLogic.shouldShowCanvasImportEntry(
                permissions = perms,
                features = MobilePlatformFeatures(ffMobileCourseCreateV2 = true, ffMobileCanvasImport = false),
                isOnline = true,
            ),
        )
        assertTrue(
            CanvasImportLogic.shouldShowCanvasImportEntry(
                permissions = perms,
                features = MobilePlatformFeatures(ffMobileCourseCreateV2 = true, ffMobileCanvasImport = true),
                isOnline = true,
            ),
        )
        assertFalse(
            CanvasImportLogic.shouldShowCanvasImportEntry(
                permissions = perms,
                features = MobilePlatformFeatures(ffMobileCourseCreateV2 = true, ffMobileCanvasImport = true),
                isOnline = false,
            ),
        )
        assertFalse(
            CanvasImportLogic.shouldShowCanvasImportEntry(
                permissions = emptyList(),
                features = MobilePlatformFeatures(ffMobileCourseCreateV2 = true, ffMobileCanvasImport = true),
                isOnline = true,
            ),
        )
    }

    @Test
    fun tokenAbsenceFromStorageHaystacks() {
        val token = "canvas-secret-token-abc"
        assertFalse(
            CanvasImportLogic.storageContainsToken(
                listOf("userDefaults:ok", "keychain:session-jwt"),
                token,
            ),
        )
        assertTrue(
            CanvasImportLogic.storageContainsToken(
                listOf("oops $token leaked"),
                token,
            ),
        )
        assertTrue(CanvasImportLogic.TOKEN_MUST_NOT_PERSIST_POLICY.contains("never persisted"))
    }

    @Test
    fun filterCourses() {
        val courses = listOf(
            CanvasCourseListItem(1, "Biology 101", "BIO101", "available", "Fall"),
            CanvasCourseListItem(2, "Chemistry", "CHEM", "unpublished", "Spring"),
        )
        assertEquals(1, CanvasImportLogic.filterCourses(courses, "bio").size)
        assertTrue(CanvasImportLogic.isUnpublished("unpublished"))
        assertFalse(CanvasImportLogic.isUnpublished("available"))
    }
}
