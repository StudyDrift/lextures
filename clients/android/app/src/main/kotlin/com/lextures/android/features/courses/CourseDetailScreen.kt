package com.lextures.android.features.courses

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Layers
import androidx.compose.material.icons.filled.Menu
import androidx.compose.material.icons.filled.Search
import com.lextures.android.R
import com.lextures.android.core.i18n.L
import com.lextures.android.features.behavior.BehaviorRosterScreen
import com.lextures.android.features.behavior.HallPassScreen
import com.lextures.android.features.behavior.MyHallPassScreen
import com.lextures.android.features.navigation.courseSectionLabelRes
import androidx.compose.material3.Button
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import kotlinx.coroutines.launch
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import com.lextures.android.features.tutor.TutorChatMode
import com.lextures.android.features.tutor.TutorChatScreen
import com.lextures.android.features.tutor.TutorFab
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.activity.compose.BackHandler
import androidx.compose.ui.window.Dialog
import androidx.compose.ui.window.DialogProperties
import com.lextures.android.features.search.UniversalSearchScreen
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.lms.AttendanceSession
import com.lextures.android.core.lms.TakeAttendanceLogic
import com.lextures.android.core.lms.TakeAttendanceRequest
import com.lextures.android.core.realtime.CourseStructureSocket
import androidx.compose.ui.platform.LocalContext
import com.lextures.android.core.design.OfflineBanner
import com.lextures.android.core.design.StalenessChip
import com.lextures.android.core.lms.CourseStructureItem
import com.lextures.android.core.lms.CourseSummary
import com.lextures.android.core.lms.GradingBacklogItem
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.offline.OfflineCacheKey
import com.lextures.android.core.offline.OfflineService
import com.lextures.android.features.grading.GradingBacklogSection
import com.lextures.android.features.grading.SubmissionsListScreen
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsEmptyState
import com.lextures.android.features.home.LmsErrorBanner
import com.lextures.android.features.home.LmsSegmentedChips
import com.lextures.android.features.home.LmsSkeletonList
import com.lextures.android.core.lms.LibraryResourceLogic
import com.lextures.android.core.lms.ModuleContentLogic
import com.lextures.android.core.lms.isOfficeHoursEnabled
import com.lextures.android.core.lms.ModulesProgressSnapshot
import com.lextures.android.core.lms.RequirementsLogic
import com.lextures.android.features.files.CourseFilesScreen
import com.lextures.android.features.files.FilePreviewScreen
import com.lextures.android.core.lms.FilePreviewTarget
import com.lextures.android.core.lms.GradeFeedbackRoute
import com.lextures.android.features.grades.GradeFeedbackScreen
import com.lextures.android.features.officehours.CourseOfficeHoursSection
import com.lextures.android.features.discussions.CourseDiscussionsSection
import com.lextures.android.features.feed.CourseFeedSection
import com.lextures.android.features.boards.BoardsListScreen
import com.lextures.android.features.boards.BoardsUnavailableScreen
import com.lextures.android.features.groups.CourseCollabDocsSection
import com.lextures.android.features.groups.CourseGroupsSection
import com.lextures.android.features.evaluations.CourseEvaluationsSection
import com.lextures.android.core.lms.EvaluationLogic
import com.lextures.android.core.lms.EvaluationStatus
import com.lextures.android.core.navigation.CourseWorkspaceContext
import com.lextures.android.core.navigation.CourseWorkspaceSection
import com.lextures.android.core.navigation.MobileDestinations
import com.lextures.android.core.navigation.MobilePlatformFeatures
import com.lextures.android.features.live.CourseLiveSection
import com.lextures.android.features.home.HomeShellState
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.TextButton

/**
 * Course home: gradient hero + segmented sections
 * (Overview · Modules · Grades · Attendance · Grading by role).
 */
@Composable
fun CourseDetailScreen(
    session: AuthSession,
    course: CourseSummary,
    onBack: () -> Unit,
    shell: HomeShellState? = null,
    initialSection: CourseWorkspaceSection? = null,
    initialThreadId: String? = null,
    modifier: Modifier = Modifier,
) {
    val accessToken by session.accessToken.collectAsState()
    val context = LocalContext.current
    val offline = remember { OfflineService.get(context) }
    val isOnline by offline.networkMonitor.isOnline.collectAsState()
    val deepLinkThreadId = initialThreadId
    var boardsDeepLinkUnavailable by remember {
        mutableStateOf(false)
    }

    // A course reached with an unaccepted invitation: gate content behind accept/decline.
    // Accepting swaps in the refreshed (active) course so the rest of the screen loads normally.
    var resolvedCourse by remember(course.courseCode) { mutableStateOf(course) }
    if (resolvedCourse.hasPendingInvitation) {
        CourseInvitationScreen(
            session = session,
            course = resolvedCourse,
            onAccepted = { resolvedCourse = it },
            onDeclined = onBack,
            onBack = onBack,
            modifier = modifier,
        )
        return
    }
    @Suppress("NAME_SHADOWING")
    val course = resolvedCourse

    var section by rememberSaveable(course.courseCode) {
        mutableStateOf(initialSection?.name ?: "modules")
    }
    var showOverflow by remember { mutableStateOf(false) }
    var items by remember { mutableStateOf<List<CourseStructureItem>>(emptyList()) }
    var progress by remember { mutableStateOf<ModulesProgressSnapshot?>(null) }
    var cacheLabel by remember { mutableStateOf<String?>(null) }
    var hasAttendanceSessions by remember { mutableStateOf(false) }
    var evaluationStatus by remember { mutableStateOf<EvaluationStatus?>(null) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var loading by remember { mutableStateOf(true) }

    var openItem by remember { mutableStateOf<CourseStructureItem?>(null) }
    var lockedItem by remember { mutableStateOf<CourseStructureItem?>(null) }
    var openAttendanceSession by remember { mutableStateOf<AttendanceSession?>(null) }
    var openTakeAttendance by remember { mutableStateOf<TakeAttendanceRequest?>(null) }
    var openBacklogItem by remember { mutableStateOf<GradingBacklogItem?>(null) }
    var openInstructorAtRisk by remember { mutableStateOf(false) }
    var openInstructorWhatsWorking by remember { mutableStateOf(false) }
    var openStudentProgress by remember { mutableStateOf<Pair<String, String>?>(null) }
    var openFilePreview by remember { mutableStateOf<FilePreviewTarget?>(null) }
    var openGradeFeedback by remember { mutableStateOf<GradeFeedbackRoute?>(null) }
    var showCourseSearch by remember { mutableStateOf(false) }
    var showTutor by remember { mutableStateOf(false) }

    val emptyCourseTitle = moduleEmptyCourseTitle()
    val emptyCourseHint = moduleEmptyCourseHint()
    val groups = remember(items) { ModuleContentLogic.buildModuleGroups(items) }

    val structureSocket = remember(course.courseCode) { CourseStructureSocket() }
    val structureRevision by structureSocket.revision.collectAsState()
    DisposableEffect(course.courseCode) {
        structureSocket.connect(course.courseCode) { accessToken }
        onDispose { structureSocket.disconnect() }
    }

    BackHandler(onBack = onBack)

    openGradeFeedback?.let { route ->
        GradeFeedbackScreen(
            session = session,
            course = course,
            column = route.column,
            onBack = { openGradeFeedback = null },
            modifier = modifier,
        )
        return
    }

    openFilePreview?.let { preview ->
        FilePreviewScreen(
            session = session,
            target = preview,
            onBack = { openFilePreview = null },
            modifier = modifier,
        )
        return
    }

    openItem?.let { selected ->
        ModuleItemRouteScreen(
            session = session,
            course = course,
            item = selected,
            onBack = { openItem = null },
            onProgressChanged = { refreshProgress(accessToken, course, offline) { progress = it } },
            nativeVibeActivitiesEnabled = shell?.platformFeatures?.ffMobileVibeActivities != false,
            nativeLibraryEnabled = shell?.platformFeatures?.ffMobileLibraryEreserves != false,
            modifier = modifier,
        )
        return
    }

    openTakeAttendance?.let { request ->
        TakeAttendanceScreen(
            session = session,
            course = course,
            offline = offline,
            initialSessionId = request.sessionId,
            onBack = { openTakeAttendance = null },
            modifier = modifier,
        )
        return
    }

    openAttendanceSession?.let { selected ->
        AttendanceSessionDetailScreen(
            session = session,
            course = course,
            attendanceSession = selected,
            onBack = { openAttendanceSession = null },
            modifier = modifier,
        )
        return
    }

    openBacklogItem?.let { selected ->
        SubmissionsListScreen(
            session = session,
            course = course,
            backlogItem = selected,
            onBack = { openBacklogItem = null },
            modifier = modifier,
        )
        return
    }

    openStudentProgress?.let { (enrollmentId, displayName) ->
        com.lextures.android.features.instructorinsights.StudentProgressScreen(
            session = session,
            course = course,
            offline = offline,
            enrollmentId = enrollmentId,
            displayName = displayName,
            onBack = { openStudentProgress = null },
            modifier = modifier,
        )
        return
    }

    if (openInstructorAtRisk) {
        com.lextures.android.features.instructorinsights.AtRiskListScreen(
            session = session,
            course = course,
            offline = offline,
            platformFeatures = shell?.platformFeatures ?: MobilePlatformFeatures(),
            onBack = { openInstructorAtRisk = false },
            onOpenStudent = { enrollmentId, name -> openStudentProgress = enrollmentId to name },
            modifier = modifier,
        )
        return
    }

    if (openInstructorWhatsWorking) {
        com.lextures.android.features.instructorinsights.WhatsWorkingScreen(
            session = session,
            course = course,
            offline = offline,
            onBack = { openInstructorWhatsWorking = false },
            modifier = modifier,
        )
        return
    }

    LaunchedEffect(accessToken, course.courseCode, structureRevision) {
        val token = accessToken ?: return@LaunchedEffect
        loading = true
        errorMessage = null
        try {
            hasAttendanceSessions = runCatching {
                LmsApi.fetchAttendanceSessions(course.courseCode, token).isNotEmpty()
            }.getOrDefault(false)
            val platformFeatures = shell?.platformFeatures ?: MobilePlatformFeatures()
            evaluationStatus = if (EvaluationLogic.evaluationsEnabled(platformFeatures)) {
                runCatching { LmsApi.fetchEvaluationStatus(course.courseCode, token) }.getOrNull()
            } else {
                null
            }
            val result = offline.cachedFetch(
                key = OfflineCacheKey.courseStructure(course.courseCode),
                accessToken = token,
                serializer = kotlinx.serialization.builtins.ListSerializer(CourseStructureItem.serializer()),
            ) {
                LmsApi.fetchCourseStructure(course.courseCode, token)
            }
            items = result.first
            val cached = result.second
            cacheLabel = if (cached != null && cached.isStale(isOnline)) cached.lastUpdatedLabel() else null
            refreshProgress(accessToken, course, offline) { progress = it }
        } catch (e: Exception) {
            errorMessage = session.mapError(e)
        } finally {
            loading = false
        }
    }

    val workspaceContext = CourseWorkspaceContext(
        course = course,
        permissions = shell?.permissions.orEmpty(),
        hasAttendanceSessions = hasAttendanceSessions,
        hasLibraryResources = LibraryResourceLogic.hasLibraryResources(items),
        evaluationStatus = evaluationStatus,
        platformFeatures = shell?.platformFeatures ?: MobilePlatformFeatures(),
    )
    val allSections = MobileDestinations.courseWorkspaceSections(workspaceContext)
    val selectedSection = shell?.activeCourseSection
        ?: (CourseWorkspaceSection.entries.firstOrNull { it.name == section } ?: CourseWorkspaceSection.Modules)

    // Publish the active course + available sections so the course drawer can drive navigation.
    // DisposableEffect clears the course context when this screen leaves the composition.
    androidx.compose.runtime.DisposableEffect(course.courseCode) {
        if (shell != null) {
            shell.activeCourse = course
            shell.activeCourseRoot = shell.rootDestination
        }
        onDispose { shell?.activeCourse = null }
    }
    LaunchedEffect(allSections, initialSection) {
        shell?.activeCourseSections = allSections
        if (initialSection == CourseWorkspaceSection.Boards && CourseWorkspaceSection.Boards !in allSections) {
            boardsDeepLinkUnavailable = true
        } else {
            boardsDeepLinkUnavailable = false
            val initial = initialSection?.takeIf { it in allSections }
            if (initial != null && shell != null) shell.activeCourseSection = initial
        }
        val current = shell?.activeCourseSection
        if (current != null && current !in allSections && !boardsDeepLinkUnavailable) {
            shell?.activeCourseSection = allSections.firstOrNull() ?: CourseWorkspaceSection.Modules
        }
    }

    if (showCourseSearch && shell != null) {
        Dialog(
            onDismissRequest = { showCourseSearch = false },
            properties = DialogProperties(usePlatformDefaultWidth = false),
        ) {
            UniversalSearchScreen(
                session = session,
                shell = shell,
                onDismiss = { showCourseSearch = false },
                courseScope = course.courseCode,
                isOnline = isOnline,
            )
        }
    }

    if (showTutor) {
        Dialog(
            onDismissRequest = { showTutor = false },
            properties = DialogProperties(usePlatformDefaultWidth = false),
        ) {
            TutorChatScreen(
                session = session,
                mode = TutorChatMode.Course(course),
                shell = shell,
                onClose = { showTutor = false },
                modifier = Modifier.fillMaxSize(),
            )
        }
    }

    Box(modifier = modifier) {
    Column(modifier = Modifier.fillMaxSize()) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(top = 8.dp, end = 16.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            IconButton(onClick = onBack) {
                Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back", tint = textPrimary())
            }
            if (shell != null) {
                IconButton(onClick = { shell.drawerState = com.lextures.android.core.navigation.DrawerState.Course }) {
                    Icon(Icons.Default.Menu, contentDescription = L.text(R.string.mobile_drawer_courseMenu), tint = textPrimary())
                }
            }
            Text(
                text = course.displayTitle,
                fontSize = 18.sp,
                fontWeight = FontWeight.SemiBold,
                color = textPrimary(),
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
                modifier = Modifier.weight(1f),
            )
            if (shell?.universalSearchEnabled == true) {
                IconButton(onClick = { showCourseSearch = true }) {
                    Icon(Icons.Default.Search, contentDescription = "Search in course", tint = textPrimary())
                }
            }
        }

        LazyColumn(
            modifier = Modifier.fillMaxSize(),
            contentPadding = PaddingValues(16.dp),
            verticalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            item {
                CourseBanner(
                    course = course,
                    accessToken = accessToken,
                )
            }

            item {
                Text(
                    text = L.text(courseSectionLabelRes(selectedSection)),
                    fontSize = 20.sp,
                    fontWeight = FontWeight.SemiBold,
                    color = textPrimary(),
                )
            }

            errorMessage?.let { message ->
                item { LmsErrorBanner(message) }
            }

            if (!isOnline) {
                item { OfflineBanner() }
            }
            cacheLabel?.let { label ->
                item { StalenessChip(label = label) }
            }

            if (boardsDeepLinkUnavailable) {
                item { BoardsUnavailableScreen() }
            } else if (shell?.iaRedesignEnabled == true) {
                when (selectedSection) {
                    CourseWorkspaceSection.Overview -> item {
                        CourseSyllabusSection(session = session, course = course)
                    }
                    CourseWorkspaceSection.Grades -> item {
                        CourseGradesSection(
                            session = session,
                            course = course,
                            onOpenFeedback = { openGradeFeedback = it },
                        )
                    }
                    CourseWorkspaceSection.Mastery -> item {
                        com.lextures.android.features.mastery.CourseMasterySection(
                            session = session,
                            course = course,
                        )
                    }
                    CourseWorkspaceSection.OfficeHours -> item {
                        CourseOfficeHoursSection(session = session, course = course)
                    }
                    CourseWorkspaceSection.Attendance -> item {
                        CourseAttendanceSection(
                            session = session,
                            course = course,
                            onTakeAttendance = if (course.viewerIsStaff) {
                                { openTakeAttendance = TakeAttendanceRequest() }
                            } else {
                                null
                            },
                            onOpenSession = { attendanceSession ->
                                if (TakeAttendanceLogic.shouldTakeSession(attendanceSession, course.viewerIsStaff)) {
                                    openTakeAttendance = TakeAttendanceRequest(sessionId = attendanceSession.id)
                                } else {
                                    openAttendanceSession = attendanceSession
                                }
                            },
                        )
                    }
                    CourseWorkspaceSection.Grading -> item {
                        GradingBacklogSection(
                            session = session,
                            course = course,
                            onOpenItem = { openBacklogItem = it },
                        )
                    }
                    CourseWorkspaceSection.InstructorInsights -> item {
                        com.lextures.android.features.instructorinsights.CourseInsightsSection(
                            session = session,
                            course = course,
                            offline = offline,
                            platformFeatures = shell?.platformFeatures ?: MobilePlatformFeatures(),
                            shell = shell,
                            onOpenAtRisk = { openInstructorAtRisk = true },
                            onOpenWhatsWorking = { openInstructorWhatsWorking = true },
                        )
                    }
                    CourseWorkspaceSection.Settings -> item {
                        com.lextures.android.features.courses.settings.CourseSettingsHostScreen(
                            session = session,
                            course = course,
                            offline = offline,
                            platformFeatures = shell?.platformFeatures ?: MobilePlatformFeatures(),
                            permissions = shell?.permissions.orEmpty(),
                            onCourseUpdated = { resolvedCourse = it },
                        )
                    }
                    CourseWorkspaceSection.Files -> item {
                        CourseFilesScreen(
                            session = session,
                            course = course,
                            onOpenPreview = { openFilePreview = it },
                        )
                    }
                    CourseWorkspaceSection.Library -> item {
                        CourseLibraryScreen(
                            course = course,
                            items = items,
                            onSelectItem = { openItem = it },
                        )
                    }
                    CourseWorkspaceSection.Discussions -> item {
                        CourseDiscussionsSection(
                            session = session,
                            course = course,
                            initialThreadId = if (selectedSection == CourseWorkspaceSection.Discussions) {
                                deepLinkThreadId
                            } else {
                                null
                            },
                        )
                    }
                    CourseWorkspaceSection.Feed -> item {
                        CourseFeedSection(session = session, course = course)
                    }
                    CourseWorkspaceSection.Groups -> item {
                        CourseGroupsSection(session = session, course = course)
                    }
                    CourseWorkspaceSection.CollabDocs -> item {
                        CourseCollabDocsSection(session = session, course = course)
                    }
                    CourseWorkspaceSection.Boards -> item {
                        BoardsListScreen(
                            session = session,
                            course = course,
                            permissions = shell?.permissions.orEmpty(),
                            currentUserId = shell?.profile?.id,
                            initialBoardId = if (selectedSection == CourseWorkspaceSection.Boards) {
                                deepLinkThreadId
                            } else {
                                null
                            },
                        )
                    }
                    CourseWorkspaceSection.People -> item {
                        CoursePeopleSection(
                            session = session,
                            course = course,
                            platformFeatures = shell?.platformFeatures ?: MobilePlatformFeatures(),
                        )
                    }
                    CourseWorkspaceSection.Live -> item {
                        CourseLiveSection(
                            session = session,
                            course = course,
                            platformFeatures = shell?.platformFeatures ?: MobilePlatformFeatures(),
                        )
                    }
                    CourseWorkspaceSection.Evaluations -> item {
                        CourseEvaluationsSection(
                            session = session,
                            course = course,
                            showResults = course.viewerIsStaff ||
                                (selectedSection == CourseWorkspaceSection.Evaluations && deepLinkThreadId == "results"),
                        )
                    }
                    CourseWorkspaceSection.Behavior -> item {
                        BehaviorRosterScreen(session = session, course = course)
                    }
                    CourseWorkspaceSection.HallPass -> item {
                        if (course.viewerIsStaff) {
                            HallPassScreen(session = session, course = course)
                        } else {
                            MyHallPassScreen(session = session, course = course)
                        }
                    }
                    CourseWorkspaceSection.Modules -> {
                        if (loading && items.isEmpty()) {
                            item { LmsSkeletonList(count = 3) }
                        } else if (groups.isEmpty() && errorMessage == null) {
                            item {
                                LmsEmptyState(
                                    icon = Icons.Default.Layers,
                                    title = emptyCourseTitle,
                                    message = emptyCourseHint,
                                )
                            }
                        } else {
                            if (shell != null) {
                                item {
                                    com.lextures.android.features.introcourse.IntroCourseProgressRail(
                                        courseCode = course.courseCode,
                                        session = session,
                                        shell = shell,
                                    )
                                }
                            }
                            item {
                                ModuleList(
                                    course = course,
                                    groups = groups,
                                    progress = progress,
                                    onSelectItem = { openItem = it },
                                    onLockedItem = { item, _ -> lockedItem = item },
                                )
                            }
                        }
                    }
                }
            } else {
                when (section) {
                    "overview" -> item { CourseSyllabusSection(session = session, course = course) }
                    "grades" -> item {
                        CourseGradesSection(
                            session = session,
                            course = course,
                            onOpenFeedback = { openGradeFeedback = it },
                        )
                    }
                    "officehours" -> item { CourseOfficeHoursSection(session = session, course = course) }
                    "attendance" -> item {
                        CourseAttendanceSection(
                            session = session,
                            course = course,
                            onTakeAttendance = if (course.viewerIsStaff) {
                                { openTakeAttendance = TakeAttendanceRequest() }
                            } else {
                                null
                            },
                            onOpenSession = { attendanceSession ->
                                if (TakeAttendanceLogic.shouldTakeSession(attendanceSession, course.viewerIsStaff)) {
                                    openTakeAttendance = TakeAttendanceRequest(sessionId = attendanceSession.id)
                                } else {
                                    openAttendanceSession = attendanceSession
                                }
                            },
                        )
                    }
                    "grading" -> item {
                        GradingBacklogSection(
                            session = session,
                            course = course,
                            onOpenItem = { openBacklogItem = it },
                        )
                    }
                    "files" -> item {
                        CourseFilesScreen(
                            session = session,
                            course = course,
                            onOpenPreview = { openFilePreview = it },
                        )
                    }
                    else -> {
                        if (loading && items.isEmpty()) {
                            item { LmsSkeletonList(count = 3) }
                        } else if (groups.isEmpty() && errorMessage == null) {
                            item {
                                LmsEmptyState(
                                    icon = Icons.Default.Layers,
                                    title = emptyCourseTitle,
                                    message = emptyCourseHint,
                                )
                            }
                        } else {
                            item {
                                ModuleList(
                                    course = course,
                                    groups = groups,
                                    progress = progress,
                                    onSelectItem = { openItem = it },
                                    onLockedItem = { item, _ -> lockedItem = item },
                                )
                            }
                        }
                    }
                }
            }
        }
    }
        TutorFab(
            course = course,
            onOpen = { showTutor = true },
            modifier = Modifier.align(Alignment.BottomEnd),
        )
    }

    lockedItem?.let { item ->
        RequirementsSheet(
            targetItem = item,
            groups = groups,
            progress = progress,
            onDismiss = { lockedItem = null },
            onGoToRequired = { itemId ->
                RequirementsLogic.findItem(itemId, groups)?.let { openItem = it }
            },
        )
    }
}

/**
 * Shown when the viewer opens a course they were invited to but have not yet accepted.
 * Accepting activates the enrollment and hands back the refreshed (active) course; declining
 * removes the enrollment and pops back to the courses list.
 */
@Composable
private fun CourseInvitationScreen(
    session: AuthSession,
    course: CourseSummary,
    onAccepted: (CourseSummary) -> Unit,
    onDeclined: () -> Unit,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val accessToken by session.accessToken.collectAsState()
    val scope = rememberCoroutineScope()
    var submitting by remember { mutableStateOf(false) }
    var errorMessage by remember { mutableStateOf<String?>(null) }

    BackHandler(onBack = onBack)

    Column(modifier = modifier.fillMaxSize()) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(top = 8.dp, end = 16.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            IconButton(onClick = onBack) {
                Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back", tint = textPrimary())
            }
            Text(
                text = course.displayTitle,
                fontSize = 18.sp,
                fontWeight = FontWeight.SemiBold,
                color = textPrimary(),
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
                modifier = Modifier.weight(1f),
            )
        }

        LazyColumn(
            modifier = Modifier.fillMaxSize(),
            contentPadding = PaddingValues(16.dp),
            verticalArrangement = Arrangement.spacedBy(14.dp),
        ) {
            item { CourseBanner(course = course, accessToken = accessToken) }
            item {
                Text(
                    text = L.text(R.string.mobile_courseInvite_title),
                    fontSize = 20.sp,
                    fontWeight = FontWeight.SemiBold,
                    color = textPrimary(),
                )
            }
            item {
                Text(text = L.text(R.string.mobile_courseInvite_body), color = textPrimary())
            }
            errorMessage?.let { message ->
                item { LmsErrorBanner(message) }
            }
            item {
                Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
                    Button(
                        onClick = {
                            val token = accessToken ?: return@Button
                            val enrollmentId = course.viewerPendingEnrollmentId ?: return@Button
                            submitting = true
                            errorMessage = null
                            scope.launch {
                                try {
                                    LmsApi.approveCourseInvitation(course.courseCode, enrollmentId, token)
                                    val refreshed = runCatching {
                                        LmsApi.fetchCourse(course.courseCode, token)
                                    }.getOrNull() ?: course.copy(
                                        viewerEnrollmentInvitationPending = false,
                                        viewerPendingEnrollmentId = null,
                                    )
                                    onAccepted(refreshed)
                                } catch (e: Exception) {
                                    errorMessage = session.mapError(e)
                                    submitting = false
                                }
                            }
                        },
                        enabled = !submitting,
                        modifier = Modifier.fillMaxWidth(),
                    ) { Text(L.text(R.string.mobile_courseInvite_accept)) }

                    TextButton(
                        onClick = {
                            val token = accessToken ?: return@TextButton
                            val enrollmentId = course.viewerPendingEnrollmentId ?: return@TextButton
                            submitting = true
                            errorMessage = null
                            scope.launch {
                                try {
                                    LmsApi.declineCourseInvitation(course.courseCode, enrollmentId, token)
                                    onDeclined()
                                } catch (e: Exception) {
                                    errorMessage = session.mapError(e)
                                    submitting = false
                                }
                            }
                        },
                        enabled = !submitting,
                        modifier = Modifier.fillMaxWidth(),
                    ) { Text(L.text(R.string.mobile_courseInvite_decline)) }
                }
            }
        }
    }
}

private fun legacySections(
    course: CourseSummary,
    hasAttendanceSessions: Boolean,
): List<Pair<String, String>> = buildList {
    add("overview" to "Overview")
    add("modules" to "Modules")
    add("files" to "Files")
    if (course.viewerIsStudent) add("grades" to "Grades")
    if (course.isOfficeHoursEnabled) add("officehours" to "Office Hours")
    if (course.viewerIsStaff || hasAttendanceSessions) add("attendance" to "Attendance")
    if (course.viewerIsStaff) add("grading" to "Grading")
}

private suspend fun refreshProgress(
    accessToken: String?,
    course: CourseSummary,
    offline: OfflineService,
    onResult: (ModulesProgressSnapshot?) -> Unit,
) {
    if (!course.viewerIsStudent) {
        onResult(null)
        return
    }
    val token = accessToken ?: return
    runCatching {
        offline.cachedFetch(
            key = OfflineCacheKey.modulesProgress(course.courseCode),
            accessToken = token,
            serializer = ModulesProgressSnapshot.serializer(),
        ) {
            LmsApi.fetchModulesProgress(course.courseCode, token) ?: ModulesProgressSnapshot()
        }.first
    }.onSuccess { snapshot ->
        onResult(
            if (snapshot.modules.isEmpty() && snapshot.enrollmentId.isEmpty()) null else snapshot,
        )
    }.onFailure {
        onResult(null)
    }
}
