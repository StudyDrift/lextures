package com.lextures.android.features.settings.admin

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Check
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.rememberModalBottomSheetState
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
import com.lextures.android.core.i18n.DateFormatting
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalePreferences
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.PeopleAdminLogic
import com.lextures.android.core.lms.PersonReport
import com.lextures.android.core.lms.RoleWithPermissions
import com.lextures.android.core.lms.RolesPermissionsAdminLogic
import com.lextures.android.features.home.HomeShellState
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsErrorBanner
import com.lextures.android.features.home.LmsSkeletonList
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun UserDetailAdminScreen(
    session: AuthSession,
    shell: HomeShellState,
    localePrefs: LocalePreferences,
    userId: String,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    val accessToken by session.accessToken.collectAsState()
    val locale = localePrefs.effectiveLocale

    var report by remember { mutableStateOf<PersonReport?>(null) }
    var roles by remember { mutableStateOf<List<RoleWithPermissions>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var statusMessage by remember { mutableStateOf<String?>(null) }
    var busy by remember { mutableStateOf(false) }
    var showAssignRoleSheet by remember { mutableStateOf(false) }
    var pendingSuspend by remember { mutableStateOf(false) }
    var pendingReactivate by remember { mutableStateOf(false) }
    var pendingResendInvite by remember { mutableStateOf(false) }
    var pendingAssignRole by remember { mutableStateOf<RoleWithPermissions?>(null) }

    val genericError = L.text(context, localePrefs, R.string.mobile_admin_people_error)
    val activeLabel = L.text(context, localePrefs, R.string.mobile_admin_people_status_active)
    val suspendedLabel = L.text(context, localePrefs, R.string.mobile_admin_people_status_suspended)
    val emDash = L.text(context, localePrefs, R.string.mobile_emDash)

    suspend fun load(token: String) {
        loading = true
        errorMessage = null
        runCatching {
            report = LmsApi.fetchPersonReport(userId, token)
            roles = LmsApi.fetchRoles(token)
        }.onFailure {
            errorMessage = PeopleAdminLogic.userFacingError(it, genericError)
        }
        loading = false
    }

    LaunchedEffect(accessToken, userId) {
        val token = accessToken ?: return@LaunchedEffect
        load(token)
    }

    val assignSheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)

    Scaffold(
        modifier = modifier.fillMaxSize(),
        topBar = {
            TopAppBar(
                title = {
                    Text(
                        report?.let { PeopleAdminLogic.personDisplayName(it) }
                            ?: L.text(context, localePrefs, R.string.mobile_admin_people_title),
                    )
                },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = null)
                    }
                },
            )
        },
    ) { padding ->
        Column(
            modifier = Modifier
                .padding(padding)
                .fillMaxSize()
                .verticalScroll(rememberScrollState())
                .padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(16.dp),
        ) {
            errorMessage?.let { LmsErrorBanner(it) }
            statusMessage?.let {
                Text(it, color = textSecondary())
            }

            when {
                loading && report == null -> LmsSkeletonList(count = 2)
                report != null -> {
                    val current = report!!
                    profileCard(
                        context = context,
                        localePrefs = localePrefs,
                        report = current,
                        activeLabel = activeLabel,
                        suspendedLabel = suspendedLabel,
                        emDash = emDash,
                        locale = locale,
                    )
                    if (!PeopleAdminLogic.isErased(current.email)) {
                        actionsSection(
                            context = context,
                            localePrefs = localePrefs,
                            report = current,
                            busy = busy,
                            shell = shell,
                            onSuspend = { pendingSuspend = true },
                            onReactivate = { pendingReactivate = true },
                            onResendInvite = { pendingResendInvite = true },
                            onSelfSuspendBlocked = {
                                errorMessage = L.text(
                                    context,
                                    localePrefs,
                                    R.string.mobile_admin_people_selfSuspendBlocked,
                                )
                            },
                        )
                    }
                    roleSection(
                        context = context,
                        localePrefs = localePrefs,
                        onAssignRole = { showAssignRoleSheet = true },
                    )
                    enrollmentsSection(context, localePrefs, current, locale, emDash)
                }
            }
        }
    }

    if (pendingSuspend && report != null) {
        val current = report!!
        AlertDialog(
            onDismissRequest = { pendingSuspend = false },
            title = { Text(L.text(context, localePrefs, R.string.mobile_admin_people_suspendConfirm)) },
            text = {
                Text("${PeopleAdminLogic.personDisplayName(current)}\n${current.email}")
            },
            confirmButton = {
                TextButton(
                    onClick = {
                        pendingSuspend = false
                        val token = accessToken ?: return@TextButton
                        scope.launch {
                            busy = true
                            runCatching {
                                LmsApi.patchPerson(userId, false, token)
                            }.onSuccess {
                                statusMessage = L.text(
                                    context,
                                    localePrefs,
                                    R.string.mobile_admin_people_suspendSuccess,
                                )
                                load(token)
                            }.onFailure {
                                errorMessage = PeopleAdminLogic.userFacingError(it, genericError)
                            }
                            busy = false
                        }
                    },
                ) {
                    Text(L.text(context, localePrefs, R.string.mobile_admin_people_suspend))
                }
            },
            dismissButton = {
                TextButton(onClick = { pendingSuspend = false }) {
                    Text(L.text(context, localePrefs, R.string.mobile_common_cancel))
                }
            },
        )
    }

    if (pendingReactivate) {
        AlertDialog(
            onDismissRequest = { pendingReactivate = false },
            title = { Text(L.text(context, localePrefs, R.string.mobile_admin_people_reactivateConfirm)) },
            confirmButton = {
                TextButton(
                    onClick = {
                        pendingReactivate = false
                        val token = accessToken ?: return@TextButton
                        scope.launch {
                            busy = true
                            runCatching {
                                LmsApi.patchPerson(userId, true, token)
                            }.onSuccess {
                                statusMessage = L.text(
                                    context,
                                    localePrefs,
                                    R.string.mobile_admin_people_reactivateSuccess,
                                )
                                load(token)
                            }.onFailure {
                                errorMessage = PeopleAdminLogic.userFacingError(it, genericError)
                            }
                            busy = false
                        }
                    },
                ) {
                    Text(L.text(context, localePrefs, R.string.mobile_admin_people_reactivate))
                }
            },
            dismissButton = {
                TextButton(onClick = { pendingReactivate = false }) {
                    Text(L.text(context, localePrefs, R.string.mobile_common_cancel))
                }
            },
        )
    }

    if (pendingResendInvite && report != null) {
        AlertDialog(
            onDismissRequest = { pendingResendInvite = false },
            title = { Text(L.text(context, localePrefs, R.string.mobile_admin_people_resendInviteConfirm)) },
            text = { Text(report!!.email) },
            confirmButton = {
                TextButton(
                    onClick = {
                        pendingResendInvite = false
                        scope.launch {
                            busy = true
                            runCatching {
                                LmsApi.resendPersonInvite(report!!.email)
                            }.onSuccess {
                                statusMessage = L.text(
                                    context,
                                    localePrefs,
                                    R.string.mobile_admin_people_resendInviteSuccess,
                                )
                            }.onFailure {
                                errorMessage = PeopleAdminLogic.userFacingError(it, genericError)
                            }
                            busy = false
                        }
                    },
                ) {
                    Text(L.text(context, localePrefs, R.string.mobile_admin_people_resendInvite))
                }
            },
            dismissButton = {
                TextButton(onClick = { pendingResendInvite = false }) {
                    Text(L.text(context, localePrefs, R.string.mobile_common_cancel))
                }
            },
        )
    }

    pendingAssignRole?.let { role ->
        val current = report ?: return@let
        AlertDialog(
            onDismissRequest = { pendingAssignRole = null },
            title = { Text(L.text(context, localePrefs, R.string.mobile_admin_people_assignRole)) },
            text = {
                Text(
                    L.format(
                        context,
                        localePrefs,
                        R.string.mobile_admin_roles_assignConfirm,
                        PeopleAdminLogic.personDisplayName(current),
                        role.name,
                    ),
                )
            },
            confirmButton = {
                TextButton(
                    onClick = {
                        val token = accessToken ?: return@TextButton
                        if (RolesPermissionsAdminLogic.blocksSelfElevation(
                                role,
                                current.id,
                                shell.profile?.id,
                            )
                        ) {
                            errorMessage = L.text(
                                context,
                                localePrefs,
                                R.string.mobile_admin_roles_selfElevationBlocked,
                            )
                            pendingAssignRole = null
                            return@TextButton
                        }
                        scope.launch {
                            busy = true
                            pendingAssignRole = null
                            runCatching {
                                LmsApi.addUserToRole(role.id, current.id, token)
                            }.onSuccess {
                                statusMessage = L.text(
                                    context,
                                    localePrefs,
                                    R.string.mobile_admin_roles_assignSuccess,
                                )
                                load(token)
                            }.onFailure {
                                errorMessage = PeopleAdminLogic.userFacingError(it, genericError)
                            }
                            busy = false
                        }
                    },
                ) {
                    Text(L.text(context, localePrefs, R.string.mobile_admin_people_assignRole))
                }
            },
            dismissButton = {
                TextButton(onClick = { pendingAssignRole = null }) {
                    Text(L.text(context, localePrefs, R.string.mobile_common_cancel))
                }
            },
        )
    }

    if (showAssignRoleSheet) {
        ModalBottomSheet(
            onDismissRequest = { showAssignRoleSheet = false },
            sheetState = assignSheetState,
        ) {
            Column(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(16.dp),
                verticalArrangement = Arrangement.spacedBy(8.dp),
            ) {
                Text(
                    L.text(context, localePrefs, R.string.mobile_admin_people_assignRoleTitle),
                    fontWeight = FontWeight.SemiBold,
                    color = textPrimary(),
                )
                roles.forEach { role ->
                    Row(
                        modifier = Modifier
                            .fillMaxWidth()
                            .clickable {
                                pendingAssignRole = role
                                showAssignRoleSheet = false
                            }
                            .padding(vertical = 8.dp),
                        horizontalArrangement = Arrangement.SpaceBetween,
                    ) {
                        Column(modifier = Modifier.weight(1f)) {
                            Text(role.name, color = textPrimary())
                            role.description?.takeIf { it.isNotBlank() }?.let {
                                Text(it, color = textSecondary())
                            }
                        }
                        if (report != null && PeopleAdminLogic.roleMatchesReport(role, report!!)) {
                            Icon(Icons.Default.Check, contentDescription = null)
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun profileCard(
    context: android.content.Context,
    localePrefs: LocalePreferences,
    report: PersonReport,
    activeLabel: String,
    suspendedLabel: String,
    emDash: String,
    locale: java.util.Locale,
) {
    LmsCard {
        Column(
            modifier = Modifier.padding(12.dp),
            verticalArrangement = Arrangement.spacedBy(6.dp),
        ) {
            Text(
                PeopleAdminLogic.personDisplayName(report),
                fontWeight = FontWeight.SemiBold,
                color = textPrimary(),
            )
            Text(report.email, color = textSecondary())
            detailRow(
                context,
                localePrefs,
                R.string.mobile_admin_people_detail_org,
                report.orgName,
            )
            detailRow(
                context,
                localePrefs,
                R.string.mobile_admin_people_detail_role,
                report.role.ifBlank { emDash },
            )
            detailRow(
                context,
                localePrefs,
                R.string.mobile_admin_people_detail_status,
                PeopleAdminLogic.statusLabel(report.active, activeLabel, suspendedLabel),
            )
            detailRow(
                context,
                localePrefs,
                R.string.mobile_admin_people_detail_joined,
                DateFormatting.formatAbsoluteShort(report.createdAt, locale).ifBlank { emDash },
            )
            detailRow(
                context,
                localePrefs,
                R.string.mobile_admin_people_detail_lastActivity,
                report.lastActivityAt?.let { DateFormatting.formatAbsoluteShort(it, locale) }
                    ?.ifBlank { emDash } ?: emDash,
            )
            detailRow(
                context,
                localePrefs,
                R.string.mobile_admin_people_detail_enrollments,
                report.enrollmentCount.toString(),
            )
        }
    }
}

@Composable
private fun detailRow(
    context: android.content.Context,
    localePrefs: LocalePreferences,
    labelRes: Int,
    value: String,
) {
    Row(modifier = Modifier.fillMaxWidth()) {
        Text(
            L.text(context, localePrefs, labelRes),
            modifier = Modifier.weight(0.4f),
            color = textSecondary(),
        )
        Text(value, modifier = Modifier.weight(0.6f), color = textPrimary())
    }
}

@Composable
private fun actionsSection(
    context: android.content.Context,
    localePrefs: LocalePreferences,
    report: PersonReport,
    busy: Boolean,
    shell: HomeShellState,
    onSuspend: () -> Unit,
    onReactivate: () -> Unit,
    onResendInvite: () -> Unit,
    onSelfSuspendBlocked: () -> Unit,
) {
    Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
        Text(
            L.text(context, localePrefs, R.string.mobile_admin_people_actions),
            fontWeight = FontWeight.SemiBold,
            color = textPrimary(),
        )
        if (report.active) {
            OutlinedButton(
                onClick = {
                    if (PeopleAdminLogic.blocksSelfSuspend(report.id, shell.profile?.id)) {
                        onSelfSuspendBlocked()
                    } else {
                        onSuspend()
                    }
                },
                enabled = !busy,
                modifier = Modifier.fillMaxWidth(),
            ) {
                Text(L.text(context, localePrefs, R.string.mobile_admin_people_suspend))
            }
        } else {
            Button(
                onClick = onReactivate,
                enabled = !busy,
                modifier = Modifier.fillMaxWidth(),
            ) {
                Text(L.text(context, localePrefs, R.string.mobile_admin_people_reactivate))
            }
        }
        OutlinedButton(
            onClick = onResendInvite,
            enabled = !busy,
            modifier = Modifier.fillMaxWidth(),
        ) {
            Text(L.text(context, localePrefs, R.string.mobile_admin_people_resendInvite))
        }
    }
}

@Composable
private fun roleSection(
    context: android.content.Context,
    localePrefs: LocalePreferences,
    onAssignRole: () -> Unit,
) {
    Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
        Text(
            L.text(context, localePrefs, R.string.mobile_admin_roles_title),
            fontWeight = FontWeight.SemiBold,
            color = textPrimary(),
        )
        OutlinedButton(
            onClick = onAssignRole,
            modifier = Modifier.fillMaxWidth(),
        ) {
            Text(L.text(context, localePrefs, R.string.mobile_admin_people_assignRole))
        }
    }
}

@Composable
private fun enrollmentsSection(
    context: android.content.Context,
    localePrefs: LocalePreferences,
    report: PersonReport,
    locale: java.util.Locale,
    emDash: String,
) {
    Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
        Text(
            L.text(context, localePrefs, R.string.mobile_admin_people_detail_enrollmentsTitle),
            fontWeight = FontWeight.SemiBold,
            color = textPrimary(),
        )
        if (report.enrollments.isEmpty()) {
            Text(
                L.text(context, localePrefs, R.string.mobile_admin_people_detail_noEnrollments),
                color = textSecondary(),
            )
        } else {
            report.enrollments.forEach { enrollment ->
                LmsCard {
                    Column(modifier = Modifier.padding(12.dp)) {
                        Text(enrollment.courseTitle, fontWeight = FontWeight.Medium, color = textPrimary())
                        Text(enrollment.courseCode, color = textSecondary())
                        Text(
                            "${enrollment.role} · ${enrollment.state} · ${
                                DateFormatting.formatAbsoluteShort(enrollment.enrolledAt, locale)
                                    .ifBlank { emDash }
                            }",
                            color = textSecondary(),
                        )
                    }
                }
            }
        }
    }
}
