package com.lextures.android.features.courses

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Layers
import androidx.compose.material3.AlertDialog
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
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontStyle
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.activity.compose.BackHandler
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.LexturesType
import com.lextures.android.core.design.coverBrush
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.lms.AttendanceSession
import androidx.compose.ui.platform.LocalContext
import com.lextures.android.core.design.OfflineBanner
import com.lextures.android.core.design.StalenessChip
import com.lextures.android.core.lms.CourseStructureItem
import com.lextures.android.core.lms.CourseSummary
import com.lextures.android.core.lms.GradingBacklogItem
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.LmsDates
import com.lextures.android.core.offline.OfflineCacheKey
import com.lextures.android.core.offline.OfflineService
import com.lextures.android.features.grading.GradingBacklogSection
import com.lextures.android.features.grading.SubmissionsListScreen
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsEmptyState
import com.lextures.android.features.home.LmsErrorBanner
import com.lextures.android.features.home.LmsSegmentedChips
import com.lextures.android.features.home.LmsSkeletonList
import com.lextures.android.core.lms.ModuleContentLogic
import com.lextures.android.core.lms.ModulesProgressSnapshot
import com.lextures.android.core.i18n.L

/**
 * Course home: gradient hero + segmented sections
 * (Overview · Modules · Grades · Attendance · Grading by role).
 */
@Composable
fun CourseDetailScreen(
    session: AuthSession,
    course: CourseSummary,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val accessToken by session.accessToken.collectAsState()
    val context = LocalContext.current
    val offline = remember { OfflineService.get(context) }
    val isOnline by offline.networkMonitor.isOnline.collectAsState()

    var section by rememberSaveable(course.courseCode) { mutableStateOf("modules") }
    var items by remember { mutableStateOf<List<CourseStructureItem>>(emptyList()) }
    var progress by remember { mutableStateOf<ModulesProgressSnapshot?>(null) }
    var cacheLabel by remember { mutableStateOf<String?>(null) }
    var hasAttendanceSessions by remember { mutableStateOf(false) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var loading by remember { mutableStateOf(true) }

    var openItem by remember { mutableStateOf<CourseStructureItem?>(null) }
    var lockDialog by remember { mutableStateOf<Pair<String, String>?>(null) }
    var openAttendanceSession by remember { mutableStateOf<AttendanceSession?>(null) }
    var openBacklogItem by remember { mutableStateOf<GradingBacklogItem?>(null) }

    BackHandler(onBack = onBack)

    openItem?.let { selected ->
        ModuleItemRouteScreen(
            session = session,
            course = course,
            item = selected,
            onBack = { openItem = null },
            onProgressChanged = { refreshProgress(accessToken, course, offline) { progress = it } },
            modifier = modifier,
        )
        return
    }

    lockDialog?.let { (title, message) ->
        AlertDialog(
            onDismissRequest = { lockDialog = null },
            title = { Text(title) },
            text = { Text(message) },
            confirmButton = {
                TextButton(onClick = { lockDialog = null }) { Text("OK") }
            },
        )
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

    val sections = buildList {
        add("overview" to "Overview")
        add("modules" to "Modules")
        if (course.viewerIsStudent) add("grades" to "Grades")
        if (course.viewerIsStaff || hasAttendanceSessions) add("attendance" to "Attendance")
        if (course.viewerIsStaff) add("grading" to "Grading")
    }

    val groups = remember(items) { ModuleContentLogic.buildModuleGroups(items) }

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
            )
        }

        LazyColumn(
            modifier = Modifier.fillMaxSize(),
            contentPadding = PaddingValues(16.dp),
            verticalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            item {
                // Gradient cover banner — matches the course's tile color across the app.
                Box(
                    modifier = Modifier
                        .fillMaxWidth()
                        .clip(RoundedCornerShape(24.dp))
                        .background(coverBrush(course.courseCode)),
                ) {
                    Box(
                        modifier = Modifier
                            .size(140.dp)
                            .offset(x = 260.dp, y = (-52).dp)
                            .clip(CircleShape)
                            .background(Color.White.copy(alpha = 0.08f)),
                    )
                    Column(
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(20.dp),
                        verticalArrangement = Arrangement.spacedBy(7.dp),
                    ) {
                        Text(
                            text = course.courseCode.uppercase(),
                            fontSize = 11.sp,
                            fontWeight = FontWeight.SemiBold,
                            letterSpacing = 1.2.sp,
                            color = Color.White.copy(alpha = 0.8f),
                        )
                        Text(
                            text = course.title,
                            style = LexturesType.display(22),
                            color = Color.White,
                        )
                        if (course.description.isNotEmpty()) {
                            Text(
                                text = course.description,
                                fontSize = 13.sp,
                                color = Color.White.copy(alpha = 0.85f),
                                maxLines = 3,
                                overflow = TextOverflow.Ellipsis,
                            )
                        }
                        Row(
                            modifier = Modifier.padding(top = 4.dp),
                            horizontalArrangement = Arrangement.spacedBy(6.dp),
                        ) {
                            LmsDates.parse(course.startsAt)?.let {
                                HeroChip("Starts ${LmsDates.shortDate(course.startsAt)}")
                            }
                            course.viewerEnrollmentRoles.orEmpty().forEach { role ->
                                HeroChip(if (role.length <= 2) role.uppercase() else role.replaceFirstChar { it.uppercase() })
                            }
                        }
                    }
                }
            }

            item {
                LmsSegmentedChips(
                    options = sections,
                    selectedId = section,
                    onSelect = { section = it },
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

            when (section) {
                "overview" -> item {
                    CourseSyllabusSection(session = session, course = course)
                }

                "grades" -> item {
                    CourseGradesSection(session = session, course = course)
                }

                "attendance" -> item {
                    CourseAttendanceSection(
                        session = session,
                        course = course,
                        onOpenSession = { openAttendanceSession = it },
                    )
                }

                "grading" -> item {
                    GradingBacklogSection(
                        session = session,
                        course = course,
                        onOpenItem = { openBacklogItem = it },
                    )
                }

                else -> {
                    if (loading && items.isEmpty()) {
                        item { LmsSkeletonList(count = 3) }
                    } else if (groups.isEmpty() && errorMessage == null) {
                        item {
                            LmsEmptyState(
                                icon = Icons.Default.Layers,
                                title = L.text("mobile.modules.emptyCourse"),
                                message = L.text("mobile.modules.emptyCourseHint"),
                            )
                        }
                    } else {
                        item {
                            ModuleList(
                                course = course,
                                groups = groups,
                                progress = progress,
                                onSelectItem = { openItem = it },
                                onLockedItem = { item, reason ->
                                    lockDialog = item.title to (
                                        reason?.message ?: L.text("mobile.modules.lockedDefault")
                                        )
                                },
                            )
                        }
                    }
                }
            }
        }
    }
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

@Composable
private fun HeroChip(text: String) {
    Text(
        text = text,
        fontSize = 12.sp,
        fontWeight = FontWeight.Medium,
        color = Color.White,
        modifier = Modifier
            .clip(RoundedCornerShape(50))
            .background(Color.White.copy(alpha = 0.16f))
            .padding(horizontal = 9.dp, vertical = 4.dp),
    )
}
