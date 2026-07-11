package com.lextures.android.features.settings.admin

import android.content.Intent
import android.net.Uri
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
import androidx.compose.material.icons.filled.Lock
import androidx.compose.material.icons.filled.OpenInBrowser
import androidx.compose.material.icons.filled.Person
import androidx.compose.material.icons.filled.Search
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.OutlinedTextField
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
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.config.AppConfiguration
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalePreferences
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.RbacUserBrief
import com.lextures.android.core.lms.RoleWithPermissions
import com.lextures.android.core.lms.RolesPermissionsAdminLogic
import com.lextures.android.features.home.HomeShellState
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsEmptyState
import com.lextures.android.features.home.LmsErrorBanner
import com.lextures.android.features.home.LmsSkeletonList
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun RolesPermissionsAdminScreen(
    session: AuthSession,
    shell: HomeShellState,
    localePrefs: LocalePreferences,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    val accessToken by session.accessToken.collectAsState()

    var roles by remember { mutableStateOf<List<RoleWithPermissions>>(emptyList()) }
    var searchText by remember { mutableStateOf("") }
    var loading by remember { mutableStateOf(true) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var selectedRole by remember { mutableStateOf<RoleWithPermissions?>(null) }

    val canView = RolesPermissionsAdminLogic.canView(shell.platformFeatures, shell.permissions)
    val filteredRoles = RolesPermissionsAdminLogic.filterRoles(roles, searchText)
    val genericError = L.text(context, localePrefs, R.string.mobile_admin_roles_error)

    suspend fun load(token: String) {
        loading = true
        errorMessage = null
        runCatching {
            roles = LmsApi.fetchRoles(token)
        }.onFailure {
            errorMessage = RolesPermissionsAdminLogic.userFacingError(it, genericError)
        }
        loading = false
    }

    LaunchedEffect(accessToken) {
        val token = accessToken ?: return@LaunchedEffect
        if (canView) load(token)
    }

    if (selectedRole != null) {
        RolesPermissionsRoleDetailScreen(
            session = session,
            shell = shell,
            localePrefs = localePrefs,
            role = selectedRole!!,
            onBack = { selectedRole = null },
            modifier = modifier,
        )
        return
    }

    Scaffold(
        modifier = modifier.fillMaxSize(),
        topBar = {
            TopAppBar(
                title = { Text(L.text(context, localePrefs, R.string.mobile_admin_roles_title)) },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = null)
                    }
                },
            )
        },
    ) { padding ->
        if (!canView) {
            LmsEmptyState(
                icon = Icons.Default.Lock,
                title = L.text(context, localePrefs, R.string.mobile_admin_roles_accessDeniedTitle),
                message = L.text(context, localePrefs, R.string.mobile_admin_roles_accessDeniedMessage),
                modifier = Modifier.padding(padding).padding(16.dp),
            )
            return@Scaffold
        }

        Column(
            modifier = Modifier
                .padding(padding)
                .fillMaxSize()
                .verticalScroll(rememberScrollState())
                .padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(16.dp),
        ) {
            Text(
                L.text(context, localePrefs, R.string.mobile_admin_roles_description),
                color = textSecondary(),
            )

            OutlinedTextField(
                value = searchText,
                onValueChange = { searchText = it },
                modifier = Modifier.fillMaxWidth(),
                label = { Text(L.text(context, localePrefs, R.string.mobile_admin_roles_search)) },
                leadingIcon = { Icon(Icons.Default.Search, contentDescription = null) },
                singleLine = true,
            )

            LmsCard {
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .clickable {
                            val url = AppConfiguration.apiUrl(
                                RolesPermissionsAdminLogic.webSettingsPath(),
                            ).toString()
                            context.startActivity(Intent(Intent.ACTION_VIEW, Uri.parse(url)))
                        }
                        .padding(12.dp),
                    horizontalArrangement = Arrangement.spacedBy(10.dp),
                ) {
                    Icon(Icons.Default.OpenInBrowser, contentDescription = null)
                    Column(modifier = Modifier.weight(1f)) {
                        Text(
                            L.text(context, localePrefs, R.string.mobile_admin_roles_webTitle),
                            fontWeight = FontWeight.SemiBold,
                            color = textPrimary(),
                        )
                        Text(
                            L.text(context, localePrefs, R.string.mobile_admin_roles_webHint),
                            color = textSecondary(),
                        )
                    }
                }
            }

            errorMessage?.let { LmsErrorBanner(message = it) }

            when {
                loading && roles.isEmpty() -> LmsSkeletonList(count = 3)
                roles.isEmpty() -> LmsEmptyState(
                    icon = Icons.Default.Person,
                    title = L.text(context, localePrefs, R.string.mobile_admin_roles_emptyTitle),
                    message = L.text(context, localePrefs, R.string.mobile_admin_roles_emptyMessage),
                )
                filteredRoles.isEmpty() -> LmsEmptyState(
                    icon = Icons.Default.Search,
                    title = L.text(context, localePrefs, R.string.mobile_admin_roles_emptyTitle),
                    message = L.text(context, localePrefs, R.string.mobile_admin_roles_emptySearch),
                )
                else -> filteredRoles.forEach { role ->
                    LmsCard {
                        Column(
                            modifier = Modifier
                                .fillMaxWidth()
                                .clickable { selectedRole = role }
                                .padding(12.dp),
                        ) {
                            Text(role.name, fontWeight = FontWeight.SemiBold, color = textPrimary())
                            role.description?.trim()?.takeIf { it.isNotEmpty() }?.let {
                                Text(it, color = textSecondary())
                            }
                            Text(
                                L.format(
                                    context,
                                    localePrefs,
                                    R.string.mobile_admin_roles_permissionCount,
                                    role.permissions.size,
                                ),
                                color = textSecondary(),
                            )
                        }
                    }
                }
            }
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun RolesPermissionsRoleDetailScreen(
    session: AuthSession,
    shell: HomeShellState,
    localePrefs: LocalePreferences,
    role: RoleWithPermissions,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    val accessToken by session.accessToken.collectAsState()
    val sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)

    var members by remember { mutableStateOf<List<RbacUserBrief>>(emptyList()) }
    var eligible by remember { mutableStateOf<List<RbacUserBrief>>(emptyList()) }
    var permissionSearch by remember { mutableStateOf("") }
    var assignSearch by remember { mutableStateOf("") }
    var loadingMembers by remember { mutableStateOf(true) }
    var loadingEligible by remember { mutableStateOf(false) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var statusMessage by remember { mutableStateOf<String?>(null) }
    var busyUserId by remember { mutableStateOf<String?>(null) }
    var showAssignSheet by remember { mutableStateOf(false) }
    var pendingAssign by remember { mutableStateOf<RbacUserBrief?>(null) }
    var pendingRemove by remember { mutableStateOf<RbacUserBrief?>(null) }

    val genericError = L.text(context, localePrefs, R.string.mobile_admin_roles_error)
    val filteredPermissions = RolesPermissionsAdminLogic.filterPermissions(role.permissions, permissionSearch)
    val filteredEligible = RolesPermissionsAdminLogic.filterUsers(eligible, assignSearch)

    suspend fun loadMembers(token: String) {
        loadingMembers = true
        errorMessage = null
        runCatching {
            members = LmsApi.fetchRoleUsers(role.id, token)
        }.onFailure {
            errorMessage = RolesPermissionsAdminLogic.userFacingError(it, genericError)
        }
        loadingMembers = false
    }

    suspend fun loadEligible(token: String) {
        loadingEligible = true
        runCatching {
            eligible = LmsApi.fetchEligibleRoleUsers(role.id, assignSearch, token)
        }.onFailure {
            errorMessage = RolesPermissionsAdminLogic.userFacingError(it, genericError)
        }
        loadingEligible = false
    }

    LaunchedEffect(accessToken, role.id) {
        val token = accessToken ?: return@LaunchedEffect
        loadMembers(token)
    }

    LaunchedEffect(accessToken, assignSearch, showAssignSheet) {
        if (!showAssignSheet) return@LaunchedEffect
        val token = accessToken ?: return@LaunchedEffect
        loadEligible(token)
    }

    pendingAssign?.let { user ->
        AlertDialog(
            onDismissRequest = { pendingAssign = null },
            title = { Text(L.text(context, localePrefs, R.string.mobile_admin_roles_assignUser)) },
            text = {
                Text(
                    L.format(
                        context,
                        localePrefs,
                        R.string.mobile_admin_roles_assignConfirm,
                        RolesPermissionsAdminLogic.userDisplayLabel(user),
                        role.name,
                    ),
                )
            },
            confirmButton = {
                TextButton(
                    onClick = {
                        val token = accessToken ?: return@TextButton
                        pendingAssign = null
                        scope.launch {
                            busyUserId = user.id
                            errorMessage = null
                            statusMessage = null
                            runCatching {
                                LmsApi.addUserToRole(role.id, user.id, token)
                                members = members + user
                                statusMessage = L.text(
                                    context,
                                    localePrefs,
                                    R.string.mobile_admin_roles_assignSuccess,
                                )
                            }.onFailure {
                                errorMessage = RolesPermissionsAdminLogic.userFacingError(it, genericError)
                            }
                            busyUserId = null
                        }
                    },
                ) {
                    Text(L.text(context, localePrefs, R.string.mobile_admin_roles_assignUser))
                }
            },
            dismissButton = {
                TextButton(onClick = { pendingAssign = null }) {
                    Text(L.text(context, localePrefs, R.string.mobile_common_cancel))
                }
            },
        )
    }

    pendingRemove?.let { user ->
        AlertDialog(
            onDismissRequest = { pendingRemove = null },
            title = { Text(L.text(context, localePrefs, R.string.mobile_admin_roles_removeUser)) },
            text = {
                Text(
                    L.format(
                        context,
                        localePrefs,
                        R.string.mobile_admin_roles_removeConfirm,
                        RolesPermissionsAdminLogic.userDisplayLabel(user),
                        role.name,
                    ),
                )
            },
            confirmButton = {
                TextButton(
                    onClick = {
                        val token = accessToken ?: return@TextButton
                        pendingRemove = null
                        scope.launch {
                            busyUserId = user.id
                            errorMessage = null
                            statusMessage = null
                            runCatching {
                                LmsApi.removeUserFromRole(role.id, user.id, token)
                                members = members.filter { it.id != user.id }
                                statusMessage = L.text(
                                    context,
                                    localePrefs,
                                    R.string.mobile_admin_roles_removeSuccess,
                                )
                            }.onFailure {
                                errorMessage = RolesPermissionsAdminLogic.userFacingError(it, genericError)
                            }
                            busyUserId = null
                        }
                    },
                ) {
                    Text(L.text(context, localePrefs, R.string.mobile_admin_roles_removeUser))
                }
            },
            dismissButton = {
                TextButton(onClick = { pendingRemove = null }) {
                    Text(L.text(context, localePrefs, R.string.mobile_common_cancel))
                }
            },
        )
    }

    if (showAssignSheet) {
        ModalBottomSheet(
            onDismissRequest = { showAssignSheet = false },
            sheetState = sheetState,
        ) {
            Column(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(horizontal = 16.dp, vertical = 8.dp),
                verticalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                Text(
                    L.text(context, localePrefs, R.string.mobile_admin_roles_assignUser),
                    fontWeight = FontWeight.SemiBold,
                )
                OutlinedTextField(
                    value = assignSearch,
                    onValueChange = { assignSearch = it },
                    modifier = Modifier.fillMaxWidth(),
                    label = { Text(L.text(context, localePrefs, R.string.mobile_admin_roles_searchUsers)) },
                    singleLine = true,
                )
                when {
                    loadingEligible -> LmsSkeletonList(count = 2)
                    filteredEligible.isEmpty() -> Text(
                        L.text(context, localePrefs, R.string.mobile_admin_roles_noEligibleUsers),
                        color = textSecondary(),
                    )
                    else -> filteredEligible.forEach { user ->
                        LmsCard {
                            Column(
                                modifier = Modifier
                                    .fillMaxWidth()
                                    .clickable {
                                        if (RolesPermissionsAdminLogic.blocksSelfElevation(
                                                role,
                                                user.id,
                                                shell.profile?.id,
                                            )
                                        ) {
                                            errorMessage = L.text(
                                                context,
                                                localePrefs,
                                                R.string.mobile_admin_roles_selfElevationBlocked,
                                            )
                                            showAssignSheet = false
                                        } else {
                                            pendingAssign = user
                                            showAssignSheet = false
                                        }
                                    }
                                    .padding(12.dp),
                            ) {
                                Text(
                                    RolesPermissionsAdminLogic.userDisplayLabel(user),
                                    fontWeight = FontWeight.Medium,
                                )
                                Text(user.email, color = textSecondary())
                            }
                        }
                    }
                }
                TextButton(onClick = { showAssignSheet = false }) {
                    Text(L.text(context, localePrefs, R.string.mobile_common_cancel))
                }
            }
        }
    }

    Scaffold(
        modifier = modifier.fillMaxSize(),
        topBar = {
            TopAppBar(
                title = { Text(role.name) },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = null)
                    }
                },
                actions = {
                    TextButton(onClick = { showAssignSheet = true }) {
                        Text(L.text(context, localePrefs, R.string.mobile_admin_roles_assignUser))
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
            Text(
                L.text(context, localePrefs, R.string.mobile_admin_roles_readOnlyHint),
                color = textSecondary(),
            )
            role.scope?.trim()?.takeIf { it.isNotEmpty() }?.let { scopeValue ->
                Text(
                    "${L.text(context, localePrefs, R.string.mobile_admin_roles_scope)}: $scopeValue",
                    color = textSecondary(),
                )
            }

            errorMessage?.let { LmsErrorBanner(message = it) }
            statusMessage?.let {
                Text(it, color = textSecondary())
            }

            OutlinedTextField(
                value = permissionSearch,
                onValueChange = { permissionSearch = it },
                modifier = Modifier.fillMaxWidth(),
                label = { Text(L.text(context, localePrefs, R.string.mobile_admin_roles_searchPermissions)) },
                singleLine = true,
            )

            Text(
                L.text(context, localePrefs, R.string.mobile_admin_roles_permissionsTitle),
                fontWeight = FontWeight.SemiBold,
                color = textPrimary(),
            )

            when {
                role.permissions.isEmpty() -> Text(
                    L.text(context, localePrefs, R.string.mobile_admin_roles_emptyPermissions),
                    color = textSecondary(),
                )
                filteredPermissions.isEmpty() -> Text(
                    L.text(context, localePrefs, R.string.mobile_admin_roles_emptyPermissionsSearch),
                    color = textSecondary(),
                )
                else -> filteredPermissions.forEach { permission ->
                    LmsCard {
                        Column(modifier = Modifier.padding(12.dp)) {
                            Text(
                                permission.permissionString,
                                fontFamily = FontFamily.Monospace,
                                color = textPrimary(),
                            )
                            if (permission.description.isNotEmpty()) {
                                Text(permission.description, color = textSecondary())
                            }
                        }
                    }
                }
            }

            Text(
                L.text(context, localePrefs, R.string.mobile_admin_roles_members),
                fontWeight = FontWeight.SemiBold,
                color = textPrimary(),
            )

            when {
                loadingMembers -> LmsSkeletonList(count = 2)
                members.isEmpty() -> Text(
                    L.text(context, localePrefs, R.string.mobile_admin_roles_noMembers),
                    color = textSecondary(),
                )
                else -> members.forEach { member ->
                    LmsCard {
                        Row(
                            modifier = Modifier
                                .fillMaxWidth()
                                .padding(12.dp),
                            horizontalArrangement = Arrangement.SpaceBetween,
                        ) {
                            Column(modifier = Modifier.weight(1f)) {
                                Text(
                                    RolesPermissionsAdminLogic.userDisplayLabel(member),
                                    fontWeight = FontWeight.Medium,
                                )
                                Text(member.email, color = textSecondary())
                            }
                            Button(
                                onClick = { pendingRemove = member },
                                enabled = busyUserId != member.id,
                            ) {
                                Text(L.text(context, localePrefs, R.string.mobile_admin_roles_removeUser))
                            }
                        }
                    }
                }
            }
        }
    }
}
