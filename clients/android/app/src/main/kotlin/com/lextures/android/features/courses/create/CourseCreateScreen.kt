package com.lextures.android.features.courses.create

import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.Close
import androidx.compose.material.icons.filled.CloudDownload
import androidx.compose.material.icons.filled.Delete
import androidx.compose.material.icons.filled.ChevronRight
import androidx.compose.material.icons.filled.ExpandMore
import androidx.compose.material.icons.filled.NoteAdd
import androidx.compose.material.icons.filled.RadioButtonUnchecked
import androidx.compose.material.icons.filled.Remove
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.ExposedDropdownMenuBox
import androidx.compose.material3.ExposedDropdownMenuDefaults
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.SegmentedButton
import androidx.compose.material3.SegmentedButtonDefaults
import androidx.compose.material3.SingleChoiceSegmentedButtonRow
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
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
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.semantics.selected
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.LexturesType
import com.lextures.android.core.design.cardBackground
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalLocalePreferences
import com.lextures.android.core.lms.AddCourseOutcomeLinkBody
import com.lextures.android.core.lms.CourseCreateDraftStore
import com.lextures.android.core.lms.CourseCreateLogic
import com.lextures.android.core.lms.CourseCreateObservability
import com.lextures.android.core.lms.CourseSummary
import com.lextures.android.core.lms.CreateCourseOutcomeBody
import com.lextures.android.core.lms.CreateCourseOutcomeSubOutcomeBody
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.OrgTerm
import com.lextures.android.core.lms.PatchCourseOutcomeBody
import com.lextures.android.core.lms.PatchCourseSyllabusRequest
import com.lextures.android.core.navigation.MobilePlatformFeatures
import com.lextures.android.features.home.HomeShellState
import com.lextures.android.features.home.LmsErrorBanner
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun CourseCreateScreen(
    session: AuthSession,
    existingCourses: List<CourseSummary>,
    shell: HomeShellState?,
    onFinished: (CourseSummary) -> Unit,
    onDismiss: () -> Unit,
    modifier: Modifier = Modifier,
    features: MobilePlatformFeatures? = null,
) {
    val accessToken by session.accessToken.collectAsState()
    val userEmail by session.userEmail.collectAsState()
    val scope = rememberCoroutineScope()
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    val draftStore = remember { CourseCreateDraftStore(context) }

    val platformFeatures = features ?: shell?.platformFeatures ?: MobilePlatformFeatures()
    val v2Enabled = CourseCreateLogic.courseCreateV2Enabled(platformFeatures)

    var step by remember { mutableStateOf(CourseCreateLogic.initialWizardStep(v2Enabled)) }
    var title by remember { mutableStateOf("") }
    var description by remember { mutableStateOf("") }
    var courseMode by remember { mutableStateOf(CourseCreateLogic.CourseMode.Traditional) }
    var selectedTermId by remember { mutableStateOf("") }
    var selectedGradeLevel by remember { mutableStateOf("") }
    var selectedTemplateId by remember { mutableStateOf(CourseCreateLogic.DEFAULT_TEMPLATE_ID) }
    var firstModuleTitle by remember { mutableStateOf("") }
    var competencies by remember { mutableStateOf(listOf(CourseCreateLogic.CompetencyDraft.empty())) }
    var createdCourse by remember { mutableStateOf<CourseSummary?>(null) }
    var terms by remember { mutableStateOf<List<OrgTerm>>(emptyList()) }
    var loadingTerms by remember { mutableStateOf(false) }
    var submitting by remember { mutableStateOf(false) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var titleError by remember { mutableStateOf<String?>(null) }
    var showCancelConfirm by remember { mutableStateOf(false) }
    var showCanvasImport by remember { mutableStateOf(false) }
    var termMenuExpanded by remember { mutableStateOf(false) }
    var gradeMenuExpanded by remember { mutableStateOf(false) }
    var draftKey by remember { mutableStateOf("") }
    var didRestoreDraft by remember { mutableStateOf(false) }
    var recordedStart by remember { mutableStateOf(false) }
    var draftReady by remember { mutableStateOf(false) }

    val isCompetency = courseMode == CourseCreateLogic.CourseMode.CompetencyBased
    val offline = remember { com.lextures.android.core.offline.OfflineService.get(context) }
    val isOnline by offline.networkMonitor.isOnline.collectAsState()

    fun clearDraft() {
        if (draftKey.isNotEmpty()) draftStore.clear(draftKey)
    }

    fun persistDraft() {
        if (!v2Enabled || draftKey.isEmpty()) return
        draftStore.save(
            draftKey,
            CourseCreateDraftStore.Draft(
                step = step.number,
                title = title,
                description = description,
                courseMode = courseMode.value,
                selectedTermId = selectedTermId,
                selectedGradeLevel = selectedGradeLevel,
                selectedTemplateId = selectedTemplateId,
                firstModuleTitle = firstModuleTitle,
                createdCourseCode = createdCourse?.courseCode,
                competencies = competencies,
                createSource = null,
            ),
        )
    }

    fun maybeRecordStarted() {
        if (!v2Enabled || recordedStart) return
        CourseCreateObservability.recordStarted(context, courseMode.value, selectedTemplateId)
        recordedStart = true
    }

    fun restoreDraft(draft: CourseCreateDraftStore.Draft) {
        var restoredStep = CourseCreateLogic.WizardStep.fromNumber(draft.step)
        if (restoredStep == CourseCreateLogic.WizardStep.Source && !v2Enabled) {
            restoredStep = CourseCreateLogic.WizardStep.Basics
        }
        step = restoredStep
        title = draft.title
        description = draft.description
        courseMode = CourseCreateLogic.CourseMode.fromCourseType(draft.courseMode)
        selectedTermId = draft.selectedTermId
        selectedGradeLevel = draft.selectedGradeLevel
        selectedTemplateId = draft.selectedTemplateId
        firstModuleTitle = draft.firstModuleTitle
        competencies = draft.competencies.ifEmpty { listOf(CourseCreateLogic.CompetencyDraft.empty()) }
        val code = draft.createdCourseCode
        if (!code.isNullOrBlank()) {
            createdCourse = CourseSummary(
                id = code,
                courseCode = code,
                title = draft.title,
                description = draft.description,
                published = false,
                courseType = draft.courseMode,
                termId = draft.selectedTermId.takeIf { it.isNotEmpty() },
                gradeLevel = draft.selectedGradeLevel.takeIf { it.isNotEmpty() },
            )
        }
    }

    fun formatCompetencyError(error: CourseCreateLogic.CompetencyValidationError): String {
        val resName = error.key.replace('.', '_')
        val resId = context.resources.getIdentifier(resName, "string", context.packageName)
        if (resId == 0) return error.key
        return when (error.args.size) {
            0 -> L.text(context, localePrefs, resId)
            else -> L.format(context, localePrefs, resId, *error.args.toTypedArray())
        }
    }

    fun attemptDismiss() {
        if (CourseCreateLogic.shouldConfirmCancel(createdCourse?.courseCode)) {
            showCancelConfirm = true
        } else {
            clearDraft()
            onDismiss()
        }
    }

    fun goBack() {
        errorMessage = null
        titleError = null
        when (step) {
            CourseCreateLogic.WizardStep.Syllabus -> {
                createdCourse?.let { created ->
                    title = created.title
                    description = created.description
                    courseMode = CourseCreateLogic.CourseMode.fromCourseType(created.courseType)
                    selectedTermId = created.termId ?: selectedTermId
                    selectedGradeLevel = created.gradeLevel ?: selectedGradeLevel
                }
                step = CourseCreateLogic.WizardStep.Basics
            }
            CourseCreateLogic.WizardStep.Finish -> step = CourseCreateLogic.WizardStep.Syllabus
            CourseCreateLogic.WizardStep.Basics, CourseCreateLogic.WizardStep.Source -> Unit
        }
    }

    fun submitBasics() {
        val token = accessToken ?: return
        titleError = CourseCreateLogic.validateTitle(title)
        if (titleError != null) {
            errorMessage = null
            return
        }
        scope.launch {
            submitting = true
            errorMessage = null
            try {
                val existing = createdCourse
                createdCourse = if (existing != null && CourseCreateLogic.shouldUpdateExistingCourse(existing.courseCode)) {
                    val body = CourseCreateLogic.buildUpdateRequest(
                        course = existing,
                        title = title,
                        description = description,
                        termId = selectedTermId,
                        gradeLevel = selectedGradeLevel,
                    )
                    LmsApi.updateCourse(existing.courseCode, body, token)
                } else {
                    val body = CourseCreateLogic.buildCreateRequest(
                        title = title,
                        description = description,
                        mode = courseMode,
                        termId = selectedTermId,
                        gradeLevel = selectedGradeLevel,
                    )
                    LmsApi.createCourse(body, token)
                }
                if (v2Enabled) {
                    CourseCreateObservability.recordStepCompleted(context, 1)
                }
                step = CourseCreateLogic.WizardStep.Syllabus
            } catch (e: Exception) {
                errorMessage = session.mapError(e)
                    .ifBlank { L.text(context, localePrefs, R.string.mobile_createCourse_error_createFailed) }
            } finally {
                submitting = false
            }
        }
    }

    fun continueFromSyllabus() {
        val course = createdCourse ?: return
        val token = accessToken ?: return
        scope.launch {
            submitting = true
            errorMessage = null
            try {
                if (CourseCreateLogic.shouldPatchSyllabus(selectedTemplateId)) {
                    val tmpl = CourseCreateLogic.template(selectedTemplateId)
                    if (tmpl != null) {
                        val sections = CourseCreateLogic.templateSectionsToSyllabus(tmpl.sections)
                        LmsApi.patchCourseSyllabus(
                            course.courseCode,
                            PatchCourseSyllabusRequest(sections = sections, requireSyllabusAcceptance = false),
                            token,
                        )
                    }
                }
                if (!isCompetency) {
                    firstModuleTitle = CourseCreateLogic.suggestedFirstModuleTitle(selectedTemplateId, firstModuleTitle)
                }
                if (v2Enabled) {
                    CourseCreateObservability.recordStepCompleted(context, 2)
                }
                step = CourseCreateLogic.WizardStep.Finish
            } catch (e: Exception) {
                errorMessage = session.mapError(e)
                    .ifBlank { L.text(context, localePrefs, R.string.mobile_createCourse_error_syllabusFailed) }
            } finally {
                submitting = false
            }
        }
    }

    fun finishTraditional(skipModule: Boolean) {
        val course = createdCourse ?: return
        val token = accessToken ?: return
        scope.launch {
            submitting = true
            errorMessage = null
            try {
                if (!skipModule) {
                    val moduleTitle = firstModuleTitle.trim()
                    if (moduleTitle.isNotEmpty()) {
                        LmsApi.createCourseModule(course.courseCode, moduleTitle, token)
                    }
                }
                if (v2Enabled) {
                    CourseCreateObservability.recordStepCompleted(context, 3)
                    CourseCreateObservability.recordFinished(context, courseMode.value, selectedTemplateId)
                }
                clearDraft()
                shell?.refresh(token)
                val refreshed = runCatching { LmsApi.fetchCourse(course.courseCode, token) }.getOrDefault(course)
                onFinished(refreshed)
            } catch (e: Exception) {
                errorMessage = session.mapError(e)
                    .ifBlank { L.text(context, localePrefs, R.string.mobile_createCourse_error_moduleFailed) }
            } finally {
                submitting = false
            }
        }
    }

    fun finishCompetencyHandoff() {
        val course = createdCourse ?: return
        val token = accessToken ?: return
        scope.launch {
            submitting = true
            errorMessage = null
            clearDraft()
            shell?.refresh(token)
            val refreshed = runCatching { LmsApi.fetchCourse(course.courseCode, token) }.getOrDefault(course)
            submitting = false
            onFinished(refreshed)
        }
    }

    fun finishCompetencyBased() {
        val course = createdCourse ?: return
        val token = accessToken ?: return
        val validation = CourseCreateLogic.validateCompetencies(competencies)
        if (validation != null) {
            errorMessage = formatCompetencyError(validation)
            return
        }
        scope.launch {
            submitting = true
            errorMessage = null
            try {
                for (comp in competencies) {
                    val module = LmsApi.createCourseModule(
                        course.courseCode,
                        comp.title.trim(),
                        token,
                    )
                    val outcome = LmsApi.createCourseOutcome(
                        course.courseCode,
                        CreateCourseOutcomeBody(
                            title = comp.title.trim(),
                            description = comp.description.trim(),
                        ),
                        token,
                    )
                    LmsApi.patchCourseOutcome(
                        course.courseCode,
                        outcome.id,
                        PatchCourseOutcomeBody(moduleStructureItemId = module.id),
                        token,
                    )
                    for (sub in comp.subOutcomes) {
                        val subRow = LmsApi.createCourseOutcomeSubOutcome(
                            course.courseCode,
                            outcome.id,
                            CreateCourseOutcomeSubOutcomeBody(
                                title = sub.title.trim(),
                                description = sub.description.trim(),
                            ),
                            token,
                        )
                        val assessmentTitle = sub.assessmentTitle.trim()
                        val item = when (sub.assessmentKind) {
                            CourseCreateLogic.AssessmentKind.Assignment ->
                                LmsApi.createModuleAssignment(
                                    course.courseCode,
                                    module.id,
                                    assessmentTitle,
                                    token,
                                )
                            CourseCreateLogic.AssessmentKind.Quiz ->
                                LmsApi.createModuleQuiz(
                                    course.courseCode,
                                    module.id,
                                    assessmentTitle,
                                    token,
                                )
                        }
                        LmsApi.addCourseOutcomeLink(
                            course.courseCode,
                            outcome.id,
                            AddCourseOutcomeLinkBody(
                                structureItemId = item.id,
                                targetKind = sub.assessmentKind.value,
                                measurementLevel = "summative",
                                intensityLevel = "high",
                                subOutcomeId = subRow.id,
                            ),
                            token,
                        )
                    }
                }
                CourseCreateObservability.recordStepCompleted(context, 3)
                CourseCreateObservability.recordFinished(context, courseMode.value, selectedTemplateId)
                clearDraft()
                shell?.refresh(token)
                val refreshed = runCatching { LmsApi.fetchCourse(course.courseCode, token) }.getOrDefault(course)
                onFinished(refreshed)
            } catch (e: Exception) {
                errorMessage = session.mapError(e)
                    .ifBlank { L.text(context, localePrefs, R.string.mobile_createCourse_error_competencyFailed) }
            } finally {
                submitting = false
            }
        }
    }

    LaunchedEffect(Unit) {
        step = CourseCreateLogic.initialWizardStep(v2Enabled)
        val orgId = CourseCreateLogic.resolveOrgId(accessToken, existingCourses)
        draftKey = draftStore.storageKey(userEmail, orgId)
        if (v2Enabled && !didRestoreDraft) {
            draftStore.load(draftKey)?.let { restoreDraft(it) }
            didRestoreDraft = true
        }
        if (step != CourseCreateLogic.WizardStep.Source) {
            maybeRecordStarted()
        }
        draftReady = true
    }

    LaunchedEffect(accessToken) {
        val token = accessToken ?: return@LaunchedEffect
        loadingTerms = true
        val orgId = CourseCreateLogic.resolveOrgId(token, existingCourses)
        terms = if (orgId != null) {
            runCatching { LmsApi.fetchOrgTerms(orgId, token) }.getOrDefault(emptyList())
        } else {
            emptyList()
        }
        loadingTerms = false
    }

    LaunchedEffect(
        draftReady,
        step,
        title,
        description,
        courseMode,
        selectedTermId,
        selectedGradeLevel,
        selectedTemplateId,
        firstModuleTitle,
        competencies,
        createdCourse?.courseCode,
    ) {
        if (draftReady) persistDraft()
    }

    if (showCanvasImport) {
        CanvasImportScreen(
            session = session,
            existingCourses = existingCourses,
            onFinished = onFinished,
            onDismiss = {
                showCanvasImport = false
                step = CourseCreateLogic.WizardStep.Source
            },
            modifier = modifier,
        )
        return
    }

    Surface(modifier = modifier.fillMaxSize()) {
        Column(Modifier.fillMaxSize()) {
            TopAppBar(
                title = { Text(stringResource(R.string.mobile_createCourse_title)) },
                navigationIcon = {
                    IconButton(onClick = { attemptDismiss() }, enabled = !submitting) {
                        Icon(Icons.Default.Close, contentDescription = stringResource(R.string.mobile_common_close))
                    }
                },
            )

            Column(
                modifier = Modifier
                    .weight(1f)
                    .verticalScroll(rememberScrollState())
                    .padding(16.dp),
                verticalArrangement = Arrangement.spacedBy(14.dp),
            ) {
                if (step != CourseCreateLogic.WizardStep.Source) {
                    Text(
                        text = stringResource(R.string.mobile_createCourse_stepOf, step.number, 3),
                        fontSize = 12.sp,
                        fontWeight = FontWeight.SemiBold,
                        color = textSecondary(),
                    )
                    Row(horizontalArrangement = Arrangement.spacedBy(8.dp), modifier = Modifier.fillMaxWidth()) {
                        CourseCreateLogic.WizardStep.progressSteps.forEach { s ->
                            val active = s.number <= step.number
                            Column(modifier = Modifier.weight(1f)) {
                                Box(
                                    modifier = Modifier
                                        .fillMaxWidth()
                                        .height(4.dp)
                                        .clip(RoundedCornerShape(2.dp))
                                        .background(if (active) LexturesColors.Primary else textSecondary().copy(alpha = 0.25f)),
                                )
                                Text(
                                    text = stringResource(
                                        when {
                                            s == CourseCreateLogic.WizardStep.Finish && isCompetency ->
                                                R.string.mobile_createCourse_step_competencies
                                            s == CourseCreateLogic.WizardStep.Basics ->
                                                R.string.mobile_createCourse_step_basics
                                            s == CourseCreateLogic.WizardStep.Syllabus ->
                                                R.string.mobile_createCourse_step_syllabus
                                            else -> R.string.mobile_createCourse_step_module
                                        },
                                    ),
                                    fontSize = 11.sp,
                                    color = if (active) textPrimary() else textSecondary(),
                                    maxLines = 1,
                                )
                            }
                        }
                    }
                }

                errorMessage?.let { LmsErrorBanner(it) }

                when (step) {
                    CourseCreateLogic.WizardStep.Source -> {
                        Text(
                            stringResource(R.string.mobile_createCourse_source_intro),
                            color = textSecondary(),
                            fontSize = 14.sp,
                        )
                        SourceOptionCard(
                            title = stringResource(R.string.mobile_createCourse_source_scratch_title),
                            summary = stringResource(R.string.mobile_createCourse_source_scratch_summary),
                            icon = Icons.Default.NoteAdd,
                            onClick = {
                                step = CourseCreateLogic.WizardStep.Basics
                                maybeRecordStarted()
                            },
                        )
                        if (CourseCreateLogic.shouldShowCanvasImportSource(
                                permissions = shell?.permissions.orEmpty(),
                                features = platformFeatures,
                                isOnline = isOnline,
                            )
                        ) {
                            SourceOptionCard(
                                title = stringResource(R.string.mobile_createCourse_source_canvas_title),
                                summary = stringResource(R.string.mobile_createCourse_source_canvas_summary),
                                icon = Icons.Default.CloudDownload,
                                onClick = { showCanvasImport = true },
                            )
                        }
                    }

                    CourseCreateLogic.WizardStep.Basics -> {
                        Text(stringResource(R.string.mobile_createCourse_field_title), fontWeight = FontWeight.SemiBold, color = textPrimary())
                        OutlinedTextField(
                            value = title,
                            onValueChange = { title = it },
                            modifier = Modifier.fillMaxWidth(),
                            singleLine = true,
                            placeholder = { Text(stringResource(R.string.mobile_createCourse_field_titlePlaceholder)) },
                            isError = titleError != null,
                            supportingText = titleError?.let {
                                { Text(stringResource(R.string.mobile_createCourse_error_titleRequired), color = LexturesColors.Error) }
                            },
                        )
                        Text(stringResource(R.string.mobile_createCourse_field_description), fontWeight = FontWeight.SemiBold, color = textPrimary())
                        OutlinedTextField(
                            value = description,
                            onValueChange = { description = it },
                            modifier = Modifier.fillMaxWidth(),
                            minLines = 3,
                            placeholder = { Text(stringResource(R.string.mobile_createCourse_field_descriptionPlaceholder)) },
                        )
                        Text(stringResource(R.string.mobile_createCourse_field_mode), fontWeight = FontWeight.SemiBold, color = textPrimary())
                        SingleChoiceSegmentedButtonRow(modifier = Modifier.fillMaxWidth()) {
                            SegmentedButton(
                                selected = !isCompetency,
                                onClick = { courseMode = CourseCreateLogic.CourseMode.Traditional },
                                shape = SegmentedButtonDefaults.itemShape(index = 0, count = 2),
                            ) { Text(stringResource(R.string.mobile_createCourse_mode_traditional)) }
                            SegmentedButton(
                                selected = isCompetency,
                                onClick = { courseMode = CourseCreateLogic.CourseMode.CompetencyBased },
                                shape = SegmentedButtonDefaults.itemShape(index = 1, count = 2),
                            ) { Text(stringResource(R.string.mobile_createCourse_mode_competency)) }
                        }
                        Text(
                            text = stringResource(
                                if (isCompetency) R.string.mobile_createCourse_mode_competencyHint
                                else R.string.mobile_createCourse_mode_traditionalHint,
                            ),
                            fontSize = 12.sp,
                            color = textSecondary(),
                        )

                        Text(stringResource(R.string.mobile_createCourse_field_term), fontWeight = FontWeight.SemiBold, color = textPrimary())
                        if (loadingTerms) {
                            CircularProgressIndicator(modifier = Modifier.size(24.dp))
                        } else {
                            ExposedDropdownMenuBox(expanded = termMenuExpanded, onExpandedChange = { termMenuExpanded = it }) {
                                OutlinedTextField(
                                    value = terms.firstOrNull { it.id == selectedTermId }?.name
                                        ?: stringResource(R.string.mobile_createCourse_term_none),
                                    onValueChange = {},
                                    readOnly = true,
                                    modifier = Modifier
                                        .menuAnchor()
                                        .fillMaxWidth(),
                                    trailingIcon = { ExposedDropdownMenuDefaults.TrailingIcon(expanded = termMenuExpanded) },
                                )
                                ExposedDropdownMenu(
                                    expanded = termMenuExpanded,
                                    onDismissRequest = { termMenuExpanded = false },
                                ) {
                                    DropdownMenuItem(
                                        text = { Text(stringResource(R.string.mobile_createCourse_term_none)) },
                                        onClick = { selectedTermId = ""; termMenuExpanded = false },
                                    )
                                    terms.forEach { term ->
                                        DropdownMenuItem(
                                            text = { Text(term.name) },
                                            onClick = { selectedTermId = term.id; termMenuExpanded = false },
                                        )
                                    }
                                }
                            }
                        }

                        Text(stringResource(R.string.mobile_createCourse_field_gradeLevel), fontWeight = FontWeight.SemiBold, color = textPrimary())
                        ExposedDropdownMenuBox(expanded = gradeMenuExpanded, onExpandedChange = { gradeMenuExpanded = it }) {
                            OutlinedTextField(
                                value = selectedGradeLevel.ifEmpty { stringResource(R.string.mobile_createCourse_gradeLevel_none) },
                                onValueChange = {},
                                readOnly = true,
                                modifier = Modifier
                                    .menuAnchor()
                                    .fillMaxWidth(),
                                trailingIcon = { ExposedDropdownMenuDefaults.TrailingIcon(expanded = gradeMenuExpanded) },
                            )
                            ExposedDropdownMenu(
                                expanded = gradeMenuExpanded,
                                onDismissRequest = { gradeMenuExpanded = false },
                            ) {
                                DropdownMenuItem(
                                    text = { Text(stringResource(R.string.mobile_createCourse_gradeLevel_none)) },
                                    onClick = { selectedGradeLevel = ""; gradeMenuExpanded = false },
                                )
                                CourseCreateLogic.gradeLevels.filter { it.isNotEmpty() }.forEach { level ->
                                    DropdownMenuItem(
                                        text = { Text(level) },
                                        onClick = { selectedGradeLevel = level; gradeMenuExpanded = false },
                                    )
                                }
                            }
                        }
                    }

                    CourseCreateLogic.WizardStep.Syllabus -> {
                        Text(
                            stringResource(R.string.mobile_createCourse_syllabus_intro),
                            color = textSecondary(),
                            fontSize = 14.sp,
                        )
                        TemplateOptionCard(
                            name = stringResource(R.string.mobile_createCourse_template_blank),
                            summary = stringResource(R.string.mobile_createCourse_template_blankSummary),
                            selected = selectedTemplateId == CourseCreateLogic.BLANK_TEMPLATE_ID,
                            onClick = { selectedTemplateId = CourseCreateLogic.BLANK_TEMPLATE_ID },
                        )
                        CourseCreateLogic.starterTemplates.forEach { tmpl ->
                            val nameRes = context.resources.getIdentifier(tmpl.nameKey, "string", context.packageName)
                            val summaryRes = context.resources.getIdentifier(tmpl.summaryKey, "string", context.packageName)
                            TemplateOptionCard(
                                name = if (nameRes != 0) stringResource(nameRes) else tmpl.id,
                                summary = if (summaryRes != 0) stringResource(summaryRes) else "",
                                selected = selectedTemplateId == tmpl.id,
                                onClick = { selectedTemplateId = tmpl.id },
                            )
                        }
                    }

                    CourseCreateLogic.WizardStep.Finish -> {
                        if (isCompetency) {
                            if (v2Enabled) {
                                CompetencyEditor(
                                    competencies = competencies,
                                    onChange = { competencies = it },
                                )
                            } else {
                                Text(
                                    stringResource(R.string.mobile_createCourse_competency_handoffTitle),
                                    style = LexturesType.display(18),
                                    color = textPrimary(),
                                )
                                Text(
                                    stringResource(R.string.mobile_createCourse_competency_handoffBody),
                                    color = textSecondary(),
                                )
                            }
                        } else {
                            Text(stringResource(R.string.mobile_createCourse_firstModule_label), fontWeight = FontWeight.SemiBold, color = textPrimary())
                            OutlinedTextField(
                                value = firstModuleTitle,
                                onValueChange = { firstModuleTitle = it },
                                modifier = Modifier.fillMaxWidth(),
                                singleLine = true,
                                placeholder = { Text(stringResource(R.string.mobile_createCourse_firstModule_placeholder)) },
                            )
                            Text(
                                stringResource(R.string.mobile_createCourse_firstModule_hint),
                                fontSize = 12.sp,
                                color = textSecondary(),
                            )
                        }
                    }
                }
            }

            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(16.dp),
                horizontalArrangement = Arrangement.spacedBy(12.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                when {
                    step != CourseCreateLogic.WizardStep.Basics && step != CourseCreateLogic.WizardStep.Source -> {
                        OutlinedButton(onClick = { goBack() }, enabled = !submitting) {
                            Text(stringResource(R.string.mobile_createCourse_action_back))
                        }
                    }
                    step == CourseCreateLogic.WizardStep.Basics && v2Enabled -> {
                        OutlinedButton(
                            onClick = { step = CourseCreateLogic.WizardStep.Source },
                            enabled = !submitting,
                        ) {
                            Text(stringResource(R.string.mobile_createCourse_action_back))
                        }
                    }
                }
                Spacer(Modifier.weight(1f))
                if (step == CourseCreateLogic.WizardStep.Finish && !isCompetency) {
                    TextButton(onClick = { finishTraditional(skipModule = true) }, enabled = !submitting) {
                        Text(stringResource(R.string.mobile_createCourse_firstModule_skip))
                    }
                }
                if (step != CourseCreateLogic.WizardStep.Source) {
                    Button(
                        onClick = {
                            when (step) {
                                CourseCreateLogic.WizardStep.Source -> Unit
                                CourseCreateLogic.WizardStep.Basics -> submitBasics()
                                CourseCreateLogic.WizardStep.Syllabus -> continueFromSyllabus()
                                CourseCreateLogic.WizardStep.Finish -> {
                                    when {
                                        isCompetency && v2Enabled -> finishCompetencyBased()
                                        isCompetency -> finishCompetencyHandoff()
                                        else -> finishTraditional(skipModule = false)
                                    }
                                }
                            }
                        },
                        enabled = !submitting,
                    ) {
                        if (submitting) {
                            CircularProgressIndicator(modifier = Modifier.size(18.dp), strokeWidth = 2.dp, color = textPrimary())
                        } else {
                            Text(
                                stringResource(
                                    when {
                                        step == CourseCreateLogic.WizardStep.Finish && isCompetency && v2Enabled ->
                                            R.string.mobile_createCourse_action_createCompetencies
                                        step == CourseCreateLogic.WizardStep.Finish ->
                                            R.string.mobile_createCourse_action_createOpen
                                        else -> R.string.mobile_createCourse_action_continue
                                    },
                                ),
                            )
                        }
                    }
                }
            }
        }
    }

    if (showCancelConfirm) {
        AlertDialog(
            onDismissRequest = { showCancelConfirm = false },
            title = { Text(stringResource(R.string.mobile_createCourse_cancel_confirm)) },
            text = { Text(stringResource(R.string.mobile_createCourse_cancel_message)) },
            confirmButton = {
                TextButton(
                    onClick = {
                        clearDraft()
                        showCancelConfirm = false
                        onDismiss()
                    },
                ) {
                    Text(stringResource(R.string.mobile_createCourse_cancel_leave))
                }
            },
            dismissButton = {
                TextButton(onClick = { showCancelConfirm = false }) {
                    Text(stringResource(R.string.mobile_common_close))
                }
            },
        )
    }
}

@Composable
private fun SourceOptionCard(
    title: String,
    summary: String,
    icon: ImageVector,
    onClick: () -> Unit,
) {
    Surface(
        modifier = Modifier
            .fillMaxWidth()
            .clickable(onClick = onClick),
        shape = RoundedCornerShape(14.dp),
        color = cardBackground(),
    ) {
        Row(
            modifier = Modifier.padding(14.dp),
            horizontalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            Icon(
                imageVector = icon,
                contentDescription = null,
                tint = LexturesColors.Primary,
                modifier = Modifier.size(28.dp),
            )
            Column(verticalArrangement = Arrangement.spacedBy(4.dp)) {
                Text(title, style = LexturesType.display(16), color = textPrimary())
                Text(summary, fontSize = 12.sp, color = textSecondary())
            }
        }
    }
}

@Composable
private fun CompetencyEditor(
    competencies: List<CourseCreateLogic.CompetencyDraft>,
    onChange: (List<CourseCreateLogic.CompetencyDraft>) -> Unit,
) {
    Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
        Text(
            stringResource(R.string.mobile_createCourse_competency_intro),
            color = textSecondary(),
            fontSize = 14.sp,
        )
        competencies.forEachIndexed { index, comp ->
            CompetencyCard(
                index = index,
                competency = comp,
                canRemove = competencies.size > 1,
                onUpdate = { updated ->
                    onChange(competencies.toMutableList().also { it[index] = updated })
                },
                onRemove = {
                    onChange(competencies.toMutableList().also { it.removeAt(index) })
                },
            )
        }
        TextButton(
            onClick = { onChange(competencies + CourseCreateLogic.CompetencyDraft.empty()) },
        ) {
            Icon(Icons.Default.Add, contentDescription = null, modifier = Modifier.size(18.dp))
            Spacer(Modifier.size(6.dp))
            Text(stringResource(R.string.mobile_createCourse_competency_add))
        }
    }
}

@Composable
private fun CompetencyCard(
    index: Int,
    competency: CourseCreateLogic.CompetencyDraft,
    canRemove: Boolean,
    onUpdate: (CourseCreateLogic.CompetencyDraft) -> Unit,
    onRemove: () -> Unit,
) {
    Surface(
        modifier = Modifier.fillMaxWidth(),
        shape = RoundedCornerShape(14.dp),
        color = cardBackground(),
    ) {
        Column(
            modifier = Modifier.padding(14.dp),
            verticalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                TextButton(
                    onClick = { onUpdate(competency.copy(expanded = !competency.expanded)) },
                ) {
                    Icon(
                        imageVector = if (competency.expanded) Icons.Default.ExpandMore else Icons.Default.ChevronRight,
                        contentDescription = null,
                        modifier = Modifier.size(18.dp),
                    )
                    Spacer(Modifier.size(4.dp))
                    Text(
                        stringResource(R.string.mobile_createCourse_competency_heading, index + 1),
                        fontWeight = FontWeight.SemiBold,
                        color = textPrimary(),
                    )
                }
                Spacer(Modifier.weight(1f))
                if (canRemove) {
                    IconButton(onClick = onRemove) {
                        Icon(
                            Icons.Default.Delete,
                            contentDescription = stringResource(R.string.mobile_createCourse_competency_remove),
                            tint = LexturesColors.Error,
                        )
                    }
                }
            }

            if (competency.expanded) {
                OutlinedTextField(
                    value = competency.title,
                    onValueChange = { onUpdate(competency.copy(title = it)) },
                    modifier = Modifier.fillMaxWidth(),
                    singleLine = true,
                    placeholder = { Text(stringResource(R.string.mobile_createCourse_competency_titlePlaceholder)) },
                )
                OutlinedTextField(
                    value = competency.description,
                    onValueChange = { onUpdate(competency.copy(description = it)) },
                    modifier = Modifier.fillMaxWidth(),
                    minLines = 2,
                    placeholder = { Text(stringResource(R.string.mobile_createCourse_competency_descriptionPlaceholder)) },
                )

                competency.subOutcomes.forEachIndexed { subIndex, sub ->
                    SubOutcomeEditor(
                        index = subIndex,
                        subOutcome = sub,
                        canRemove = competency.subOutcomes.size > 1,
                        onUpdate = { updated ->
                            onUpdate(
                                competency.copy(
                                    subOutcomes = competency.subOutcomes.toMutableList().also { it[subIndex] = updated },
                                ),
                            )
                        },
                        onRemove = {
                            onUpdate(
                                competency.copy(
                                    subOutcomes = competency.subOutcomes.toMutableList().also { it.removeAt(subIndex) },
                                ),
                            )
                        },
                    )
                }

                TextButton(
                    onClick = {
                        onUpdate(
                            competency.copy(
                                subOutcomes = competency.subOutcomes + CourseCreateLogic.SubOutcomeDraft.empty(),
                            ),
                        )
                    },
                ) {
                    Text(stringResource(R.string.mobile_createCourse_competency_addSubOutcome))
                }
            }
        }
    }
}

@Composable
private fun SubOutcomeEditor(
    index: Int,
    subOutcome: CourseCreateLogic.SubOutcomeDraft,
    canRemove: Boolean,
    onUpdate: (CourseCreateLogic.SubOutcomeDraft) -> Unit,
    onRemove: () -> Unit,
) {
    Surface(
        modifier = Modifier.fillMaxWidth(),
        shape = RoundedCornerShape(10.dp),
        color = cardBackground(),
        border = BorderStroke(1.dp, textSecondary().copy(alpha = 0.25f)),
    ) {
        Column(
            modifier = Modifier.padding(10.dp),
            verticalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text(
                    stringResource(R.string.mobile_createCourse_competency_subOutcomeHeading, index + 1),
                    fontWeight = FontWeight.SemiBold,
                    fontSize = 14.sp,
                    color = textPrimary(),
                )
                Spacer(Modifier.weight(1f))
                if (canRemove) {
                    IconButton(onClick = onRemove, modifier = Modifier.size(36.dp)) {
                        Icon(Icons.Default.Remove, contentDescription = null, tint = LexturesColors.Error)
                    }
                }
            }
            OutlinedTextField(
                value = subOutcome.title,
                onValueChange = { onUpdate(subOutcome.copy(title = it)) },
                modifier = Modifier.fillMaxWidth(),
                singleLine = true,
                placeholder = { Text(stringResource(R.string.mobile_createCourse_competency_subOutcomeTitlePlaceholder)) },
            )
            OutlinedTextField(
                value = subOutcome.description,
                onValueChange = { onUpdate(subOutcome.copy(description = it)) },
                modifier = Modifier.fillMaxWidth(),
                minLines = 2,
                placeholder = { Text(stringResource(R.string.mobile_createCourse_competency_subOutcomeDescriptionPlaceholder)) },
            )
            OutlinedTextField(
                value = subOutcome.assessmentTitle,
                onValueChange = { onUpdate(subOutcome.copy(assessmentTitle = it)) },
                modifier = Modifier.fillMaxWidth(),
                singleLine = true,
                placeholder = { Text(stringResource(R.string.mobile_createCourse_competency_assessmentTitlePlaceholder)) },
            )
            Text(
                stringResource(R.string.mobile_createCourse_competency_assessmentKind),
                fontWeight = FontWeight.SemiBold,
                fontSize = 13.sp,
                color = textPrimary(),
            )
            SingleChoiceSegmentedButtonRow(modifier = Modifier.fillMaxWidth()) {
                SegmentedButton(
                    selected = subOutcome.assessmentKind == CourseCreateLogic.AssessmentKind.Quiz,
                    onClick = { onUpdate(subOutcome.copy(assessmentKind = CourseCreateLogic.AssessmentKind.Quiz)) },
                    shape = SegmentedButtonDefaults.itemShape(index = 0, count = 2),
                ) { Text(stringResource(R.string.mobile_createCourse_competency_assessment_quiz)) }
                SegmentedButton(
                    selected = subOutcome.assessmentKind == CourseCreateLogic.AssessmentKind.Assignment,
                    onClick = { onUpdate(subOutcome.copy(assessmentKind = CourseCreateLogic.AssessmentKind.Assignment)) },
                    shape = SegmentedButtonDefaults.itemShape(index = 1, count = 2),
                ) { Text(stringResource(R.string.mobile_createCourse_competency_assessment_assignment)) }
            }
        }
    }
}

@Composable
private fun TemplateOptionCard(
    name: String,
    summary: String,
    selected: Boolean,
    onClick: () -> Unit,
) {
    Surface(
        modifier = Modifier
            .fillMaxWidth()
            .semantics { this.selected = selected }
            .clickable(onClick = onClick),
        shape = RoundedCornerShape(14.dp),
        color = cardBackground(),
        border = if (selected) BorderStroke(2.dp, LexturesColors.Primary) else null,
    ) {
        Row(
            modifier = Modifier.padding(14.dp),
            horizontalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            Icon(
                imageVector = if (selected) Icons.Default.CheckCircle else Icons.Default.RadioButtonUnchecked,
                contentDescription = null,
                tint = if (selected) LexturesColors.Primary else textSecondary(),
            )
            Column(verticalArrangement = Arrangement.spacedBy(4.dp)) {
                Text(name, style = LexturesType.display(16), color = textPrimary())
                Text(summary, fontSize = 12.sp, color = textSecondary())
            }
        }
    }
}
