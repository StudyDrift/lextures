package com.lextures.android.core.lms

import com.lextures.android.core.navigation.MobilePlatformFeatures
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test

class BoardsAdvancedLogicTest {
    @Before
    fun setUp() {
        BoardsAdvancedObservability.resetForTests()
    }

    @Test
    fun advancedGatingRequiresFlagAndCourse() {
        assertFalse(
            BoardsAdvancedLogic.isAdvancedEnabled(true, MobilePlatformFeatures(ffMobileBoardsAdvanced = false)),
        )
        assertTrue(
            BoardsAdvancedLogic.isAdvancedEnabled(true, MobilePlatformFeatures(ffMobileBoardsAdvanced = true)),
        )
        assertFalse(
            BoardsAdvancedLogic.isAdvancedEnabled(false, MobilePlatformFeatures(ffMobileBoardsAdvanced = true)),
        )
        assertTrue(
            BoardsAdvancedLogic.canUseTemplates(true, MobilePlatformFeatures(ffMobileBoardsAdvanced = true), true),
        )
        assertFalse(
            BoardsAdvancedLogic.canUseTemplates(true, MobilePlatformFeatures(ffMobileBoardsAdvanced = true), false),
        )
    }

    @Test
    fun filterTemplatesByScopeAndQuery() {
        val templates = listOf(
            BoardTemplate(id = "1", scope = "builtin", title = "KWL Chart", description = "Know Want Learned", tags = listOf("kwl")),
            BoardTemplate(id = "2", scope = "course", title = "Exit ticket", description = "Quick check"),
            BoardTemplate(id = "3", scope = "org", title = "Brainstorm", description = "Ideas wall", tags = listOf("ideas")),
        )
        assertEquals(
            listOf("1"),
            BoardsAdvancedLogic.filterTemplates(templates, BoardTemplateScope.Builtin, "").map { it.id },
        )
        assertEquals(
            listOf("2"),
            BoardsAdvancedLogic.filterTemplates(templates, null, "exit").map { it.id },
        )
    }

    @Test
    fun pollDelayAndTerminal() {
        assertEquals(0.5, BoardsAdvancedLogic.pollDelaySeconds(0), 0.001)
        assertEquals(8.0, BoardsAdvancedLogic.pollDelaySeconds(10), 0.001)
        assertTrue(BoardsAdvancedLogic.isExportTerminal("done"))
        assertTrue(BoardsAdvancedLogic.isCopyTerminal("completed"))
        assertEquals("png", BoardsAdvancedLogic.exportFileExtension(BoardExportFormat.Image))
    }

    @Test
    fun orderedPostsForPresent() {
        val sections = listOf(
            BoardSection(id = "s2", boardId = "b", title = "B", sortIndex = 2.0),
            BoardSection(id = "s1", boardId = "b", title = "A", sortIndex = 1.0),
        )
        val posts = listOf(
            BoardPost(id = "p3", boardId = "b", contentType = "text", sectionId = "s2", sortIndex = 0.0),
            BoardPost(id = "p1", boardId = "b", contentType = "text", sectionId = "s1", sortIndex = 2.0),
            BoardPost(id = "p2", boardId = "b", contentType = "text", sectionId = "s1", sortIndex = 1.0),
            BoardPost(id = "p4", boardId = "b", contentType = "text", sectionId = null, sortIndex = 0.0),
        )
        assertEquals(
            listOf("p2", "p1", "p3", "p4"),
            BoardsAdvancedLogic.orderedPostsForPresent(posts, sections).map { it.id },
        )
    }

    @Test
    fun governanceGating() {
        val on = MobilePlatformFeatures(ffMobileAdminConsole = true, ffMobileBoardsAdvanced = true)
        assertTrue(BoardsGovernanceAdminLogic.canView(on, listOf("global:app:rbac:manage")))
        assertFalse(BoardsGovernanceAdminLogic.canView(on, emptyList()))
        val off = MobilePlatformFeatures(ffMobileAdminConsole = true, ffMobileBoardsAdvanced = false)
        assertFalse(BoardsGovernanceAdminLogic.canView(off, listOf("global:app:rbac:manage")))
    }

    @Test
    fun observabilityAndCapDraft() {
        BoardsAdvancedObservability.record("board_presented")
        assertEquals(1, BoardsAdvancedObservability.count("board_presented"))
        assertEquals(12, BoardsAdvancedLogic.parseBoardCapDraft("12"))
        assertNull(BoardsAdvancedLogic.parseBoardCapDraft(""))
    }
}
