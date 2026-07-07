package com.lextures.android.core.lms

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

class CourseBlueprintLogicTest {
    @Test
    fun canManageBlueprintRequiresOrgAndPermission() {
        val course = sampleCourse(orgId = "org-1")
        assertFalse(CourseBlueprintLogic.canManageBlueprint(course, emptyList()))
        assertTrue(CourseBlueprintLogic.canManageBlueprint(course, listOf(CourseBlueprintLogic.GLOBAL_ADMIN_PERMISSION)))
        assertTrue(CourseBlueprintLogic.canManageBlueprint(course, listOf(CourseBlueprintLogic.ORG_UNITS_ADMIN_PERMISSION)))
        assertFalse(
            CourseBlueprintLogic.canManageBlueprint(
                sampleCourse(orgId = null),
                listOf(CourseBlueprintLogic.GLOBAL_ADMIN_PERMISSION),
            ),
        )
    }

    @Test
    fun blueprintRole() {
        val master = sampleCourse(orgId = "org-1").copy(isBlueprint = true)
        assertEquals(CourseBlueprintLogic.BlueprintRole.Master, CourseBlueprintLogic.blueprintRole(master).role)

        val child = sampleCourse(orgId = "org-1").copy(blueprintParentCourseCode = "BP-001")
        assertEquals(CourseBlueprintLogic.BlueprintRole.Child, CourseBlueprintLogic.blueprintRole(child).role)
        assertEquals("BP-001", CourseBlueprintLogic.blueprintRole(child).parentCode)

        assertEquals(CourseBlueprintLogic.BlueprintRole.None, CourseBlueprintLogic.blueprintRole(sampleCourse("org-1")).role)
    }

    @Test
    fun shouldLoadBlueprintDetails() {
        val master = sampleCourse(orgId = "org-1").copy(isBlueprint = true)
        assertTrue(CourseBlueprintLogic.shouldLoadBlueprintDetails(master, canManage = true))
        assertFalse(CourseBlueprintLogic.shouldLoadBlueprintDetails(master, canManage = false))
    }

    @Test
    fun pushDisabledWhenOffline() {
        assertNotNull(CourseBlueprintLogic.pushDisabledReason(isOnline = false, childCount = 2))
    }

    @Test
    fun pushDisabledWithoutChildren() {
        assertNotNull(CourseBlueprintLogic.pushDisabledReason(isOnline = true, childCount = 0))
    }

    @Test
    fun pushEnabledWhenOnlineWithChildren() {
        assertNull(CourseBlueprintLogic.pushDisabledReason(isOnline = true, childCount = 1))
    }

    @Test
    fun mutationsDisabledWhenOffline() {
        assertNotNull(CourseBlueprintLogic.mutationsDisabledReason(isOnline = false))
        assertNull(CourseBlueprintLogic.mutationsDisabledReason(isOnline = true))
    }

    @Test
    fun cacheKey() {
        assertEquals("course:C-1:blueprint", CourseBlueprintLogic.cacheKeyBlueprintData("C-1"))
    }

    private fun sampleCourse(orgId: String?) = CourseSummary(
        id = "1",
        courseCode = "C-1",
        title = "Intro",
        orgId = orgId,
    )
}
