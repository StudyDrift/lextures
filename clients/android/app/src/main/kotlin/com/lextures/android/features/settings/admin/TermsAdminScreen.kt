package com.lextures.android.features.settings.admin

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.CalendarMonth
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.DatePicker
import androidx.compose.material3.DatePickerDialog
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.rememberDatePickerState
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
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalePreferences
import com.lextures.android.core.lms.AdminOrgRow
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.OrgStructureAdminLogic
import com.lextures.android.core.lms.OrgTerm
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsEmptyState
import com.lextures.android.features.home.LmsErrorBanner
import com.lextures.android.features.home.LmsSkeletonList
import kotlinx.coroutines.launch
import java.util.Date

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun TermsAdminScreen(
    session: AuthSession,
    localePrefs: LocalePreferences,
    orgId: String,
    organizations: List<AdminOrgRow>,
    canPickOrg: Boolean,
    selectedOrgId: String,
    onOrgSelected: (String) -> Unit,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    val accessToken by session.accessToken.collectAsState()
    val effectiveOrgId = if (canPickOrg) selectedOrgId else orgId
    val genericError = L.text(context, localePrefs, R.string.mobile_admin_orgStructure_error)

    var terms by remember { mutableStateOf<List<OrgTerm>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var showCreateDialog by remember { mutableStateOf(false) }
    var editTarget by remember { mutableStateOf<OrgTerm?>(null) }

    suspend fun loadTerms(token: String) {
        if (effectiveOrgId.isEmpty()) {
            terms = emptyList()
            loading = false
            return
        }
        loading = true
        errorMessage = null
        runCatching {
            terms = LmsApi.fetchOrgTerms(effectiveOrgId, token)
        }.onFailure {
            terms = emptyList()
            errorMessage = OrgStructureAdminLogic.userFacingError(it, genericError)
        }
        loading = false
    }

    LaunchedEffect(accessToken, effectiveOrgId) {
        val token = accessToken ?: return@LaunchedEffect
        loadTerms(token)
    }

    Column(
        modifier = modifier
            .verticalScroll(rememberScrollState())
            .padding(16.dp),
        verticalArrangement = Arrangement.spacedBy(16.dp),
    ) {
        WebLinkCard(
            localePrefs = localePrefs,
            hintRes = R.string.mobile_admin_orgStructure_webHint,
            path = OrgStructureAdminLogic.webTermsPath(),
        )

        if (canPickOrg && organizations.isNotEmpty()) {
            OrgPickerDropdown(
                organizations = organizations,
                selectedOrgId = selectedOrgId,
                onOrgSelected = onOrgSelected,
                localePrefs = localePrefs,
            )
        }

        Button(
            onClick = { showCreateDialog = true },
            enabled = effectiveOrgId.isNotEmpty(),
            modifier = Modifier.fillMaxWidth(),
        ) {
            Text(L.text(context, localePrefs, R.string.mobile_admin_orgStructure_terms_add))
        }

        errorMessage?.let { LmsErrorBanner(message = it) }

        when {
            loading && terms.isEmpty() -> LmsSkeletonList(count = 3)
            terms.isEmpty() -> LmsEmptyState(
                icon = Icons.Default.CalendarMonth,
                title = L.text(context, localePrefs, R.string.mobile_admin_orgStructure_terms_emptyTitle),
                message = L.text(context, localePrefs, R.string.mobile_admin_orgStructure_terms_emptyMessage),
            )
            else -> terms.forEach { term ->
                LmsCard {
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.SpaceBetween,
                    ) {
                        Column(verticalArrangement = Arrangement.spacedBy(4.dp)) {
                            Text(term.name, fontWeight = FontWeight.SemiBold, color = textPrimary())
                            Text(
                                OrgStructureAdminLogic.formatDateRange(term.startDate, term.endDate),
                                color = textSecondary(),
                            )
                            term.status?.takeIf { it.isNotBlank() }?.let {
                                Text(it.replaceFirstChar { c -> c.uppercase() }, color = textSecondary())
                            }
                        }
                        TextButton(onClick = { editTarget = term }) {
                            Text(L.text(context, localePrefs, R.string.mobile_admin_orgStructure_terms_edit))
                        }
                    }
                }
            }
        }
    }

    if (showCreateDialog) {
        TermEditorDialog(
            mode = TermEditorMode.Create,
            term = null,
            localePrefs = localePrefs,
            onDismiss = { showCreateDialog = false },
            onSave = { name, start, end ->
                val token = accessToken ?: return@TermEditorDialog
                scope.launch {
                    runCatching {
                        LmsApi.createAcademicTerm(
                            effectiveOrgId,
                            OrgStructureAdminLogic.createTermRequest(name, OrgStructureAdminLogic.DEFAULT_TERM_TYPE, start, end),
                            token,
                        )
                        showCreateDialog = false
                        loadTerms(token)
                    }.onFailure {
                        errorMessage = OrgStructureAdminLogic.userFacingError(it, genericError)
                    }
                }
            },
        )
    }

    editTarget?.let { term ->
        TermEditorDialog(
            mode = TermEditorMode.Edit,
            term = term,
            localePrefs = localePrefs,
            onDismiss = { editTarget = null },
            onSave = { _, start, end ->
                val token = accessToken ?: return@TermEditorDialog
                scope.launch {
                    runCatching {
                        LmsApi.patchAcademicTerm(
                            effectiveOrgId,
                            term.id,
                            OrgStructureAdminLogic.patchTermDatesRequest(start, end),
                            token,
                        )
                        editTarget = null
                        loadTerms(token)
                    }.onFailure {
                        errorMessage = OrgStructureAdminLogic.userFacingError(it, genericError)
                    }
                }
            },
        )
    }
}

private enum class TermEditorMode { Create, Edit }

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun TermEditorDialog(
    mode: TermEditorMode,
    term: OrgTerm?,
    localePrefs: LocalePreferences,
    onDismiss: () -> Unit,
    onSave: (name: String, start: String, end: String) -> Unit,
) {
    val context = LocalContext.current
    var name by remember { mutableStateOf(term?.name.orEmpty()) }
    var startDate by remember {
        mutableStateOf(OrgStructureAdminLogic.dateFromIso(term?.startDate) ?: Date())
    }
    var endDate by remember {
        mutableStateOf(OrgStructureAdminLogic.dateFromIso(term?.endDate) ?: Date())
    }
    var showStartPicker by remember { mutableStateOf(false) }
    var showEndPicker by remember { mutableStateOf(false) }
    var showConfirm by remember { mutableStateOf(false) }

    fun submit() {
        val start = OrgStructureAdminLogic.isoDateString(startDate)
        val end = OrgStructureAdminLogic.isoDateString(endDate)
        onSave(name, start, end)
    }

    val canSave = if (mode == TermEditorMode.Create) {
        OrgStructureAdminLogic.isValidTermName(name) &&
            OrgStructureAdminLogic.isValidDateRange(
                OrgStructureAdminLogic.isoDateString(startDate),
                OrgStructureAdminLogic.isoDateString(endDate),
            )
    } else {
        OrgStructureAdminLogic.isValidDateRange(
            OrgStructureAdminLogic.isoDateString(startDate),
            OrgStructureAdminLogic.isoDateString(endDate),
        )
    }

    AlertDialog(
        onDismissRequest = onDismiss,
        title = {
            Text(
                L.text(
                    context,
                    localePrefs,
                    if (mode == TermEditorMode.Create) {
                        R.string.mobile_admin_orgStructure_terms_addTitle
                    } else {
                        R.string.mobile_admin_orgStructure_terms_editTitle
                    },
                ),
            )
        },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                if (mode == TermEditorMode.Create) {
                    OutlinedTextField(
                        value = name,
                        onValueChange = { name = it },
                        label = { Text(L.text(context, localePrefs, R.string.mobile_admin_orgStructure_terms_name)) },
                        singleLine = true,
                        modifier = Modifier.fillMaxWidth(),
                    )
                } else {
                    Text(term?.name.orEmpty(), fontWeight = FontWeight.SemiBold)
                }
                TextButton(onClick = { showStartPicker = true }, modifier = Modifier.fillMaxWidth()) {
                    Text(
                        "${L.text(context, localePrefs, R.string.mobile_admin_orgStructure_terms_startDate)}: " +
                            OrgStructureAdminLogic.isoDateString(startDate),
                    )
                }
                TextButton(onClick = { showEndPicker = true }, modifier = Modifier.fillMaxWidth()) {
                    Text(
                        "${L.text(context, localePrefs, R.string.mobile_admin_orgStructure_terms_endDate)}: " +
                            OrgStructureAdminLogic.isoDateString(endDate),
                    )
                }
            }
        },
        confirmButton = {
            Button(
                enabled = canSave,
                onClick = {
                    if (mode == TermEditorMode.Edit) {
                        showConfirm = true
                    } else {
                        submit()
                    }
                },
            ) {
                Text(L.text(context, localePrefs, R.string.mobile_admin_orgStructure_terms_save))
            }
        },
        dismissButton = {
            TextButton(onClick = onDismiss) {
                Text(L.text(context, localePrefs, R.string.mobile_common_cancel))
            }
        },
    )

    if (showConfirm) {
        AlertDialog(
            onDismissRequest = { showConfirm = false },
            title = { Text(L.text(context, localePrefs, R.string.mobile_admin_orgStructure_terms_saveConfirm)) },
            text = { Text(L.text(context, localePrefs, R.string.mobile_admin_orgStructure_terms_saveConfirmMessage)) },
            confirmButton = {
                Button(onClick = {
                    showConfirm = false
                    submit()
                }) {
                    Text(L.text(context, localePrefs, R.string.mobile_admin_orgStructure_terms_save))
                }
            },
            dismissButton = {
                TextButton(onClick = { showConfirm = false }) {
                    Text(L.text(context, localePrefs, R.string.mobile_common_cancel))
                }
            },
        )
    }

    if (showStartPicker) {
        val state = rememberDatePickerState(initialSelectedDateMillis = startDate.time)
        DatePickerDialog(
            onDismissRequest = { showStartPicker = false },
            confirmButton = {
                TextButton(onClick = {
                    state.selectedDateMillis?.let { startDate = Date(it) }
                    showStartPicker = false
                }) { Text(L.text(context, localePrefs, R.string.mobile_admin_orgStructure_terms_save)) }
            },
            dismissButton = {
                TextButton(onClick = { showStartPicker = false }) {
                    Text(L.text(context, localePrefs, R.string.mobile_common_cancel))
                }
            },
        ) { DatePicker(state = state) }
    }

    if (showEndPicker) {
        val state = rememberDatePickerState(initialSelectedDateMillis = endDate.time)
        DatePickerDialog(
            onDismissRequest = { showEndPicker = false },
            confirmButton = {
                TextButton(onClick = {
                    state.selectedDateMillis?.let { endDate = Date(it) }
                    showEndPicker = false
                }) { Text(L.text(context, localePrefs, R.string.mobile_admin_orgStructure_terms_save)) }
            },
            dismissButton = {
                TextButton(onClick = { showEndPicker = false }) {
                    Text(L.text(context, localePrefs, R.string.mobile_common_cancel))
                }
            },
        ) { DatePicker(state = state) }
    }
}
