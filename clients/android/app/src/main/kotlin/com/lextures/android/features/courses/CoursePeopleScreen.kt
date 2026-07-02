package com.lextures.android.features.courses

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.ChevronRight
import androidx.compose.material.icons.filled.People
import androidx.compose.material.icons.filled.Search
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.OfflineBanner
import com.lextures.android.core.design.ProfileAvatar
import com.lextures.android.core.design.StalenessChip
import com.lextures.android.core.design.accentColor
import com.lextures.android.core.design.isDarkTheme
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.lms.CourseEnrollment
import com.lextures.android.core.lms.CoursePeopleGroupKind
import com.lextures.android.core.lms.CoursePeopleLogic
import com.lextures.android.core.lms.CoursePeopleRoleFilter
import com.lextures.android.core.lms.CourseSection
import com.lextures.android.core.lms.CourseSummary
import com.lextures.android.core.lms.EnrollmentMessageBody
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.LmsDates
import com.lextures.android.core.offline.OfflineCacheKey
import com.lextures.android.core.offline.OfflineService
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsEmptyState
import com.lextures.android.features.home.LmsErrorBanner
import com.lextures.android.features.home.LmsSegmentedChips
import com.lextures.android.features.home.LmsSkeletonList
import com.lextures.android.core.design.LexturesColors
import kotlinx.coroutines.launch
import kotlinx.serialization.builtins.ListSerializer

/** Staff course roster: search, filters, message, and remove (M11.4). */
@Composable
fun CoursePeopleSection(
    session: AuthSession,
    course: CourseSummary,
    modifier: Modifier = Modifier,
) {
    val accessToken by session.accessToken.collectAsState()
    val context = LocalContext.current
    val appContext = context.applicationContext
    val offline = remember { OfflineService.get(context) }
    val isOnline by offline.networkMonitor.isOnline.collectAsState()
    val scope = rememberCoroutineScope()

    var enrollments by remember { mutableStateOf<List<CourseEnrollment>>(emptyList()) }
    var sections by remember { mutableStateOf<List<CourseSection>>(emptyList()) }
    var permissions by remember { mutableStateOf<List<String>>(emptyList()) }
    var cacheLabel by remember { mutableStateOf<String?>(null) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var successMessage by remember { mutableStateOf<String?>(null) }
    var loading by remember { mutableStateOf(true) }
    var searchText by remember { mutableStateOf("") }
    var roleFilter by remember { mutableStateOf(CoursePeopleRoleFilter.All) }
    var sectionFilter by remember { mutableStateOf("") }
    var selectedEnrollment by remember { mutableStateOf<CourseEnrollment?>(null) }
    var removeTarget by remember { mutableStateOf<CourseEnrollment?>(null) }
    var composeMode by remember { mutableStateOf(false) }
    var messageSubject by remember { mutableStateOf("") }
    var messageBody by remember { mutableStateOf("") }
    var actionBusy by remember { mutableStateOf(false) }

    val canRemove = remember(course.courseCode, permissions) {
        CoursePeopleLogic.canUpdateEnrollments(course.courseCode, permissions)
    }

    val filteredEnrollments = remember(enrollments, searchText, roleFilter, sectionFilter) {
        CoursePeopleLogic.filter(
            enrollments = enrollments,
            search = searchText,
            roleFilter = roleFilter,
            sectionId = sectionFilter.ifEmpty { null },
        )
    }

    val groupedSections = remember(filteredEnrollments) {
        CoursePeopleLogic.groupedSections(filteredEnrollments)
    }

    LaunchedEffect(accessToken, course.courseCode) {
        val token = accessToken ?: return@LaunchedEffect
        if (!course.viewerIsStaff) {
            loading = false
            return@LaunchedEffect
        }
        loading = true
        errorMessage = null
        try {
            permissions = runCatching { LmsApi.fetchMyPermissions(token) }.getOrDefault(emptyList())
            val result = offline.cachedFetch(
                key = OfflineCacheKey.courseEnrollments(course.courseCode),
                accessToken = token,
                serializer = ListSerializer(CourseEnrollment.serializer()),
            ) {
                LmsApi.fetchCourseEnrollments(course.courseCode, token)
            }
            enrollments = result.first
            val cached = result.second
            cacheLabel = if (cached != null && cached.isStale(isOnline)) cached.lastUpdatedLabel() else null
            if (course.isSectionsEnabled) {
                sections = runCatching { LmsApi.fetchCourseSections(course.courseCode, token) }.getOrDefault(emptyList())
            }
        } catch (e: Exception) {
            errorMessage = session.mapError(e)
        } finally {
            loading = false
        }
    }

    Column(modifier = modifier, verticalArrangement = Arrangement.spacedBy(10.dp)) {
        if (!isOnline) OfflineBanner()
        cacheLabel?.let { StalenessChip(label = it) }
        errorMessage?.let { LmsErrorBanner(it) }
        successMessage?.let {
            Text(
                text = it,
                fontSize = 12.sp,
                fontWeight = FontWeight.SemiBold,
                color = LexturesColors.BrandTeal,
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(12.dp),
            )
        }

        when {
            loading && enrollments.isEmpty() -> LmsSkeletonList(count = 4)
            else -> {
                OutlinedTextField(
                    value = searchText,
                    onValueChange = { searchText = it },
                    modifier = Modifier.fillMaxWidth(),
                    placeholder = { Text(L.text(R.string.mobile_people_search)) },
                    leadingIcon = { Icon(Icons.Default.Search, contentDescription = null) },
                    singleLine = true,
                )

                LmsSegmentedChips(
                    options = listOf(
                        CoursePeopleRoleFilter.All.name to L.text(R.string.mobile_people_filter_allRoles),
                        CoursePeopleRoleFilter.Staff.name to L.text(R.string.mobile_people_filter_staff),
                        CoursePeopleRoleFilter.Students.name to L.text(R.string.mobile_people_filter_students),
                    ),
                    selectedId = roleFilter.name,
                    onSelect = { selected ->
                        roleFilter = CoursePeopleRoleFilter.entries.firstOrNull { it.name == selected }
                            ?: CoursePeopleRoleFilter.All
                    },
                )

                if (course.isSectionsEnabled && sections.isNotEmpty()) {
                    LmsSegmentedChips(
                        options = listOf("" to L.text(R.string.mobile_people_filter_allSections)) +
                            sections.map { it.id to it.displayName },
                        selectedId = sectionFilter,
                        onSelect = { sectionFilter = it },
                    )
                }

                if (filteredEnrollments.isEmpty()) {
                    LmsEmptyState(
                        icon = Icons.Default.People,
                        title = if (enrollments.isEmpty()) {
                            L.text(R.string.mobile_people_empty)
                        } else {
                            L.text(R.string.mobile_people_noResults)
                        },
                        message = if (enrollments.isEmpty()) {
                            L.text(R.string.mobile_people_emptyHint)
                        } else {
                            L.text(R.string.mobile_people_noResultsHint)
                        },
                    )
                } else {
                    groupedSections.forEach { group ->
                        LmsCard {
                            Text(
                                text = groupTitle(group.kind),
                                fontSize = 17.sp,
                                fontWeight = FontWeight.SemiBold,
                                color = textPrimary(),
                            )
                            group.enrollments.forEachIndexed { index, enrollment ->
                                if (index > 0) HorizontalDivider()
                                RosterRow(
                                    enrollment = enrollment,
                                    onClick = {
                                        selectedEnrollment = enrollment
                                        composeMode = false
                                        messageSubject = ""
                                        messageBody = ""
                                        successMessage = null
                                    },
                                )
                            }
                        }
                    }
                }
            }
        }
    }

    selectedEnrollment?.let { enrollment ->
        EnrollmentDetailDialog(
            enrollment = enrollment,
            composeMode = composeMode,
            messageSubject = messageSubject,
            messageBody = messageBody,
            actionBusy = actionBusy,
            canRemove = canRemove,
            isOnline = isOnline,
            onDismiss = { selectedEnrollment = null },
            onCompose = { composeMode = true },
            onSubjectChange = { messageSubject = it },
            onBodyChange = { messageBody = it },
            onSend = {
                scope.launch {
                    val token = accessToken ?: return@launch
                    if (!isOnline) return@launch
                    actionBusy = true
                    errorMessage = null
                    try {
                        LmsApi.sendEnrollmentMessage(
                            courseCode = course.courseCode,
                            enrollmentId = enrollment.id,
                            payload = EnrollmentMessageBody(
                                subject = messageSubject.trim(),
                                body = messageBody,
                            ),
                            accessToken = token,
                        )
                        selectedEnrollment = null
                        composeMode = false
                        successMessage = appContext.getString(R.string.mobile_people_message_success)
                    } catch (e: Exception) {
                        errorMessage = session.mapError(e)
                    } finally {
                        actionBusy = false
                    }
                }
            },
            onRemove = {
                removeTarget = enrollment
            },
        )
    }

    removeTarget?.let { target ->
        AlertDialog(
            onDismissRequest = { removeTarget = null },
            title = { Text(L.text(R.string.mobile_people_remove_confirmTitle)) },
            text = {
                Text(
                    L.format(
                        R.string.mobile_people_remove_confirmMessage,
                        CoursePeopleLogic.displayName(target),
                    ),
                )
            },
            confirmButton = {
                TextButton(
                    onClick = {
                        scope.launch {
                            val token = accessToken ?: return@launch
                            if (!isOnline) return@launch
                            actionBusy = true
                            errorMessage = null
                            removeTarget = null
                            try {
                                LmsApi.removeCourseEnrollment(
                                    courseCode = course.courseCode,
                                    enrollmentId = target.id,
                                    accessToken = token,
                                )
                                enrollments = enrollments.filterNot { it.id == target.id }
                                if (selectedEnrollment?.id == target.id) {
                                    selectedEnrollment = null
                                }
                                successMessage = appContext.getString(R.string.mobile_people_remove_success)
                            } catch (e: Exception) {
                                errorMessage = session.mapError(e)
                            } finally {
                                actionBusy = false
                            }
                        }
                    },
                ) {
                    Text(L.text(R.string.mobile_people_remove_confirm))
                }
            },
            dismissButton = {
                TextButton(onClick = { removeTarget = null }) {
                    Text(L.text(R.string.mobile_people_remove_cancel))
                }
            },
        )
    }
}

@Composable
private fun groupTitle(kind: CoursePeopleGroupKind): String =
    when (kind) {
        CoursePeopleGroupKind.Teachers -> L.text(R.string.mobile_people_role_teachers)
        CoursePeopleGroupKind.Tas -> L.text(R.string.mobile_people_role_tas)
        CoursePeopleGroupKind.Students -> L.text(R.string.mobile_people_role_students)
        CoursePeopleGroupKind.Other -> L.text(R.string.mobile_people_role_other)
    }

@Composable
private fun RosterRow(
    enrollment: CourseEnrollment,
    onClick: () -> Unit,
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clickable(onClick = onClick)
            .padding(vertical = 6.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        ProfileAvatar(
            avatarUrl = enrollment.avatarUrl,
            initials = CoursePeopleLogic.initials(enrollment),
            size = 40.dp,
        )
        Column(modifier = Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(2.dp)) {
            Row(horizontalArrangement = Arrangement.spacedBy(6.dp), verticalAlignment = Alignment.CenterVertically) {
                Text(
                    text = CoursePeopleLogic.displayName(enrollment),
                    fontSize = 14.sp,
                    fontWeight = FontWeight.Medium,
                    color = textPrimary(),
                )
                if (enrollment.invitationPending == true) {
                    InvitedBadge()
                }
            }
            Text(
                text = localizedRoleLabel(enrollment),
                fontSize = 12.sp,
                color = textSecondary(),
            )
            CoursePeopleLogic.sectionLabel(enrollment)?.let { section ->
                Text(
                    text = L.format(R.string.mobile_people_section, section),
                    fontSize = 11.sp,
                    color = textSecondary(),
                )
            }
        }
        Icon(
            imageVector = Icons.Default.ChevronRight,
            contentDescription = null,
            tint = textSecondary(),
        )
    }
}

@Composable
private fun localizedRoleLabel(enrollment: CourseEnrollment): String {
    val custom = enrollment.roleDisplay?.trim().orEmpty()
    if (custom.isNotEmpty()) return custom
    return when (CoursePeopleLogic.normalizedRole(enrollment.role)) {
        "owner", "teacher", "instructor" -> L.text(R.string.mobile_people_role_teacher)
        "ta" -> L.text(R.string.mobile_people_role_ta)
        "student" -> L.text(R.string.mobile_people_role_student)
        else -> enrollment.role
    }
}

@Composable
private fun InvitedBadge() {
    Text(
        text = L.text(R.string.mobile_people_invited),
        fontSize = 11.sp,
        fontWeight = FontWeight.SemiBold,
        color = LexturesColors.Amber,
        modifier = Modifier.padding(horizontal = 6.dp, vertical = 2.dp),
    )
}

@Composable
private fun EnrollmentDetailDialog(
    enrollment: CourseEnrollment,
    composeMode: Boolean,
    messageSubject: String,
    messageBody: String,
    actionBusy: Boolean,
    canRemove: Boolean,
    isOnline: Boolean,
    onDismiss: () -> Unit,
    onCompose: () -> Unit,
    onSubjectChange: (String) -> Unit,
    onBodyChange: (String) -> Unit,
    onSend: () -> Unit,
    onRemove: () -> Unit,
) {
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text(L.text(R.string.mobile_people_detail_title)) },
        text = {
            Column(
                modifier = Modifier.verticalScroll(rememberScrollState()),
                verticalArrangement = Arrangement.spacedBy(8.dp),
            ) {
                Text(
                    text = CoursePeopleLogic.displayName(enrollment),
                    fontWeight = FontWeight.SemiBold,
                    color = textPrimary(),
                )
                Text(
                    text = localizedRoleLabel(enrollment),
                    fontSize = 13.sp,
                    color = textSecondary(),
                )
                if (enrollment.invitationPending == true) {
                    InvitedBadge()
                }
                CoursePeopleLogic.sectionLabel(enrollment)?.let { section ->
                    DetailLine(L.text(R.string.mobile_people_detail_section), section)
                }
                enrollment.lastCourseAccessAt?.takeIf { it.isNotEmpty() }?.let { lastAccess ->
                    DetailLine(
                        L.text(R.string.mobile_people_detail_lastAccess),
                        LmsDates.relative(lastAccess),
                    )
                }
                enrollment.state?.takeIf { it.isNotEmpty() }?.let { state ->
                    DetailLine(
                        L.text(R.string.mobile_people_detail_state),
                        state.replaceFirstChar { if (it.isLowerCase()) it.titlecase() else it.toString() },
                    )
                }

                if (composeMode) {
                    OutlinedTextField(
                        value = messageSubject,
                        onValueChange = onSubjectChange,
                        label = { Text(L.text(R.string.mobile_people_message_subject)) },
                        modifier = Modifier.fillMaxWidth(),
                    )
                    OutlinedTextField(
                        value = messageBody,
                        onValueChange = onBodyChange,
                        label = { Text(L.text(R.string.mobile_people_message_body)) },
                        modifier = Modifier.fillMaxWidth(),
                        minLines = 4,
                    )
                } else if (!isOnline) {
                    Text(
                        text = L.text(R.string.mobile_people_message_offline),
                        fontSize = 12.sp,
                        color = textSecondary(),
                    )
                }
            }
        },
        confirmButton = {
            when {
                composeMode -> TextButton(
                    onClick = onSend,
                    enabled = !actionBusy && messageBody.trim().isNotEmpty() && isOnline,
                ) {
                    Text(
                        if (actionBusy) {
                            L.text(R.string.mobile_people_message_sending)
                        } else {
                            L.text(R.string.mobile_people_message_send)
                        },
                    )
                }
                else -> TextButton(onClick = onCompose, enabled = isOnline) {
                    Text(L.text(R.string.mobile_people_message))
                }
            }
        },
        dismissButton = {
            Row {
                if (!composeMode && canRemove) {
                    TextButton(onClick = onRemove, enabled = isOnline) {
                        Text(L.text(R.string.mobile_people_remove))
                    }
                }
                TextButton(onClick = onDismiss) {
                    Text(L.text(R.string.mobile_people_detail_done))
                }
            }
        },
    )
}

@Composable
private fun DetailLine(label: String, value: String) {
    Column {
        Text(text = label, fontSize = 11.sp, fontWeight = FontWeight.SemiBold, color = textSecondary())
        Text(text = value, fontSize = 13.sp, color = textPrimary())
    }
}