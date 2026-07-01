package com.lextures.android.core.navigation

import com.lextures.android.core.lms.CourseSummary
import com.lextures.android.core.lms.PlatformFeatures
import com.lextures.android.core.lms.TutorLogic
import com.lextures.android.core.lms.isOfficeHoursEnabled
import com.lextures.android.core.routing.CourseDeepLinkSection

/** Effective navigation persona derived from server-authoritative signals. */
enum class MobileRoleKind {
    Student,
    Instructor,
    Parent,
    SelfLearner,
}

/** Persisted active context for multi-role users. */
enum class MobileRoleContext {
    Learning,
    Teaching,
    Parent,
}

/** Primary bottom-bar destinations (≤5 per shell). */
enum class ShellTab(val labelRes: String, val iconName: String) {
    Home("tabs_home", "home"),
    Courses("tabs_courses", "courses"),
    Notebooks("tabs_notebooks", "notebooks"),
    Inbox("tabs_inbox", "inbox"),
    Profile("tabs_profile", "profile"),
    Teach("mobile_ia_tabs_teach", "teach"),
    Children("mobile_ia_tabs_children", "children"),
    Calendar("mobile_ia_tabs_calendar", "calendar"),
}

/** Secondary destinations surfaced from Profile / More hub. */
enum class MoreDestination(val labelRes: String) {
    Calendar("mobile_ia_more_calendar"),
    Planner("mobile_ia_more_planner"),
    Catalog("mobile_ia_more_catalog"),
    Paths("mobile_ia_more_paths"),
    Library("mobile_ia_more_library"),
    Reading("mobile_ia_more_reading"),
    Portfolio("mobile_ia_more_portfolio"),
    Credentials("mobile_ia_more_credentials"),
    Advising("mobile_ia_more_advising"),
    Settings("mobile_ia_more_settings"),
    AskAi("mobile_tutor_askAi"),
    PeerReviews("mobile_peerReview_title"),
}

/** Course-scoped workspace chips (registry-driven). */
enum class CourseWorkspaceSection(val labelRes: String, val deepLinkSegment: String?) {
    Overview("mobile_ia_course_overview", "overview"),
    Modules("mobile_ia_course_modules", "modules"),
    Grades("mobile_ia_course_grades", "grades"),
    Discussions("mobile_ia_course_discussions", "discussions"),
    Feed("mobile_ia_course_feed", "feed"),
    Live("mobile_ia_course_live", "live"),
    People("mobile_ia_course_people", "people"),
    Files("mobile_ia_course_files", "files"),
    Attendance("mobile_ia_course_attendance", "attendance"),
    Evaluations("mobile_ia_course_evaluations", "evaluations"),
    Library("mobile_ia_course_library", "library"),
    OfficeHours("mobile_ia_course_officeHours", "office-hours"),
    Grading("mobile_ia_course_grading", "grading"),
    ;

    companion object {
        fun from(section: CourseDeepLinkSection?): CourseWorkspaceSection? = when (section) {
            CourseDeepLinkSection.Overview -> Overview
            CourseDeepLinkSection.Modules -> Modules
            CourseDeepLinkSection.Grades -> Grades
            CourseDeepLinkSection.Feed -> Feed
            CourseDeepLinkSection.Discussions -> Discussions
            CourseDeepLinkSection.OfficeHours -> OfficeHours
            CourseDeepLinkSection.Live -> Live
            CourseDeepLinkSection.Files -> Files
            CourseDeepLinkSection.Attendance -> Attendance
            CourseDeepLinkSection.People -> People
            CourseDeepLinkSection.Evaluations -> Evaluations
            CourseDeepLinkSection.Library -> Library
            null -> null
        }
    }
}

data class RoleSnapshot(
    val hasStudentEnrollment: Boolean = false,
    val hasStaffEnrollment: Boolean = false,
    val hasParentDashboard: Boolean = false,
    val hasSelfPacedEnrollment: Boolean = false,
) {
    val availableContexts: List<MobileRoleContext>
        get() = buildList {
            if (hasParentDashboard) add(MobileRoleContext.Parent)
            if (hasStaffEnrollment) add(MobileRoleContext.Teaching)
            if (hasStudentEnrollment || hasSelfPacedEnrollment) add(MobileRoleContext.Learning)
            if (isEmpty()) add(MobileRoleContext.Learning)
        }

    fun defaultContext(): MobileRoleContext = availableContexts.first()

    fun resolvedContext(stored: MobileRoleContext?): MobileRoleContext {
        if (stored != null && stored in availableContexts) return stored
        return defaultContext()
    }
}

data class MobilePlatformFeatures(
    val ffLibrary: Boolean = false,
    val ffCourseEvaluations: Boolean = false,
    val ffMobileIaRedesign: Boolean = false,
    val ffMobileVibeActivities: Boolean = true,
    val ffMobileUniversalSearch: Boolean = false,
    val ffMobileProfileDepth: Boolean = false,
    val ffMobileLibraryEreserves: Boolean = true,
    val oerLibraryEnabled: Boolean = false,
    val customFieldsEnabled: Boolean = false,
    val ffDemographics: Boolean = false,
    val ffResearchConsent: Boolean = false,
    val ffPersistentTutor: Boolean = false,
    val ffAiStudyBuddy: Boolean = false,
    val ragNotebookEnabled: Boolean = false,
    val aiStudyBuddyEnabled: Boolean = false,
    val aiDisclosureEnabled: Boolean = false,
    val ffPeerReview: Boolean = false,
) {
    val libraryBrowseEnabled: Boolean
        get() = ffMobileLibraryEreserves && (ffLibrary || oerLibraryEnabled)

    companion object {
        fun from(features: PlatformFeatures?): MobilePlatformFeatures = MobilePlatformFeatures(
            ffLibrary = features?.ffLibrary == true,
            ffCourseEvaluations = features?.ffCourseEvaluations == true,
            ffMobileIaRedesign = features?.ffMobileIaRedesign == true,
            ffMobileVibeActivities = features?.ffMobileVibeActivities != false,
            ffMobileUniversalSearch = features?.ffMobileUniversalSearch == true,
            ffMobileProfileDepth = features?.ffMobileProfileDepth == true,
            ffMobileLibraryEreserves = features?.ffMobileLibraryEreserves != false,
            oerLibraryEnabled = features?.oerLibraryEnabled == true,
            customFieldsEnabled = features?.customFieldsEnabled == true,
            ffDemographics = features?.ffDemographics == true,
            ffResearchConsent = features?.ffResearchConsent == true,
            ffPersistentTutor = features?.ffPersistentTutor == true,
            ffAiStudyBuddy = features?.ffAiStudyBuddy == true,
            ragNotebookEnabled = features?.ragNotebookEnabled == true,
            aiStudyBuddyEnabled = features?.aiStudyBuddyEnabled == true,
            aiDisclosureEnabled = features?.aiDisclosureEnabled == true,
            ffPeerReview = features?.ffPeerReview == true,
        )
    }
}

data class CourseWorkspaceContext(
    val course: CourseSummary,
    val hasAttendanceSessions: Boolean = false,
    val hasLibraryResources: Boolean = false,
    val platformFeatures: MobilePlatformFeatures = MobilePlatformFeatures(),
)

/** Registry: role-aware shell tabs, More hub, and course workspace chips. */
object MobileDestinations {
    const val MAX_PRIMARY_CHIPS = 6
    const val PARENT_DASHBOARD_PERMISSION = "app:user:account-parent-dashboard"

    fun shellTabs(context: MobileRoleContext): List<ShellTab> = when (context) {
        MobileRoleContext.Teaching -> listOf(
            ShellTab.Home,
            ShellTab.Courses,
            ShellTab.Teach,
            ShellTab.Inbox,
            ShellTab.Profile,
        )
        MobileRoleContext.Parent -> listOf(
            ShellTab.Home,
            ShellTab.Children,
            ShellTab.Calendar,
            ShellTab.Inbox,
            ShellTab.Profile,
        )
        MobileRoleContext.Learning -> listOf(
            ShellTab.Home,
            ShellTab.Courses,
            ShellTab.Notebooks,
            ShellTab.Inbox,
            ShellTab.Profile,
        )
    }

    fun moreDestinations(
        context: MobileRoleContext,
        platform: MobilePlatformFeatures,
    ): List<MoreDestination> = buildList {
        when (context) {
            MobileRoleContext.Learning -> {
                if (TutorLogic.askAiEnabled(platform)) add(MoreDestination.AskAi)
                if (platform.ffPeerReview) add(MoreDestination.PeerReviews)
                add(MoreDestination.Calendar)
                add(MoreDestination.Planner)
                add(MoreDestination.Catalog)
                add(MoreDestination.Paths)
                add(MoreDestination.Reading)
                if (platform.libraryBrowseEnabled) add(MoreDestination.Library)
                add(MoreDestination.Portfolio)
                add(MoreDestination.Credentials)
                add(MoreDestination.Advising)
                add(MoreDestination.Settings)
            }
            MobileRoleContext.Teaching -> {
                add(MoreDestination.Calendar)
                add(MoreDestination.Planner)
                if (platform.libraryBrowseEnabled) add(MoreDestination.Library)
                add(MoreDestination.Advising)
                add(MoreDestination.Settings)
            }
            MobileRoleContext.Parent -> {
                add(MoreDestination.Calendar)
                add(MoreDestination.Advising)
                add(MoreDestination.Settings)
            }
        }
    }

    fun courseWorkspaceSections(ctx: CourseWorkspaceContext): List<CourseWorkspaceSection> = buildList {
        add(CourseWorkspaceSection.Overview)
        add(CourseWorkspaceSection.Modules)
        if (ctx.course.isFilesEnabled) add(CourseWorkspaceSection.Files)
        if (ctx.course.viewerIsStudent) add(CourseWorkspaceSection.Grades)
        if (ctx.course.isDiscussionsEnabled) add(CourseWorkspaceSection.Discussions)
        if (ctx.course.isFeedEnabled) add(CourseWorkspaceSection.Feed)
        if (ctx.course.isLiveSessionsEnabled) add(CourseWorkspaceSection.Live)
        if (ctx.course.viewerIsStaff && ctx.course.isSectionsEnabled) add(CourseWorkspaceSection.People)
        if (ctx.course.isOfficeHoursEnabled) add(CourseWorkspaceSection.OfficeHours)
        if (ctx.course.isAttendanceEnabled && (ctx.course.viewerIsStaff || ctx.hasAttendanceSessions)) {
            add(CourseWorkspaceSection.Attendance)
        }
        if (ctx.course.viewerIsStaff) {
            add(CourseWorkspaceSection.Grading)
            if (ctx.platformFeatures.ffCourseEvaluations) add(CourseWorkspaceSection.Evaluations)
        }
        if (ctx.platformFeatures.ffMobileLibraryEreserves &&
            ctx.platformFeatures.ffLibrary &&
            ctx.hasLibraryResources
        ) {
            add(CourseWorkspaceSection.Library)
        }
    }

    data class ChipSplit(
        val visible: List<CourseWorkspaceSection>,
        val overflow: List<CourseWorkspaceSection>,
    )

    fun splitCourseChips(sections: List<CourseWorkspaceSection>): ChipSplit {
        if (sections.size <= MAX_PRIMARY_CHIPS) {
            return ChipSplit(sections, emptyList())
        }
        return ChipSplit(
            visible = sections.take(MAX_PRIMARY_CHIPS),
            overflow = sections.drop(MAX_PRIMARY_CHIPS),
        )
    }

    fun buildRoleSnapshot(
        permissions: List<String>,
        courses: List<CourseSummary>,
        selfPacedEnrollmentCount: Int = 0,
    ): RoleSnapshot = RoleSnapshot(
        hasStudentEnrollment = courses.any { it.viewerIsStudent },
        hasStaffEnrollment = courses.any { it.viewerIsStaff },
        hasParentDashboard = permissions.contains(PARENT_DASHBOARD_PERMISSION),
        hasSelfPacedEnrollment = selfPacedEnrollmentCount > 0,
    )
}