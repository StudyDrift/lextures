package com.lextures.android.features.courses.settings

import androidx.compose.foundation.clickable
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Lock
import androidx.compose.material.icons.filled.Settings
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.material3.Button
import androidx.compose.material3.FilterChip
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
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
import com.lextures.android.core.design.UnsavedChangesBanner
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalLocalePreferences
import com.lextures.android.core.lms.CourseGeneralFormState
import com.lextures.android.core.lms.CourseSettingsLogic
import com.lextures.android.core.lms.CourseStructureItem
import com.lextures.android.core.lms.CourseSummary
import com.lextures.android.core.lms.CourseUpdateRequest
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.navigation.MobilePlatformFeatures
import com.lextures.android.core.offline.OfflineService
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsEmptyState
import com.lextures.android.features.home.LmsErrorBanner
import com.lextures.android.features.home.LmsSkeletonList
import kotlinx.coroutines.launch
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json

private val settingsJson = Json { ignoreUnknownKeys = true }

@Composable
fun CourseSettingsHostScreen(
    session: AuthSession,
    course: CourseSummary,
    offline: OfflineService,
    platformFeatures: MobilePlatformFeatures,
    permissions: List<String>,
    onCourseUpdated: (CourseSummary) -> Unit,
) {
    var selected by remember { mutableStateOf(CourseSettingsLogic.CourseSettingsSection.General) }
    val canManage = CourseSettingsLogic.canManageCourse(course.courseCode, permissions)
    val sections = CourseSettingsLogic.visibleSettingsSections(course, platformFeatures)

    if (!canManage) {
        LmsEmptyState(
            icon = Icons.Filled.Lock,
            title = L.text(R.string.mobile_courseSettings_accessDeniedTitle),
            message = L.text(R.string.mobile_courseSettings_accessDeniedMessage),
        )
        return
    }

    Row(modifier = Modifier.fillMaxSize()) {
        Column(modifier = Modifier.weight(0.35f).padding(8.dp), verticalArrangement = Arrangement.spacedBy(4.dp)) {
            sections.forEach { section ->
                val selectedThis = section == selected
                Text(
                    text = L.text(sectionLabelRes(section)),
                    fontWeight = if (selectedThis) FontWeight.SemiBold else FontWeight.Normal,
                    modifier = Modifier
                        .fillMaxWidth()
                        .clickable { selected = section }
                        .padding(horizontal = 10.dp, vertical = 10.dp),
                )
            }
        }
        HorizontalDivider(modifier = Modifier.fillMaxSize())
        Column(modifier = Modifier.weight(0.65f)) {
            when (selected) {
                CourseSettingsLogic.CourseSettingsSection.General -> CourseGeneralSettingsScreen(
                    session = session,
                    course = course,
                    offline = offline,
                    onCourseUpdated = onCourseUpdated,
                )
                CourseSettingsLogic.CourseSettingsSection.ImportExport -> CourseImportExportScreen(
                    session = session,
                    course = course,
                )
                CourseSettingsLogic.CourseSettingsSection.Blueprint -> CourseBlueprintSettingsScreen(
                    session = session,
                    course = course,
                    offline = offline,
                    permissions = permissions,
                    onCourseUpdated = onCourseUpdated,
                )
                CourseSettingsLogic.CourseSettingsSection.Archive -> CourseArchivedContentScreen(
                    session = session,
                    course = course,
                    offline = offline,
                    permissions = permissions,
                )
                else -> LmsEmptyState(
                    icon = Icons.Filled.Settings,
                    title = L.text(sectionLabelRes(selected)),
                    message = L.text(R.string.mobile_courseSettings_sectionComingSoon),
                )
            }
        }
    }
}

private fun sectionLabelRes(section: CourseSettingsLogic.CourseSettingsSection): Int = when (section) {
    CourseSettingsLogic.CourseSettingsSection.General -> R.string.mobile_courseSettings_section_general
    CourseSettingsLogic.CourseSettingsSection.Features -> R.string.mobile_courseSettings_section_features
    CourseSettingsLogic.CourseSettingsSection.Sections -> R.string.mobile_courseSettings_section_sections
    CourseSettingsLogic.CourseSettingsSection.Grading -> R.string.mobile_courseSettings_section_grading
    CourseSettingsLogic.CourseSettingsSection.Outcomes -> R.string.mobile_courseSettings_section_outcomes
    CourseSettingsLogic.CourseSettingsSection.GradingAgents -> R.string.mobile_courseSettings_section_gradingAgents
    CourseSettingsLogic.CourseSettingsSection.Plagiarism -> R.string.mobile_courseSettings_section_plagiarism
    CourseSettingsLogic.CourseSettingsSection.Accessibility -> R.string.mobile_courseSettings_section_accessibility
    CourseSettingsLogic.CourseSettingsSection.Translations -> R.string.mobile_courseSettings_section_translations
    CourseSettingsLogic.CourseSettingsSection.ImportExport -> R.string.mobile_courseSettings_section_importExport
    CourseSettingsLogic.CourseSettingsSection.Blueprint -> R.string.mobile_courseSettings_section_blueprint
    CourseSettingsLogic.CourseSettingsSection.Archive -> R.string.mobile_courseSettings_section_archive
}

@Composable
fun CourseGeneralSettingsScreen(
    session: AuthSession,
    course: CourseSummary,
    offline: OfflineService,
    onCourseUpdated: (CourseSummary) -> Unit,
) {
    val scope = rememberCoroutineScope()
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    val isOnline by offline.networkMonitor.isOnline.collectAsState()
    var serverCourse by remember(course.courseCode) { mutableStateOf(course) }
    var form by remember(course.courseCode) { mutableStateOf(CourseSettingsLogic.applyCourseToForm(course)) }
    var structureItems by remember { mutableStateOf<List<CourseStructureItem>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var loadError by remember { mutableStateOf<String?>(null) }
    var cacheLabel by remember { mutableStateOf<String?>(null) }
    var validationError by remember { mutableStateOf<CourseSettingsLogic.ValidationError?>(null) }
    var saving by remember { mutableStateOf(false) }
    var showHeroEditor by remember { mutableStateOf(false) }

    val isDirty = CourseSettingsLogic.isGeneralFormDirty(form, serverCourse)
    val contentPages = CourseSettingsLogic.contentPages(structureItems)

    LaunchedEffect(course.courseCode) {
        val token = session.accessToken.value ?: return@LaunchedEffect
        loading = true
        loadError = null
        runCatching {
            val result = offline.cachedFetch(
                key = CourseSettingsLogic.cacheKeySettings(course.courseCode),
                accessToken = token,
                serializer = CourseSummary.serializer(),
            ) {
                LmsApi.fetchCourse(course.courseCode, token)
            }
            serverCourse = result.first
            form = CourseSettingsLogic.applyCourseToForm(result.first)
            cacheLabel = result.second?.takeIf { it.isStale(isOnline) }?.lastUpdatedLabel()
            structureItems = runCatching { LmsApi.fetchCourseStructure(course.courseCode, token) }.getOrDefault(emptyList())
        }.onFailure { loadError = it.message }
        loading = false
    }

    Column(modifier = Modifier.fillMaxSize()) {
        LazyColumn(
            modifier = Modifier.weight(1f),
            verticalArrangement = Arrangement.spacedBy(12.dp),
            contentPadding = androidx.compose.foundation.layout.PaddingValues(16.dp),
        ) {
            if (!isOnline) item { OfflineBanner() }
            loadError?.let { msg -> item { LmsErrorBanner(message = msg) } }
            cacheLabel?.let { label -> item { StalenessChip(label = label) } }

            if (loading) {
                item { LmsSkeletonList(count = 4) }
            } else {
                item {
                    LmsCard {
                        Text(L.text(R.string.mobile_courseSettings_basicInfo), fontWeight = FontWeight.SemiBold)
                        OutlinedTextField(
                            value = form.title,
                            onValueChange = { form = form.copy(title = it) },
                            label = { Text(L.text(R.string.mobile_courseSettings_title)) },
                            modifier = Modifier.fillMaxWidth(),
                        )
                        if (validationError?.title != null) {
                            Text(L.text(R.string.mobile_courseSettings_validation_titleRequired), color = androidx.compose.ui.graphics.Color.Red)
                        }
                        OutlinedTextField(
                            value = form.description,
                            onValueChange = { form = form.copy(description = it) },
                            label = { Text(L.text(R.string.mobile_courseSettings_description)) },
                            modifier = Modifier.fillMaxWidth(),
                        )
                    }
                }
                item {
                    LmsCard {
                        Row(
                            modifier = Modifier.fillMaxWidth(),
                            horizontalArrangement = Arrangement.SpaceBetween,
                        ) {
                            Column {
                                Text(L.text(R.string.mobile_courseSettings_published), fontWeight = FontWeight.SemiBold)
                                Text(L.text(R.string.mobile_courseSettings_publishedHint))
                            }
                            Switch(
                                checked = form.published,
                                onCheckedChange = { form = form.copy(published = it) },
                            )
                        }
                    }
                }
                item {
                    LmsCard {
                        Text(L.text(R.string.mobile_courseSettings_courseHome), fontWeight = FontWeight.SemiBold)
                        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                            CourseSettingsLogic.CourseHomeLanding.entries.forEach { landing ->
                                FilterChip(
                                    selected = form.courseHomeLanding == landing,
                                    onClick = {
                                        form = form.copy(
                                            courseHomeLanding = landing,
                                            courseHomeContentItemId = if (landing == CourseSettingsLogic.CourseHomeLanding.content_page) {
                                                form.courseHomeContentItemId
                                            } else "",
                                        )
                                    },
                                    label = { Text(landing.name) },
                                )
                            }
                        }
                        if (form.courseHomeLanding == CourseSettingsLogic.CourseHomeLanding.content_page) {
                            contentPages.forEach { page ->
                                Text(
                                    text = page.title,
                                    modifier = Modifier
                                        .fillMaxWidth()
                                        .clickable { form = form.copy(courseHomeContentItemId = page.id) }
                                        .padding(vertical = 6.dp),
                                    fontWeight = if (form.courseHomeContentItemId == page.id) FontWeight.Bold else FontWeight.Normal,
                                )
                            }
                            if (validationError?.courseHome != null) {
                                Text(L.text(R.string.mobile_courseSettings_validation_contentPageRequired), color = androidx.compose.ui.graphics.Color.Red)
                            }
                        }
                    }
                }
                item {
                    LmsCard {
                        Text(L.text(R.string.mobile_courseSettings_heroImage), fontWeight = FontWeight.SemiBold)
                        Button(onClick = { showHeroEditor = true }) {
                            Text(L.text(R.string.mobile_courseSettings_editHeroImage))
                        }
                    }
                }
                item {
                    LmsCard {
                        Text(L.text(R.string.mobile_courseSettings_readingTheme), fontWeight = FontWeight.SemiBold)
                        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                            CourseSettingsLogic.markdownThemePresets.filter { it != "custom" }.forEach { preset ->
                                FilterChip(
                                    selected = form.markdownThemePreset == preset,
                                    onClick = { form = form.copy(markdownThemePreset = preset) },
                                    label = { Text(preset.replaceFirstChar { it.uppercase() }) },
                                )
                            }
                        }
                    }
                }
            }
        }

        if (isDirty) {
            UnsavedChangesBanner(
                isSaving = saving,
                onDiscard = {
                    form = CourseSettingsLogic.applyCourseToForm(serverCourse)
                    validationError = null
                },
                onSave = {
                    scope.launch {
                        validationError = CourseSettingsLogic.validateGeneralForm(
                            form.title,
                            form.courseHomeLanding,
                            form.courseHomeContentItemId,
                        )
                        if (validationError != null) return@launch
                        val token = session.accessToken.value ?: return@launch
                        saving = true
                        runCatching {
                            if (CourseSettingsLogic.courseNeedsUpdate(form, serverCourse)) {
                                val body = CourseSettingsLogic.buildCourseUpdateRequest(form)
                                offline.enqueueMutation(
                                    method = "PUT",
                                    path = "/api/v1/courses/${course.courseCode}",
                                    bodyJson = settingsJson.encodeToString(CourseUpdateRequest.serializer(), body),
                                    label = L.text(context, localePrefs, R.string.mobile_courseSettings_saveLabel),
                                    accessToken = token,
                                    idempotencyKey = "course-settings:${course.courseCode}:general",
                                )
                            }
                            if (CourseSettingsLogic.themeNeedsUpdate(form, serverCourse)) {
                                val patch = CourseSettingsLogic.buildMarkdownThemePatch(form)
                                offline.enqueueMutation(
                                    method = "PATCH",
                                    path = "/api/v1/courses/${course.courseCode}/markdown-theme",
                                    bodyJson = settingsJson.encodeToString(
                                        com.lextures.android.core.lms.CourseMarkdownThemePatch.serializer(),
                                        patch,
                                    ),
                                    label = L.text(context, localePrefs, R.string.mobile_courseSettings_saveThemeLabel),
                                    accessToken = token,
                                    idempotencyKey = "course-settings:${course.courseCode}:theme",
                                )
                            }
                            val refreshed = LmsApi.fetchCourse(course.courseCode, token)
                            serverCourse = refreshed
                            form = CourseSettingsLogic.applyCourseToForm(refreshed)
                            onCourseUpdated(refreshed)
                        }.onFailure { loadError = it.message }
                        saving = false
                    }
                },
            )
        }
    }

    if (showHeroEditor) {
        CourseHeroImageEditorSheet(
            session = session,
            course = serverCourse,
            offline = offline,
            onDismiss = { showHeroEditor = false },
            onSaved = {
                serverCourse = it
                onCourseUpdated(it)
                showHeroEditor = false
            },
        )
    }
}
