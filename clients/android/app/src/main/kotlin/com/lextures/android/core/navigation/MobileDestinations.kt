package com.lextures.android.core.navigation

import com.lextures.android.core.design.UIMode
import com.lextures.android.core.lms.AdvisingLogic
import com.lextures.android.core.lms.CourseSettingsLogic
import com.lextures.android.core.lms.CourseSummary
import com.lextures.android.core.lms.EvaluationLogic
import com.lextures.android.core.lms.EvaluationStatus
import com.lextures.android.core.lms.ImmersiveReaderCapabilities
import com.lextures.android.core.lms.InstructorInsightsLogic
import com.lextures.android.core.lms.PlatformFeatures
import com.lextures.android.core.lms.TutorLogic
import com.lextures.android.core.lms.WalletLogic
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

/**
 * Two-level left-drawer state machine shared by the shell.
 * None = no drawer; Course = course-scoped menu; Global = app-wide menu.
 */
enum class DrawerState { None, Course, Global }

/**
 * Top-level app destinations reachable from the global drawer.
 * Replaces the former bottom-bar [ShellTab] selection. `labelRes`/`iconName`
 * are resolved to R.string / ImageVector in the Compose layer.
 */
enum class RootDestination(val labelRes: String, val iconName: String) {
    Dashboard("mobile_drawer_dashboard", "dashboard"),
    Courses("tabs_courses", "courses"),
    Calendar("mobile_ia_tabs_calendar", "calendar"),
    Todos("mobile_drawer_todos", "todos"),
    Review("mobile_drawer_review", "review"),
    Insights("mobile_drawer_insights", "insights"),
    Notebooks("mobile_drawer_notebooks", "notebooks"),
    GlobalNotebook("mobile_drawer_globalNotebook", "globalNotebook"),
    Accommodations("mobile_drawer_accommodations", "accommodations"),
    Inbox("tabs_inbox", "inbox"),
    Settings("mobile_ia_more_settings", "settings"),
    Profile("tabs_profile", "profile"),
    Teach("mobile_ia_tabs_teach", "teach"),
    Children("mobile_ia_tabs_children", "children"),
    ;

    val showsInboxBadge: Boolean get() = this == Inbox
}

/** A titled section of the global drawer. `titleRes == null` renders header-less. */
data class DrawerGroup(val titleRes: String?, val items: List<RootDestination>)

/** A titled section of the course drawer, grouping existing workspace sections. */
data class CourseDrawerGroup(val titleRes: String, val sections: List<CourseWorkspaceSection>)

/** Secondary destinations surfaced from Profile / More hub. */
enum class MoreDestination(val labelRes: String) {
    Calendar("mobile_ia_more_calendar"),
    Planner("mobile_ia_more_planner"),
    Catalog("mobile_ia_more_catalog"),
    Marketplace("mobile_ia_more_marketplace"),
    Paths("mobile_ia_more_paths"),
    Library("mobile_ia_more_library"),
    Reading("mobile_ia_more_reading"),
    Portfolio("mobile_ia_more_portfolio"),
    Credentials("mobile_ia_more_credentials"),
    Wallet("mobile_ia_more_wallet"),
    Gamification("mobile_ia_more_gamification"),
    Advising("mobile_ia_more_advising"),
    Settings("mobile_ia_more_settings"),
    AskAi("mobile_tutor_askAi"),
    PeerReviews("mobile_peerReview_title"),
    ReportCards("mobile_mastery_reportCards"),
    Insights("mobile_ia_more_insights"),
}

/** Course-scoped workspace chips (registry-driven). */
enum class CourseWorkspaceSection(val labelRes: String, val deepLinkSegment: String?) {
    Overview("mobile_ia_course_overview", "overview"),
    Modules("mobile_ia_course_modules", "modules"),
    Grades("mobile_ia_course_grades", "grades"),
    Mastery("mobile_ia_course_mastery", "mastery"),
    Discussions("mobile_ia_course_discussions", "discussions"),
    Feed("mobile_ia_course_feed", "feed"),
    Live("mobile_ia_course_live", "live"),
    People("mobile_ia_course_people", "people"),
    Files("mobile_ia_course_files", "files"),
    Attendance("mobile_ia_course_attendance", "attendance"),
    Evaluations("mobile_ia_course_evaluations", "evaluations"),
    Library("mobile_ia_course_library", "library"),
    OfficeHours("mobile_ia_course_officeHours", "office-hours"),
    Groups("mobile_ia_course_groups", "groups"),
    CollabDocs("mobile_ia_course_collabDocs", "collab-docs"),
    Boards("mobile_ia_course_boards", "boards"),
    Grading("mobile_ia_course_grading", "grading"),
    InstructorInsights("mobile_ia_course_insights", "insights"),
    Settings("mobile_ia_course_settings", "settings"),
    Behavior("mobile_ia_course_behavior", "behavior"),
    HallPass("mobile_ia_course_hallPass", "hall-pass"),
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
            CourseDeepLinkSection.Groups -> Groups
            CourseDeepLinkSection.CollabDocs -> CollabDocs
            CourseDeepLinkSection.Boards -> Boards
            CourseDeepLinkSection.Behavior -> Behavior
            CourseDeepLinkSection.HallPass -> HallPass
            CourseDeepLinkSection.Insights -> InstructorInsights
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
    val ffMobileCourseEvaluations: Boolean = true,
    val ffMobileIaRedesign: Boolean = false,
    val ffMobileVibeActivities: Boolean = true,
    val ffMobileUniversalSearch: Boolean = false,
    val ffMobileProfileDepth: Boolean = false,
    val ffMobileLibraryEreserves: Boolean = true,
    val ffMobileImmersiveReader: Boolean = true,
    val ffMobileLiveMeetings: Boolean = true,
    val readAloudEnabled: Boolean = false,
    val ffReadAloud: Boolean = false,
    val videoCaptionsEnabled: Boolean = false,
    val autoCaptioningEnabled: Boolean = false,
    val translationMemoryEnabled: Boolean = false,
    val ffReadingPreferences: Boolean = false,
    /** AN.2 kill-switch; default on when unset. */
    val ffMotionNavigation: Boolean = true,
    /** AN.3 kill-switch; default on when unset. */
    val ffMotionReveal: Boolean = true,
    /** AN.4 kill-switch; default on when unset. */
    val ffMotionLists: Boolean = true,
    val oerLibraryEnabled: Boolean = false,
    val xapiEmissionEnabled: Boolean = false,
    val customFieldsEnabled: Boolean = false,
    val ffDemographics: Boolean = false,
    val ffResearchConsent: Boolean = false,
    val ffPersistentTutor: Boolean = false,
    val ffAiStudyBuddy: Boolean = false,
    val ragNotebookEnabled: Boolean = false,
    val aiStudyBuddyEnabled: Boolean = false,
    val aiDisclosureEnabled: Boolean = false,
    val ffPeerReview: Boolean = false,
    val ffLearningPaths: Boolean = false,
    val selfReflectionEnabled: Boolean = false,
    val ffPublicCatalog: Boolean = false,
    val ffCourseMarketplace: Boolean = false,
    val ffSelfPacedMode: Boolean = false,
    val ffCourseReviews: Boolean = false,
    val ffCompletionCredentials: Boolean = false,
    val ffCoCurricularTranscript: Boolean = false,
    val ffTranscripts: Boolean = false,
    val ffCeuTracking: Boolean = false,
    val ffEportfolio: Boolean = false,
    val ffGamification: Boolean = false,
    val ffStripeBilling: Boolean = false,
    val ffPaymentsEnabled: Boolean = false,
    val ffTaxCollection: Boolean = false,
    val ffAdvisingIntegration: Boolean = false,
    val ffMobileAdvising: Boolean = true,
    val ffParentPortal: Boolean = false,
    val ffConferenceScheduling: Boolean = false,
    val ffClassroomSignals: Boolean = false,
    val ffBroadcasts: Boolean = false,
    val ffUiMode: Boolean = false,
    val atRiskAlertsEnabled: Boolean = false,
    val instructorInsightsEnabled: Boolean = false,
    val studentProgressEnabled: Boolean = true,
    val ffMobileInstructorInsights: Boolean = true,
    val ffMobileCourseSettings: Boolean = false,
    val ffMobileCreateCourse: Boolean = false,
    val ffConsortiumSharing: Boolean = false,
    val graderAgentEnabled: Boolean = false,
    val ffPlagiarismChecks: Boolean = false,
    val altTextEnforcementEnabled: Boolean = false,
    val learnerProfileEnabled: Boolean = false,
    val ffMobileLearnerProfile: Boolean = true,
    val introCourseEnabled: Boolean = false,
    val ffApiTokens: Boolean = false,
    val ffCalendarFeeds: Boolean = true,
    val ffMobileSettingsIntegrations: Boolean = false,
    val ffMobileAdminSettings: Boolean = false,
    val ffFeedback: Boolean = true,
) {
    val libraryBrowseEnabled: Boolean
        get() = ffMobileLibraryEreserves && (ffLibrary || oerLibraryEnabled)

    val immersiveReader: ImmersiveReaderCapabilities
        get() {
            if (!ffMobileImmersiveReader) {
                return ImmersiveReaderCapabilities(toolbarEnabled = false)
            }
            return ImmersiveReaderCapabilities(
                toolbarEnabled = true,
                readAloudEnabled = readAloudEnabled && ffReadAloud,
                translationEnabled = translationMemoryEnabled,
                captionsEnabled = videoCaptionsEnabled,
                preferencesEnabled = ffReadingPreferences || (readAloudEnabled && ffReadAloud),
            )
        }

    companion object {
        fun from(features: PlatformFeatures?): MobilePlatformFeatures = MobilePlatformFeatures(
            ffLibrary = features?.ffLibrary == true,
            ffCourseEvaluations = features?.ffCourseEvaluations == true,
            ffMobileCourseEvaluations = features?.ffMobileCourseEvaluations != false,
            ffMobileIaRedesign = features?.ffMobileIaRedesign == true,
            ffMobileVibeActivities = features?.ffMobileVibeActivities != false,
            ffMobileUniversalSearch = features?.ffMobileUniversalSearch == true,
            ffMobileProfileDepth = features?.ffMobileProfileDepth == true,
            ffMobileLibraryEreserves = features?.ffMobileLibraryEreserves != false,
            ffMobileImmersiveReader = features?.ffMobileImmersiveReader != false,
            ffMobileLiveMeetings = features?.ffMobileLiveMeetings != false,
            readAloudEnabled = features?.readAloudEnabled == true,
            ffReadAloud = features?.ffReadAloud == true,
            videoCaptionsEnabled = features?.videoCaptionsEnabled == true || features?.autoCaptioningEnabled == true,
            autoCaptioningEnabled = features?.autoCaptioningEnabled == true,
            translationMemoryEnabled = features?.translationMemoryEnabled == true,
            ffReadingPreferences = features?.ffReadingPreferences == true,
            ffMotionNavigation = features?.ffMotionNavigation != false,
            ffMotionReveal = features?.ffMotionReveal != false,
            ffMotionLists = features?.ffMotionLists != false,
            oerLibraryEnabled = features?.oerLibraryEnabled == true,
            xapiEmissionEnabled = features?.xapiEmissionEnabled == true,
            customFieldsEnabled = features?.customFieldsEnabled == true,
            ffDemographics = features?.ffDemographics == true,
            ffResearchConsent = features?.ffResearchConsent == true,
            ffPersistentTutor = features?.ffPersistentTutor == true,
            ffAiStudyBuddy = features?.ffAiStudyBuddy == true,
            ragNotebookEnabled = features?.ragNotebookEnabled == true,
            aiStudyBuddyEnabled = features?.aiStudyBuddyEnabled == true,
            aiDisclosureEnabled = features?.aiDisclosureEnabled == true,
            ffPeerReview = features?.ffPeerReview == true,
            ffLearningPaths = features?.ffLearningPaths == true,
            selfReflectionEnabled = features?.selfReflectionEnabled == true,
            ffPublicCatalog = features?.ffPublicCatalog == true,
            ffCourseMarketplace = features?.ffCourseMarketplace == true,
            ffSelfPacedMode = features?.ffSelfPacedMode == true,
            ffCourseReviews = features?.ffCourseReviews == true,
            ffCompletionCredentials = features?.ffCompletionCredentials == true,
            ffCoCurricularTranscript = features?.ffCoCurricularTranscript == true,
            ffTranscripts = features?.ffTranscripts == true,
            ffCeuTracking = features?.ffCeuTracking == true,
            ffEportfolio = features?.ffEportfolio == true,
            ffGamification = features?.ffGamification == true,
            ffStripeBilling = features?.ffStripeBilling == true,
            ffPaymentsEnabled = features?.ffPaymentsEnabled == true,
            ffTaxCollection = features?.ffTaxCollection == true,
            ffAdvisingIntegration = features?.ffAdvisingIntegration == true,
            ffMobileAdvising = features?.ffMobileAdvising != false,
            ffParentPortal = features?.ffParentPortal == true,
            ffConferenceScheduling = features?.ffConferenceScheduling == true,
            ffClassroomSignals = features?.ffClassroomSignals == true,
            ffBroadcasts = features?.ffBroadcasts == true,
            ffUiMode = features?.ffUiMode == true,
            atRiskAlertsEnabled = features?.atRiskAlertsEnabled == true,
            instructorInsightsEnabled = features?.instructorInsightsEnabled == true,
            studentProgressEnabled = features?.studentProgressEnabled != false,
            ffMobileInstructorInsights = features?.ffMobileInstructorInsights != false,
            ffMobileCourseSettings = features?.ffMobileCourseSettings == true,
            ffMobileCreateCourse = features?.ffMobileCreateCourse == true,
            ffConsortiumSharing = features?.ffConsortiumSharing == true,
            graderAgentEnabled = features?.graderAgentEnabled == true,
            ffPlagiarismChecks = features?.ffPlagiarismChecks == true,
            altTextEnforcementEnabled = features?.altTextEnforcementEnabled == true,
            learnerProfileEnabled = features?.learnerProfileEnabled == true,
            ffMobileLearnerProfile = features?.ffMobileLearnerProfile != false,
            introCourseEnabled = features?.introCourseEnabled != false,
            ffApiTokens = features?.ffApiTokens == true,
            ffCalendarFeeds = features?.ffCalendarFeeds != false,
            ffMobileSettingsIntegrations = features?.ffMobileSettingsIntegrations == true,
            ffMobileAdminSettings = features?.ffMobileAdminSettings == true,
            ffFeedback = features?.ffFeedback != false,
        )
    }
}

data class CourseWorkspaceContext(
    val course: CourseSummary,
    val permissions: List<String> = emptyList(),
    val hasAttendanceSessions: Boolean = false,
    val hasLibraryResources: Boolean = false,
    val evaluationStatus: EvaluationStatus? = null,
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

    /** Role-aware grouped destinations for the global drawer, mirroring the web sidebar. */
    fun globalDrawerGroups(
        context: MobileRoleContext,
        platform: MobilePlatformFeatures,
        uiMode: UIMode = UIMode.Standard,
    ): List<DrawerGroup> = when (context) {
        MobileRoleContext.Learning -> when (uiMode) {
            UIMode.K2 -> youngK2DrawerGroups()
            UIMode.Elementary -> youngElementaryDrawerGroups(platform)
            UIMode.Standard -> listOf(
                DrawerGroup(null, listOf(RootDestination.Dashboard, RootDestination.Courses, RootDestination.Calendar, RootDestination.Todos)),
                DrawerGroup("mobile_drawer_group_learning", learningDrawerItems(platform)),
                DrawerGroup("mobile_drawer_group_notes", listOf(RootDestination.Notebooks, RootDestination.GlobalNotebook)),
                DrawerGroup("mobile_drawer_group_administration", listOf(RootDestination.Accommodations)),
                DrawerGroup("mobile_drawer_group_account", listOf(RootDestination.Inbox, RootDestination.Settings)),
            )
        }
        MobileRoleContext.Teaching -> listOf(
            DrawerGroup(null, listOf(RootDestination.Dashboard, RootDestination.Courses, RootDestination.Calendar)),
            DrawerGroup("mobile_drawer_group_teaching", listOf(RootDestination.Teach)),
            DrawerGroup("mobile_drawer_group_notes", listOf(RootDestination.Notebooks)),
            DrawerGroup("mobile_drawer_group_account", listOf(RootDestination.Inbox, RootDestination.Settings)),
        )
        MobileRoleContext.Parent -> listOf(
            DrawerGroup(null, listOf(RootDestination.Dashboard, RootDestination.Children, RootDestination.Calendar)),
            DrawerGroup("mobile_drawer_group_account", listOf(RootDestination.Inbox, RootDestination.Settings)),
        )
    }

    /**
     * Regroups the existing course workspace sections under web-style headers.
     * Only sections available for the viewer (from [courseWorkspaceSections]) appear.
     */
    fun courseDrawerGroups(sections: List<CourseWorkspaceSection>): List<CourseDrawerGroup> {
        fun filtered(group: List<CourseWorkspaceSection>) = group.filter { it in sections }
        return listOf(
            "mobile_drawer_course_content" to filtered(
                listOf(CourseWorkspaceSection.Overview, CourseWorkspaceSection.Modules, CourseWorkspaceSection.Files, CourseWorkspaceSection.Library),
            ),
            "mobile_drawer_course_collaboration" to filtered(
                listOf(
                    CourseWorkspaceSection.Discussions,
                    CourseWorkspaceSection.Feed,
                    CourseWorkspaceSection.Groups,
                    CourseWorkspaceSection.CollabDocs,
                    CourseWorkspaceSection.Boards,
                    CourseWorkspaceSection.Live,
                    CourseWorkspaceSection.OfficeHours,
                ),
            ),
            "mobile_drawer_course_grades" to filtered(
                listOf(CourseWorkspaceSection.Grades, CourseWorkspaceSection.Mastery),
            ),
            "mobile_drawer_course_people" to filtered(listOf(CourseWorkspaceSection.People)),
            "mobile_drawer_course_manage" to filtered(
                listOf(
                    CourseWorkspaceSection.Grading,
                    CourseWorkspaceSection.InstructorInsights,
                    CourseWorkspaceSection.Settings,
                    CourseWorkspaceSection.Attendance,
                    CourseWorkspaceSection.Evaluations,
                    CourseWorkspaceSection.Behavior,
                    CourseWorkspaceSection.HallPass,
                ),
            ),
        ).mapNotNull { (key, list) -> if (list.isEmpty()) null else CourseDrawerGroup(key, list) }
    }

    private fun learningDrawerItems(platform: MobilePlatformFeatures): List<RootDestination> = buildList {
        add(RootDestination.Review)
        if (platform.selfReflectionEnabled) add(RootDestination.Insights)
    }

    private fun youngK2DrawerGroups(): List<DrawerGroup> = listOf(
        DrawerGroup(null, listOf(RootDestination.Dashboard, RootDestination.Courses, RootDestination.Todos)),
        DrawerGroup("mobile_drawer_group_account", listOf(RootDestination.Inbox, RootDestination.Settings)),
    )

    private fun youngElementaryDrawerGroups(platform: MobilePlatformFeatures): List<DrawerGroup> = listOf(
        DrawerGroup(null, listOf(RootDestination.Dashboard, RootDestination.Courses, RootDestination.Calendar, RootDestination.Todos)),
        DrawerGroup("mobile_drawer_group_learning", learningDrawerItems(platform)),
        DrawerGroup("mobile_drawer_group_account", listOf(RootDestination.Inbox, RootDestination.Settings)),
    )

    fun showsUniversalSearch(uiMode: UIMode): Boolean = uiMode == UIMode.Standard

    fun moreDestinations(
        context: MobileRoleContext,
        platform: MobilePlatformFeatures,
        uiMode: UIMode = UIMode.Standard,
    ): List<MoreDestination> = buildList {
        when (context) {
            MobileRoleContext.Learning -> {
                when (uiMode) {
                    UIMode.K2 -> {
                        if (platform.ffLibrary) add(MoreDestination.Reading)
                        add(MoreDestination.Settings)
                        return@buildList
                    }
                    UIMode.Elementary -> {
                        add(MoreDestination.ReportCards)
                        if (platform.ffLibrary) add(MoreDestination.Reading)
                        add(MoreDestination.Calendar)
                        add(MoreDestination.Settings)
                        return@buildList
                    }
                    UIMode.Standard -> Unit
                }
                if (TutorLogic.askAiEnabled(platform)) add(MoreDestination.AskAi)
                if (platform.ffPeerReview) add(MoreDestination.PeerReviews)
                add(MoreDestination.ReportCards)
                if (platform.selfReflectionEnabled) add(MoreDestination.Insights)
                add(MoreDestination.Calendar)
                add(MoreDestination.Planner)
                add(MoreDestination.Catalog)
                if (platform.ffCourseMarketplace) add(MoreDestination.Marketplace)
                add(MoreDestination.Paths)
                if (platform.ffLibrary) add(MoreDestination.Reading)
                if (platform.libraryBrowseEnabled) add(MoreDestination.Library)
                if (platform.ffEportfolio) add(MoreDestination.Portfolio)
                if (WalletLogic.walletEnabled(platform)) {
                    add(MoreDestination.Wallet)
                } else if (platform.ffCompletionCredentials) {
                    add(MoreDestination.Credentials)
                }
                if (platform.ffGamification) add(MoreDestination.Gamification)
                if (AdvisingLogic.advisingEnabled(platform)) add(MoreDestination.Advising)
                add(MoreDestination.Settings)
            }
            MobileRoleContext.Teaching -> {
                add(MoreDestination.Calendar)
                add(MoreDestination.Planner)
                if (platform.libraryBrowseEnabled) add(MoreDestination.Library)
                if (AdvisingLogic.advisingEnabled(platform)) add(MoreDestination.Advising)
                add(MoreDestination.Settings)
            }
            MobileRoleContext.Parent -> {
                add(MoreDestination.Calendar)
                if (AdvisingLogic.advisingEnabled(platform)) add(MoreDestination.Advising)
                add(MoreDestination.Settings)
            }
        }
    }

    fun courseWorkspaceSections(ctx: CourseWorkspaceContext): List<CourseWorkspaceSection> = buildList {
        add(CourseWorkspaceSection.Overview)
        add(CourseWorkspaceSection.Modules)
        if (ctx.course.isFilesEnabled) add(CourseWorkspaceSection.Files)
        if (ctx.course.viewerIsStudent) add(CourseWorkspaceSection.Grades)
        if (ctx.course.viewerIsStudent && ctx.course.isMasteryEnabled) add(CourseWorkspaceSection.Mastery)
        if (ctx.course.isDiscussionsEnabled) add(CourseWorkspaceSection.Discussions)
        if (ctx.course.isFeedEnabled) add(CourseWorkspaceSection.Feed)
        if (ctx.course.isLiveSessionsEnabled) add(CourseWorkspaceSection.Live)
        if (ctx.course.viewerIsStaff && ctx.course.isSectionsEnabled) add(CourseWorkspaceSection.People)
        if (ctx.course.isOfficeHoursEnabled) add(CourseWorkspaceSection.OfficeHours)
        if (ctx.course.isGroupSpacesEnabled) add(CourseWorkspaceSection.Groups)
        if (ctx.course.isCollabDocsEnabled) add(CourseWorkspaceSection.CollabDocs)
        if (ctx.course.isVisualBoardsEnabled) add(CourseWorkspaceSection.Boards)
        if (ctx.course.isAttendanceEnabled && (ctx.course.viewerIsStaff || ctx.hasAttendanceSessions)) {
            add(CourseWorkspaceSection.Attendance)
        }
        if (EvaluationLogic.shouldShowWorkspaceSection(ctx.course, ctx.evaluationStatus, ctx.platformFeatures)) {
            add(CourseWorkspaceSection.Evaluations)
        }
        if (ctx.course.viewerIsStaff) {
            add(CourseWorkspaceSection.Grading)
        }
        if (InstructorInsightsLogic.shouldShowWorkspaceSection(ctx.course, ctx.platformFeatures)) {
            add(CourseWorkspaceSection.InstructorInsights)
        }
        if (CourseSettingsLogic.shouldShowWorkspaceSection(ctx.course, ctx.permissions, ctx.platformFeatures)) {
            add(CourseWorkspaceSection.Settings)
        }
        if (ctx.platformFeatures.ffClassroomSignals && ctx.course.viewerIsStaff) {
            add(CourseWorkspaceSection.Behavior)
        }
        if (ctx.platformFeatures.ffClassroomSignals &&
            ctx.course.isSectionsEnabled &&
            (ctx.course.viewerIsStaff || ctx.course.viewerIsStudent)
        ) {
            add(CourseWorkspaceSection.HallPass)
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