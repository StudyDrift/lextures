package com.lextures.android.features.courses.settings

import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.GridView
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.HorizontalDivider
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
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.OfflineBanner
import com.lextures.android.core.design.StalenessChip
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalLocalePreferences
import com.lextures.android.core.lms.CourseEnrollment
import com.lextures.android.core.lms.CoursePeopleLogic
import com.lextures.android.core.lms.CourseSection
import com.lextures.android.core.lms.CourseSectionsCachedPayload
import com.lextures.android.core.lms.CourseSectionsLogic
import com.lextures.android.core.lms.CourseStructureItem
import com.lextures.android.core.lms.CourseSummary
import com.lextures.android.core.lms.CreateCourseSectionBody
import com.lextures.android.core.lms.EnrollmentSectionPatchBody
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.PatchCourseSectionBody
import com.lextures.android.core.lms.SectionAssignmentOverrideBody
import com.lextures.android.core.offline.OfflineService
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsEmptyState
import com.lextures.android.features.home.LmsErrorBanner
import com.lextures.android.features.home.LmsSkeletonList
import kotlinx.coroutines.launch
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json

private val sectionsJson = Json { ignoreUnknownKeys = true }

@Composable
fun CourseSectionsSettingsScreen(
    session: AuthSession,
    course: CourseSummary,
    offline: OfflineService,
    permissions: List<String>,
    onCourseUpdated: (CourseSummary) -> Unit,
) {
    val scope = rememberCoroutineScope()
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    val isOnline by offline.networkMonitor.isOnline.collectAsState()

    var serverCourse by remember(course.courseCode) { mutableStateOf(course) }
    var sections by remember { mutableStateOf<List<CourseSection>>(emptyList()) }
    var enrollments by remember { mutableStateOf<List<CourseEnrollment>>(emptyList()) }
    var assignments by remember { mutableStateOf<List<CourseStructureItem>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var loadError by remember { mutableStateOf<String?>(null) }
    var actionError by remember { mutableStateOf<String?>(null) }
    var actionSuccess by remember { mutableStateOf<String?>(null) }
    var cacheLabel by remember { mutableStateOf<String?>(null) }
    var busy by remember { mutableStateOf(false) }

    var newSectionCode by remember { mutableStateOf("") }
    var newSectionName by remember { mutableStateOf("") }
    var overrideSectionId by remember { mutableStateOf("") }
    var overrideItemId by remember { mutableStateOf("") }
    var overrideDue by remember { mutableStateOf("") }

    var selectedSection by remember { mutableStateOf<CourseSection?>(null) }
    var editSectionCode by remember { mutableStateOf("") }
    var editSectionName by remember { mutableStateOf("") }
    var pendingArchive by remember { mutableStateOf<CourseSection?>(null) }
    var pendingMoveEnrollment by remember { mutableStateOf<CourseEnrollment?>(null) }

    val showEditors = CourseSectionsLogic.shouldShowEditors(serverCourse.isSectionsEnabled)
    val canAssignStudents = CourseSectionsLogic.canAssignStudents(course.courseCode, permissions)
    val activeSections = remember(sections) { CourseSectionsLogic.activeSections(sections) }

    suspend fun reload() {
        val token = session.accessToken.value ?: return
        loading = sections.isEmpty()
        loadError = null
        runCatching {
            val result = offline.cachedFetch(
                key = CourseSectionsLogic.cacheKeySections(course.courseCode),
                accessToken = token,
                serializer = CourseSectionsCachedPayload.serializer(),
            ) { LmsApi.fetchCourseSectionsPayload(course.courseCode, token) }
            sections = result.first.sections
            enrollments = result.first.enrollments
            assignments = result.first.assignments
            cacheLabel = result.second?.takeIf { it.isStale(isOnline) }?.lastUpdatedLabel()
            LmsApi.fetchCourse(course.courseCode, token).also {
                serverCourse = it
                onCourseUpdated(it)
            }
        }.onFailure { loadError = session.mapError(it) }
        loading = false
    }

    LaunchedEffect(course.courseCode, showEditors) {
        if (showEditors) reload()
        else loading = false
    }

    if (!showEditors) {
        LmsEmptyState(
            icon = Icons.Default.GridView,
            title = L.text(R.string.mobile_courseSettings_sections_disabledTitle),
            message = L.text(R.string.mobile_courseSettings_sections_disabledMessage),
        )
        return
    }

    LazyColumn(
        modifier = Modifier.fillMaxSize(),
        verticalArrangement = Arrangement.spacedBy(12.dp),
        contentPadding = androidx.compose.foundation.layout.PaddingValues(16.dp),
    ) {
        if (!isOnline) item { OfflineBanner() }
        cacheLabel?.let { label -> item { StalenessChip(label = label) } }
        loadError?.let { msg -> item { LmsErrorBanner(msg) } }
        actionError?.let { msg -> item { LmsErrorBanner(msg) } }
        actionSuccess?.let { msg ->
            item {
                Text(msg, fontWeight = FontWeight.SemiBold, color = androidx.compose.ui.graphics.Color(0xFF0D9488))
            }
        }
        CourseSectionsLogic.mutationsDisabledReason(isOnline)?.let { reason ->
            item { Text(L.text(R.string.mobile_courseSettings_sections_offlineMutationsDisabled)) }
        }

        if (loading) {
            item { LmsSkeletonList(count = 3) }
        } else {
            item {
                LmsCard {
                    Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                        Text(L.text(R.string.mobile_courseSettings_sections_listTitle), fontWeight = FontWeight.SemiBold)
                        Text(L.text(R.string.mobile_courseSettings_sections_listDescription))
                        if (sections.isEmpty()) {
                            Text(L.text(R.string.mobile_courseSettings_sections_empty))
                        } else {
                            sections.forEach { section ->
                                Row(
                                    modifier = Modifier
                                        .fillMaxWidth()
                                        .clickable {
                                            selectedSection = section
                                            editSectionCode = section.sectionCode
                                            editSectionName = section.name.orEmpty()
                                        }
                                        .padding(vertical = 6.dp),
                                    horizontalArrangement = Arrangement.SpaceBetween,
                                ) {
                                    Column {
                                        Text(section.displayLabel, fontWeight = FontWeight.Medium)
                                        Text(
                                            L.format(
                                                R.string.mobile_courseSettings_sections_rosterCount,
                                                CourseSectionsLogic.rosterCount(section.id, enrollments),
                                            ),
                                        )
                                    }
                                }
                            }
                        }
                    }
                }
            }

            item {
                LmsCard {
                    Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                        Text(L.text(R.string.mobile_courseSettings_sections_createTitle), fontWeight = FontWeight.SemiBold)
                        OutlinedTextField(
                            value = newSectionCode,
                            onValueChange = { newSectionCode = it },
                            modifier = Modifier.fillMaxWidth(),
                            label = { Text(L.text(R.string.mobile_courseSettings_sections_sectionCode)) },
                        )
                        OutlinedTextField(
                            value = newSectionName,
                            onValueChange = { newSectionName = it },
                            modifier = Modifier.fillMaxWidth(),
                            label = { Text(L.text(R.string.mobile_courseSettings_sections_sectionNameOptional)) },
                        )
                        Button(
                            onClick = {
                                scope.launch {
                                    val token = session.accessToken.value ?: return@launch
                                    val code = newSectionCode.trim()
                                    if (CourseSectionsLogic.validateCreateSection(code) != null) return@launch
                                    busy = true
                                    runCatching {
                                        offline.enqueueMutation(
                                            method = "POST",
                                            path = "/api/v1/courses/${course.courseCode}/sections",
                                            bodyJson = sectionsJson.encodeToString(
                                                CreateCourseSectionBody(
                                                    sectionCode = code,
                                                    name = newSectionName.trim().ifEmpty { null },
                                                ),
                                            ),
                                            label = L.text(context, localePrefs, R.string.mobile_courseSettings_sections_createLabel),
                                            accessToken = token,
                                            idempotencyKey = CourseSectionsLogic.createSectionIdempotencyKey(course.courseCode, code),
                                        )
                                        newSectionCode = ""
                                        newSectionName = ""
                                        actionSuccess = L.text(R.string.mobile_courseSettings_sections_createSuccess)
                                        reload()
                                    }.onFailure { actionError = session.mapError(it) }
                                    busy = false
                                }
                            },
                            enabled = !busy && newSectionCode.trim().isNotEmpty(),
                            modifier = Modifier.fillMaxWidth(),
                        ) {
                            Text(L.text(R.string.mobile_courseSettings_sections_createButton))
                        }
                    }
                }
            }

            item {
                LmsCard {
                    Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                        Text(L.text(R.string.mobile_courseSettings_sections_overrideTitle), fontWeight = FontWeight.SemiBold)
                        Text(L.text(R.string.mobile_courseSettings_sections_overrideDescription))
                        activeSections.forEach { section ->
                            Text(
                                text = section.displayLabel,
                                fontWeight = if (overrideSectionId == section.id) FontWeight.Bold else FontWeight.Normal,
                                modifier = Modifier
                                    .fillMaxWidth()
                                    .clickable { overrideSectionId = section.id }
                                    .padding(vertical = 4.dp),
                            )
                        }
                        assignments.forEach { item ->
                            Text(
                                text = item.title,
                                fontWeight = if (overrideItemId == item.id) FontWeight.Bold else FontWeight.Normal,
                                modifier = Modifier
                                    .fillMaxWidth()
                                    .clickable { overrideItemId = item.id }
                                    .padding(vertical = 4.dp),
                            )
                        }
                        OutlinedTextField(
                            value = overrideDue,
                            onValueChange = { overrideDue = it },
                            modifier = Modifier.fillMaxWidth(),
                            label = { Text(L.text(R.string.mobile_courseSettings_sections_overrideDueLocal)) },
                        )
                        Button(
                            onClick = {
                                scope.launch {
                                    val token = session.accessToken.value ?: return@launch
                                    val body = CourseSectionsLogic.buildOverrideBody(overrideDue)
                                        ?: run {
                                            actionError = L.text(R.string.mobile_courseSettings_sections_overrideInvalidDate)
                                            return@launch
                                        }
                                    busy = true
                                    runCatching {
                                        offline.enqueueMutation(
                                            method = "PUT",
                                            path = "/api/v1/sections/$overrideSectionId/overrides/$overrideItemId",
                                            bodyJson = sectionsJson.encodeToString(body),
                                            label = L.text(context, localePrefs, R.string.mobile_courseSettings_sections_overrideLabel),
                                            accessToken = token,
                                            idempotencyKey = CourseSectionsLogic.overrideIdempotencyKey(overrideSectionId, overrideItemId),
                                        )
                                        overrideDue = ""
                                        actionSuccess = L.text(R.string.mobile_courseSettings_sections_overrideSuccess)
                                    }.onFailure { actionError = session.mapError(it) }
                                    busy = false
                                }
                            },
                            enabled = !busy && overrideSectionId.isNotEmpty() && overrideItemId.isNotEmpty(),
                            modifier = Modifier.fillMaxWidth(),
                        ) {
                            Text(L.text(R.string.mobile_courseSettings_sections_overrideSave))
                        }
                    }
                }
            }

            item {
                CourseCrossListingSection(
                    session = session,
                    course = serverCourse,
                    sections = sections,
                    permissions = permissions,
                    onReload = { reload() },
                )
            }
        }
    }

    selectedSection?.let { section ->
        AlertDialog(
            onDismissRequest = { selectedSection = null },
            title = { Text(section.displayLabel) },
            text = {
                Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    OutlinedTextField(
                        value = editSectionCode,
                        onValueChange = { editSectionCode = it },
                        modifier = Modifier.fillMaxWidth(),
                        label = { Text(L.text(R.string.mobile_courseSettings_sections_sectionCode)) },
                    )
                    OutlinedTextField(
                        value = editSectionName,
                        onValueChange = { editSectionName = it },
                        modifier = Modifier.fillMaxWidth(),
                        label = { Text(L.text(R.string.mobile_courseSettings_sections_sectionNameOptional)) },
                    )
                    if (section.isActive) {
                        TextButton(
                            onClick = {
                                scope.launch {
                                    val token = session.accessToken.value ?: return@launch
                                    busy = true
                                    runCatching {
                                        offline.enqueueMutation(
                                            method = "PATCH",
                                            path = "/api/v1/courses/${course.courseCode}/sections/${section.id}",
                                            bodyJson = sectionsJson.encodeToString(
                                                PatchCourseSectionBody(
                                                    sectionCode = editSectionCode.trim().ifEmpty { null },
                                                    name = editSectionName.trim().ifEmpty { null },
                                                ),
                                            ),
                                            label = L.text(context, localePrefs, R.string.mobile_courseSettings_sections_renameLabel),
                                            accessToken = token,
                                            idempotencyKey = CourseSectionsLogic.patchSectionIdempotencyKey(course.courseCode, section.id),
                                        )
                                        actionSuccess = L.text(R.string.mobile_courseSettings_sections_renameSuccess)
                                        reload()
                                    }.onFailure { actionError = session.mapError(it) }
                                    busy = false
                                }
                            },
                        ) { Text(L.text(R.string.mobile_courseSettings_sections_saveRename)) }
                        TextButton(onClick = { pendingArchive = section }) {
                            Text(L.text(R.string.mobile_courseSettings_sections_archive))
                        }
                    }
                    Text(L.text(R.string.mobile_courseSettings_sections_sectionRosterTitle), fontWeight = FontWeight.SemiBold)
                    enrollments.filter { it.sectionId == section.id && CoursePeopleLogic.isStudentRole(it.role) }
                        .forEach { enrollment ->
                            Row(
                                modifier = Modifier.fillMaxWidth(),
                                horizontalArrangement = Arrangement.SpaceBetween,
                            ) {
                                Text(CoursePeopleLogic.displayName(enrollment))
                                if (canAssignStudents && section.isActive) {
                                    TextButton(onClick = { pendingMoveEnrollment = enrollment }) {
                                        Text(L.text(R.string.mobile_courseSettings_sections_moveStudent))
                                    }
                                }
                            }
                        }
                }
            },
            confirmButton = {
                TextButton(onClick = { selectedSection = null }) {
                    Text(L.text(R.string.mobile_courseSettings_sections_done))
                }
            },
        )
    }

    pendingArchive?.let { section ->
        AlertDialog(
            onDismissRequest = { pendingArchive = null },
            title = { Text(L.text(R.string.mobile_courseSettings_sections_archiveConfirmTitle)) },
            text = { Text(L.format(R.string.mobile_courseSettings_sections_archiveConfirmMessage, section.displayLabel)) },
            confirmButton = {
                TextButton(onClick = {
                    scope.launch {
                        val token = session.accessToken.value ?: return@launch
                        busy = true
                        runCatching {
                            offline.enqueueMutation(
                                method = "DELETE",
                                path = "/api/v1/courses/${course.courseCode}/sections/${section.id}",
                                bodyJson = null,
                                label = L.text(context, localePrefs, R.string.mobile_courseSettings_sections_archiveLabel),
                                accessToken = token,
                                idempotencyKey = CourseSectionsLogic.archiveSectionIdempotencyKey(course.courseCode, section.id),
                            )
                            selectedSection = null
                            actionSuccess = L.text(R.string.mobile_courseSettings_sections_archiveSuccess)
                            reload()
                        }.onFailure { actionError = session.mapError(it) }
                        busy = false
                        pendingArchive = null
                    }
                }) { Text(L.text(R.string.mobile_courseSettings_sections_archive)) }
            },
            dismissButton = {
                TextButton(onClick = { pendingArchive = null }) {
                    Text(L.text(R.string.mobile_courseSettings_sections_cancel))
                }
            },
        )
    }

    pendingMoveEnrollment?.let { enrollment ->
        AlertDialog(
            onDismissRequest = { pendingMoveEnrollment = null },
            title = { Text(L.text(R.string.mobile_courseSettings_sections_moveStudentTitle)) },
            text = {
                Column {
                    activeSections.forEach { section ->
                        Text(
                            text = section.displayLabel,
                            modifier = Modifier
                                .fillMaxWidth()
                                .clickable {
                                    scope.launch {
                                        val token = session.accessToken.value ?: return@launch
                                        busy = true
                                        runCatching {
                                            offline.enqueueMutation(
                                                method = "PATCH",
                                                path = "/api/v1/enrollments/${enrollment.id}/section",
                                                bodyJson = sectionsJson.encodeToString(
                                                    EnrollmentSectionPatchBody(section.id),
                                                ),
                                                label = L.text(context, localePrefs, R.string.mobile_courseSettings_sections_moveStudentLabel),
                                                accessToken = token,
                                                idempotencyKey = CourseSectionsLogic.enrollmentSectionIdempotencyKey(
                                                    enrollment.id,
                                                    section.id,
                                                ),
                                            )
                                            actionSuccess = L.text(R.string.mobile_courseSettings_sections_moveStudentSuccess)
                                            reload()
                                        }.onFailure { actionError = session.mapError(it) }
                                        busy = false
                                        pendingMoveEnrollment = null
                                    }
                                }
                                .padding(vertical = 6.dp),
                        )
                    }
                }
            },
            confirmButton = {},
            dismissButton = {
                TextButton(onClick = { pendingMoveEnrollment = null }) {
                    Text(L.text(R.string.mobile_courseSettings_sections_cancel))
                }
            },
        )
    }
}
