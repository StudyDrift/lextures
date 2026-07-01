package com.lextures.android.features.courses

import androidx.compose.foundation.layout.Arrangement
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
import androidx.compose.material.icons.filled.Search
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
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
import com.lextures.android.core.navigation.CourseWorkspaceContext
import com.lextures.android.core.navigation.CourseWorkspaceSection
import com.lextures.android.core.navigation.MobileDestinations
import com.lextures.android.core.navigation.MobilePlatformFeatures
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
    modifier: Modifier = Modifier,
) {
    val accessToken by session.accessToken.collectAsState()
    val context = LocalContext.current
    val offline = remember { OfflineService.get(context) }
    val isOnline by offline.networkMonitor.isOnline.collectAsState()

    var section by rememberSaveable(course.courseCode) {
        mutableStateOf(initialSection?.name ?: "modules")
    }
    var showOverflow by remember { mutableStateOf(false) }
    var items by remember { mutableStateOf<List<CourseStructureItem>>(emptyList()) }
    var progress by remember { mutableStateOf<ModulesProgressSnapshot?>(null) }
    var cacheLabel by remember { mutableStateOf<String?>(null) }
    var hasAttendanceSessions by remember { mutableStateOf(false) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var loading by remember { mutableStateOf(true) }

    var openItem by remember { mutableStateOf<CourseStructureItem?>(null) }
    var lockedItem by remember { mutableStateOf<CourseStructureItem?>(null) }
    var openAttendanceSession by remember { mutableStateOf<AttendanceSession?>(null) }
    var openTakeAttendance by remember { mutableStateOf<TakeAttendanceRequest?>(null) }
    var openBacklogItem by remember { mutableStateOf<GradingBacklogItem?>(null) }
    var openFilePreview by remember { mutableStateOf<FilePreviewTarget?>(null) }
    var openGradeFeedback by remember { mutableStateOf<GradeFeedbackRoute?>(null) }
    var showCourseSearch by remember { mutableStateOf(false) }

    val emptyCourseTitle = moduleEmptyCourseTitle()
    val emptyCourseHint = moduleEmptyCourseHint()
    val groups = remember(items) { ModuleContentLogic.buildModuleGroups(items) }

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

    LaunchedEffect(accessToken, course.courseCode) {
        val token = accessToken ?: return@LaunchedEffect
        loading = true
        errorMessage = null
        try {
            hasAttendanceSessions = runCatching {
                LmsApi.fetchAttendanceSessions(course.courseCode, token).isNotEmpty()
            }.getOrDefault(false)
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
        hasAttendanceSessions = hasAttendanceSessions,
        hasLibraryResources = LibraryResourceLogic.hasLibraryResources(items),
        platformFeatures = shell?.platformFeatures ?: MobilePlatformFeatures(),
    )
    val allSections = MobileDestinations.courseWorkspaceSections(workspaceContext)
    val chipSplit = MobileDestinations.splitCourseChips(allSections)
    val selectedSection = CourseWorkspaceSection.entries.firstOrNull { it.name == section }
        ?: CourseWorkspaceSection.Modules

    if (showOverflow) {
        AlertDialog(
            onDismissRequest = { showOverflow = false },
            title = { Text("More") },
            text = {
                Column {
                    chipSplit.overflow.forEach { item ->
                        TextButton(onClick = {
                            section = item.name
                            showOverflow = false
                        }) {
                            Text(item.name)
                        }
                    }
                }
            },
            confirmButton = {
                TextButton(onClick = { showOverflow = false }) { Text("Close") }
            },
        )
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

    Column(modifier = modifier) {
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
                if (shell?.iaRedesignEnabled == true) {
                    CourseWorkspaceNav(
                        sections = chipSplit.visible,
                        overflow = chipSplit.overflow,
                        selected = selectedSection,
                        onSelect = { section = it.name },
                        onOpenOverflow = { showOverflow = true },
                    )
                } else {
                    LmsSegmentedChips(
                        options = legacySections(course, hasAttendanceSessions),
                        selectedId = section,
                        onSelect = { section = it },
                    )
                }
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

            if (shell?.iaRedesignEnabled == true) {
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
                    CourseWorkspaceSection.Discussions,
                    CourseWorkspaceSection.Feed,
                    CourseWorkspaceSection.Live,
                    CourseWorkspaceSection.People,
                    CourseWorkspaceSection.Evaluations,
                    -> item { CourseDestinationPlaceholder(section = selectedSection) }
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
