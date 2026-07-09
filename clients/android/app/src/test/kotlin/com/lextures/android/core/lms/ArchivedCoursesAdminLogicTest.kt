package com.lextures.android.core.lms

import com.lextures.android.core.navigation.MobilePlatformFeatures
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class ArchivedCoursesAdminLogicTest {
    @Test
    fun adminSettingsEnabledRequiresFlag() {
        val off = MobilePlatformFeatures()
        assertFalse(ArchivedCoursesAdminLogic.adminSettingsEnabled(off))
        val on = MobilePlatformFeatures(ffMobileAdminSettings = true)
        assertTrue(ArchivedCoursesAdminLogic.adminSettingsEnabled(on))
    }

    @Test
    fun canManageArchivedCoursesRequiresRbacManage() {
        assertFalse(ArchivedCoursesAdminLogic.canManageArchivedCourses(emptyList()))
        assertTrue(
            ArchivedCoursesAdminLogic.canManageArchivedCourses(
                listOf(ArchivedCoursesAdminLogic.RBAC_MANAGE_PERMISSION),
            ),
        )
    }

    @Test
    fun shouldShowEntryRequiresFlagAndPermission() {
        val features = MobilePlatformFeatures(ffMobileAdminSettings = true)
        assertFalse(
            ArchivedCoursesAdminLogic.shouldShowEntry(features, emptyList()),
        )
        assertTrue(
            ArchivedCoursesAdminLogic.shouldShowEntry(
                features,
                listOf(ArchivedCoursesAdminLogic.RBAC_MANAGE_PERMISSION),
            ),
        )
    }

    @Test
    fun filterRowsMatchesTitleAndCode() {
        val rows = listOf(
            ArchivedCourseRow(id = "1", courseCode = "C-ALG101", title = "Algebra I"),
            ArchivedCourseRow(id = "2", courseCode = "C-BIO201", title = "Biology"),
        )
        assertEquals(
            listOf("C-ALG101"),
            ArchivedCoursesAdminLogic.filterRows(rows, "alg").map { it.courseCode },
        )
    }

    @Test
    fun deleteConfirmMatchesCourseCode() {
        val row = ArchivedCourseRow(id = "1", courseCode = "C-DEL01", title = "Delete me")
        assertFalse(ArchivedCoursesAdminLogic.deleteConfirmMatches("wrong", row))
        assertTrue(ArchivedCoursesAdminLogic.deleteConfirmMatches("c-del01", row))
    }

    @Test
    fun archivedByLabelPrefersNameThenEmail() {
        val withName = ArchivedCourseRow(
            id = "1",
            courseCode = "C-1",
            title = "T",
            archivedByName = "  Pat  ",
            archivedByEmail = "pat@example.com",
        )
        assertEquals("Pat", ArchivedCoursesAdminLogic.archivedByLabel(withName))

        val withEmail = ArchivedCourseRow(
            id = "2",
            courseCode = "C-2",
            title = "T",
            archivedByEmail = "admin@example.com",
        )
        assertEquals("admin@example.com", ArchivedCoursesAdminLogic.archivedByLabel(withEmail))
    }
}