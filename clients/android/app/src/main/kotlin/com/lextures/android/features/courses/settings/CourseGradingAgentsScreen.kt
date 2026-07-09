package com.lextures.android.features.courses.settings

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FilterChip
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.rememberModalBottomSheetState
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
import androidx.compose.ui.graphics.Color
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
import com.lextures.android.core.lms.CourseGradingAgentSummary
import com.lextures.android.core.lms.CourseGradingAgentsLogic
import com.lextures.android.core.lms.CourseStructureItem
import com.lextures.android.core.lms.CourseSummary
import com.lextures.android.core.lms.GraderAgentTemplateSummary
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.offline.OfflineService
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsErrorBanner
import com.lextures.android.features.home.LmsSkeletonList
import kotlinx.coroutines.launch
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json

private val gradingAgentsJson = Json { ignoreUnknownKeys = true }

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun CourseGradingAgentsScreen(
    session: AuthSession,
    course: CourseSummary,
    offline: OfflineService,
) {
    val scope = rememberCoroutineScope()
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    val isOnline by offline.networkMonitor.isOnline.collectAsState()

    var agents by remember { mutableStateOf<List<CourseGradingAgentSummary>>(emptyList()) }
    var templates by remember { mutableStateOf<List<GraderAgentTemplateSummary>>(emptyList()) }
    var structure by remember { mutableStateOf<List<CourseStructureItem>>(emptyList()) }
    var filterQuery by remember { mutableStateOf("") }
    var loading by remember { mutableStateOf(true) }
    var loadError by remember { mutableStateOf<String?>(null) }
    var cacheLabel by remember { mutableStateOf<String?>(null) }
    var selectedAgent by remember { mutableStateOf<CourseGradingAgentSummary?>(null) }
    var createSheetOpen by remember { mutableStateOf(false) }

    val existingItemIds = remember(agents) { agents.map { it.itemId }.toSet() }
    val gradableOptions = remember(structure, existingItemIds) {
        CourseGradingAgentsLogic.gradableOptions(structure, existingItemIds)
    }
    val filteredAgents = remember(agents, filterQuery) {
        CourseGradingAgentsLogic.filteredAgents(agents, filterQuery)
    }

    LaunchedEffect(course.courseCode) {
        val token = session.accessToken.value ?: return@LaunchedEffect
        loading = true
        loadError = null
        runCatching {
            val cached = offline.cachedFetch(
                key = CourseGradingAgentsLogic.cacheKeyAgents(course.courseCode),
                accessToken = token,
                serializer = com.lextures.android.core.lms.CourseGradingAgentsListResponse.serializer(),
            ) {
                LmsApi.fetchCourseGradingAgents(course.courseCode, token)
            }
            agents = cached.first.agents
            templates = LmsApi.fetchGraderAgentTemplates(course.courseCode, token).templates
            structure = runCatching { LmsApi.fetchCourseStructure(course.courseCode, token) }.getOrDefault(emptyList())
            cacheLabel = cached.second?.takeIf { it.isStale(isOnline) }?.lastUpdatedLabel()
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
                        Text(L.text(R.string.mobile_courseSettings_gradingAgents_introTitle), fontWeight = FontWeight.SemiBold)
                        Text(L.text(R.string.mobile_courseSettings_gradingAgents_introDescription))
                    }
                }
                item {
                    LmsCard {
                        Text(L.text(R.string.mobile_courseSettings_gradingAgents_templatesTitle), fontWeight = FontWeight.SemiBold)
                        if (templates.isEmpty()) {
                            Text(L.text(R.string.mobile_courseSettings_gradingAgents_templatesEmpty))
                        } else {
                            templates.forEach { template ->
                                Row(
                                    modifier = Modifier.fillMaxWidth(),
                                    horizontalArrangement = Arrangement.SpaceBetween,
                                ) {
                                    Text(template.name, fontWeight = FontWeight.Medium)
                                    if (template.isBuiltin == true) {
                                        Text(L.text(R.string.mobile_courseSettings_gradingAgents_templateBuiltin))
                                    }
                                }
                            }
                            Text(L.text(R.string.mobile_courseSettings_gradingAgents_templatesHint))
                        }
                    }
                }
                item {
                    LmsCard {
                        Text(L.text(R.string.mobile_courseSettings_gradingAgents_agentsTitle), fontWeight = FontWeight.SemiBold)
                        if (agents.isEmpty()) {
                            Text(L.text(R.string.mobile_courseSettings_gradingAgents_empty))
                        } else {
                            OutlinedTextField(
                                value = filterQuery,
                                onValueChange = { filterQuery = it },
                                label = { Text(L.text(R.string.mobile_courseSettings_gradingAgents_filterPlaceholder)) },
                                modifier = Modifier.fillMaxWidth(),
                            )
                            if (filteredAgents.isEmpty()) {
                                Text(L.text(R.string.mobile_courseSettings_gradingAgents_noMatch))
                            } else {
                                filteredAgents.forEachIndexed { index, agent ->
                                    GradingAgentRow(agent = agent, onClick = { selectedAgent = agent })
                                    if (index < filteredAgents.lastIndex) HorizontalDivider()
                                }
                            }
                        }
                    }
                }
            }
        }

        if (!loading) {
            Button(
                onClick = { createSheetOpen = true },
                enabled = gradableOptions.isNotEmpty(),
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(horizontal = 16.dp, vertical = 8.dp),
            ) {
                Text(L.text(R.string.mobile_courseSettings_gradingAgents_createButton))
            }
        }
    }

    selectedAgent?.let { agent ->
        GradingAgentEditorSheet(
            session = session,
            course = course,
            offline = offline,
            itemId = agent.itemId,
            itemKind = CourseGradingAgentsLogic.normalizedItemKind(agent.itemKind),
            assignmentTitle = agent.assignmentTitle,
            initialDraft = null,
            loadExistingConfig = true,
            onDismiss = { selectedAgent = null },
            onSaved = {
                selectedAgent = null
                scope.launch {
                    val token = session.accessToken.value ?: return@launch
                    runCatching {
                        agents = LmsApi.fetchCourseGradingAgents(course.courseCode, token).agents
                    }
                }
            },
            onDeleted = {
                selectedAgent = null
                scope.launch {
                    val token = session.accessToken.value ?: return@launch
                    runCatching {
                        agents = LmsApi.fetchCourseGradingAgents(course.courseCode, token).agents
                    }
                }
            },
        )
    }

    if (createSheetOpen) {
        CreateGradingAgentSheet(
            session = session,
            course = course,
            offline = offline,
            gradableOptions = gradableOptions,
            templates = templates,
            onDismiss = { createSheetOpen = false },
            onCreated = {
                createSheetOpen = false
                scope.launch {
                    val token = session.accessToken.value ?: return@launch
                    runCatching {
                        agents = LmsApi.fetchCourseGradingAgents(course.courseCode, token).agents
                    }
                }
            },
        )
    }
}

@Composable
private fun GradingAgentRow(
    agent: CourseGradingAgentSummary,
    onClick: () -> Unit,
) {
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .clickable(onClick = onClick)
            .padding(vertical = 8.dp),
        verticalArrangement = Arrangement.spacedBy(4.dp),
    ) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Text(agent.assignmentTitle, fontWeight = FontWeight.SemiBold, modifier = Modifier.weight(1f))
            CourseGradingAgentsLogic.kindLabelRes(agent.itemKind)?.let { res ->
                Text(L.text(res), color = Color(0xFF0D9488))
            }
        }
        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            Text(L.text(CourseGradingAgentsLogic.statusLabelRes(agent.status)), fontWeight = FontWeight.Medium)
            Text(
                if (agent.autoGradeNew) {
                    L.text(R.string.mobile_courseSettings_gradingAgents_autoGradeOn)
                } else {
                    L.text(R.string.mobile_courseSettings_gradingAgents_autoGradeOff)
                },
            )
        }
        if (agent.assignmentArchived) {
            Text(L.text(R.string.mobile_courseSettings_gradingAgents_archivedAssignment))
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun CreateGradingAgentSheet(
    session: AuthSession,
    course: CourseSummary,
    offline: OfflineService,
    gradableOptions: List<CourseGradingAgentsLogic.GradableOption>,
    templates: List<GraderAgentTemplateSummary>,
    onDismiss: () -> Unit,
    onCreated: () -> Unit,
) {
    val scope = rememberCoroutineScope()
    val sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)
    var itemKind by remember { mutableStateOf("assignment") }
    var selectedItemId by remember { mutableStateOf("") }
    var useTemplate by remember { mutableStateOf(false) }
    var selectedTemplateId by remember { mutableStateOf("") }
    var opening by remember { mutableStateOf(false) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var editorTarget by remember { mutableStateOf<GradingAgentEditorTarget?>(null) }

    val filteredOptions = remember(gradableOptions, itemKind) {
        gradableOptions.filter { it.kind == itemKind }
    }

    LaunchedEffect(filteredOptions) {
        if (selectedItemId.isEmpty() || filteredOptions.none { it.id == selectedItemId }) {
            selectedItemId = filteredOptions.firstOrNull()?.id.orEmpty()
        }
    }
    LaunchedEffect(templates) {
        if (selectedTemplateId.isEmpty()) {
            selectedTemplateId = templates.firstOrNull()?.id.orEmpty()
        }
    }

    ModalBottomSheet(onDismissRequest = onDismiss, sheetState = sheetState) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .verticalScroll(rememberScrollState())
                .padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            Text(L.text(R.string.mobile_courseSettings_gradingAgents_createTitle), fontWeight = FontWeight.SemiBold)
            errorMessage?.let { LmsErrorBanner(message = it) }
            Text(L.text(R.string.mobile_courseSettings_gradingAgents_createScopeTitle), fontWeight = FontWeight.Medium)
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                FilterChip(
                    selected = itemKind == "assignment",
                    onClick = { itemKind = "assignment" },
                    label = { Text(L.text(R.string.mobile_courseSettings_gradingAgents_itemKindAssignment)) },
                )
                FilterChip(
                    selected = itemKind == "quiz",
                    onClick = { itemKind = "quiz" },
                    label = { Text(L.text(R.string.mobile_courseSettings_gradingAgents_itemKindQuiz)) },
                )
            }
            if (filteredOptions.isEmpty()) {
                Text(L.text(R.string.mobile_courseSettings_gradingAgents_noAvailableItems))
            } else {
                filteredOptions.forEach { option ->
                    Text(
                        text = option.label,
                        modifier = Modifier
                            .fillMaxWidth()
                            .clickable { selectedItemId = option.id }
                            .padding(vertical = 8.dp),
                        fontWeight = if (selectedItemId == option.id) FontWeight.Bold else FontWeight.Normal,
                    )
                }
            }
            if (templates.isNotEmpty()) {
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween,
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Text(L.text(R.string.mobile_courseSettings_gradingAgents_useTemplate))
                    Switch(checked = useTemplate, onCheckedChange = { useTemplate = it })
                }
                if (useTemplate) {
                    templates.forEach { template ->
                        Text(
                            text = template.name,
                            modifier = Modifier
                                .fillMaxWidth()
                                .clickable { selectedTemplateId = template.id }
                                .padding(vertical = 6.dp),
                            fontWeight = if (selectedTemplateId == template.id) FontWeight.Bold else FontWeight.Normal,
                        )
                    }
                }
            }
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                TextButton(onClick = onDismiss) {
                    Text(L.text(R.string.mobile_common_cancel))
                }
                Button(
                    onClick = {
                        scope.launch {
                            val token = session.accessToken.value ?: return@launch
                            val option = filteredOptions.firstOrNull { it.id == selectedItemId } ?: return@launch
                            opening = true
                            errorMessage = null
                            runCatching {
                                var seedDraft = CourseGradingAgentsLogic.draft(null)
                                if (useTemplate && selectedTemplateId.isNotEmpty()) {
                                    val template = LmsApi.fetchGraderAgentTemplate(course.courseCode, selectedTemplateId, token)
                                    seedDraft = CourseGradingAgentsLogic.draft(template)
                                }
                                editorTarget = GradingAgentEditorTarget(
                                    itemId = option.id,
                                    itemKind = option.kind,
                                    assignmentTitle = option.label,
                                    seedDraft = seedDraft,
                                )
                            }.onFailure { errorMessage = it.message }
                            opening = false
                        }
                    },
                    enabled = !opening && selectedItemId.isNotEmpty(),
                ) {
                    if (opening) {
                        CircularProgressIndicator(modifier = Modifier.padding(4.dp))
                    } else {
                        Text(L.text(R.string.mobile_courseSettings_gradingAgents_continueButton))
                    }
                }
            }
        }
    }

    editorTarget?.let { target ->
        GradingAgentEditorSheet(
            session = session,
            course = course,
            offline = offline,
            itemId = target.itemId,
            itemKind = target.itemKind,
            assignmentTitle = target.assignmentTitle,
            initialDraft = target.seedDraft,
            loadExistingConfig = false,
            onDismiss = { editorTarget = null },
            onSaved = {
                editorTarget = null
                onCreated()
            },
            onDeleted = null,
        )
    }
}

private data class GradingAgentEditorTarget(
    val itemId: String,
    val itemKind: String,
    val assignmentTitle: String,
    val seedDraft: CourseGradingAgentsLogic.AgentDraft,
)

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun GradingAgentEditorSheet(
    session: AuthSession,
    course: CourseSummary,
    offline: OfflineService,
    itemId: String,
    itemKind: String,
    assignmentTitle: String,
    initialDraft: CourseGradingAgentsLogic.AgentDraft?,
    loadExistingConfig: Boolean,
    onDismiss: () -> Unit,
    onSaved: () -> Unit,
    onDeleted: (() -> Unit)?,
) {
    val scope = rememberCoroutineScope()
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    val isOnline by offline.networkMonitor.isOnline.collectAsState()
    val sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)

    var baseline by remember(itemId) { mutableStateOf(initialDraft ?: CourseGradingAgentsLogic.draft(null)) }
    var form by remember(itemId) { mutableStateOf(initialDraft ?: CourseGradingAgentsLogic.draft(null)) }
    var loadingConfig by remember(itemId) { mutableStateOf(loadExistingConfig && initialDraft == null) }
    var loadError by remember { mutableStateOf<String?>(null) }
    var actionError by remember { mutableStateOf<String?>(null) }
    var actionSuccess by remember { mutableStateOf<String?>(null) }
    var saving by remember { mutableStateOf(false) }
    var deleting by remember { mutableStateOf(false) }
    var showDeleteConfirm by remember { mutableStateOf(false) }

    val isDirty = CourseGradingAgentsLogic.isDirty(form, baseline)

    LaunchedEffect(itemId, loadExistingConfig, initialDraft) {
        if (initialDraft != null) {
            baseline = initialDraft
            form = initialDraft
            loadingConfig = false
            return@LaunchedEffect
        }
        if (!loadExistingConfig) {
            loadingConfig = false
            return@LaunchedEffect
        }
        val token = session.accessToken.value ?: return@LaunchedEffect
        loadingConfig = true
        loadError = null
        runCatching {
            val config = LmsApi.fetchGraderAgentConfig(course.courseCode, itemId, itemKind, token)
            val draft = CourseGradingAgentsLogic.draft(config)
            baseline = draft
            form = draft
        }.onFailure { loadError = it.message }
        loadingConfig = false
    }

    ModalBottomSheet(onDismissRequest = onDismiss, sheetState = sheetState) {
        Column(modifier = Modifier.fillMaxSize()) {
            Column(
                modifier = Modifier
                    .weight(1f)
                    .verticalScroll(rememberScrollState())
                    .padding(16.dp),
                verticalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                Text(L.text(R.string.mobile_courseSettings_gradingAgents_editTitle), fontWeight = FontWeight.SemiBold)
                if (!isOnline) OfflineBanner()
                loadError?.let { LmsErrorBanner(message = it) }
                actionError?.let { LmsErrorBanner(message = it) }
                actionSuccess?.let { Text(it, color = Color(0xFF0D9488), fontWeight = FontWeight.SemiBold) }

                if (loadingConfig) {
                    CircularProgressIndicator()
                } else {
                    LmsCard {
                        Text(L.text(R.string.mobile_courseSettings_gradingAgents_scopeLabel), fontWeight = FontWeight.SemiBold)
                        Text(assignmentTitle)
                        CourseGradingAgentsLogic.kindLabelRes(itemKind)?.let { res ->
                            Text(L.text(res), color = Color(0xFF0D9488))
                        }
                    }
                    LmsCard {
                        Text(L.text(R.string.mobile_courseSettings_gradingAgents_promptLabel), fontWeight = FontWeight.SemiBold)
                        OutlinedTextField(
                            value = form.prompt,
                            onValueChange = { form = form.copy(prompt = it) },
                            modifier = Modifier.fillMaxWidth(),
                            minLines = 6,
                        )
                        Row(
                            modifier = Modifier.fillMaxWidth(),
                            horizontalArrangement = Arrangement.SpaceBetween,
                            verticalAlignment = Alignment.CenterVertically,
                        ) {
                            Text(L.text(R.string.mobile_courseSettings_gradingAgents_includeContent))
                            Switch(
                                checked = form.includeAssignmentContent,
                                onCheckedChange = { form = form.copy(includeAssignmentContent = it) },
                            )
                        }
                        Row(
                            modifier = Modifier.fillMaxWidth(),
                            horizontalArrangement = Arrangement.SpaceBetween,
                            verticalAlignment = Alignment.CenterVertically,
                        ) {
                            Text(L.text(R.string.mobile_courseSettings_gradingAgents_includeRubric))
                            Switch(
                                checked = form.includeRubric,
                                onCheckedChange = { form = form.copy(includeRubric = it) },
                            )
                        }
                        Row(
                            modifier = Modifier.fillMaxWidth(),
                            horizontalArrangement = Arrangement.SpaceBetween,
                            verticalAlignment = Alignment.CenterVertically,
                        ) {
                            Text(L.text(R.string.mobile_courseSettings_gradingAgents_autoGradeNew))
                            Switch(
                                checked = form.autoGradeNew,
                                onCheckedChange = { form = form.copy(autoGradeNew = it) },
                            )
                        }
                        Text(L.text(R.string.mobile_courseSettings_gradingAgents_statusLabel), fontWeight = FontWeight.Medium)
                        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                            CourseGradingAgentsLogic.AgentStatus.entries.forEach { status ->
                                FilterChip(
                                    selected = form.status == status.apiValue,
                                    onClick = { form = form.copy(status = status.apiValue) },
                                    label = { Text(L.text(status.labelRes)) },
                                )
                            }
                        }
                        Text(L.text(R.string.mobile_courseSettings_gradingAgents_workflowHint))
                    }
                    if (onDeleted != null) {
                        TextButton(
                            onClick = { showDeleteConfirm = true },
                            enabled = !deleting,
                        ) {
                            Text(L.text(R.string.mobile_courseSettings_gradingAgents_deleteButton), color = Color.Red)
                        }
                    }
                }
            }

            if (isDirty) {
                UnsavedChangesBanner(
                    isSaving = saving,
                    onDiscard = { form = baseline },
                    onSave = {
                        scope.launch {
                            val token = session.accessToken.value ?: return@launch
                            if (CourseGradingAgentsLogic.validateDraft(form) != null) {
                                actionError = L.text(context, localePrefs, R.string.mobile_courseSettings_gradingAgents_validation_promptRequired)
                                return@launch
                            }
                            saving = true
                            actionError = null
                            actionSuccess = null
                            runCatching {
                                val body = CourseGradingAgentsLogic.buildPutBody(form, itemKind)
                                offline.enqueueMutation(
                                    method = "PUT",
                                    path = CourseGradingAgentsLogic.graderAgentPath(course.courseCode, itemId, itemKind),
                                    bodyJson = gradingAgentsJson.encodeToString(body),
                                    label = L.text(context, localePrefs, R.string.mobile_courseSettings_gradingAgents_saveLabel),
                                    accessToken = token,
                                    idempotencyKey = CourseGradingAgentsLogic.saveIdempotencyKey(course.courseCode, itemId, itemKind),
                                )
                                val refreshed = LmsApi.fetchGraderAgentConfig(course.courseCode, itemId, itemKind, token)
                                val draft = CourseGradingAgentsLogic.draft(refreshed)
                                baseline = draft
                                form = draft
                                actionSuccess = L.text(context, localePrefs, R.string.mobile_courseSettings_gradingAgents_saved)
                                onSaved()
                            }.onFailure { actionError = it.message }
                            saving = false
                        }
                    },
                )
            }
        }
    }

    if (showDeleteConfirm) {
        AlertDialog(
            onDismissRequest = { showDeleteConfirm = false },
            title = { Text(L.text(R.string.mobile_courseSettings_gradingAgents_deleteConfirmTitle)) },
            text = { Text(L.text(R.string.mobile_courseSettings_gradingAgents_deleteConfirmMessage)) },
            confirmButton = {
                TextButton(
                    onClick = {
                        scope.launch {
                            val token = session.accessToken.value ?: return@launch
                            deleting = true
                            runCatching {
                                offline.enqueueMutation(
                                    method = "DELETE",
                                    path = CourseGradingAgentsLogic.graderAgentPath(course.courseCode, itemId, itemKind),
                                    bodyJson = null,
                                    label = L.text(context, localePrefs, R.string.mobile_courseSettings_gradingAgents_deleteLabel),
                                    accessToken = token,
                                    idempotencyKey = CourseGradingAgentsLogic.deleteIdempotencyKey(course.courseCode, itemId, itemKind),
                                )
                                showDeleteConfirm = false
                                onDeleted?.invoke()
                            }.onFailure { actionError = it.message }
                            deleting = false
                        }
                    },
                ) {
                    Text(L.text(R.string.mobile_courseSettings_gradingAgents_deleteButton))
                }
            },
            dismissButton = {
                TextButton(onClick = { showDeleteConfirm = false }) {
                    Text(L.text(R.string.mobile_common_cancel))
                }
            },
        )
    }
}
