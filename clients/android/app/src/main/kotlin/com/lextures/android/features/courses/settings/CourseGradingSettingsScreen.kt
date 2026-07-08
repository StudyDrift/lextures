package com.lextures.android.features.courses.settings

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.ExposedDropdownMenuBox
import androidx.compose.material3.ExposedDropdownMenuDefaults
import androidx.compose.material3.HorizontalDivider
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
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.OfflineBanner
import com.lextures.android.core.design.StalenessChip
import com.lextures.android.core.design.UnsavedChangesBanner
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalLocalePreferences
import com.lextures.android.core.lms.CourseGradingLogic
import com.lextures.android.core.lms.CourseStructureItem
import com.lextures.android.core.lms.CourseSummary
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.PatchItemAssignmentGroupBody
import com.lextures.android.core.offline.OfflineService
import com.lextures.android.core.offline.OutboxStatus
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsErrorBanner
import com.lextures.android.features.home.LmsSkeletonList
import kotlinx.coroutines.launch
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json

private val gradingJson = Json { ignoreUnknownKeys = true }

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun CourseGradingSettingsScreen(
    session: AuthSession,
    course: CourseSummary,
    offline: OfflineService,
) {
    val scope = rememberCoroutineScope()
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    val isOnline by offline.networkMonitor.isOnline.collectAsState()

    var baseline by remember(course.courseCode) {
        mutableStateOf(
            CourseGradingLogic.FormBaseline(
                gradingScale = "letter_standard",
                groups = CourseGradingLogic.defaultGroups(),
                schemeType = "points",
                bands = CourseGradingLogic.defaultBands(),
                passMinPct = "60",
                completeMinPct = "50",
            ),
        )
    }
    var form by remember(course.courseCode) { mutableStateOf(baseline) }
    var structure by remember { mutableStateOf<List<CourseStructureItem>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var loadError by remember { mutableStateOf<String?>(null) }
    var actionError by remember { mutableStateOf<String?>(null) }
    var actionSuccess by remember { mutableStateOf<String?>(null) }
    var cacheLabel by remember { mutableStateOf<String?>(null) }
    var saving by remember { mutableStateOf(false) }
    var itemPatchingId by remember { mutableStateOf<String?>(null) }
    var pendingItemIds by remember { mutableStateOf(setOf<String>()) }

    val weightTotal = CourseGradingLogic.weightTotal(form.groups)
    val isDirty = CourseGradingLogic.isSettingsDirty(form, baseline) || CourseGradingLogic.isSchemeDirty(form, baseline)
    val gradableRows = CourseGradingLogic.gradableRows(structure)

    LaunchedEffect(course.courseCode) {
        val token = session.accessToken.value ?: return@LaunchedEffect
        loading = true
        loadError = null
        runCatching {
            val settingsResult = offline.cachedFetch(
                key = CourseGradingLogic.cacheKeyGrading(course.courseCode),
                accessToken = token,
                serializer = com.lextures.android.core.lms.CourseGradingSettings.serializer(),
            ) {
                LmsApi.fetchCourseGradingSettings(course.courseCode, token)
            }
            val scheme = runCatching { LmsApi.fetchCourseGradingScheme(course.courseCode, token) }.getOrNull()
            val loaded = CourseGradingLogic.baseline(settingsResult.first, scheme)
            baseline = loaded
            form = loaded
            structure = runCatching { LmsApi.fetchCourseStructure(course.courseCode, token) }.getOrDefault(emptyList())
            cacheLabel = settingsResult.second?.takeIf { it.isStale(isOnline) }?.lastUpdatedLabel()
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
                        Text(L.text(R.string.mobile_courseSettings_grading_scaleTitle), fontWeight = FontWeight.SemiBold)
                        Text(L.text(R.string.mobile_courseSettings_grading_scaleDescription))
                        CourseGradingLogic.gradingScaleOptions.forEach { option ->
                            val optionId = option.id.name
                            Row(
                                modifier = Modifier
                                    .fillMaxWidth()
                                    .clickable { form = form.copy(gradingScale = optionId) }
                                    .padding(vertical = 8.dp),
                                verticalAlignment = Alignment.CenterVertically,
                            ) {
                                RadioButton(
                                    selected = form.gradingScale == optionId,
                                    onClick = { form = form.copy(gradingScale = optionId) },
                                )
                                Column {
                                    Text(
                                        L.text(context, localePrefs, CourseGradingLogic.gradingScaleLabelRes(option.id)),
                                        fontWeight = FontWeight.SemiBold,
                                    )
                                    Text(L.text(context, localePrefs, CourseGradingLogic.gradingScaleDescriptionRes(option.id)))
                                }
                            }
                        }
                    }
                }

                item {
                    LmsCard {
                        Text(L.text(R.string.mobile_courseSettings_grading_schemeTitle), fontWeight = FontWeight.SemiBold)
                        Text(L.text(R.string.mobile_courseSettings_grading_schemeDescription))
                        SchemeTypePicker(form = form, onChange = { form = it })
                        when (form.schemeType) {
                            "letter", "gpa" -> {
                                Text(L.text(R.string.mobile_courseSettings_grading_bandsTitle), fontWeight = FontWeight.SemiBold)
                                form.bands.forEach { band ->
                                    Row(horizontalArrangement = Arrangement.spacedBy(8.dp), verticalAlignment = Alignment.CenterVertically) {
                                        OutlinedTextField(
                                            value = band.label,
                                            onValueChange = { value ->
                                                form = form.copy(
                                                    bands = form.bands.map {
                                                        if (it.clientKey == band.clientKey) it.copy(label = value) else it
                                                    },
                                                )
                                            },
                                            label = { Text(L.text(R.string.mobile_courseSettings_grading_bandLabel)) },
                                            modifier = Modifier.weight(1f),
                                        )
                                        OutlinedTextField(
                                            value = band.minPct,
                                            onValueChange = { value ->
                                                form = form.copy(
                                                    bands = form.bands.map {
                                                        if (it.clientKey == band.clientKey) it.copy(minPct = value) else it
                                                    },
                                                )
                                            },
                                            label = { Text(L.text(R.string.mobile_courseSettings_grading_bandMinPct)) },
                                            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Decimal),
                                            modifier = Modifier.weight(1f),
                                        )
                                        if (form.bands.size > 1) {
                                            TextButton(onClick = {
                                                form = form.copy(bands = form.bands.filter { it.clientKey != band.clientKey })
                                            }) {
                                                Text(L.text(R.string.mobile_courseSettings_grading_removeBand))
                                            }
                                        }
                                    }
                                }
                                Button(onClick = {
                                    form = form.copy(
                                        bands = form.bands + CourseGradingLogic.GradingSchemeBand(
                                            CourseGradingLogic.newClientKey(), "", "0",
                                        ),
                                    )
                                }) {
                                    Text(L.text(R.string.mobile_courseSettings_grading_addBand))
                                }
                            }
                            "pass_fail" -> {
                                OutlinedTextField(
                                    value = form.passMinPct,
                                    onValueChange = { form = form.copy(passMinPct = it) },
                                    label = { Text(L.text(R.string.mobile_courseSettings_grading_passMinPct)) },
                                    keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Decimal),
                                    modifier = Modifier.fillMaxWidth(),
                                )
                            }
                            "complete_incomplete" -> {
                                OutlinedTextField(
                                    value = form.completeMinPct,
                                    onValueChange = { form = form.copy(completeMinPct = it) },
                                    label = { Text(L.text(R.string.mobile_courseSettings_grading_completeMinPct)) },
                                    keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Decimal),
                                    modifier = Modifier.fillMaxWidth(),
                                )
                            }
                        }
                    }
                }

                item {
                    LmsCard {
                        Text(L.text(R.string.mobile_courseSettings_grading_groupsTitle), fontWeight = FontWeight.SemiBold)
                        Text(L.text(R.string.mobile_courseSettings_grading_groupsDescription))
                        form.groups.forEach { group ->
                            Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                                OutlinedTextField(
                                    value = group.name,
                                    onValueChange = { value ->
                                        form = form.copy(
                                            groups = form.groups.map {
                                                if (it.clientKey == group.clientKey) it.copy(name = value) else it
                                            },
                                        )
                                    },
                                    label = { Text(L.text(R.string.mobile_courseSettings_grading_groupName)) },
                                    modifier = Modifier.fillMaxWidth(),
                                )
                                Row(horizontalArrangement = Arrangement.spacedBy(8.dp), verticalAlignment = Alignment.CenterVertically) {
                                    OutlinedTextField(
                                        value = group.weightPercent,
                                        onValueChange = { value ->
                                            form = form.copy(
                                                groups = form.groups.map {
                                                    if (it.clientKey == group.clientKey) it.copy(weightPercent = value) else it
                                                },
                                            )
                                        },
                                        label = { Text(L.text(R.string.mobile_courseSettings_grading_groupWeight)) },
                                        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Decimal),
                                        modifier = Modifier.weight(1f),
                                    )
                                    if (form.groups.size > 1) {
                                        TextButton(onClick = {
                                            form = form.copy(groups = form.groups.filter { it.clientKey != group.clientKey })
                                        }) {
                                            Text(L.text(R.string.mobile_courseSettings_grading_removeGroup))
                                        }
                                    }
                                }
                                HorizontalDivider()
                            }
                        }
                        Button(onClick = {
                            form = form.copy(
                                groups = form.groups + CourseGradingLogic.EditableAssignmentGroup(
                                    CourseGradingLogic.newClientKey(), null, "", form.groups.size, "0",
                                ),
                            )
                        }) {
                            Text(L.text(R.string.mobile_courseSettings_grading_addGroup))
                        }
                        Text(
                            text = L.format(context, localePrefs, R.string.mobile_courseSettings_grading_weightTotal, "%.2f".format(weightTotal)),
                            fontWeight = FontWeight.SemiBold,
                            color = if (CourseGradingLogic.hasWeightWarning(weightTotal)) Color(0xFFE67E22) else Color.Unspecified,
                        )
                        if (CourseGradingLogic.hasWeightWarning(weightTotal)) {
                            Text(L.text(R.string.mobile_courseSettings_grading_weightWarning))
                        }
                    }
                }

                item {
                    LmsCard {
                        Text(L.text(R.string.mobile_courseSettings_grading_mappingTitle), fontWeight = FontWeight.SemiBold)
                        Text(L.text(R.string.mobile_courseSettings_grading_mappingDescription))
                        if (gradableRows.isEmpty()) {
                            Text(L.text(R.string.mobile_courseSettings_grading_mappingEmpty))
                        } else {
                            val namedGroups = CourseGradingLogic.namedGroupsWithIds(form.groups)
                            gradableRows.forEach { row ->
                                MappingRow(
                                    row = row,
                                    selectedGroupId = structure.firstOrNull { it.id == row.item.id }?.assignmentGroupId.orEmpty(),
                                    namedGroups = namedGroups,
                                    isPatching = itemPatchingId == row.item.id,
                                    isPending = pendingItemIds.contains(row.item.id),
                                    onGroupSelected = { groupId ->
                                        scope.launch {
                                            val token = session.accessToken.value ?: return@launch
                                            itemPatchingId = row.item.id
                                            actionError = null
                                            val previous = structure
                                            structure = structure.map { item ->
                                                if (item.id == row.item.id) item.copy(assignmentGroupId = groupId) else item
                                            }
                                            runCatching {
                                                val outboxItem = offline.enqueueMutation(
                                                    method = "PATCH",
                                                    path = "/api/v1/courses/${course.courseCode}/structure/items/${row.item.id}/assignment-group",
                                                    bodyJson = gradingJson.encodeToString(
                                                        PatchItemAssignmentGroupBody(groupId),
                                                    ),
                                                    label = L.text(context, localePrefs, R.string.mobile_courseSettings_grading_mappingSaveLabel),
                                                    accessToken = token,
                                                    idempotencyKey = CourseGradingLogic.itemMappingIdempotencyKey(course.courseCode, row.item.id),
                                                )
                                                if (outboxItem.outboxStatus() != OutboxStatus.Synced) {
                                                    pendingItemIds = pendingItemIds + row.item.id
                                                } else {
                                                    pendingItemIds = pendingItemIds - row.item.id
                                                    structure = runCatching {
                                                        LmsApi.fetchCourseStructure(course.courseCode, token)
                                                    }.getOrDefault(structure)
                                                }
                                            }.onFailure {
                                                structure = previous
                                                actionError = it.message
                                            }
                                            itemPatchingId = null
                                        }
                                    },
                                )
                                HorizontalDivider()
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
                    form = baseline
                    actionError = null
                    actionSuccess = null
                },
                onSave = {
                    scope.launch {
                        val token = session.accessToken.value ?: return@launch
                        actionError = null
                        actionSuccess = null
                        val settingsDirty = CourseGradingLogic.isSettingsDirty(form, baseline)
                        val schemeDirty = CourseGradingLogic.isSchemeDirty(form, baseline)
                        if (settingsDirty && CourseGradingLogic.validateGroups(form.groups) != null) {
                            actionError = L.text(context, localePrefs, R.string.mobile_courseSettings_grading_validation_groupsNeedNames)
                            return@launch
                        }
                        if (schemeDirty) {
                            when (val error = CourseGradingLogic.validateScheme(form)) {
                                is CourseGradingLogic.ValidationError.BandsInvalid ->
                                    actionError = L.text(context, localePrefs, R.string.mobile_courseSettings_grading_validation_bandsInvalid)
                                is CourseGradingLogic.ValidationError.SchemeInvalid ->
                                    actionError = L.text(context, localePrefs, R.string.mobile_courseSettings_grading_validation_schemeInvalid)
                                else -> Unit
                            }
                            if (actionError != null) return@launch
                        }
                        saving = true
                        runCatching {
                            if (settingsDirty) {
                                offline.enqueueMutation(
                                    method = "PUT",
                                    path = "/api/v1/courses/${course.courseCode}/grading",
                                    bodyJson = gradingJson.encodeToString(CourseGradingLogic.buildPutSettingsBody(form)),
                                    label = L.text(context, localePrefs, R.string.mobile_courseSettings_grading_saveSettingsLabel),
                                    accessToken = token,
                                    idempotencyKey = CourseGradingLogic.settingsIdempotencyKey(course.courseCode),
                                )
                            }
                            if (schemeDirty) {
                                offline.enqueueMutation(
                                    method = "PUT",
                                    path = "/api/v1/courses/${course.courseCode}/grading-scheme",
                                    bodyJson = gradingJson.encodeToString(CourseGradingLogic.buildPutSchemeBody(form)),
                                    label = L.text(context, localePrefs, R.string.mobile_courseSettings_grading_saveSchemeLabel),
                                    accessToken = token,
                                    idempotencyKey = CourseGradingLogic.schemeIdempotencyKey(course.courseCode),
                                )
                            }
                            val settings = LmsApi.fetchCourseGradingSettings(course.courseCode, token)
                            val scheme = runCatching { LmsApi.fetchCourseGradingScheme(course.courseCode, token) }.getOrNull()
                            val loaded = CourseGradingLogic.baseline(settings, scheme)
                            baseline = loaded
                            form = loaded
                            structure = runCatching { LmsApi.fetchCourseStructure(course.courseCode, token) }.getOrDefault(structure)
                            actionSuccess = when {
                                settingsDirty && schemeDirty -> L.text(context, localePrefs, R.string.mobile_courseSettings_grading_savedBoth)
                                schemeDirty -> L.text(context, localePrefs, R.string.mobile_courseSettings_grading_savedScheme)
                                else -> L.text(context, localePrefs, R.string.mobile_courseSettings_grading_savedSettings)
                            }
                        }.onFailure { actionError = it.message }
                        saving = false
                    }
                },
            )
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun SchemeTypePicker(
    form: CourseGradingLogic.FormBaseline,
    onChange: (CourseGradingLogic.FormBaseline) -> Unit,
) {
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    var expanded by remember { mutableStateOf(false) }
    val selected = CourseGradingLogic.schemeDisplayTypes.firstOrNull { it.id.name == form.schemeType }
    ExposedDropdownMenuBox(expanded = expanded, onExpandedChange = { expanded = it }) {
        OutlinedTextField(
            value = selected?.let { L.text(context, localePrefs, CourseGradingLogic.schemeTypeLabelRes(it.id)) }.orEmpty(),
            onValueChange = {},
            readOnly = true,
            label = { Text(L.text(R.string.mobile_courseSettings_grading_schemeDisplayAs)) },
            trailingIcon = { ExposedDropdownMenuDefaults.TrailingIcon(expanded) },
            modifier = Modifier.menuAnchor().fillMaxWidth(),
        )
        ExposedDropdownMenu(expanded = expanded, onDismissRequest = { expanded = false }) {
            CourseGradingLogic.schemeDisplayTypes.forEach { type ->
                DropdownMenuItem(
                    text = { Text(L.text(context, localePrefs, CourseGradingLogic.schemeTypeLabelRes(type.id))) },
                    onClick = {
                        onChange(form.copy(schemeType = type.id.name))
                        expanded = false
                    },
                )
            }
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun MappingRow(
    row: CourseGradingLogic.GradableRow,
    selectedGroupId: String,
    namedGroups: List<CourseGradingLogic.EditableAssignmentGroup>,
    isPatching: Boolean,
    isPending: Boolean,
    onGroupSelected: (String?) -> Unit,
) {
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    var expanded by remember { mutableStateOf(false) }
    val selectedLabel = when {
        selectedGroupId.isBlank() -> L.text(R.string.mobile_courseSettings_grading_mappingNone)
        else -> namedGroups.firstOrNull { it.id == selectedGroupId }?.name.orEmpty()
    }

    Column(verticalArrangement = Arrangement.spacedBy(6.dp), modifier = Modifier.padding(vertical = 8.dp)) {
        if (row.moduleTitle.isNotBlank()) Text(row.moduleTitle)
        Row(verticalAlignment = Alignment.CenterVertically) {
            Column(modifier = Modifier.weight(1f)) {
                Text(row.item.title, fontWeight = FontWeight.SemiBold)
                Text(L.text(context, localePrefs, CourseGradingLogic.kindLabelRes(row.item.kind)))
            }
            if (isPatching) CircularProgressIndicator()
            else if (isPending) Text(L.text(R.string.mobile_courseSettings_grading_pending))
        }
        ExposedDropdownMenuBox(expanded = expanded, onExpandedChange = { if (!isPatching) expanded = it }) {
            OutlinedTextField(
                value = selectedLabel,
                onValueChange = {},
                readOnly = true,
                enabled = !isPatching && namedGroups.isNotEmpty(),
                modifier = Modifier.menuAnchor().fillMaxWidth(),
                trailingIcon = { ExposedDropdownMenuDefaults.TrailingIcon(expanded) },
            )
            ExposedDropdownMenu(expanded = expanded, onDismissRequest = { expanded = false }) {
                DropdownMenuItem(
                    text = { Text(L.text(R.string.mobile_courseSettings_grading_mappingNone)) },
                    onClick = {
                        onGroupSelected(null)
                        expanded = false
                    },
                )
                namedGroups.forEach { group ->
                    DropdownMenuItem(
                        text = { Text(group.name) },
                        onClick = {
                            onGroupSelected(group.id)
                            expanded = false
                        },
                    )
                }
            }
        }
    }
}