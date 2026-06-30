package com.lextures.android.core.lms

import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class RequirementsLogicTest {
    private fun item(
        id: String,
        sort: Int,
        kind: String = "content_page",
        title: String,
        parent: String,
    ) = CourseStructureItem(id, sort, kind, title, parent, true)

    private fun moduleItem(id: String, sort: Int, title: String) =
        CourseStructureItem(id, sort, "module", title, null, true)

    @Test
    fun sequentialRequirementsListsPriorSteps() {
        val items = listOf(
            moduleItem("m1", 0, "Module 1"),
            item("p1", 1, title = "Reading 1", parent = "m1"),
            item("p2", 2, title = "Reading 2", parent = "m1"),
            item("q1", 3, kind = "quiz", title = "Quiz 1", parent = "m1"),
        )
        val groups = ModuleContentLogic.buildModuleGroups(items)
        val progress = ModulesProgressSnapshot(
            enrollmentId = "e1",
            modules = listOf(
                ModuleLockState(
                    moduleId = "m1",
                    title = "Module 1",
                    items = listOf(
                        ItemLockState(itemId = "p1", locked = false, complete = true),
                        ItemLockState(
                            itemId = "p2",
                            locked = true,
                            complete = false,
                            reason = LockReason(
                                code = "sequential_order",
                                message = "Complete \"Reading 1\" first.",
                                itemId = "p1",
                                title = "Reading 1",
                            ),
                        ),
                        ItemLockState(
                            itemId = "q1",
                            locked = true,
                            complete = false,
                            reason = LockReason(
                                code = "sequential_order",
                                message = "Complete \"Reading 2\" first.",
                                itemId = "p2",
                                title = "Reading 2",
                            ),
                        ),
                    ),
                ),
            ),
        )

        val target = items.first { it.id == "q1" }!!
        val summary = RequirementsLogic.buildRequirements(target, groups, progress)

        assertEquals(1, summary.metCount)
        assertEquals(2, summary.totalCount)
        assertEquals("p2", summary.nextRequiredItemId)
        assertTrue(summary.rows.any { it.id == "item:p1" && it.met })
        assertTrue(summary.rows.any { it.id == "item:p2" && !it.met })
    }

    @Test
    fun modulePrerequisiteRequirementsIncludeIncompleteItems() {
        val items = listOf(
            moduleItem("m1", 0, "Module A"),
            item("a1", 1, title = "Lesson A", parent = "m1"),
            moduleItem("m2", 2, "Module B"),
            item("b1", 3, title = "Lesson B", parent = "m2"),
        )
        val groups = ModuleContentLogic.buildModuleGroups(items)
        val progress = ModulesProgressSnapshot(
            enrollmentId = "e1",
            modules = listOf(
                ModuleLockState(
                    moduleId = "m1",
                    title = "Module A",
                    complete = false,
                    items = listOf(ItemLockState(itemId = "a1", locked = false, complete = false)),
                ),
                ModuleLockState(
                    moduleId = "m2",
                    title = "Module B",
                    locked = true,
                    complete = false,
                    reason = LockReason(
                        code = "module_prerequisite",
                        message = "Complete module \"Module A\" to unlock.",
                        title = "Module A",
                    ),
                    items = listOf(
                        ItemLockState(
                            itemId = "b1",
                            locked = true,
                            complete = false,
                            reason = LockReason(
                                code = "module_prerequisite",
                                message = "Complete module \"Module A\" to unlock.",
                                title = "Module A",
                            ),
                        ),
                    ),
                ),
            ),
        )

        val target = items.first { it.id == "b1" }!!
        val summary = RequirementsLogic.buildRequirements(target, groups, progress)

        assertEquals("a1", summary.nextRequiredItemId)
        assertTrue(summary.rows.any { it.id == "module:m1" && !it.met })
        assertTrue(summary.rows.any { it.id == "item:a1" && !it.met })
    }

    @Test
    fun findItemReturnsMatch() {
        val items = listOf(
            moduleItem("m1", 0, "Module 1"),
            item("p1", 1, title = "Reading 1", parent = "m1"),
        )
        val groups = ModuleContentLogic.buildModuleGroups(items)
        assertEquals("Reading 1", RequirementsLogic.findItem("p1", groups)?.title)
    }
}
