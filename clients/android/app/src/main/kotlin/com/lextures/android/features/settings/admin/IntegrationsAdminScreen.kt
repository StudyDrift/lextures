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
import androidx.compose.material.icons.automirrored.filled.KeyboardArrowRight
import androidx.compose.material.icons.filled.Cloud
import androidx.compose.material.icons.filled.Extension
import androidx.compose.material.icons.filled.Link
import androidx.compose.material.icons.filled.Lock
import androidx.compose.material.icons.filled.MenuBook
import androidx.compose.material.icons.filled.People
import androidx.compose.material.icons.filled.AccountTree
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.ExposedDropdownMenuBox
import androidx.compose.material3.ExposedDropdownMenuDefaults
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Switch
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
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.config.AppConfiguration
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalePreferences
import com.lextures.android.core.lms.AdminOrgRow
import com.lextures.android.core.lms.CloudProviderStatus
import com.lextures.android.core.lms.IntegrationsAdminLogic
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.LrsEndpointStatus
import com.lextures.android.core.lms.LtiExternalTool
import com.lextures.android.core.lms.LtiParentPlatform
import com.lextures.android.core.lms.OerProviderStatus
import com.lextures.android.core.lms.ScimEventRow
import com.lextures.android.core.lms.ScimTokenRow
import com.lextures.android.features.home.HomeShellState
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsEmptyState
import com.lextures.android.features.home.LmsErrorBanner
import com.lextures.android.features.home.LmsSkeletonList
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun IntegrationsAdminScreen(
    session: AuthSession,
    shell: HomeShellState,
    localePrefs: LocalePreferences,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    val canView = IntegrationsAdminLogic.canView(shell.platformFeatures, shell.permissions)
    val accessToken by session.accessToken.collectAsState()
    var scimEnabled by remember { mutableStateOf(false) }
    var scimFlagLoaded by remember { mutableStateOf(false) }
    var openSection by remember { mutableStateOf<IntegrationsAdminLogic.Section?>(null) }

    LaunchedEffect(accessToken, canView) {
        if (!canView) {
            scimFlagLoaded = true
            return@LaunchedEffect
        }
        val token = accessToken
        if (token == null) {
            scimFlagLoaded = true
            return@LaunchedEffect
        }
        scimEnabled = runCatching { LmsApi.fetchPlatformScimEnabled(token) }.getOrDefault(false)
        scimFlagLoaded = true
    }

    when (openSection) {
        IntegrationsAdminLogic.Section.LTI -> {
            LtiIntegrationsAdminScreen(
                session = session,
                localePrefs = localePrefs,
                onBack = { openSection = null },
            )
            return
        }
        IntegrationsAdminLogic.Section.SCIM -> {
            ScimIntegrationsAdminScreen(
                session = session,
                localePrefs = localePrefs,
                onBack = { openSection = null },
            )
            return
        }
        IntegrationsAdminLogic.Section.CLOUD -> {
            CloudProvidersAdminScreen(
                session = session,
                localePrefs = localePrefs,
                onBack = { openSection = null },
            )
            return
        }
        IntegrationsAdminLogic.Section.LRS -> {
            LrsIntegrationsAdminScreen(
                session = session,
                localePrefs = localePrefs,
                onBack = { openSection = null },
            )
            return
        }
        IntegrationsAdminLogic.Section.OER -> {
            OerProvidersAdminScreen(
                session = session,
                localePrefs = localePrefs,
                onBack = { openSection = null },
            )
            return
        }
        null -> Unit
    }

    if (!canView) {
        Scaffold(
            modifier = modifier.fillMaxSize(),
            topBar = {
                TopAppBar(
                    title = { Text(L.text(context, localePrefs, R.string.mobile_admin_integrations_hub_title)) },
                    navigationIcon = {
                        IconButton(onClick = onBack) {
                            Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = null)
                        }
                    },
                )
            },
        ) { padding ->
            LmsEmptyState(
                modifier = Modifier.padding(padding).padding(16.dp),
                icon = Icons.Default.Lock,
                title = L.text(context, localePrefs, R.string.mobile_admin_integrations_accessDenied_title),
                message = L.text(context, localePrefs, R.string.mobile_admin_integrations_accessDenied_message),
            )
        }
        return
    }

    val sections = IntegrationsAdminLogic.visibleSections(shell.platformFeatures, scimEnabled)

    Scaffold(
        modifier = modifier.fillMaxSize(),
        topBar = {
            TopAppBar(
                title = { Text(L.text(context, localePrefs, R.string.mobile_admin_integrations_hub_title)) },
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
                .verticalScroll(rememberScrollState())
                .padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(16.dp),
        ) {
            Text(
                text = L.text(context, localePrefs, R.string.mobile_admin_integrations_hub_description),
                fontSize = 14.sp,
                color = textSecondary(),
            )
            if (!scimFlagLoaded) {
                LmsSkeletonList(count = 3)
            } else if (sections.isEmpty()) {
                LmsEmptyState(
                    icon = Icons.Default.Link,
                    title = L.text(context, localePrefs, R.string.mobile_admin_integrations_emptyTitle),
                    message = L.text(context, localePrefs, R.string.mobile_admin_integrations_emptyMessage),
                )
            } else {
                LmsCard {
                    Column {
                        sections.forEach { section ->
                            HubNavRow(
                                icon = sectionIcon(section),
                                title = sectionTitle(context, localePrefs, section),
                                subtitle = sectionSubtitle(context, localePrefs, section),
                                onClick = { openSection = section },
                            )
                        }
                    }
                }
            }
        }
    }
}

private fun sectionIcon(section: IntegrationsAdminLogic.Section): ImageVector = when (section) {
    IntegrationsAdminLogic.Section.LTI -> Icons.Default.Extension
    IntegrationsAdminLogic.Section.SCIM -> Icons.Default.People
    IntegrationsAdminLogic.Section.CLOUD -> Icons.Default.Cloud
    IntegrationsAdminLogic.Section.LRS -> Icons.Default.AccountTree
    IntegrationsAdminLogic.Section.OER -> Icons.Default.MenuBook
}

@Composable
private fun sectionTitle(
    context: android.content.Context,
    localePrefs: LocalePreferences,
    section: IntegrationsAdminLogic.Section,
): String = L.text(
    context,
    localePrefs,
    when (section) {
        IntegrationsAdminLogic.Section.LTI -> R.string.mobile_admin_integrations_lti_title
        IntegrationsAdminLogic.Section.SCIM -> R.string.mobile_admin_integrations_scim_title
        IntegrationsAdminLogic.Section.CLOUD -> R.string.mobile_admin_integrations_cloud_title
        IntegrationsAdminLogic.Section.LRS -> R.string.mobile_admin_integrations_lrs_title
        IntegrationsAdminLogic.Section.OER -> R.string.mobile_admin_integrations_oer_title
    },
)

@Composable
private fun sectionSubtitle(
    context: android.content.Context,
    localePrefs: LocalePreferences,
    section: IntegrationsAdminLogic.Section,
): String = L.text(
    context,
    localePrefs,
    when (section) {
        IntegrationsAdminLogic.Section.LTI -> R.string.mobile_admin_integrations_lti_entry_subtitle
        IntegrationsAdminLogic.Section.SCIM -> R.string.mobile_admin_integrations_scim_entry_subtitle
        IntegrationsAdminLogic.Section.CLOUD -> R.string.mobile_admin_integrations_cloud_entry_subtitle
        IntegrationsAdminLogic.Section.LRS -> R.string.mobile_admin_integrations_lrs_entry_subtitle
        IntegrationsAdminLogic.Section.OER -> R.string.mobile_admin_integrations_oer_entry_subtitle
    },
)

@Composable
private fun HubNavRow(
    icon: ImageVector,
    title: String,
    subtitle: String,
    onClick: () -> Unit,
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clickable(onClick = onClick)
            .padding(vertical = 12.dp),
        horizontalArrangement = Arrangement.spacedBy(12.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Icon(icon, contentDescription = null, tint = textPrimary())
        Column(modifier = Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(2.dp)) {
            Text(text = title, fontSize = 14.sp, fontWeight = FontWeight.SemiBold, color = textPrimary())
            Text(text = subtitle, fontSize = 12.sp, color = textSecondary())
        }
        Icon(
            Icons.AutoMirrored.Filled.KeyboardArrowRight,
            contentDescription = null,
            tint = textSecondary().copy(alpha = 0.6f),
        )
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun DetailScaffold(
    title: String,
    description: String,
    webPath: String,
    onBack: () -> Unit,
    localePrefs: LocalePreferences,
    content: @Composable () -> Unit,
) {
    val context = LocalContext.current
    Scaffold(
        modifier = Modifier.fillMaxSize(),
        topBar = {
            TopAppBar(
                title = { Text(title) },
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
                .verticalScroll(rememberScrollState())
                .padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(16.dp),
        ) {
            Text(text = description, fontSize = 14.sp, color = textSecondary())
            content()
            Button(
                onClick = {
                    context.startActivity(
                        Intent(Intent.ACTION_VIEW, Uri.parse(AppConfiguration.webUrl(webPath))),
                    )
                },
                modifier = Modifier.fillMaxWidth(),
            ) {
                Text(L.text(context, localePrefs, R.string.mobile_admin_integrations_configureOnWeb))
            }
        }
    }
}

@Composable
private fun StatusToggleRow(
    title: String,
    subtitle: String,
    enabled: Boolean,
    busy: Boolean,
    onToggle: () -> Unit,
) {
    LmsCard {
        Row(
            modifier = Modifier.fillMaxWidth(),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            Column(modifier = Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(4.dp)) {
                Text(title, fontWeight = FontWeight.SemiBold, color = textPrimary())
                if (subtitle.isNotEmpty()) {
                    Text(subtitle, fontSize = 12.sp, color = textSecondary())
                }
            }
            Switch(checked = enabled, onCheckedChange = { onToggle() }, enabled = !busy)
        }
    }
}

@Composable
private fun ConfirmToggleDialog(
    enable: Boolean,
    localePrefs: LocalePreferences,
    onConfirm: () -> Unit,
    onDismiss: () -> Unit,
) {
    val context = LocalContext.current
    AlertDialog(
        onDismissRequest = onDismiss,
        title = {
            Text(
                L.text(
                    context,
                    localePrefs,
                    if (enable) {
                        R.string.mobile_admin_integrations_confirm_enable
                    } else {
                        R.string.mobile_admin_integrations_confirm_disable
                    },
                ),
            )
        },
        text = {
            Text(L.text(context, localePrefs, R.string.mobile_admin_integrations_confirm_message))
        },
        confirmButton = {
            TextButton(onClick = onConfirm) {
                Text(L.text(context, localePrefs, R.string.mobile_admin_integrations_confirm))
            }
        },
        dismissButton = {
            TextButton(onClick = onDismiss) {
                Text(L.text(context, localePrefs, R.string.mobile_cancel))
            }
        },
    )
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun LtiIntegrationsAdminScreen(
    session: AuthSession,
    localePrefs: LocalePreferences,
    onBack: () -> Unit,
) {
    val context = LocalContext.current
    val accessToken by session.accessToken.collectAsState()
    val scope = rememberCoroutineScope()
    var platforms by remember { mutableStateOf<List<LtiParentPlatform>>(emptyList()) }
    var tools by remember { mutableStateOf<List<LtiExternalTool>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var statusMessage by remember { mutableStateOf<String?>(null) }
    var busyId by remember { mutableStateOf<String?>(null) }
    var pendingPlatform by remember { mutableStateOf<LtiParentPlatform?>(null) }
    var pendingTool by remember { mutableStateOf<LtiExternalTool?>(null) }

    fun load(token: String) {
        scope.launch {
            loading = true
            errorMessage = null
            runCatching { LmsApi.fetchLtiRegistrations(token) }
                .onSuccess {
                    platforms = it.parentPlatforms
                    tools = it.externalTools
                }
                .onFailure {
                    errorMessage = IntegrationsAdminLogic.userFacingError(
                        it,
                        L.text(context, localePrefs, R.string.mobile_admin_integrations_error),
                    )
                }
            loading = false
        }
    }

    LaunchedEffect(accessToken) {
        val token = accessToken ?: return@LaunchedEffect
        load(token)
    }

    DetailScaffold(
        title = L.text(context, localePrefs, R.string.mobile_admin_integrations_lti_title),
        description = L.text(context, localePrefs, R.string.mobile_admin_integrations_lti_description),
        webPath = IntegrationsAdminLogic.Section.LTI.webPath,
        onBack = onBack,
        localePrefs = localePrefs,
    ) {
        errorMessage?.let { LmsErrorBanner(it) }
        statusMessage?.let { Text(it, color = textSecondary(), fontSize = 12.sp) }
        when {
            loading && platforms.isEmpty() && tools.isEmpty() -> LmsSkeletonList(count = 3)
            platforms.isEmpty() && tools.isEmpty() -> LmsEmptyState(
                icon = Icons.Default.Extension,
                title = L.text(context, localePrefs, R.string.mobile_admin_integrations_lti_emptyTitle),
                message = L.text(context, localePrefs, R.string.mobile_admin_integrations_lti_emptyMessage),
            )
            else -> {
                if (platforms.isNotEmpty()) {
                    Text(
                        L.text(context, localePrefs, R.string.mobile_admin_integrations_lti_platforms),
                        fontWeight = FontWeight.Bold,
                    )
                    platforms.forEach { row ->
                        StatusToggleRow(
                            title = row.name.ifEmpty { row.platformIss },
                            subtitle = row.clientId,
                            enabled = row.active,
                            busy = busyId == row.id,
                            onToggle = { pendingPlatform = row },
                        )
                    }
                }
                if (tools.isNotEmpty()) {
                    Text(
                        L.text(context, localePrefs, R.string.mobile_admin_integrations_lti_tools),
                        fontWeight = FontWeight.Bold,
                    )
                    tools.forEach { row ->
                        StatusToggleRow(
                            title = row.name.ifEmpty { row.toolIssuer },
                            subtitle = row.clientId,
                            enabled = row.active,
                            busy = busyId == row.id,
                            onToggle = { pendingTool = row },
                        )
                    }
                }
            }
        }
    }

    pendingPlatform?.let { row ->
        ConfirmToggleDialog(
            enable = !row.active,
            localePrefs = localePrefs,
            onConfirm = {
                pendingPlatform = null
                val token = accessToken ?: return@ConfirmToggleDialog
                val desired = !row.active
                scope.launch {
                    busyId = row.id
                    errorMessage = null
                    statusMessage = null
                    runCatching { LmsApi.setLtiParentPlatformActive(row.id, desired, token) }
                        .onSuccess {
                            platforms = IntegrationsAdminLogic.applyingLtiPlatformActive(platforms, row.id, desired)
                            statusMessage = L.text(context, localePrefs, R.string.mobile_admin_integrations_saved)
                        }
                        .onFailure {
                            errorMessage = IntegrationsAdminLogic.userFacingError(
                                it,
                                L.text(context, localePrefs, R.string.mobile_admin_integrations_toggleError),
                            )
                            load(token)
                        }
                    busyId = null
                }
            },
            onDismiss = { pendingPlatform = null },
        )
    }
    pendingTool?.let { row ->
        ConfirmToggleDialog(
            enable = !row.active,
            localePrefs = localePrefs,
            onConfirm = {
                pendingTool = null
                val token = accessToken ?: return@ConfirmToggleDialog
                val desired = !row.active
                scope.launch {
                    busyId = row.id
                    errorMessage = null
                    statusMessage = null
                    runCatching { LmsApi.setLtiExternalToolActive(row.id, desired, token) }
                        .onSuccess {
                            tools = IntegrationsAdminLogic.applyingLtiToolActive(tools, row.id, desired)
                            statusMessage = L.text(context, localePrefs, R.string.mobile_admin_integrations_saved)
                        }
                        .onFailure {
                            errorMessage = IntegrationsAdminLogic.userFacingError(
                                it,
                                L.text(context, localePrefs, R.string.mobile_admin_integrations_toggleError),
                            )
                            load(token)
                        }
                    busyId = null
                }
            },
            onDismiss = { pendingTool = null },
        )
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun ScimIntegrationsAdminScreen(
    session: AuthSession,
    localePrefs: LocalePreferences,
    onBack: () -> Unit,
) {
    val context = LocalContext.current
    val accessToken by session.accessToken.collectAsState()
    val scope = rememberCoroutineScope()
    var institutions by remember { mutableStateOf<List<AdminOrgRow>>(emptyList()) }
    var selectedInstitutionId by remember { mutableStateOf<String?>(null) }
    var tokens by remember { mutableStateOf<List<ScimTokenRow>>(emptyList()) }
    var events by remember { mutableStateOf<List<ScimEventRow>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var institutionMenuExpanded by remember { mutableStateOf(false) }

    fun loadStatus(token: String, institutionId: String) {
        scope.launch {
            loading = true
            errorMessage = null
            runCatching {
                val t = LmsApi.fetchScimTokens(institutionId, token)
                val e = LmsApi.fetchScimEvents(institutionId, token)
                t to e
            }.onSuccess { (t, e) ->
                tokens = t
                events = e
            }.onFailure {
                errorMessage = IntegrationsAdminLogic.userFacingError(
                    it,
                    L.text(context, localePrefs, R.string.mobile_admin_integrations_error),
                )
            }
            loading = false
        }
    }

    LaunchedEffect(accessToken) {
        val token = accessToken ?: return@LaunchedEffect
        loading = true
        institutions = runCatching { LmsApi.fetchAdminOrganizations(token) }.getOrDefault(emptyList())
        if (selectedInstitutionId == null) {
            selectedInstitutionId = institutions.firstOrNull()?.id
        }
        val institutionId = selectedInstitutionId
        if (institutionId != null) {
            loadStatus(token, institutionId)
        } else {
            loading = false
        }
    }

    DetailScaffold(
        title = L.text(context, localePrefs, R.string.mobile_admin_integrations_scim_title),
        description = L.text(context, localePrefs, R.string.mobile_admin_integrations_scim_description),
        webPath = IntegrationsAdminLogic.Section.SCIM.webPath,
        onBack = onBack,
        localePrefs = localePrefs,
    ) {
        errorMessage?.let { LmsErrorBanner(it) }
        if (institutions.size > 1) {
            ExposedDropdownMenuBox(
                expanded = institutionMenuExpanded,
                onExpandedChange = { institutionMenuExpanded = it },
            ) {
                val selected = institutions.firstOrNull { it.id == selectedInstitutionId }
                OutlinedTextField(
                    value = selected?.name?.ifEmpty { selected.id } ?: "",
                    onValueChange = {},
                    readOnly = true,
                    label = { Text(L.text(context, localePrefs, R.string.mobile_admin_integrations_scim_institution)) },
                    trailingIcon = { ExposedDropdownMenuDefaults.TrailingIcon(expanded = institutionMenuExpanded) },
                    modifier = Modifier.menuAnchor().fillMaxWidth(),
                )
                ExposedDropdownMenu(
                    expanded = institutionMenuExpanded,
                    onDismissRequest = { institutionMenuExpanded = false },
                ) {
                    institutions.forEach { org ->
                        DropdownMenuItem(
                            text = { Text(org.name.ifEmpty { org.id }) },
                            onClick = {
                                selectedInstitutionId = org.id
                                institutionMenuExpanded = false
                                accessToken?.let { loadStatus(it, org.id) }
                            },
                        )
                    }
                }
            }
        } else {
            institutions.firstOrNull()?.let { org ->
                Text(
                    "${L.text(context, localePrefs, R.string.mobile_admin_integrations_scim_institution)}: ${org.name.ifEmpty { org.id }}",
                    color = textSecondary(),
                    fontSize = 13.sp,
                )
            }
        }

        if (loading) {
            LmsSkeletonList(count = 2)
        } else {
            LmsCard {
                Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    Text(
                        L.text(context, localePrefs, R.string.mobile_admin_integrations_scim_summary),
                        fontWeight = FontWeight.SemiBold,
                    )
                    Text(
                        "${L.text(context, localePrefs, R.string.mobile_admin_integrations_scim_activeTokens)}: ${IntegrationsAdminLogic.activeTokenCount(tokens)}",
                        color = textSecondary(),
                    )
                    Text(
                        "${L.text(context, localePrefs, R.string.mobile_admin_integrations_scim_lastEvent)}: ${IntegrationsAdminLogic.lastEventAt(events) ?: "—"}",
                        color = textSecondary(),
                    )
                }
            }
            if (events.isNotEmpty()) {
                Text(
                    L.text(context, localePrefs, R.string.mobile_admin_integrations_scim_recentEvents),
                    fontWeight = FontWeight.Bold,
                )
                events.take(20).forEach { event ->
                    LmsCard {
                        Column(verticalArrangement = Arrangement.spacedBy(4.dp)) {
                            Text("${event.operation} · ${event.scimResource}", fontWeight = FontWeight.SemiBold)
                            event.userEmail?.takeIf { it.isNotEmpty() }?.let {
                                Text(it, fontSize = 12.sp, color = textSecondary())
                            }
                            Text(event.createdAt, fontSize = 11.sp, color = textSecondary())
                        }
                    }
                }
            } else if (!loading) {
                LmsEmptyState(
                    icon = Icons.Default.People,
                    title = L.text(context, localePrefs, R.string.mobile_admin_integrations_scim_emptyTitle),
                    message = L.text(context, localePrefs, R.string.mobile_admin_integrations_scim_emptyMessage),
                )
            }
            Text(
                L.text(context, localePrefs, R.string.mobile_admin_integrations_scim_tokensNote),
                fontSize = 12.sp,
                color = textSecondary(),
            )
        }
    }
}

@Composable
private fun CloudProvidersAdminScreen(
    session: AuthSession,
    localePrefs: LocalePreferences,
    onBack: () -> Unit,
) {
    val context = LocalContext.current
    val accessToken by session.accessToken.collectAsState()
    val scope = rememberCoroutineScope()
    var providers by remember { mutableStateOf<List<CloudProviderStatus>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var statusMessage by remember { mutableStateOf<String?>(null) }
    var busyProvider by remember { mutableStateOf<String?>(null) }
    var pending by remember { mutableStateOf<CloudProviderStatus?>(null) }

    fun load(token: String) {
        scope.launch {
            loading = true
            errorMessage = null
            runCatching { LmsApi.fetchAdminCloudProviders(token) }
                .onSuccess { providers = it }
                .onFailure {
                    errorMessage = IntegrationsAdminLogic.userFacingError(
                        it,
                        L.text(context, localePrefs, R.string.mobile_admin_integrations_error),
                    )
                }
            loading = false
        }
    }

    LaunchedEffect(accessToken) {
        val token = accessToken ?: return@LaunchedEffect
        load(token)
    }

    DetailScaffold(
        title = L.text(context, localePrefs, R.string.mobile_admin_integrations_cloud_title),
        description = L.text(context, localePrefs, R.string.mobile_admin_integrations_cloud_description),
        webPath = IntegrationsAdminLogic.Section.CLOUD.webPath,
        onBack = onBack,
        localePrefs = localePrefs,
    ) {
        Text(
            L.text(context, localePrefs, R.string.mobile_admin_integrations_secretsOmitted),
            fontSize = 12.sp,
            color = textSecondary(),
        )
        errorMessage?.let { LmsErrorBanner(it) }
        statusMessage?.let { Text(it, color = textSecondary(), fontSize = 12.sp) }
        when {
            loading && providers.isEmpty() -> LmsSkeletonList(count = 3)
            providers.isEmpty() -> LmsEmptyState(
                icon = Icons.Default.Cloud,
                title = L.text(context, localePrefs, R.string.mobile_admin_integrations_cloud_emptyTitle),
                message = L.text(context, localePrefs, R.string.mobile_admin_integrations_cloud_emptyMessage),
            )
            else -> providers.forEach { row ->
                val label = providerLabel(context, localePrefs, IntegrationsAdminLogic.cloudProviderLabelKey(row.provider), row.provider)
                StatusToggleRow(
                    title = label,
                    subtitle = row.updatedAt.orEmpty(),
                    enabled = row.enabled,
                    busy = busyProvider == row.provider,
                    onToggle = { pending = row },
                )
            }
        }
    }

    pending?.let { row ->
        ConfirmToggleDialog(
            enable = !row.enabled,
            localePrefs = localePrefs,
            onConfirm = {
                pending = null
                val token = accessToken ?: return@ConfirmToggleDialog
                val desired = !row.enabled
                scope.launch {
                    busyProvider = row.provider
                    errorMessage = null
                    statusMessage = null
                    runCatching { LmsApi.setCloudProviderEnabled(row.provider, desired, token) }
                        .onSuccess {
                            providers = IntegrationsAdminLogic.applyingCloudEnabled(providers, row.provider, desired)
                            statusMessage = L.text(context, localePrefs, R.string.mobile_admin_integrations_saved)
                        }
                        .onFailure {
                            errorMessage = IntegrationsAdminLogic.userFacingError(
                                it,
                                L.text(context, localePrefs, R.string.mobile_admin_integrations_toggleError),
                            )
                            load(token)
                        }
                    busyProvider = null
                }
            },
            onDismiss = { pending = null },
        )
    }
}

@Composable
private fun LrsIntegrationsAdminScreen(
    session: AuthSession,
    localePrefs: LocalePreferences,
    onBack: () -> Unit,
) {
    val context = LocalContext.current
    val accessToken by session.accessToken.collectAsState()
    val scope = rememberCoroutineScope()
    var endpoints by remember { mutableStateOf<List<LrsEndpointStatus>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var statusMessage by remember { mutableStateOf<String?>(null) }
    var busyId by remember { mutableStateOf<String?>(null) }
    var pending by remember { mutableStateOf<LrsEndpointStatus?>(null) }

    fun load(token: String) {
        scope.launch {
            loading = true
            errorMessage = null
            runCatching { LmsApi.fetchAdminLrsEndpoints(token) }
                .onSuccess { endpoints = it }
                .onFailure {
                    errorMessage = IntegrationsAdminLogic.userFacingError(
                        it,
                        L.text(context, localePrefs, R.string.mobile_admin_integrations_error),
                    )
                }
            loading = false
        }
    }

    LaunchedEffect(accessToken) {
        val token = accessToken ?: return@LaunchedEffect
        load(token)
    }

    DetailScaffold(
        title = L.text(context, localePrefs, R.string.mobile_admin_integrations_lrs_title),
        description = L.text(context, localePrefs, R.string.mobile_admin_integrations_lrs_description),
        webPath = IntegrationsAdminLogic.Section.LRS.webPath,
        onBack = onBack,
        localePrefs = localePrefs,
    ) {
        Text(
            L.text(context, localePrefs, R.string.mobile_admin_integrations_secretsOmitted),
            fontSize = 12.sp,
            color = textSecondary(),
        )
        errorMessage?.let { LmsErrorBanner(it) }
        statusMessage?.let { Text(it, color = textSecondary(), fontSize = 12.sp) }
        when {
            loading && endpoints.isEmpty() -> LmsSkeletonList(count = 3)
            endpoints.isEmpty() -> LmsEmptyState(
                icon = Icons.Default.AccountTree,
                title = L.text(context, localePrefs, R.string.mobile_admin_integrations_lrs_emptyTitle),
                message = L.text(context, localePrefs, R.string.mobile_admin_integrations_lrs_emptyMessage),
            )
            else -> endpoints.forEach { row ->
                StatusToggleRow(
                    title = row.label.ifEmpty { row.endpointUrl },
                    subtitle = row.endpointUrl,
                    enabled = row.enabled,
                    busy = busyId == row.id,
                    onToggle = { pending = row },
                )
            }
        }
    }

    pending?.let { row ->
        ConfirmToggleDialog(
            enable = !row.enabled,
            localePrefs = localePrefs,
            onConfirm = {
                pending = null
                val token = accessToken ?: return@ConfirmToggleDialog
                val desired = !row.enabled
                scope.launch {
                    busyId = row.id
                    errorMessage = null
                    statusMessage = null
                    runCatching { LmsApi.setLrsEndpointEnabled(row.id, desired, token) }
                        .onSuccess {
                            endpoints = IntegrationsAdminLogic.applyingLrsEnabled(endpoints, row.id, desired)
                            statusMessage = L.text(context, localePrefs, R.string.mobile_admin_integrations_saved)
                        }
                        .onFailure {
                            errorMessage = IntegrationsAdminLogic.userFacingError(
                                it,
                                L.text(context, localePrefs, R.string.mobile_admin_integrations_toggleError),
                            )
                            load(token)
                        }
                    busyId = null
                }
            },
            onDismiss = { pending = null },
        )
    }
}

@Composable
private fun OerProvidersAdminScreen(
    session: AuthSession,
    localePrefs: LocalePreferences,
    onBack: () -> Unit,
) {
    val context = LocalContext.current
    val accessToken by session.accessToken.collectAsState()
    val scope = rememberCoroutineScope()
    var providers by remember { mutableStateOf<List<OerProviderStatus>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var statusMessage by remember { mutableStateOf<String?>(null) }
    var busyProvider by remember { mutableStateOf<String?>(null) }
    var pending by remember { mutableStateOf<OerProviderStatus?>(null) }

    fun load(token: String) {
        scope.launch {
            loading = true
            errorMessage = null
            runCatching { LmsApi.fetchAdminOerProviders(token) }
                .onSuccess { providers = it }
                .onFailure {
                    errorMessage = IntegrationsAdminLogic.userFacingError(
                        it,
                        L.text(context, localePrefs, R.string.mobile_admin_integrations_error),
                    )
                }
            loading = false
        }
    }

    LaunchedEffect(accessToken) {
        val token = accessToken ?: return@LaunchedEffect
        load(token)
    }

    DetailScaffold(
        title = L.text(context, localePrefs, R.string.mobile_admin_integrations_oer_title),
        description = L.text(context, localePrefs, R.string.mobile_admin_integrations_oer_description),
        webPath = IntegrationsAdminLogic.Section.OER.webPath,
        onBack = onBack,
        localePrefs = localePrefs,
    ) {
        errorMessage?.let { LmsErrorBanner(it) }
        statusMessage?.let { Text(it, color = textSecondary(), fontSize = 12.sp) }
        when {
            loading && providers.isEmpty() -> LmsSkeletonList(count = 3)
            providers.isEmpty() -> LmsEmptyState(
                icon = Icons.Default.MenuBook,
                title = L.text(context, localePrefs, R.string.mobile_admin_integrations_oer_emptyTitle),
                message = L.text(context, localePrefs, R.string.mobile_admin_integrations_oer_emptyMessage),
            )
            else -> providers.forEach { row ->
                val label = providerLabel(context, localePrefs, IntegrationsAdminLogic.oerProviderLabelKey(row.provider), row.provider)
                StatusToggleRow(
                    title = label,
                    subtitle = row.updatedAt.orEmpty(),
                    enabled = row.enabled,
                    busy = busyProvider == row.provider,
                    onToggle = { pending = row },
                )
            }
        }
    }

    pending?.let { row ->
        ConfirmToggleDialog(
            enable = !row.enabled,
            localePrefs = localePrefs,
            onConfirm = {
                pending = null
                val token = accessToken ?: return@ConfirmToggleDialog
                val desired = !row.enabled
                scope.launch {
                    busyProvider = row.provider
                    errorMessage = null
                    statusMessage = null
                    runCatching { LmsApi.setOerProviderEnabled(row.provider, desired, token) }
                        .onSuccess {
                            providers = IntegrationsAdminLogic.applyingOerEnabled(providers, row.provider, desired)
                            statusMessage = L.text(context, localePrefs, R.string.mobile_admin_integrations_saved)
                        }
                        .onFailure {
                            errorMessage = IntegrationsAdminLogic.userFacingError(
                                it,
                                L.text(context, localePrefs, R.string.mobile_admin_integrations_toggleError),
                            )
                            load(token)
                        }
                    busyProvider = null
                }
            },
            onDismiss = { pending = null },
        )
    }
}

@Composable
private fun providerLabel(
    context: android.content.Context,
    localePrefs: LocalePreferences,
    resName: String,
    fallback: String,
): String {
    val id = context.resources.getIdentifier(resName, "string", context.packageName)
    return if (id != 0) L.text(context, localePrefs, id) else fallback
}
