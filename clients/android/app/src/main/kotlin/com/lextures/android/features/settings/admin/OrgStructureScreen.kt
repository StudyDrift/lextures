package com.lextures.android.features.settings.admin

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Folder
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalePreferences
import com.lextures.android.core.lms.AdminOrgRow
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.OrgStructureAdminLogic
import com.lextures.android.core.lms.OrgUnitTreeNode
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsEmptyState
import com.lextures.android.features.home.LmsErrorBanner
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExposedDropdownMenuBox
import androidx.compose.material3.ExposedDropdownMenuDefaults
import androidx.compose.runtime.rememberCoroutineScope
import com.lextures.android.features.home.LmsSkeletonList
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun OrgStructureScreen(
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

    var tree by remember { mutableStateOf<List<OrgUnitTreeNode>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var renameTarget by remember { mutableStateOf<OrgUnitTreeNode?>(null) }
    var renameDraft by remember { mutableStateOf("") }
    var savingRename by remember { mutableStateOf(false) }

    suspend fun loadTree(token: String) {
        if (effectiveOrgId.isEmpty()) {
            tree = emptyList()
            loading = false
            return
        }
        loading = true
        errorMessage = null
        runCatching {
            tree = LmsApi.fetchOrgUnitTree(effectiveOrgId, token)
        }.onFailure {
            tree = emptyList()
            errorMessage = OrgStructureAdminLogic.userFacingError(it, genericError)
        }
        loading = false
    }

    LaunchedEffect(accessToken, effectiveOrgId) {
        val token = accessToken ?: return@LaunchedEffect
        loadTree(token)
    }

    Column(
        modifier = modifier
            .verticalScroll(rememberScrollState())
            .padding(16.dp),
        verticalArrangement = Arrangement.spacedBy(16.dp),
    ) {
        WebLinkCard(
            localePrefs = localePrefs,
            hintRes = R.string.mobile_admin_orgStructure_webHintOrgUnits,
            path = OrgStructureAdminLogic.webOrgUnitsPath(),
        )

        if (canPickOrg && organizations.isNotEmpty()) {
            OrgPickerDropdown(
                organizations = organizations,
                selectedOrgId = selectedOrgId,
                onOrgSelected = onOrgSelected,
                localePrefs = localePrefs,
            )
        }

        errorMessage?.let { LmsErrorBanner(message = it) }

        when {
            loading && tree.isEmpty() -> LmsSkeletonList(count = 4)
            tree.isEmpty() -> LmsEmptyState(
                icon = Icons.Default.Folder,
                title = L.text(context, localePrefs, R.string.mobile_admin_orgStructure_orgUnits_emptyTitle),
                message = L.text(context, localePrefs, R.string.mobile_admin_orgStructure_orgUnits_emptyMessage),
            )
            else -> tree.forEach { node ->
                OrgUnitTreeBranch(
                    node = node,
                    depth = 0,
                    localePrefs = localePrefs,
                    onRename = {
                        renameTarget = it
                        renameDraft = it.name
                    },
                )
            }
        }
    }

    renameTarget?.let { unit ->
        AlertDialog(
            onDismissRequest = { if (!savingRename) renameTarget = null },
            title = { Text(L.text(context, localePrefs, R.string.mobile_admin_orgStructure_orgUnits_renameTitle)) },
            text = {
                OutlinedTextField(
                    value = renameDraft,
                    onValueChange = { renameDraft = it },
                    singleLine = true,
                    modifier = Modifier.fillMaxSize(),
                )
            },
            confirmButton = {
                Button(
                    enabled = !savingRename && OrgStructureAdminLogic.isValidTermName(renameDraft),
                    onClick = {
                        val token = accessToken ?: return@Button
                        savingRename = true
                        scope.launch {
                            runCatching {
                                LmsApi.patchOrgUnit(
                                    effectiveOrgId,
                                    unit.id,
                                    OrgStructureAdminLogic.patchOrgUnitNameRequest(renameDraft),
                                    token,
                                )
                                renameTarget = null
                                loadTree(token)
                            }.onFailure {
                                errorMessage = OrgStructureAdminLogic.userFacingError(it, genericError)
                            }
                            savingRename = false
                        }
                    },
                ) {
                    Text(L.text(context, localePrefs, R.string.mobile_admin_orgStructure_orgUnits_renameSave))
                }
            },
            dismissButton = {
                TextButton(onClick = { renameTarget = null }) {
                    Text(L.text(context, localePrefs, R.string.mobile_common_cancel))
                }
            },
        )
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
internal fun OrgPickerDropdown(
    organizations: List<AdminOrgRow>,
    selectedOrgId: String,
    onOrgSelected: (String) -> Unit,
    localePrefs: LocalePreferences,
) {
    val context = LocalContext.current
    var expanded by remember { mutableStateOf(false) }
    val selectedName = organizations.firstOrNull { it.id == selectedOrgId }?.name.orEmpty()

    LmsCard {
        ExposedDropdownMenuBox(expanded = expanded, onExpandedChange = { expanded = it }) {
            OutlinedTextField(
                value = selectedName,
                onValueChange = {},
                readOnly = true,
                label = { Text(L.text(context, localePrefs, R.string.mobile_admin_orgStructure_selectOrg)) },
                trailingIcon = { ExposedDropdownMenuDefaults.TrailingIcon(expanded = expanded) },
                modifier = Modifier
                    .menuAnchor()
                    .fillMaxWidth(),
            )
            ExposedDropdownMenu(expanded = expanded, onDismissRequest = { expanded = false }) {
                organizations.forEach { org ->
                    DropdownMenuItem(
                        text = { Text(org.name) },
                        onClick = {
                            onOrgSelected(org.id)
                            expanded = false
                        },
                    )
                }
            }
        }
    }
}

@Composable
private fun OrgUnitTreeBranch(
    node: OrgUnitTreeNode,
    depth: Int,
    localePrefs: LocalePreferences,
    onRename: (OrgUnitTreeNode) -> Unit,
) {
    val context = LocalContext.current
    var expanded by remember { mutableStateOf(true) }
    val children = node.children.orEmpty()

    Column(modifier = Modifier.padding(start = (depth * 14).dp)) {
        Row(
            modifier = Modifier
                .fillMaxSize()
                .padding(vertical = 6.dp),
            horizontalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            TextButton(onClick = { expanded = !expanded }, enabled = children.isNotEmpty()) {
                Text(if (expanded) "▼" else "▶")
            }
            Column(modifier = Modifier.weight(1f)) {
                Text(node.name, fontWeight = FontWeight.SemiBold, color = textPrimary())
                Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    Text(node.unitType, fontSize = 12.sp, color = textSecondary())
                    node.childCourseCount?.takeIf { it > 0 }?.let { count ->
                        Text(
                            L.format(context, localePrefs, R.string.mobile_admin_orgStructure_orgUnits_courseCount, count),
                            fontSize = 12.sp,
                            color = textSecondary(),
                        )
                    }
                }
            }
            TextButton(onClick = { onRename(node) }) {
                Text(L.text(context, localePrefs, R.string.mobile_admin_orgStructure_orgUnits_rename))
            }
        }
        if (expanded) {
            children.forEach { child ->
                OrgUnitTreeBranch(child, depth + 1, localePrefs, onRename)
            }
        }
    }
}
