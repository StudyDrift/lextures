package com.lextures.android.features.profile

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.ExposedDropdownMenuBox
import androidx.compose.material3.ExposedDropdownMenuDefaults
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.OutlinedTextField
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
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.LexturesType
import com.lextures.android.core.design.accentColor
import com.lextures.android.core.design.sceneBackground
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.ProfileDepthLogic
import com.lextures.android.core.lms.ProfileFieldDefinition
import com.lextures.android.core.lms.ProfileFieldsPatch
import com.lextures.android.core.lms.StudentDemographicsPatch
import com.lextures.android.core.network.ApiError
import com.lextures.android.features.home.HomeShellState
import com.lextures.android.features.home.LmsCard
import kotlinx.coroutines.launch

private val demographicsBoolKeys = listOf(
    "freeLunch",
    "reducedLunch",
    "ellStatus",
    "disabilityStatus",
    "homelessIndicator",
    "migrantIndicator",
)

/** Demographics and org custom profile fields (M1.5). */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ProfilePersonalDetailsScreen(
    session: AuthSession,
    shell: HomeShellState,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val accessToken by session.accessToken.collectAsState()
    val scope = rememberCoroutineScope()
    val saveErrorText = L.text(R.string.mobile_profileDepth_saveError)
    val requiredMessage = L.text(R.string.mobile_profileDepth_requiredField)
    val invalidNumberMessage = L.text(R.string.mobile_profileDepth_invalidNumber)
    val invalidDateMessage = L.text(R.string.mobile_profileDepth_invalidDate)
    val invalidSelectMessage = L.text(R.string.mobile_profileDepth_invalidSelect)
    val invalidBooleanMessage = L.text(R.string.mobile_profileDepth_invalidBoolean)

    var loading by remember { mutableStateOf(true) }
    var loadFailed by remember { mutableStateOf(false) }
    var saving by remember { mutableStateOf(false) }
    var saved by remember { mutableStateOf(false) }
    var errorMessage by remember { mutableStateOf<String?>(null) }

    var fieldDefinitions by remember { mutableStateOf<List<ProfileFieldDefinition>>(emptyList()) }
    var customDraft by remember { mutableStateOf<Map<String, String>>(emptyMap()) }
    var raceSelection by remember { mutableStateOf("") }
    var boolDraft by remember { mutableStateOf<Map<String, String>>(emptyMap()) }
    var fieldErrors by remember { mutableStateOf<Map<String, String>>(emptyMap()) }

    fun load() {
        val token = accessToken ?: run {
            loading = false
            loadFailed = true
            return
        }
        scope.launch {
            loading = true
            loadFailed = false
            saved = false
            errorMessage = null
            try {
                if (shell.platformFeatures.customFieldsEnabled) {
                    val response = LmsApi.fetchMyProfileFields(token)
                    fieldDefinitions = response.fields
                    customDraft = ProfileDepthLogic.draftFromValues(response.fields, response.values)
                } else {
                    fieldDefinitions = emptyList()
                    customDraft = emptyMap()
                }
                if (shell.platformFeatures.ffDemographics) {
                    val demographics = LmsApi.fetchMyDemographics(token)
                    raceSelection = demographics.raceEthnicityCode?.takeUnless {
                        it == ProfileDepthLogic.PREFER_NOT_TO_SAY_RACE_CODE
                    }.orEmpty()
                    boolDraft = demographicsBoolKeys.associateWith { key ->
                        triStateString(
                            when (key) {
                                "freeLunch" -> demographics.freeLunch
                                "reducedLunch" -> demographics.reducedLunch
                                "ellStatus" -> demographics.ellStatus
                                "disabilityStatus" -> demographics.disabilityStatus
                                "homelessIndicator" -> demographics.homelessIndicator
                                "migrantIndicator" -> demographics.migrantIndicator
                                else -> null
                            },
                        )
                    }
                } else {
                    raceSelection = ""
                    boolDraft = emptyMap()
                }
            } catch (_: Exception) {
                loadFailed = true
            } finally {
                loading = false
            }
        }
    }

    LaunchedEffect(accessToken) { load() }

    fun save() {
        val token = accessToken ?: return
        fieldErrors = ProfileDepthLogic.validateCustomFields(
            definitions = fieldDefinitions,
            draft = customDraft,
            requiredMessage = requiredMessage,
            invalidNumberMessage = invalidNumberMessage,
            invalidDateMessage = invalidDateMessage,
            invalidSelectMessage = invalidSelectMessage,
            invalidBooleanMessage = invalidBooleanMessage,
        )
        if (fieldErrors.isNotEmpty()) return

        scope.launch {
            saving = true
            saved = false
            errorMessage = null
            try {
                if (shell.platformFeatures.customFieldsEnabled && fieldDefinitions.isNotEmpty()) {
                    val encoded = ProfileDepthLogic.encodeCustomFieldValues(fieldDefinitions, customDraft)
                    LmsApi.updateMyProfileFields(ProfileFieldsPatch(values = encoded), token)
                }
                if (shell.platformFeatures.ffDemographics) {
                    LmsApi.updateMyDemographics(
                        StudentDemographicsPatch(
                            freeLunch = parseTriState(boolDraft["freeLunch"]),
                            reducedLunch = parseTriState(boolDraft["reducedLunch"]),
                            ellStatus = parseTriState(boolDraft["ellStatus"]),
                            disabilityStatus = parseTriState(boolDraft["disabilityStatus"]),
                            raceEthnicityCode = raceSelection.ifEmpty {
                                ProfileDepthLogic.PREFER_NOT_TO_SAY_RACE_CODE
                            },
                            homelessIndicator = parseTriState(boolDraft["homelessIndicator"]),
                            migrantIndicator = parseTriState(boolDraft["migrantIndicator"]),
                        ),
                        token,
                    )
                }
                saved = true
                load()
            } catch (e: ApiError.HttpStatus) {
                errorMessage = e.apiMessage ?: saveErrorText
            } catch (_: Exception) {
                errorMessage = saveErrorText
            } finally {
                saving = false
            }
        }
    }

    Column(
        modifier = modifier
            .fillMaxSize()
            .background(sceneBackground()),
    ) {
        TopAppBar(
            title = { Text(L.text(R.string.mobile_profileDepth_personalDetails_title)) },
            navigationIcon = {
                IconButton(onClick = onBack) {
                    Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = null)
                }
            },
        )

        when {
            loading -> Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                CircularProgressIndicator(color = accentColor())
            }

            loadFailed -> Column(
                modifier = Modifier.fillMaxSize().padding(32.dp),
                horizontalAlignment = Alignment.CenterHorizontally,
                verticalArrangement = Arrangement.Center,
            ) {
                Text(L.text(R.string.mobile_profileDepth_loadError), color = textSecondary())
                TextButton(onClick = { load() }) { Text(L.text(R.string.mobile_common_retry)) }
            }

            else -> Column(
                modifier = Modifier
                    .fillMaxSize()
                    .verticalScroll(rememberScrollState())
                    .padding(16.dp),
                verticalArrangement = Arrangement.spacedBy(16.dp),
            ) {
                Text(
                    text = L.text(R.string.mobile_profileDepth_personalDetails_description),
                    fontSize = 12.sp,
                    color = textSecondary(),
                )

                if (shell.platformFeatures.ffDemographics) {
                    DemographicsSection(
                        raceSelection = raceSelection,
                        onRaceSelectionChange = { raceSelection = it },
                        boolDraft = boolDraft,
                        onBoolDraftChange = { key, value ->
                            boolDraft = boolDraft.toMutableMap().apply { put(key, value) }
                        },
                    )
                }

                if (fieldDefinitions.isNotEmpty()) {
                    CustomFieldsSection(
                        definitions = fieldDefinitions,
                        draft = customDraft,
                        errors = fieldErrors,
                        onDraftChange = { key, value ->
                            customDraft = customDraft.toMutableMap().apply { put(key, value) }
                        },
                    )
                }

                errorMessage?.let {
                    Text(text = it, color = LexturesColors.Error, fontSize = 13.sp)
                }

                if (saved) {
                    Text(
                        text = L.text(R.string.mobile_profileDepth_saved),
                        fontSize = 12.sp,
                        fontWeight = FontWeight.SemiBold,
                        color = accentColor(),
                    )
                }

                Button(
                    onClick = { save() },
                    enabled = !saving,
                    modifier = Modifier.fillMaxWidth(),
                    colors = ButtonDefaults.buttonColors(
                        containerColor = if (saved) LexturesColors.BrandTeal else LexturesColors.PrimaryDeep,
                    ),
                ) {
                    if (saving) {
                        CircularProgressIndicator(color = Color.White, modifier = Modifier.size(20.dp))
                    } else {
                        Text(
                            text = if (saved) {
                                L.text(R.string.mobile_profileDepth_saved)
                            } else {
                                L.text(R.string.mobile_common_save)
                            },
                            color = Color.White,
                            fontWeight = FontWeight.SemiBold,
                        )
                    }
                }
            }
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun DemographicsSection(
    raceSelection: String,
    onRaceSelectionChange: (String) -> Unit,
    boolDraft: Map<String, String>,
    onBoolDraftChange: (String, String) -> Unit,
) {
    LmsCard {
        Text(
            text = L.text(R.string.mobile_profileDepth_demographics_section),
            style = LexturesType.display(17),
            color = textPrimary(),
        )
        Text(
            text = L.text(R.string.mobile_profileDepth_demographics_optional),
            fontSize = 12.sp,
            color = textSecondary(),
            modifier = Modifier.padding(top = 4.dp, bottom = 8.dp),
        )

        val raceOptions = remember {
            ProfileDepthLogic.raceEthnicityOptions.filter {
                it.first != ProfileDepthLogic.PREFER_NOT_TO_SAY_RACE_CODE
            }
        }
        TriStateDropdown(
            label = L.text(R.string.mobile_profileDepth_demographics_raceEthnicityCode),
            value = raceSelection,
            displayValue = raceSelection.takeIf { it.isNotEmpty() }?.let { code ->
                raceOptions.firstOrNull { it.first == code }?.let { profileDepthRaceLabel(it.second) }
            } ?: L.text(R.string.mobile_profileDepth_preferNotToSay),
            options = listOf("" to L.text(R.string.mobile_profileDepth_preferNotToSay)) +
                raceOptions.map { (code, labelKey) -> code to profileDepthRaceLabel(labelKey) },
            onValueChange = onRaceSelectionChange,
        )

        demographicsBoolKeys.forEachIndexed { index, key ->
            if (index > 0) HorizontalDivider(modifier = Modifier.padding(vertical = 8.dp))
            TriStateDropdown(
                label = profileDepthDemographicsLabel(key),
                value = boolDraft[key].orEmpty().ifEmpty { "prefer" },
                displayValue = triStateDisplay(boolDraft[key].orEmpty().ifEmpty { "prefer" }),
                options = triStateOptions(),
                onValueChange = { onBoolDraftChange(key, it) },
            )
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun CustomFieldsSection(
    definitions: List<ProfileFieldDefinition>,
    draft: Map<String, String>,
    errors: Map<String, String>,
    onDraftChange: (String, String) -> Unit,
) {
    LmsCard {
        Text(
            text = L.text(R.string.mobile_profileDepth_customFields_section),
            style = LexturesType.display(17),
            color = textPrimary(),
        )

        definitions.forEachIndexed { index, def ->
            if (index > 0) HorizontalDivider(modifier = Modifier.padding(vertical = 8.dp))
            CustomFieldEditor(
                definition = def,
                value = draft[def.key].orEmpty(),
                onValueChange = { onDraftChange(def.key, it) },
            )
            errors[def.key]?.let { error ->
                Text(
                    text = error,
                    fontSize = 12.sp,
                    color = LexturesColors.Error,
                    modifier = Modifier.padding(top = 4.dp),
                )
            }
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun CustomFieldEditor(
    definition: ProfileFieldDefinition,
    value: String,
    onValueChange: (String) -> Unit,
) {
    val label = if (definition.isRequired) "${definition.label} *" else definition.label
    when (definition.fieldType) {
        "boolean" -> {
            val triState = value.ifEmpty { "prefer" }
            TriStateDropdown(
                label = label,
                value = triState,
                displayValue = triStateDisplay(triState),
                options = triStateOptions(),
                onValueChange = { onValueChange(if (it == "prefer") "" else it) },
            )
        }
        "select" -> {
            val options = definition.selectOptions.orEmpty()
            TriStateDropdown(
                label = label,
                value = value,
                displayValue = value.ifEmpty { L.text(R.string.mobile_emDash) },
                options = listOf("" to L.text(R.string.mobile_emDash)) + options.map { it to it },
                onValueChange = onValueChange,
            )
        }
        else -> {
            OutlinedTextField(
                value = value,
                onValueChange = onValueChange,
                label = { Text(label) },
                singleLine = true,
                placeholder = {
                    if (definition.fieldType == "date") {
                        Text("YYYY-MM-DD")
                    }
                },
                keyboardOptions = when (definition.fieldType) {
                    "number" -> KeyboardOptions(keyboardType = KeyboardType.Decimal)
                    "date" -> KeyboardOptions(keyboardType = KeyboardType.Number)
                    else -> KeyboardOptions.Default
                },
                modifier = Modifier.fillMaxWidth().padding(top = 8.dp),
            )
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun TriStateDropdown(
    label: String,
    value: String,
    displayValue: String,
    options: List<Pair<String, String>>,
    onValueChange: (String) -> Unit,
) {
    var expanded by remember { mutableStateOf(false) }
    ExposedDropdownMenuBox(
        expanded = expanded,
        onExpandedChange = { expanded = it },
        modifier = Modifier.fillMaxWidth().padding(top = 8.dp),
    ) {
        OutlinedTextField(
            value = displayValue,
            onValueChange = {},
            readOnly = true,
            label = { Text(label) },
            trailingIcon = { ExposedDropdownMenuDefaults.TrailingIcon(expanded = expanded) },
            modifier = Modifier.menuAnchor().fillMaxWidth(),
        )
        ExposedDropdownMenu(
            expanded = expanded,
            onDismissRequest = { expanded = false },
        ) {
            options.forEach { (optionValue, optionLabel) ->
                DropdownMenuItem(
                    text = { Text(optionLabel) },
                    onClick = {
                        expanded = false
                        onValueChange(optionValue)
                    },
                )
            }
        }
    }
}

@Composable
private fun triStateOptions(): List<Pair<String, String>> = listOf(
    "prefer" to L.text(R.string.mobile_profileDepth_preferNotToSay),
    "true" to L.text(R.string.mobile_profileDepth_yes),
    "false" to L.text(R.string.mobile_profileDepth_no),
)

@Composable
private fun triStateDisplay(raw: String): String = when (raw) {
    "true" -> L.text(R.string.mobile_profileDepth_yes)
    "false" -> L.text(R.string.mobile_profileDepth_no)
    else -> L.text(R.string.mobile_profileDepth_preferNotToSay)
}

private fun triStateString(value: Boolean?): String = when (value) {
    true -> "true"
    false -> "false"
    null -> "prefer"
}

private fun parseTriState(raw: String?): Boolean? {
    if (raw == null || raw == "prefer") return null
    return raw == "true"
}