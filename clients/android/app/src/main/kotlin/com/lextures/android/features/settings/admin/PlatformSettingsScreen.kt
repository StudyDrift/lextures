package com.lextures.android.features.settings.admin

import android.content.Intent
import android.net.Uri
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
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
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
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.config.AppConfiguration
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalePreferences
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.PlatformFeatureDefinition
import com.lextures.android.core.lms.PlatformSettingsAdminLogic
import com.lextures.android.core.lms.PlatformSettingsSnapshot
import com.lextures.android.features.home.HomeShellState
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsEmptyState
import com.lextures.android.features.home.LmsErrorBanner
import com.lextures.android.features.home.LmsSkeletonList
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun PlatformSettingsScreen(
    session: AuthSession,
    shell: HomeShellState,
    localePrefs: LocalePreferences,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    val accessToken by session.accessToken.collectAsState()
    var settings by remember { mutableStateOf<PlatformSettingsSnapshot?>(null) }
    var loading by remember { mutableStateOf(true) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var savedMessage by remember { mutableStateOf<String?>(null) }
    var busyKey by remember { mutableStateOf<String?>(null) }
    var pendingFeature by remember { mutableStateOf<PlatformFeatureDefinition?>(null) }
    val canView = PlatformSettingsAdminLogic.canView(shell.platformFeatures, shell.permissions)

    fun stringByName(name: String): String {
        val id = context.resources.getIdentifier(name, "string", context.packageName)
        return if (id == 0) name else L.text(context, localePrefs, id)
    }

    suspend fun load(token: String) {
        loading = true
        errorMessage = null
        runCatching { settings = LmsApi.fetchPlatformSettings(token) }
            .onFailure { errorMessage = L.text(context, localePrefs, R.string.mobile_admin_platform_error) }
        loading = false
    }

    LaunchedEffect(accessToken, canView) {
        val token = accessToken ?: return@LaunchedEffect
        if (canView) load(token)
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text(L.text(context, localePrefs, R.string.mobile_admin_platform_title)) },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = null)
                    }
                },
            )
        },
        modifier = modifier,
    ) { padding ->
        if (!canView) {
            LmsEmptyState(
                icon = Icons.Filled.Lock,
                title = L.text(context, localePrefs, R.string.mobile_admin_platform_accessDenied_title),
                message = L.text(context, localePrefs, R.string.mobile_admin_platform_accessDenied_message),
                modifier = Modifier.padding(padding).padding(16.dp),
            )
        } else {
            Column(
                modifier = Modifier.fillMaxSize().padding(padding).verticalScroll(rememberScrollState()).padding(16.dp),
                verticalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                Text(L.text(context, localePrefs, R.string.mobile_admin_platform_description), color = textSecondary())
                errorMessage?.let { LmsErrorBanner(message = it) }
                savedMessage?.let { Text(it, color = LexturesColors.BrandTeal, fontWeight = FontWeight.SemiBold) }
                val current = settings
                if (loading && current == null) {
                    LmsSkeletonList(count = 4)
                } else if (current != null) {
                    PlatformConfigCard(current, localePrefs)
                    Text(
                        L.text(context, localePrefs, R.string.mobile_admin_platform_features_title),
                        fontWeight = FontWeight.Bold,
                    )
                    Text(
                        L.text(context, localePrefs, R.string.mobile_admin_platform_features_allowlist),
                        color = textSecondary(),
                    )
                    PlatformSettingsAdminLogic.FEATURE_DEFINITIONS.forEach { feature ->
                        val enabled = PlatformSettingsAdminLogic.value(feature.key, current)
                        LmsCard {
                            Row(
                                modifier = Modifier.fillMaxWidth(),
                                verticalAlignment = Alignment.CenterVertically,
                                horizontalArrangement = Arrangement.spacedBy(12.dp),
                            ) {
                                Column(modifier = Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(4.dp)) {
                                    Text(stringByName(feature.labelResName), fontWeight = FontWeight.SemiBold)
                                    Text(stringByName(feature.descriptionResName), color = textSecondary())
                                    Text(
                                        L.text(context, localePrefs, if (enabled) R.string.mobile_enabled else R.string.mobile_disabled),
                                        color = textSecondary(),
                                    )
                                }
                                Switch(
                                    checked = enabled,
                                    onCheckedChange = { pendingFeature = feature },
                                    enabled = busyKey == null,
                                )
                            }
                        }
                    }
                    Button(
                        onClick = {
                            context.startActivity(Intent(Intent.ACTION_VIEW, Uri.parse(
                                AppConfiguration.webUrl(PlatformSettingsAdminLogic.webSettingsPath()),
                            )))
                        },
                        modifier = Modifier.fillMaxWidth(),
                    ) { Text(L.text(context, localePrefs, R.string.mobile_admin_platform_editOnWeb)) }
                }
            }
        }
    }

    pendingFeature?.let { feature ->
        val current = settings ?: return@let
        val desired = !PlatformSettingsAdminLogic.value(feature.key, current)
        AlertDialog(
            onDismissRequest = { pendingFeature = null },
            title = {
                Text(context.getString(
                    if (desired) R.string.mobile_admin_platform_confirm_enable else R.string.mobile_admin_platform_confirm_disable,
                    stringByName(feature.labelResName),
                ))
            },
            text = { Text(L.text(context, localePrefs, R.string.mobile_admin_platform_confirm_message)) },
            confirmButton = {
                TextButton(onClick = {
                    pendingFeature = null
                    val token = accessToken ?: return@TextButton
                    scope.launch {
                        busyKey = feature.key
                        errorMessage = null
                        savedMessage = null
                        runCatching { LmsApi.setPlatformFeature(feature.key, desired, token) }
                            .onSuccess { updated ->
                                if (PlatformSettingsAdminLogic.value(feature.key, updated) == desired) {
                                    settings = updated
                                    savedMessage = L.text(context, localePrefs, R.string.mobile_admin_platform_saved)
                                } else {
                                    errorMessage = L.text(context, localePrefs, R.string.mobile_admin_platform_toggleError)
                                    load(token)
                                }
                            }
                            .onFailure {
                                errorMessage = L.text(context, localePrefs, R.string.mobile_admin_platform_toggleError)
                                load(token)
                            }
                        busyKey = null
                    }
                }) { Text(L.text(context, localePrefs, R.string.mobile_admin_platform_confirm)) }
            },
            dismissButton = {
                TextButton(onClick = { pendingFeature = null }) {
                    Text(L.text(context, localePrefs, R.string.mobile_cancel))
                }
            },
        )
    }
}

@Composable
private fun PlatformConfigCard(settings: PlatformSettingsSnapshot, localePrefs: LocalePreferences) {
    val context = LocalContext.current
    @Composable fun row(labelRes: Int, value: String) {
        Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
            Text(L.text(context, localePrefs, labelRes), fontWeight = FontWeight.SemiBold, modifier = Modifier.weight(1f))
            Text(
                value.ifBlank { L.text(context, localePrefs, R.string.mobile_admin_platform_notConfigured) },
                color = textSecondary(), modifier = Modifier.weight(1f),
            )
        }
    }
    LmsCard {
        Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
            Text(L.text(context, localePrefs, R.string.mobile_admin_platform_config_title), fontWeight = FontWeight.Bold)
            row(R.string.mobile_admin_platform_config_saml, L.text(context, localePrefs, if (settings.samlSsoEnabled) R.string.mobile_enabled else R.string.mobile_disabled))
            row(R.string.mobile_admin_platform_config_mfa, if (settings.mfaEnabled) settings.mfaEnforcement else L.text(context, localePrefs, R.string.mobile_disabled))
            row(R.string.mobile_admin_platform_config_baseUrl, settings.samlPublicBaseUrl)
            row(R.string.mobile_admin_platform_config_entityId, settings.samlSpEntityId)
            row(R.string.mobile_admin_platform_config_smtp, if (settings.smtpHost.isBlank()) "" else "${settings.smtpHost}:${settings.smtpPort}")
            row(R.string.mobile_admin_platform_config_from, settings.smtpFrom)
            Text(L.text(context, localePrefs, R.string.mobile_admin_platform_config_secretsOmitted), color = textSecondary())
        }
    }
}

