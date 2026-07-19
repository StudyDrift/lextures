package com.lextures.android.core.navigation

import com.lextures.android.core.lms.CourseSummary
import com.lextures.android.core.lms.PlatformFeatures
import com.lextures.android.core.routing.CourseDeepLinkSection
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class MobileDestinationsTest {
    @Test
    fun instructorShellTabsForegroundTeach() {
        val tabs = MobileDestinations.shellTabs(MobileRoleContext.Teaching)
        assertEquals(
            listOf(ShellTab.Home, ShellTab.Courses, ShellTab.Teach, ShellTab.Inbox, ShellTab.Profile),
            tabs,
        )
    }

    @Test
    fun parentShellTabsScopeChildren() {
        val tabs = MobileDestinations.shellTabs(MobileRoleContext.Parent)
        assertTrue(tabs.contains(ShellTab.Children))
        assertFalse(tabs.contains(ShellTab.Notebooks))
    }

    @Test
    fun courseWorkspaceShowsGroupsAndCollabDocsWhenEnabled() {
        val course = CourseSummary(
            id = "1",
            courseCode = "demo",
            title = "Demo",
            viewerEnrollmentRoles = listOf("student"),
            groupSpacesEnabled = true,
            collabDocsEnabled = true,
        )
        val sections = MobileDestinations.courseWorkspaceSections(CourseWorkspaceContext(course = course))
        assertTrue(sections.contains(CourseWorkspaceSection.Groups))
        assertTrue(sections.contains(CourseWorkspaceSection.CollabDocs))
    }

    @Test
    fun courseWorkspaceShowsBoardsWhenVisualBoardsEnabled() {
        val on = CourseSummary(
            id = "1",
            courseCode = "demo",
            title = "Demo",
            viewerEnrollmentRoles = listOf("student"),
            visualBoardsEnabled = true,
        )
        val off = CourseSummary(
            id = "1",
            courseCode = "demo",
            title = "Demo",
            viewerEnrollmentRoles = listOf("student"),
            visualBoardsEnabled = false,
        )
        assertTrue(
            MobileDestinations.courseWorkspaceSections(CourseWorkspaceContext(course = on))
                .contains(CourseWorkspaceSection.Boards),
        )
        assertFalse(
            MobileDestinations.courseWorkspaceSections(CourseWorkspaceContext(course = off))
                .contains(CourseWorkspaceSection.Boards),
        )
        assertEquals(CourseWorkspaceSection.Boards, CourseWorkspaceSection.from(CourseDeepLinkSection.Boards))
    }

    @Test
    fun courseWorkspaceShowsLiveQuizzesWhenCourseAndMobileFlagsEnabled() {
        val on = CourseSummary(
            id = "1",
            courseCode = "demo",
            title = "Demo",
            viewerEnrollmentRoles = listOf("student"),
            interactiveQuizzesEnabled = true,
        )
        val off = CourseSummary(
            id = "1",
            courseCode = "demo",
            title = "Demo",
            viewerEnrollmentRoles = listOf("student"),
            interactiveQuizzesEnabled = false,
        )
        val featuresOn = MobilePlatformFeatures(ffMobileLiveQuiz = true)
        val featuresOff = MobilePlatformFeatures(ffMobileLiveQuiz = false)
        assertTrue(
            MobileDestinations.courseWorkspaceSections(
                CourseWorkspaceContext(course = on, platformFeatures = featuresOn),
            ).contains(CourseWorkspaceSection.LiveQuizzes),
        )
        assertFalse(
            MobileDestinations.courseWorkspaceSections(
                CourseWorkspaceContext(course = off, platformFeatures = featuresOn),
            ).contains(CourseWorkspaceSection.LiveQuizzes),
        )
        assertFalse(
            MobileDestinations.courseWorkspaceSections(
                CourseWorkspaceContext(course = on, platformFeatures = featuresOff),
            ).contains(CourseWorkspaceSection.LiveQuizzes),
        )
        assertEquals(
            CourseWorkspaceSection.LiveQuizzes,
            CourseWorkspaceSection.from(CourseDeepLinkSection.LiveQuizzes),
        )
    }

    @Test
    fun platformFeaturesMapsMobileWhiteboardEdit() {
        val on = PlatformFeatures(ffMobileWhiteboardEdit = true)
        val off = PlatformFeatures(ffMobileWhiteboardEdit = false)
        assertTrue(MobilePlatformFeatures.from(on).ffMobileWhiteboardEdit)
        assertFalse(MobilePlatformFeatures.from(off).ffMobileWhiteboardEdit)
        assertFalse(MobilePlatformFeatures.from(null).ffMobileWhiteboardEdit)
    }

    @Test
    fun courseWorkspaceHidesDisabledFeatures() {
        val course = CourseSummary(
            id = "1",
            courseCode = "demo",
            title = "Demo",
            viewerEnrollmentRoles = listOf("student"),
            feedEnabled = true,
            discussionsEnabled = false,
            liveSessionsEnabled = true,
            filesEnabled = true,
            attendanceEnabled = false,
        )
        val sections = MobileDestinations.courseWorkspaceSections(
            CourseWorkspaceContext(course = course, hasAttendanceSessions = false),
        )
        assertTrue(sections.contains(CourseWorkspaceSection.Feed))
        assertTrue(sections.contains(CourseWorkspaceSection.Live))
        assertFalse(sections.contains(CourseWorkspaceSection.Discussions))
    }

    @Test
    fun roleSnapshotResolvesStoredContext() {
        val snapshot = RoleSnapshot(
            hasStudentEnrollment = true,
            hasStaffEnrollment = true,
        )
        assertEquals(MobileRoleContext.Teaching, snapshot.resolvedContext(MobileRoleContext.Teaching))
        assertEquals(MobileRoleContext.Teaching, snapshot.resolvedContext(MobileRoleContext.Parent))
    }

    @Test
    fun deepLinkMapsToWorkspaceSection() {
        assertEquals(CourseWorkspaceSection.Feed, CourseWorkspaceSection.from(CourseDeepLinkSection.Feed))
        assertEquals(CourseWorkspaceSection.Attendance, CourseWorkspaceSection.from(CourseDeepLinkSection.Attendance))
        assertEquals(CourseWorkspaceSection.Groups, CourseWorkspaceSection.from(CourseDeepLinkSection.Groups))
        assertEquals(CourseWorkspaceSection.CollabDocs, CourseWorkspaceSection.from(CourseDeepLinkSection.CollabDocs))
        assertEquals(CourseWorkspaceSection.Boards, CourseWorkspaceSection.from(CourseDeepLinkSection.Boards))
        assertEquals(CourseWorkspaceSection.Behavior, CourseWorkspaceSection.from(CourseDeepLinkSection.Behavior))
        assertEquals(CourseWorkspaceSection.HallPass, CourseWorkspaceSection.from(CourseDeepLinkSection.HallPass))
        assertEquals(CourseWorkspaceSection.InstructorInsights, CourseWorkspaceSection.from(CourseDeepLinkSection.Insights))
    }

    @Test
    fun courseWorkspaceShowsInsightsForStaffWhenEnabled() {
        val course = CourseSummary(
            id = "1",
            courseCode = "demo",
            title = "Demo",
            viewerEnrollmentRoles = listOf("teacher"),
        )
        val features = MobilePlatformFeatures(atRiskAlertsEnabled = true)
        val sections = MobileDestinations.courseWorkspaceSections(
            CourseWorkspaceContext(course = course, platformFeatures = features),
        )
        assertTrue(sections.contains(CourseWorkspaceSection.InstructorInsights))
    }

    @Test
    fun courseWorkspaceShowsBehaviorForStaffWhenClassroomSignalsOn() {
        val course = CourseSummary(
            id = "1",
            courseCode = "demo",
            title = "Demo",
            viewerEnrollmentRoles = listOf("teacher"),
            sectionsEnabled = true,
        )
        val features = MobilePlatformFeatures(ffClassroomSignals = true)
        val sections = MobileDestinations.courseWorkspaceSections(
            CourseWorkspaceContext(course = course, platformFeatures = features),
        )
        assertTrue(sections.contains(CourseWorkspaceSection.Behavior))
        assertTrue(sections.contains(CourseWorkspaceSection.HallPass))
    }

    @Test
    fun courseWorkspaceShowsLibraryWhenResourcesPresent() {
        val course = CourseSummary(
            id = "1",
            courseCode = "demo",
            title = "Demo",
            viewerEnrollmentRoles = listOf("student"),
        )
        val features = MobilePlatformFeatures(ffLibrary = true, ffMobileLibraryEreserves = true)
        val sections = MobileDestinations.courseWorkspaceSections(
            CourseWorkspaceContext(course = course, hasLibraryResources = true, platformFeatures = features),
        )
        assertTrue(sections.contains(CourseWorkspaceSection.Library))
    }
}