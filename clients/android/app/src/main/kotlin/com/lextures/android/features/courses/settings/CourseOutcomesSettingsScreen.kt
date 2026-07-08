package com.lextures.android.features.courses.settings

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
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.ExposedDropdownMenuBox
import androidx.compose.material3.ExposedDropdownMenuDefaults
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.RadioButton
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
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.OfflineBanner
import com.lextures.android.core.design.StalenessChip
import com.lextures.android.core.design.UnsavedChangesBanner
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalLocalePreferences
import com.lextures.android.core.lms.CourseOutcomeLink
import com.lextures.android.core.lms.CourseOutcome
import com.lextures.android.core.lms.CourseOutcomesLogic
import com.lextures.android.core.lms.CourseStructureItem
import com.lextures.android.core.lms.CourseSummary
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.offline.OfflineService
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsErrorBanner
import com.lextures.android.features.home.LmsSkeletonList
import kotlinx.coroutines.launch
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json

private val outcomesJson = Json { ignoreUnknownKeys = true }

@Composable
fun CourseOutcomesSettingsScreen(
    session: AuthSession,
    course: CourseSummary,
    offline: OfflineService,
) {
    val scope = rememberCoroutineScope()
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    val isOnline by offline.networkMonitor.isOnline.collectAsState()

    var outcomes by remember { mutableStateOf<List<CourseOutcome>>(emptyList()) }
    var enrolledLearners by remember { mutableStateOf(0) }
    var structure by remember { mutableStateOf<List<CourseStructureItem>>(emptyList()) }
    var drafts by remember { mutableStateOf<Map<String, CourseOutcomesLogic.OutcomeDraft>>(emptyMap()) }
    var loading by remember { mutableStateOf(true) }
    var loadError by remember { mutableStateOf<String?>(null) }
    var actionError by remember { mutableStateOf<String?>(null) }
    var actionSuccess by remember { mutableStateOf<String?>(null) }
    var cacheLabel by remember { mutableStateOf<String?>(null) }
    var saving by remember { mutableStateOf(false) }
    var creating by remember { mutableStateOf(false) }
    var newTitle by remember { mutableStateOf("") }
    var newDescription by remember { mutableStateOf("") }
    var deleteOutcomeId by remember { mutableStateOf<String?>(null) }
    var showAddAnother by remember { mutableStateOf(false) }

    val gradableOptions = CourseOutcomesLogic.gradableOptions(structure)
    val isDirty = CourseOutcomesLogic.isDirty(drafts, outcomes)

    LaunchedEffect(course.courseCode) {
        val token = session.accessToken.value ?: return@LaunchedEffect
        loading = true
        loadError = null
        runCatching {
            val result = offline.cachedFetch(
                key = CourseOutcomesLogic.cacheKeyOutcomes(course.courseCode),
                accessToken = token,
                serializer = com.lextures.android.core.lms.CourseOutcomesListResponse.serializer(),
            ) {
                LmsApi.fetchCourseOutcomes(course.courseCode, token)
            }
            outcomes = result.first.outcomes
            enrolledLearners = result.first.enrolledLearners
            drafts = CourseOutcomesLogic.drafts(outcomes)
            structure = runCatching { LmsApi.fetchCourseStructure(course.courseCode, token) }.getOrDefault(emptyList())
            cacheLabel = result.second?.takeIf { it.isStale(isOnline) }?.lastUpdatedLabel()
        }.onFailure { loadError = it.message }
        loading = false
    }

    suspend fun reload() {
        val token = session.accessToken.value ?: return
        runCatching {
            val response = LmsApi.fetchCourseOutcomes(course.courseCode, token)
            outcomes = response.outcomes
            enrolledLearners = response.enrolledLearners
            drafts = CourseOutcomesLogic.drafts(outcomes)
        }.onFailure { actionError = it.message }
    }

    Column(modifier = Modifier.fillMaxSize()) {
        LazyColumn(
            modifier = Modifier.weight(1f),
            verticalArrangement = Arrangement.spacedBy(12.dp),
            contentPadding = androidx.compose.foundation.layout.PaddingValues(16.dp),
        ) {
            if (!isOnline) item { OfflineBanner() }
            cacheLabel?.let { label -> item { StalenessChip(label = label) } }
            loadError?.let { msg -> item { LmsErrorBanner(message = msg) } }
            actionError?.let { msg -> item { LmsErrorBanner(message = msg) } }
            actionSuccess?.let { msg ->
                item {
                    LmsCard {
                        Text(msg, fontWeight = FontWeight.SemiBold)
                    }
                }
            }

            if (loading) {
                item { LmsSkeletonList(count = 4) }
            } else {
                item {
                    LmsCard {
                        Text(L.text(R.string.mobile_courseSettings_outcomes_introTitle), fontWeight = FontWeight.SemiBold)
                        Text(L.text(R.string.mobile_courseSettings_outcomes_introDescription))
                    }
                }

                if (outcomes.isNotEmpty()) {
                    item {
                        Text(L.format(R.string.mobile_courseSettings_outcomes_listCount, outcomes.size.toString()))
                    }
                }

                outcomes.forEach { outcome ->
                    item(key = outcome.id) {
                        OutcomeCard(
                            session = session,
                            course = course,
                            outcome = outcome,
                            enrolledLearners = enrolledLearners,
                            gradableOptions = gradableOptions,
                            draft = drafts[outcome.id] ?: CourseOutcomesLogic.OutcomeDraft(outcome.title, outcome.description),
                            onDraftChange = { updated ->
                                drafts = drafts + (outcome.id to updated)
                            },
                            onDelete = { deleteOutcomeId = outcome.id },
                            onLinksChanged = { scope.launch { reload() } },
                            offline = offline,
                        )
                    }
                }

                item {
                    if (outcomes.isEmpty()) {
                        CreateOutcomeForm(
                            compact = false,
                            newTitle = newTitle,
                            newDescription = newDescription,
                            creating = creating,
                            onTitleChange = { newTitle = it },
                            onDescriptionChange = { newDescription = it },
                            onCreate = {
                                scope.launch {
                                    if (CourseOutcomesLogic.validateCreateTitle(newTitle) != null) {
                                        actionError = L.text(context, localePrefs, R.string.mobile_courseSettings_outcomes_validation_titleRequired)
                                        return@launch
                                    }
                                    val token = session.accessToken.value ?: return@launch
                                    creating = true
                                    actionError = null
                                    runCatching {
                                        val body = CourseOutcomesLogic.buildCreateBody(newTitle, newDescription)
                                        offline.enqueueMutation(
                                            method = "POST",
                                            path = "/api/v1/courses/${course.courseCode}/outcomes",
                                            bodyJson = outcomesJson.encodeToString(body),
                                            label = L.text(context, localePrefs, R.string.mobile_courseSettings_outcomes_createButton),
                                            accessToken = token,
                                            idempotencyKey = CourseOutcomesLogic.createOutcomeIdempotencyKey(course.courseCode),
                                        )
                                        reload()
                                        newTitle = ""
                                        newDescription = ""
                                        actionSuccess = L.text(context, localePrefs, R.string.mobile_courseSettings_outcomes_created)
                                    }.onFailure { actionError = it.message }
                                    creating = false
                                }
                            },
                        )
                    } else if (showAddAnother) {
                        CreateOutcomeForm(
                            compact = true,
                            newTitle = newTitle,
                            newDescription = newDescription,
                            creating = creating,
                            onTitleChange = { newTitle = it },
                            onDescriptionChange = { newDescription = it },
                            onCreate = {
                                scope.launch {
                                    if (CourseOutcomesLogic.validateCreateTitle(newTitle) != null) {
                                        actionError = L.text(context, localePrefs, R.string.mobile_courseSettings_outcomes_validation_titleRequired)
                                        return@launch
                                    }
                                    val token = session.accessToken.value ?: return@launch
                                    creating = true
                                    actionError = null
                                    runCatching {
                                        val body = CourseOutcomesLogic.buildCreateBody(newTitle, newDescription)
                                        offline.enqueueMutation(
                                            method = "POST",
                                            path = "/api/v1/courses/${course.courseCode}/outcomes",
                                            bodyJson = outcomesJson.encodeToString(body),
                                            label = L.text(context, localePrefs, R.string.mobile_courseSettings_outcomes_createButton),
                                            accessToken = token,
                                            idempotencyKey = CourseOutcomesLogic.createOutcomeIdempotencyKey(course.courseCode),
                                        )
                                        reload()
                                        newTitle = ""
                                        newDescription = ""
                                        showAddAnother = false
                                        actionSuccess = L.text(context, localePrefs, R.string.mobile_courseSettings_outcomes_created)
                                    }.onFailure { actionError = it.message }
                                    creating = false
                                }
                            },
                        )
                    } else {
                        TextButton(onClick = { showAddAnother = true }) {
                            Text(L.text(R.string.mobile_courseSettings_outcomes_addAnother))
                        }
                    }
                }
            }
        }

        if (isDirty) {
            UnsavedChangesBanner(
                isSaving = saving,
                onSave = {
                    scope.launch {
                        if (CourseOutcomesLogic.validateDrafts(drafts, outcomes) != null) {
                            actionError = L.text(context, localePrefs, R.string.mobile_courseSettings_outcomes_validation_titleRequired)
                            return@launch
                        }
                        val token = session.accessToken.value ?: return@launch
                        saving = true
                        actionError = null
                        runCatching {
                            CourseOutcomesLogic.dirtyOutcomeIds(drafts, outcomes).forEach { id ->
                                val draft = drafts[id] ?: return@forEach
                                offline.enqueueMutation(
                                    method = "PATCH",
                                    path = "/api/v1/courses/${course.courseCode}/outcomes/$id",
                                    bodyJson = outcomesJson.encodeToString(CourseOutcomesLogic.buildPatchBody(draft)),
                                    label = L.text(context, localePrefs, R.string.mobile_courseSettings_outcomes_saveLabel),
                                    accessToken = token,
                                    idempotencyKey = "${CourseOutcomesLogic.saveIdempotencyKey(course.courseCode)}:$id",
                                )
                            }
                            reload()
                            actionSuccess = L.text(context, localePrefs, R.string.mobile_courseSettings_outcomes_saved)
                        }.onFailure { actionError = it.message }
                        saving = false
                    }
                },
                onDiscard = {
                    drafts = CourseOutcomesLogic.drafts(outcomes)
                    actionError = null
                },
            )
        }
    }

    deleteOutcomeId?.let { outcomeId ->
        AlertDialog(
            onDismissRequest = { deleteOutcomeId = null },
            title = { Text(L.text(R.string.mobile_courseSettings_outcomes_deleteConfirmTitle)) },
            text = { Text(L.text(R.string.mobile_courseSettings_outcomes_deleteConfirmMessage)) },
            confirmButton = {
                TextButton(onClick = {
                    scope.launch {
                        val token = session.accessToken.value ?: return@launch
                        deleteOutcomeId = null
                        runCatching {
                            offline.enqueueMutation(
                                method = "DELETE",
                                path = "/api/v1/courses/${course.courseCode}/outcomes/$outcomeId",
                                bodyJson = null,
                                label = L.text(context, localePrefs, R.string.mobile_courseSettings_outcomes_deleteOutcomeLabel),
                                accessToken = token,
                                idempotencyKey = CourseOutcomesLogic.deleteOutcomeIdempotencyKey(course.courseCode, outcomeId),
                            )
                            outcomes = outcomes.filter { it.id != outcomeId }
                            drafts = drafts - outcomeId
                        }.onFailure {
                            actionError = it.message
                            reload()
                        }
                    }
                }) {
                    Text(L.text(R.string.mobile_courseSettings_outcomes_deleteButton))
                }
            },
            dismissButton = {
                TextButton(onClick = { deleteOutcomeId = null }) {
                    Text(L.text(R.string.mobile_common_cancel))
                }
            },
        )
    }
}

@Composable
private fun CreateOutcomeForm(
    compact: Boolean,
    newTitle: String,
    newDescription: String,
    creating: Boolean,
    onTitleChange: (String) -> Unit,
    onDescriptionChange: (String) -> Unit,
    onCreate: () -> Unit,
) {
    LmsCard {
        if (!compact) {
            Text(L.text(R.string.mobile_courseSettings_outcomes_emptyTitle), fontWeight = FontWeight.SemiBold)
            Text(L.text(R.string.mobile_courseSettings_outcomes_emptyDescription))
        }
        OutlinedTextField(
            value = newTitle,
            onValueChange = onTitleChange,
            label = { Text(L.text(R.string.mobile_courseSettings_outcomes_titleLabel)) },
            placeholder = { Text(L.text(R.string.mobile_courseSettings_outcomes_titlePlaceholder)) },
            modifier = Modifier.fillMaxWidth(),
        )
        OutlinedTextField(
            value = newDescription,
            onValueChange = onDescriptionChange,
            label = { Text(L.text(R.string.mobile_courseSettings_outcomes_descriptionLabel)) },
            placeholder = { Text(L.text(R.string.mobile_courseSettings_outcomes_descriptionPlaceholder)) },
            modifier = Modifier.fillMaxWidth(),
        )
        Button(
            onClick = onCreate,
            enabled = !creating && newTitle.trim().isNotEmpty(),
        ) {
            if (creating) {
                CircularProgressIndicator(modifier = Modifier.padding(end = 8.dp))
            }
            Text(L.text(R.string.mobile_courseSettings_outcomes_createButton))
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun OutcomeCard(
    session: AuthSession,
    course: CourseSummary,
    outcome: CourseOutcome,
    enrolledLearners: Int,
    gradableOptions: List<CourseOutcomesLogic.GradableOption>,
    draft: CourseOutcomesLogic.OutcomeDraft,
    onDraftChange: (CourseOutcomesLogic.OutcomeDraft) -> Unit,
    onDelete: () -> Unit,
    onLinksChanged: () -> Unit,
    offline: OfflineService,
) {
    val scope = rememberCoroutineScope()
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    var itemId by remember(outcome.id) { mutableStateOf("") }
    var quizScopeWhole by remember(outcome.id) { mutableStateOf(true) }
    var questionId by remember(outcome.id) { mutableStateOf("") }
    var quizQuestions by remember { mutableStateOf<List<CourseOutcomesLogic.QuizQuestionOption>>(emptyList()) }
    var loadingQuiz by remember { mutableStateOf(false) }
    var addingLink by remember { mutableStateOf(false) }
    var localError by remember { mutableStateOf<String?>(null) }
    var measurementLevel by remember(outcome.id) { mutableStateOf(CourseOutcomesLogic.defaultMeasurement.name) }
    var intensityLevel by remember(outcome.id) { mutableStateOf(CourseOutcomesLogic.defaultIntensity.name) }
    var itemMenuExpanded by remember { mutableStateOf(false) }
    var questionMenuExpanded by remember { mutableStateOf(false) }

    val selectedGradable = gradableOptions.firstOrNull { it.id == itemId }

    LaunchedEffect(itemId, selectedGradable?.kind) {
        val token = session.accessToken.value
        if (token == null || itemId.isEmpty() || selectedGradable?.kind != "quiz") {
            quizQuestions = emptyList()
            questionId = ""
            return@LaunchedEffect
        }
        loadingQuiz = true
        runCatching {
            val quiz = LmsApi.fetchModuleQuiz(course.courseCode, itemId, null, token)
            quizQuestions = CourseOutcomesLogic.questionOptions(quiz.questions)
            if (quizQuestions.none { it.id == questionId }) {
                questionId = quizQuestions.firstOrNull()?.id.orEmpty()
            }
        }.onFailure {
            quizQuestions = emptyList()
            questionId = ""
        }
        loadingQuiz = false
    }

    LmsCard {
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.Top,
        ) {
            Column(modifier = Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                OutlinedTextField(
                    value = draft.title,
                    onValueChange = { onDraftChange(draft.copy(title = it)) },
                    label = { Text(L.text(R.string.mobile_courseSettings_outcomes_titleLabel)) },
                    modifier = Modifier.fillMaxWidth(),
                )
                OutlinedTextField(
                    value = draft.description,
                    onValueChange = { onDraftChange(draft.copy(description = it)) },
                    label = { Text(L.text(R.string.mobile_courseSettings_outcomes_descriptionLabel)) },
                    modifier = Modifier.fillMaxWidth(),
                )
            }
            TextButton(onClick = onDelete) {
                Text(L.text(R.string.mobile_courseSettings_outcomes_deleteButton))
            }
        }

        Text(L.text(R.string.mobile_courseSettings_outcomes_classProgressTitle), fontWeight = FontWeight.SemiBold)
        val rollup = CourseOutcomesLogic.rollupPercentLabel(outcome.rollupAvgScorePercent)
        if (rollup != null) {
            Text(L.format(R.string.mobile_courseSettings_outcomes_classProgressRollup, rollup))
            LinearProgressIndicator(
                progress = { (outcome.rollupAvgScorePercent ?: 0.0).coerceIn(0.0, 100.0).toFloat() / 100f },
                modifier = Modifier.fillMaxWidth(),
            )
        } else {
            Text(L.format(R.string.mobile_courseSettings_outcomes_classProgressEmpty, enrolledLearners.toString()))
        }

        Text(L.text(R.string.mobile_courseSettings_outcomes_linksTitle), fontWeight = FontWeight.SemiBold)
        Text(L.text(R.string.mobile_courseSettings_outcomes_linksDescription))
        if (outcome.links.isEmpty()) {
            Text(L.text(R.string.mobile_courseSettings_outcomes_linksEmpty))
        } else {
            outcome.links.forEach { link ->
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween,
                    verticalAlignment = Alignment.Top,
                ) {
                    Column(modifier = Modifier.weight(1f)) {
                        Text(outcomeLinkSummary(link), fontWeight = FontWeight.Medium)
                        Text(
                            L.format(
                                R.string.mobile_courseSettings_outcomes_progressLabel,
                                CourseOutcomesLogic.progressPercentLabel(link.progress.avgScorePercent),
                                link.progress.gradedLearners,
                                link.progress.enrolledLearners,
                            ),
                        )
                    }
                    TextButton(onClick = {
                        scope.launch {
                            val token = session.accessToken.value ?: return@launch
                            localError = null
                            runCatching {
                                offline.enqueueMutation(
                                    method = "DELETE",
                                    path = "/api/v1/courses/${course.courseCode}/outcomes/${outcome.id}/links/${link.id}",
                                    bodyJson = null,
                                    label = L.text(context, localePrefs, R.string.mobile_courseSettings_outcomes_deleteLinkLabel),
                                    accessToken = token,
                                    idempotencyKey = CourseOutcomesLogic.deleteLinkIdempotencyKey(
                                        course.courseCode,
                                        outcome.id,
                                        link.id,
                                    ),
                                )
                                onLinksChanged()
                            }.onFailure { localError = it.message }
                        }
                    }) {
                        Text(L.text(R.string.mobile_courseSettings_outcomes_removeLink))
                    }
                }
            }
        }

        Text(L.text(R.string.mobile_courseSettings_outcomes_addLinkTitle), fontWeight = FontWeight.SemiBold)
        localError?.let { Text(it, color = androidx.compose.ui.graphics.Color.Red) }

        ExposedDropdownMenuBox(expanded = itemMenuExpanded, onExpandedChange = { itemMenuExpanded = it }) {
            OutlinedTextField(
                value = selectedGradable?.let {
                    val prefix = L.text(CourseOutcomesLogic.kindLabelRes(it.kind))
                    "$prefix: ${it.label}"
                } ?: L.text(R.string.mobile_courseSettings_outcomes_selectItemPlaceholder),
                onValueChange = {},
                readOnly = true,
                label = { Text(L.text(R.string.mobile_courseSettings_outcomes_selectItem)) },
                trailingIcon = { ExposedDropdownMenuDefaults.TrailingIcon(expanded = itemMenuExpanded) },
                modifier = Modifier.menuAnchor().fillMaxWidth(),
            )
            ExposedDropdownMenu(expanded = itemMenuExpanded, onDismissRequest = { itemMenuExpanded = false }) {
                gradableOptions.forEach { option ->
                    DropdownMenuItem(
                        text = {
                            Text("${L.text(CourseOutcomesLogic.kindLabelRes(option.kind))}: ${option.label}")
                        },
                        onClick = {
                            itemId = option.id
                            quizScopeWhole = true
                            questionId = ""
                            itemMenuExpanded = false
                        },
                    )
                }
            }
        }

        if (selectedGradable?.kind == "quiz") {
            Row(verticalAlignment = Alignment.CenterVertically) {
                RadioButton(selected = quizScopeWhole, onClick = { quizScopeWhole = true })
                Text(L.text(R.string.mobile_courseSettings_outcomes_quizScopeWhole))
                RadioButton(selected = !quizScopeWhole, onClick = { quizScopeWhole = false })
                Text(L.text(R.string.mobile_courseSettings_outcomes_quizScopeQuestion))
            }
            if (!quizScopeWhole) {
                if (loadingQuiz) {
                    CircularProgressIndicator()
                } else {
                    ExposedDropdownMenuBox(expanded = questionMenuExpanded, onExpandedChange = { questionMenuExpanded = it }) {
                        OutlinedTextField(
                            value = quizQuestions.firstOrNull { it.id == questionId }?.prompt.orEmpty(),
                            onValueChange = {},
                            readOnly = true,
                            label = { Text(L.text(R.string.mobile_courseSettings_outcomes_selectQuestion)) },
                            trailingIcon = { ExposedDropdownMenuDefaults.TrailingIcon(expanded = questionMenuExpanded) },
                            modifier = Modifier.menuAnchor().fillMaxWidth(),
                        )
                        ExposedDropdownMenu(expanded = questionMenuExpanded, onDismissRequest = { questionMenuExpanded = false }) {
                            quizQuestions.forEach { question ->
                                DropdownMenuItem(
                                    text = { Text(question.prompt) },
                                    onClick = {
                                        questionId = question.id
                                        questionMenuExpanded = false
                                    },
                                )
                            }
                        }
                    }
                }
            }
        }

        var measurementExpanded by remember { mutableStateOf(false) }
        ExposedDropdownMenuBox(expanded = measurementExpanded, onExpandedChange = { measurementExpanded = it }) {
            OutlinedTextField(
                value = L.text(CourseOutcomesLogic.measurementLabelRes(measurementLevel)),
                onValueChange = {},
                readOnly = true,
                label = { Text(L.text(R.string.mobile_courseSettings_outcomes_measurementLevel)) },
                trailingIcon = { ExposedDropdownMenuDefaults.TrailingIcon(expanded = measurementExpanded) },
                modifier = Modifier.menuAnchor().fillMaxWidth(),
            )
            ExposedDropdownMenu(expanded = measurementExpanded, onDismissRequest = { measurementExpanded = false }) {
                CourseOutcomesLogic.MeasurementLevelId.entries.forEach { level ->
                    DropdownMenuItem(
                        text = { Text(L.text(CourseOutcomesLogic.measurementLabelRes(level.name))) },
                        onClick = {
                            measurementLevel = level.name
                            measurementExpanded = false
                        },
                    )
                }
            }
        }

        var intensityExpanded by remember { mutableStateOf(false) }
        ExposedDropdownMenuBox(expanded = intensityExpanded, onExpandedChange = { intensityExpanded = it }) {
            OutlinedTextField(
                value = L.text(CourseOutcomesLogic.intensityLabelRes(intensityLevel)),
                onValueChange = {},
                readOnly = true,
                label = { Text(L.text(R.string.mobile_courseSettings_outcomes_intensityLevel)) },
                trailingIcon = { ExposedDropdownMenuDefaults.TrailingIcon(expanded = intensityExpanded) },
                modifier = Modifier.menuAnchor().fillMaxWidth(),
            )
            ExposedDropdownMenu(expanded = intensityExpanded, onDismissRequest = { intensityExpanded = false }) {
                CourseOutcomesLogic.IntensityLevelId.entries.forEach { level ->
                    DropdownMenuItem(
                        text = { Text(L.text(CourseOutcomesLogic.intensityLabelRes(level.name))) },
                        onClick = {
                            intensityLevel = level.name
                            intensityExpanded = false
                        },
                    )
                }
            }
        }

        Button(
            onClick = {
                scope.launch {
                    val token = session.accessToken.value ?: return@launch
                    localError = null
                    val selected = selectedGradable
                    if (selected == null) {
                        localError = L.text(context, localePrefs, R.string.mobile_courseSettings_outcomes_validation_selectItem)
                        return@launch
                    }
                    val targetKind = CourseOutcomesLogic.targetKind(selected.kind, quizScopeWhole)
                    if (targetKind == "quiz_question" && questionId.trim().isEmpty()) {
                        localError = L.text(context, localePrefs, R.string.mobile_courseSettings_outcomes_validation_selectQuestion)
                        return@launch
                    }
                    addingLink = true
                    runCatching {
                        val body = CourseOutcomesLogic.buildAddLinkBody(
                            structureItemId = itemId,
                            targetKind = targetKind,
                            quizQuestionId = if (targetKind == "quiz_question") questionId else null,
                            measurementLevel = measurementLevel,
                            intensityLevel = intensityLevel,
                        )
                        offline.enqueueMutation(
                            method = "POST",
                            path = "/api/v1/courses/${course.courseCode}/outcomes/${outcome.id}/links",
                            bodyJson = outcomesJson.encodeToString(body),
                            label = L.text(context, localePrefs, R.string.mobile_courseSettings_outcomes_createLinkLabel),
                            accessToken = token,
                            idempotencyKey = CourseOutcomesLogic.addLinkIdempotencyKey(course.courseCode, outcome.id, itemId),
                        )
                        itemId = ""
                        quizScopeWhole = true
                        questionId = ""
                        measurementLevel = CourseOutcomesLogic.defaultMeasurement.name
                        intensityLevel = CourseOutcomesLogic.defaultIntensity.name
                        onLinksChanged()
                    }.onFailure { localError = it.message }
                    addingLink = false
                }
            },
            enabled = !addingLink && itemId.isNotEmpty(),
        ) {
            if (addingLink) {
                CircularProgressIndicator(modifier = Modifier.padding(end = 8.dp))
            }
            Text(L.text(R.string.mobile_courseSettings_outcomes_addLinkButton))
        }
    }
}

@Composable
private fun outcomeLinkSummary(link: CourseOutcomeLink): String {
    val itemTitle = CourseOutcomesLogic.linkItemTitleRes(link)?.let { res ->
        L.format(res, link.itemTitle)
    } ?: link.itemTitle
    val levels = L.format(
        R.string.mobile_courseSettings_outcomes_levelsSummary,
        L.text(CourseOutcomesLogic.measurementLabelRes(link.measurementLevel)),
        L.text(CourseOutcomesLogic.intensityLabelRes(link.intensityLevel)),
    )
    return "$itemTitle · $levels"
}