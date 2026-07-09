package com.lextures.android.features.settings

import android.content.ClipData
import android.content.ClipboardManager
import android.content.Context
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
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
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
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalePreferences
import com.lextures.android.core.lms.AccessKeyScopeDef
import com.lextures.android.core.lms.AccessKeySummary
import com.lextures.android.core.lms.AccountIntegrationsLogic
import com.lextures.android.core.lms.CalendarTokenCreated
import com.lextures.android.core.lms.CalendarTokenInfo
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.MCPConfigResponse
import com.lextures.android.core.lms.OneTimeSecretReveal
import com.lextures.android.core.navigation.MobilePlatformFeatures
import com.lextures.android.features.home.HomeShellState
import com.lextures.android.features.home.LmsCard
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun IntegrationsScreen(
    session: AuthSession,
    shell: HomeShellState,
    localePrefs: LocalePreferences,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    val accessToken by session.accessToken.collectAsState()
    val features = shell.platformFeatures

    var accessKeys by remember { mutableStateOf<List<AccessKeySummary>>(emptyList()) }
    var scopes by remember { mutableStateOf<List<AccessKeyScopeDef>>(emptyList()) }
    var serviceTokens by remember { mutableStateOf<List<AccessKeySummary>>(emptyList()) }
    var serviceTokensForbidden by remember { mutableStateOf(true) }
    var mcpConfig by remember { mutableStateOf<MCPConfigResponse?>(null) }
    var tokenInfo by remember { mutableStateOf<CalendarTokenInfo?>(null) }
    var createdCalendarToken by remember { mutableStateOf<CalendarTokenCreated?>(null) }

    var loading by remember { mutableStateOf(true) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var statusMessage by remember { mutableStateOf<String?>(null) }

    var revealSecret by remember { mutableStateOf<OneTimeSecretReveal?>(null) }
    var showCreateKey by remember { mutableStateOf(false) }
    var showCreateServiceToken by remember { mutableStateOf(false) }
    var confirmingRevokeKeyId by remember { mutableStateOf<String?>(null) }
    var confirmingRotateKeyId by remember { mutableStateOf<String?>(null) }
    var confirmingRevokeServiceTokenId by remember { mutableStateOf<String?>(null) }
    var confirmingRegenerateCalendar by remember { mutableStateOf(false) }

    var createLabel by remember { mutableStateOf("") }
    var selectedScopes by remember { mutableStateOf(AccountIntegrationsLogic.defaultCreateScopes.toSet()) }
    var creating by remember { mutableStateOf(false) }

    var serviceAccountName by remember { mutableStateOf("") }
    var serviceTokenLabel by remember { mutableStateOf("") }
    var serviceTokenScopes by remember { mutableStateOf(setOf("enrollments:read")) }
    var mcpTokenDraft by remember { mutableStateOf("") }

    suspend fun loadAll(token: String) {
        loading = true
        errorMessage = null
        try {
            if (AccountIntegrationsLogic.accessKeysEnabled(features)) {
                accessKeys = LmsApi.fetchAccessKeys(token)
                scopes = LmsApi.fetchAccessKeyScopes(token)
                mcpConfig = LmsApi.fetchMCPConfig(token)
            }
            if (AccountIntegrationsLogic.calendarSubscriptionsEnabled(features)) {
                tokenInfo = LmsApi.fetchCalendarTokenInfo(token)
            }
            if (AccountIntegrationsLogic.canManageServiceTokens(shell.permissions)) {
                val tokens = LmsApi.fetchServiceTokens(token)
                if (tokens == null) {
                    serviceTokens = emptyList()
                    serviceTokensForbidden = true
                } else {
                    serviceTokens = tokens
                    serviceTokensForbidden = false
                }
            }
        } catch (_: Exception) {
            errorMessage = L.text(context, localePrefs, R.string.mobile_integrations_error)
        } finally {
            loading = false
        }
    }

    LaunchedEffect(accessToken, features) {
        val token = accessToken ?: return@LaunchedEffect
        loadAll(token)
    }

    fun copySecure(text: String, statusRes: Int) {
        val clipboard = context.getSystemService(Context.CLIPBOARD_SERVICE) as ClipboardManager
        clipboard.setPrimaryClip(ClipData.newPlainText("lextures-secret", text))
        statusMessage = L.text(context, localePrefs, statusRes)
        scope.launch {
            delay(60_000)
            if (clipboard.primaryClip?.getItemAt(0)?.text?.toString() == text) {
                clipboard.setPrimaryClip(ClipData.newPlainText("", ""))
            }
        }
    }

    Scaffold(
        modifier = modifier.fillMaxSize(),
        topBar = {
            TopAppBar(
                title = { Text(L.text(context, localePrefs, R.string.mobile_integrations_title)) },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = L.text(context, localePrefs, R.string.mobile_ia_close))
                    }
                },
            )
        },
    ) { padding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
                .verticalScroll(rememberScrollState())
                .padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(16.dp),
        ) {
            Text(
                text = L.text(context, localePrefs, R.string.mobile_integrations_description),
                fontSize = 14.sp,
                color = textSecondary(),
            )
            errorMessage?.let {
                Text(text = it, color = LexturesColors.Error, fontSize = 13.sp)
            }
            statusMessage?.let {
                Text(text = it, color = LexturesColors.Primary, fontSize = 12.sp)
            }

            if (AccountIntegrationsLogic.accessKeysEnabled(features)) {
                AccessKeysSection(
                    context = context,
                    localePrefs = localePrefs,
                    loading = loading,
                    accessKeys = accessKeys,
                    onCreate = {
                        createLabel = ""
                        selectedScopes = AccountIntegrationsLogic.defaultCreateScopes.toSet()
                        showCreateKey = true
                    },
                    onRevoke = { confirmingRevokeKeyId = it },
                    onRotate = { confirmingRotateKeyId = it },
                )
            }

            if (AccountIntegrationsLogic.calendarSubscriptionsEnabled(features)) {
                CalendarSection(
                    context = context,
                    localePrefs = localePrefs,
                    tokenInfo = tokenInfo,
                    createdToken = createdCalendarToken,
                    onCopy = { copySecure(it, R.string.mobile_integrations_calendar_copied) },
                    onGenerate = {
                        val token = accessToken ?: return@CalendarSection
                        if (createdCalendarToken != null || tokenInfo?.hasToken == true) {
                            confirmingRegenerateCalendar = true
                        } else {
                            scope.launch {
                                try {
                                    createdCalendarToken = LmsApi.createCalendarToken(token)
                                    tokenInfo = LmsApi.fetchCalendarTokenInfo(token)
                                } catch (_: Exception) {
                                    errorMessage = L.text(context, localePrefs, R.string.mobile_integrations_calendar_error)
                                }
                            }
                        }
                    },
                )
            }

            if (mcpConfig != null || AccountIntegrationsLogic.accessKeysEnabled(features)) {
                McpSection(
                    context = context,
                    localePrefs = localePrefs,
                    loading = loading,
                    config = mcpConfig,
                    tokenDraft = mcpTokenDraft,
                    onTokenDraftChange = { mcpTokenDraft = it },
                    onCopy = { copySecure(it, R.string.mobile_integrations_mcp_copied) },
                )
            }

            if (AccountIntegrationsLogic.shouldShowServiceTokensSection(shell.permissions, serviceTokensForbidden)) {
                ServiceTokensSection(
                    context = context,
                    localePrefs = localePrefs,
                    tokens = serviceTokens,
                    onCreate = {
                        serviceAccountName = ""
                        serviceTokenLabel = ""
                        serviceTokenScopes = setOf("enrollments:read")
                        showCreateServiceToken = true
                    },
                    onRevoke = { confirmingRevokeServiceTokenId = it },
                )
            }
        }
    }

    revealSecret?.let { secret ->
        SecretRevealDialog(
            context = context,
            localePrefs = localePrefs,
            secret = secret,
            onDismiss = { revealSecret = null },
            onCopy = {
                copySecure(secret.token, R.string.mobile_integrations_secret_copied)
            },
        )
    }

    if (showCreateKey) {
        CreateKeyDialog(
            context = context,
            localePrefs = localePrefs,
            label = createLabel,
            onLabelChange = { createLabel = it },
            scopes = scopes,
            selectedScopes = selectedScopes,
            onToggleScope = { id, on ->
                selectedScopes = if (on) selectedScopes + id else selectedScopes - id
            },
            creating = creating,
            onDismiss = { showCreateKey = false },
            onCreate = {
                val token = accessToken ?: return@CreateKeyDialog
                scope.launch {
                    creating = true
                    try {
                        val label = createLabel.trim()
                        val created = LmsApi.createAccessKey(label, selectedScopes.toList(), token)
                        showCreateKey = false
                        created.token?.let { revealSecret = OneTimeSecretReveal(it, created.label ?: label) }
                        statusMessage = L.text(context, localePrefs, R.string.mobile_integrations_accessKeys_created)
                        accessKeys = LmsApi.fetchAccessKeys(token)
                    } catch (_: Exception) {
                        errorMessage = L.text(context, localePrefs, R.string.mobile_integrations_accessKeys_error)
                    } finally {
                        creating = false
                    }
                }
            },
        )
    }

    if (showCreateServiceToken) {
        CreateServiceTokenDialog(
            context = context,
            localePrefs = localePrefs,
            accountName = serviceAccountName,
            onAccountNameChange = { serviceAccountName = it },
            label = serviceTokenLabel,
            onLabelChange = { serviceTokenLabel = it },
            scopes = scopes,
            selectedScopes = serviceTokenScopes,
            onToggleScope = { id, on ->
                serviceTokenScopes = if (on) serviceTokenScopes + id else serviceTokenScopes - id
            },
            creating = creating,
            onDismiss = { showCreateServiceToken = false },
            onCreate = {
                val token = accessToken ?: return@CreateServiceTokenDialog
                scope.launch {
                    creating = true
                    try {
                        val account = serviceAccountName.trim()
                        val label = serviceTokenLabel.trim().ifEmpty { account }
                        val created = LmsApi.createServiceToken(account, label, serviceTokenScopes.toList(), token)
                        showCreateServiceToken = false
                        created.token?.let { revealSecret = OneTimeSecretReveal(it, created.label ?: label) }
                        statusMessage = L.text(context, localePrefs, R.string.mobile_integrations_serviceTokens_created)
                        serviceTokens = LmsApi.fetchServiceTokens(token).orEmpty()
                    } catch (_: Exception) {
                        errorMessage = L.text(context, localePrefs, R.string.mobile_integrations_serviceTokens_error)
                    } finally {
                        creating = false
                    }
                }
            },
        )
    }

    confirmingRevokeKeyId?.let { id ->
        ConfirmDialog(
            context = context,
            localePrefs = localePrefs,
            title = R.string.mobile_integrations_accessKeys_revokeConfirm,
            confirmLabel = R.string.mobile_integrations_accessKeys_revoke,
            onDismiss = { confirmingRevokeKeyId = null },
            onConfirm = {
                val token = accessToken ?: return@ConfirmDialog
                scope.launch {
                    try {
                        LmsApi.revokeAccessKey(id, token)
                        statusMessage = L.text(context, localePrefs, R.string.mobile_integrations_accessKeys_revoked)
                        accessKeys = LmsApi.fetchAccessKeys(token)
                    } catch (_: Exception) {
                        errorMessage = L.text(context, localePrefs, R.string.mobile_integrations_accessKeys_error)
                    } finally {
                        confirmingRevokeKeyId = null
                    }
                }
            },
        )
    }

    confirmingRotateKeyId?.let { id ->
        ConfirmDialog(
            context = context,
            localePrefs = localePrefs,
            title = R.string.mobile_integrations_accessKeys_rotateConfirm,
            confirmLabel = R.string.mobile_integrations_accessKeys_rotate,
            onDismiss = { confirmingRotateKeyId = null },
            onConfirm = {
                val token = accessToken ?: return@ConfirmDialog
                scope.launch {
                    try {
                        val rotated = LmsApi.rotateAccessKey(id, token)
                        rotated.token?.let { revealSecret = OneTimeSecretReveal(it, rotated.label ?: "Rotated key") }
                        statusMessage = L.text(context, localePrefs, R.string.mobile_integrations_accessKeys_created)
                        accessKeys = LmsApi.fetchAccessKeys(token)
                    } catch (_: Exception) {
                        errorMessage = L.text(context, localePrefs, R.string.mobile_integrations_accessKeys_error)
                    } finally {
                        confirmingRotateKeyId = null
                    }
                }
            },
        )
    }

    confirmingRevokeServiceTokenId?.let { id ->
        ConfirmDialog(
            context = context,
            localePrefs = localePrefs,
            title = R.string.mobile_integrations_serviceTokens_revokeConfirm,
            confirmLabel = R.string.mobile_integrations_serviceTokens_revoke,
            onDismiss = { confirmingRevokeServiceTokenId = null },
            onConfirm = {
                val token = accessToken ?: return@ConfirmDialog
                scope.launch {
                    try {
                        LmsApi.revokeServiceToken(id, token)
                        statusMessage = L.text(context, localePrefs, R.string.mobile_integrations_serviceTokens_revoked)
                        serviceTokens = LmsApi.fetchServiceTokens(token).orEmpty()
                    } catch (_: Exception) {
                        errorMessage = L.text(context, localePrefs, R.string.mobile_integrations_serviceTokens_error)
                    } finally {
                        confirmingRevokeServiceTokenId = null
                    }
                }
            },
        )
    }

    if (confirmingRegenerateCalendar) {
        ConfirmDialog(
            context = context,
            localePrefs = localePrefs,
            title = R.string.mobile_integrations_calendar_regenerateConfirm,
            message = R.string.mobile_integrations_calendar_regenerateMessage,
            confirmLabel = R.string.mobile_integrations_calendar_regenerate,
            onDismiss = { confirmingRegenerateCalendar = false },
            onConfirm = {
                val token = accessToken ?: return@ConfirmDialog
                scope.launch {
                    try {
                        createdCalendarToken = LmsApi.createCalendarToken(token)
                        tokenInfo = LmsApi.fetchCalendarTokenInfo(token)
                    } catch (_: Exception) {
                        errorMessage = L.text(context, localePrefs, R.string.mobile_integrations_calendar_error)
                    } finally {
                        confirmingRegenerateCalendar = false
                    }
                }
            },
        )
    }
}

@Composable
private fun AccessKeysSection(
    context: Context,
    localePrefs: LocalePreferences,
    loading: Boolean,
    accessKeys: List<AccessKeySummary>,
    onCreate: () -> Unit,
    onRevoke: (String) -> Unit,
    onRotate: (String) -> Unit,
) {
    LmsCard {
        SectionHeader(
            context = context,
            localePrefs = localePrefs,
            titleRes = R.string.mobile_integrations_accessKeys_title,
            subtitleRes = R.string.mobile_integrations_accessKeys_description,
        )
        if (loading && accessKeys.isEmpty()) {
            CircularProgressIndicator(modifier = Modifier.padding(vertical = 8.dp))
        } else {
            val active = AccountIntegrationsLogic.activeAccessKeys(accessKeys)
            if (active.isEmpty()) {
                Text(
                    text = L.text(context, localePrefs, R.string.mobile_integrations_accessKeys_empty),
                    fontSize = 14.sp,
                    color = textSecondary(),
                )
            } else {
                active.forEach { key ->
                    Column(modifier = Modifier.padding(vertical = 4.dp)) {
                        Text(key.label, fontWeight = FontWeight.SemiBold, color = textPrimary())
                        Text(key.tokenMask, fontSize = 12.sp, fontFamily = FontFamily.Monospace, color = textSecondary())
                        Text(key.scopes.joinToString(", "), fontSize = 12.sp, color = textSecondary())
                        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                            TextButton(onClick = { onRotate(key.id) }) {
                                Text(L.text(context, localePrefs, R.string.mobile_integrations_accessKeys_rotate))
                            }
                            TextButton(onClick = { onRevoke(key.id) }) {
                                Text(L.text(context, localePrefs, R.string.mobile_integrations_accessKeys_revoke))
                            }
                        }
                    }
                }
            }
            Button(onClick = onCreate, modifier = Modifier.padding(top = 8.dp)) {
                Text(L.text(context, localePrefs, R.string.mobile_integrations_accessKeys_create))
            }
        }
    }
}

@Composable
private fun CalendarSection(
    context: Context,
    localePrefs: LocalePreferences,
    tokenInfo: CalendarTokenInfo?,
    createdToken: CalendarTokenCreated?,
    onCopy: (String) -> Unit,
    onGenerate: () -> Unit,
) {
    LmsCard {
        SectionHeader(
            context = context,
            localePrefs = localePrefs,
            titleRes = R.string.mobile_integrations_calendar_title,
            subtitleRes = R.string.mobile_integrations_calendar_description,
        )
        val personalUrl = AccountIntegrationsLogic.resolvedPersonalFeedUrl(tokenInfo, createdToken)
        if (personalUrl != null) {
            FeedUrlRow(
                context = context,
                localePrefs = localePrefs,
                title = L.text(context, localePrefs, R.string.mobile_integrations_calendar_personalFeed),
                url = personalUrl,
                onCopy = onCopy,
            )
        } else if (tokenInfo?.hasToken == true) {
            Text(
                text = L.text(context, localePrefs, R.string.mobile_integrations_calendar_activeHint),
                fontSize = 14.sp,
                color = textSecondary(),
            )
        } else {
            Text(
                text = L.text(context, localePrefs, R.string.mobile_integrations_calendar_emptyHint),
                fontSize = 14.sp,
                color = textSecondary(),
            )
        }
        createdToken?.token?.let { token ->
            tokenInfo?.courseFeeds?.forEach { feed ->
                AccountIntegrationsLogic.resolvedCourseFeedUrl(feed.feedUrl, token)?.let { url ->
                    FeedUrlRow(
                        context = context,
                        localePrefs = localePrefs,
                        title = feed.title,
                        url = url,
                        onCopy = onCopy,
                    )
                }
            }
        }
        Text(
            text = L.text(context, localePrefs, R.string.mobile_integrations_calendar_privacyWarning),
            fontSize = 12.sp,
            color = LexturesColors.Amber,
        )
        Button(onClick = onGenerate) {
            Text(
                L.text(
                    context,
                    localePrefs,
                    if (createdToken != null || tokenInfo?.hasToken == true) {
                        R.string.mobile_integrations_calendar_regenerate
                    } else {
                        R.string.mobile_integrations_calendar_generate
                    },
                ),
            )
        }
    }
}

@Composable
private fun FeedUrlRow(
    context: Context,
    localePrefs: LocalePreferences,
    title: String,
    url: String,
    onCopy: (String) -> Unit,
) {
    Column(modifier = Modifier.padding(vertical = 4.dp)) {
        Text(title, fontWeight = FontWeight.SemiBold, color = textPrimary())
        Text(url, fontSize = 11.sp, fontFamily = FontFamily.Monospace, color = textSecondary())
        TextButton(onClick = { onCopy(url) }) {
            Text(L.text(context, localePrefs, R.string.mobile_integrations_calendar_copy))
        }
    }
}

@Composable
private fun McpSection(
    context: Context,
    localePrefs: LocalePreferences,
    loading: Boolean,
    config: MCPConfigResponse?,
    tokenDraft: String,
    onTokenDraftChange: (String) -> Unit,
    onCopy: (String) -> Unit,
) {
    LmsCard {
        SectionHeader(
            context = context,
            localePrefs = localePrefs,
            titleRes = R.string.mobile_integrations_mcp_title,
            subtitleRes = R.string.mobile_integrations_mcp_description,
        )
        if (config == null && loading) {
            CircularProgressIndicator(modifier = Modifier.padding(vertical = 8.dp))
        } else if (config != null) {
            config.instructions.forEach { step ->
                Text("• $step", fontSize = 12.sp, color = textSecondary())
            }
            OutlinedTextField(
                value = tokenDraft,
                onValueChange = onTokenDraftChange,
                label = { Text(L.text(context, localePrefs, R.string.mobile_integrations_mcp_tokenPlaceholder)) },
                modifier = Modifier.fillMaxWidth(),
            )
            val json = AccountIntegrationsLogic.mcpConfigJson(config.cursorConfig, tokenDraft)
            Text(json, fontSize = 11.sp, fontFamily = FontFamily.Monospace, color = textSecondary())
            TextButton(onClick = { onCopy(json) }) {
                Text(L.text(context, localePrefs, R.string.mobile_integrations_mcp_copyConfig))
            }
            Text(
                text = L.text(context, localePrefs, R.string.mobile_integrations_mcp_apiBaseUrl),
                fontWeight = FontWeight.SemiBold,
                fontSize = 12.sp,
            )
            Text(config.apiBaseUrl, fontSize = 12.sp, fontFamily = FontFamily.Monospace, color = textSecondary())
        }
    }
}

@Composable
private fun ServiceTokensSection(
    context: Context,
    localePrefs: LocalePreferences,
    tokens: List<AccessKeySummary>,
    onCreate: () -> Unit,
    onRevoke: (String) -> Unit,
) {
    LmsCard {
        SectionHeader(
            context = context,
            localePrefs = localePrefs,
            titleRes = R.string.mobile_integrations_serviceTokens_title,
            subtitleRes = R.string.mobile_integrations_serviceTokens_description,
        )
        if (tokens.isEmpty()) {
            Text(
                text = L.text(context, localePrefs, R.string.mobile_integrations_serviceTokens_empty),
                fontSize = 14.sp,
                color = textSecondary(),
            )
        } else {
            tokens.forEach { token ->
                Column(modifier = Modifier.padding(vertical = 4.dp)) {
                    Text(token.label, fontWeight = FontWeight.SemiBold, color = textPrimary())
                    token.serviceAccountName?.let {
                        Text(it, fontSize = 12.sp, color = textSecondary())
                    }
                    Text(token.tokenMask, fontSize = 12.sp, fontFamily = FontFamily.Monospace, color = textSecondary())
                    TextButton(onClick = { onRevoke(token.id) }) {
                        Text(L.text(context, localePrefs, R.string.mobile_integrations_serviceTokens_revoke))
                    }
                }
            }
        }
        Button(onClick = onCreate, modifier = Modifier.padding(top = 8.dp)) {
            Text(L.text(context, localePrefs, R.string.mobile_integrations_serviceTokens_create))
        }
    }
}

@Composable
private fun SectionHeader(
    context: Context,
    localePrefs: LocalePreferences,
    titleRes: Int,
    subtitleRes: Int,
) {
    Column(modifier = Modifier.padding(bottom = 4.dp)) {
        Text(
            text = L.text(context, localePrefs, titleRes),
            fontWeight = FontWeight.SemiBold,
            fontSize = 17.sp,
            color = textPrimary(),
        )
        Text(
            text = L.text(context, localePrefs, subtitleRes),
            fontSize = 12.sp,
            color = textSecondary(),
        )
    }
}

@Composable
private fun SecretRevealDialog(
    context: Context,
    localePrefs: LocalePreferences,
    secret: OneTimeSecretReveal,
    onDismiss: () -> Unit,
    onCopy: () -> Unit,
) {
    AlertDialog(
        onDismissRequest = {},
        title = { Text(L.text(context, localePrefs, R.string.mobile_integrations_secret_title)) },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                Text(L.text(context, localePrefs, R.string.mobile_integrations_secret_warning))
                Text(secret.label, fontWeight = FontWeight.SemiBold)
                Text(secret.token, fontFamily = FontFamily.Monospace, fontSize = 12.sp)
            }
        },
        confirmButton = {
            Button(onClick = onCopy) {
                Text(L.text(context, localePrefs, R.string.mobile_integrations_secret_copy))
            }
        },
        dismissButton = {
            TextButton(onClick = onDismiss) {
                Text(L.text(context, localePrefs, R.string.mobile_integrations_secret_dismiss))
            }
        },
    )
}

@Composable
private fun CreateKeyDialog(
    context: Context,
    localePrefs: LocalePreferences,
    label: String,
    onLabelChange: (String) -> Unit,
    scopes: List<AccessKeyScopeDef>,
    selectedScopes: Set<String>,
    onToggleScope: (String, Boolean) -> Unit,
    creating: Boolean,
    onDismiss: () -> Unit,
    onCreate: () -> Unit,
) {
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text(L.text(context, localePrefs, R.string.mobile_integrations_accessKeys_createTitle)) },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                OutlinedTextField(
                    value = label,
                    onValueChange = onLabelChange,
                    label = { Text(L.text(context, localePrefs, R.string.mobile_integrations_accessKeys_label)) },
                    modifier = Modifier.fillMaxWidth(),
                )
                scopes.forEach { scope ->
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.SpaceBetween,
                    ) {
                        Text(scope.label, fontSize = 14.sp)
                        TextButton(onClick = { onToggleScope(scope.id, !selectedScopes.contains(scope.id)) }) {
                            Text(if (selectedScopes.contains(scope.id)) "✓" else "+")
                        }
                    }
                }
            }
        },
        confirmButton = {
            Button(
                onClick = onCreate,
                enabled = !creating && label.isNotBlank() && selectedScopes.isNotEmpty(),
            ) {
                Text(
                    L.text(
                        context,
                        localePrefs,
                        if (creating) R.string.mobile_integrations_creating else R.string.mobile_integrations_create,
                    ),
                )
            }
        },
        dismissButton = {
            TextButton(onClick = onDismiss) {
                Text(L.text(context, localePrefs, R.string.mobile_integrations_cancel))
            }
        },
    )
}

@Composable
private fun CreateServiceTokenDialog(
    context: Context,
    localePrefs: LocalePreferences,
    accountName: String,
    onAccountNameChange: (String) -> Unit,
    label: String,
    onLabelChange: (String) -> Unit,
    scopes: List<AccessKeyScopeDef>,
    selectedScopes: Set<String>,
    onToggleScope: (String, Boolean) -> Unit,
    creating: Boolean,
    onDismiss: () -> Unit,
    onCreate: () -> Unit,
) {
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text(L.text(context, localePrefs, R.string.mobile_integrations_serviceTokens_createTitle)) },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                OutlinedTextField(
                    value = accountName,
                    onValueChange = onAccountNameChange,
                    label = { Text(L.text(context, localePrefs, R.string.mobile_integrations_serviceTokens_accountName)) },
                    modifier = Modifier.fillMaxWidth(),
                )
                OutlinedTextField(
                    value = label,
                    onValueChange = onLabelChange,
                    label = { Text(L.text(context, localePrefs, R.string.mobile_integrations_accessKeys_label)) },
                    modifier = Modifier.fillMaxWidth(),
                )
                scopes.forEach { scope ->
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.SpaceBetween,
                    ) {
                        Text(scope.label, fontSize = 14.sp)
                        TextButton(onClick = { onToggleScope(scope.id, !selectedScopes.contains(scope.id)) }) {
                            Text(if (selectedScopes.contains(scope.id)) "✓" else "+")
                        }
                    }
                }
            }
        },
        confirmButton = {
            Button(
                onClick = onCreate,
                enabled = !creating && accountName.isNotBlank() && selectedScopes.isNotEmpty(),
            ) {
                Text(
                    L.text(
                        context,
                        localePrefs,
                        if (creating) R.string.mobile_integrations_creating else R.string.mobile_integrations_create,
                    ),
                )
            }
        },
        dismissButton = {
            TextButton(onClick = onDismiss) {
                Text(L.text(context, localePrefs, R.string.mobile_integrations_cancel))
            }
        },
    )
}

@Composable
private fun ConfirmDialog(
    context: Context,
    localePrefs: LocalePreferences,
    title: Int,
    confirmLabel: Int,
    onDismiss: () -> Unit,
    onConfirm: () -> Unit,
    message: Int? = null,
) {
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text(L.text(context, localePrefs, title)) },
        text = message?.let { { Text(L.text(context, localePrefs, it)) } },
        confirmButton = {
            Button(onClick = onConfirm) {
                Text(L.text(context, localePrefs, confirmLabel))
            }
        },
        dismissButton = {
            TextButton(onClick = onDismiss) {
                Text(L.text(context, localePrefs, R.string.mobile_integrations_cancel))
            }
        },
    )
}