package com.lextures.android.core.lms

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class ModuleContentModelsTest {
    @Test
    fun buildModuleGroupsOrdersChildren() {
        val items = listOf(
            CourseStructureItem("m1", 0, "module", "Module 1", null, true),
            CourseStructureItem("p2", 2, "content_page", "Reading 2", "m1", true),
            CourseStructureItem("p1", 1, "content_page", "Reading 1", "m1", true),
        )
        val groups = ModuleContentLogic.buildModuleGroups(items)
        assertEquals(1, groups.size)
        assertEquals(listOf("p1", "p2"), groups[0].items.map { it.id })
    }

    @Test
    fun itemLockStateLookup() {
        val progress = ModulesProgressSnapshot(
            enrollmentId = "e1",
            modules = listOf(
                ModuleLockState(
                    moduleId = "m1",
                    title = "Module 1",
                    items = listOf(
                        ItemLockState(
                            itemId = "q1",
                            locked = true,
                            complete = false,
                            reason = LockReason(message = "Complete Reading 1"),
                        ),
                    ),
                ),
            ),
        )
        val lock = ModuleContentLogic.itemLockState(progress, "q1")
        assertEquals(true, lock?.locked)
        assertEquals("Complete Reading 1", lock?.reason?.message)
        assertTrue(ModuleContentLogic.isLocked(progress, "q1"))
    }

    @Test
    fun destinationRouting() {
        assertEquals(ModuleItemDestination.ContentPage, ModuleContentLogic.destination("content_page"))
        assertEquals(ModuleItemDestination.Quiz, ModuleContentLogic.destination("quiz"))
        assertEquals(ModuleItemDestination.Interactive, ModuleContentLogic.destination("h5p"))
        assertTrue(ModuleContentLogic.isNavigable("external_link"))
        assertFalse(ModuleContentLogic.isNavigable("heading"))
    }
}
