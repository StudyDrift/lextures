package com.lextures.android.features.settings.admin

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.ExposedDropdownMenuBox
import androidx.compose.material3.ExposedDropdownMenuDefaults
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
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
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalePreferences
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.OrgBrandingAdminLogic
import com.lextures.android.core.network.ApiError
import com.lextures.android.features.home.LmsCard
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun AiProviderSettingsScreen(
    session: AuthSession,
    localePrefs: LocalePreferences,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    val accessToken by session.accessToken.collectAsState()

    var available by remember { mutableStateOf(false) }
    var providers by remember { mutableStateOf<List<String>>(emptyList()) }
    var modelAliases by remember { mutableStateOf<List<String>>(emptyList()) }
    var provider by remember { mutableStateOf("openrouter") }
    var modelAlias by remember { mutableStateOf("claude-3-5-sonnet") }
    var fallbackProvider by remember { mutableStateOf("") }
    var byokKey by remember { mutableStateOf("") }
    var byokConfigured by remember { mutableStateOf(false) }
    var loading by remember { mutableStateOf(true) }
    var saving by remember { mutableStateOf(false) }
    var testing by remember { mutableStateOf(false) }
    var statusMessage by remember { mutableStateOf<String?>(null) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var testMessage by remember { mutableStateOf<String?>(null) }

    LaunchedEffect(accessToken) {
        val token = accessToken ?: return@LaunchedEffect
        loading = true
        try {
            val response = LmsApi.fetchAiProviderSettings(token)
            available = true
            providers = response.providers ?: OrgBrandingAdminLogic.PROVIDER_LABELS.keys.toList()
            modelAliases = response.modelAliases ?: listOf("claude-3-5-sonnet", "gpt-4o", "gemini-1.5-pro")
            provider = response.provider ?: provider
            modelAlias = response.modelAlias ?: modelAlias
            fallbackProvider = response.fallbackProvider.orEmpty()
            byokConfigured = response.byokConfigured == true
            byokKey = OrgBrandingAdminLogic.byokFieldValue(byokConfigured, byokKey)
        } catch (error: Throwable) {
            if (error is ApiError.HttpStatus && (error.code == 404 || error.code == 403)) {
                available = false
            } else {
                available = true
                errorMessage = OrgBrandingAdminLogic.userFacingError(
                    error,
                    L.text(context, localePrefs, R.string.mobile_admin_orgBranding_aiProvider_loadError),
                )
            }
        } finally {
            loading = false
        }
    }

    if (!available) return

    LmsCard(modifier = modifier) {
        Column(
            modifier = Modifier.padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            Text(
                text = L.text(context, localePrefs, R.string.mobile_admin_orgBranding_aiProvider_title),
                fontWeight = FontWeight.SemiBold,
                color = textPrimary(),
            )
            Text(
                text = L.text(context, localePrefs, R.string.mobile_admin_orgBranding_aiProvider_intro),
                fontSize = 14.sp,
                color = textSecondary(),
            )

            if (loading) {
                CircularProgressIndicator(modifier = Modifier.align(androidx.compose.ui.Alignment.CenterHorizontally))
            } else {
                DropdownField(
                    label = L.text(context, localePrefs, R.string.mobile_admin_orgBranding_aiProvider_provider),
                    value = OrgBrandingAdminLogic.providerLabel(provider),
                    options = providers.map { it to OrgBrandingAdminLogic.providerLabel(it) },
                    onSelect = { provider = it },
                )
                DropdownField(
                    label = L.text(context, localePrefs, R.string.mobile_admin_orgBranding_aiProvider_modelAlias),
                    value = modelAlias,
                    options = modelAliases.map { it to it },
                    onSelect = { modelAlias = it },
                )
                DropdownField(
                    label = L.text(context, localePrefs, R.string.mobile_admin_orgBranding_aiProvider_fallback),
                    value = fallbackProvider.takeIf { it.isNotEmpty() }
                        ?: L.text(context, localePrefs, R.string.mobile_admin_orgBranding_aiProvider_fallbackNone),
                    options = listOf("" to L.text(context, localePrefs, R.string.mobile_admin_orgBranding_aiProvider_fallbackNone)) +
                        providers.map { it to OrgBrandingAdminLogic.providerLabel(it) },
                    onSelect = { fallbackProvider = it },
                )

                OutlinedTextField(
                    value = byokKey,
                    onValueChange = { byokKey = it },
                    label = { Text(L.text(context, localePrefs, R.string.mobile_admin_orgBranding_aiProvider_byokKey)) },
                    visualTransformation = PasswordVisualTransformation(),
                    modifier = Modifier.fillMaxWidth(),
                )
                if (byokConfigured) {
                    Text(
                        text = L.text(context, localePrefs, R.string.mobile_admin_orgBranding_aiProvider_byokConfigured),
                        fontSize = 12.sp,
                        color = androidx.compose.ui.graphics.Color(0xFF059669),
                    )
                }
                Text(
                    text = L.text(context, localePrefs, R.string.mobile_admin_orgBranding_aiProvider_byokHint),
                    fontSize = 12.sp,
                    color = textSecondary(),
                )

                Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                    Button(
                        onClick = {
                            val token = accessToken ?: return@Button
                            scope.launch {
                                saving = true
                                errorMessage = null
                                statusMessage = null
                                try {
                                    val request = OrgBrandingAdminLogic.buildAiProviderSaveRequest(
                                        provider,
                                        modelAlias,
                                        fallbackProvider,
                                        byokKey,
                                    )
                                    val response = LmsApi.putAiProviderSettings(request, token)
                                    provider = response.provider ?: provider
                                    modelAlias = response.modelAlias ?: modelAlias
                                    fallbackProvider = response.fallbackProvider.orEmpty()
                                    byokConfigured = response.byokConfigured == true
                                    byokKey = OrgBrandingAdminLogic.PLATFORM_SECRET_PLACEHOLDER
                                    statusMessage = L.text(context, localePrefs, R.string.mobile_admin_orgBranding_aiProvider_saved)
                                } catch (error: Throwable) {
                                    errorMessage = OrgBrandingAdminLogic.userFacingError(
                                        error,
                                        L.text(context, localePrefs, R.string.mobile_admin_orgBranding_aiProvider_saveError),
                                    )
                                } finally {
                                    saving = false
                                }
                            }
                        },
                        enabled = !saving && !testing,
                    ) {
                        if (saving) {
                            CircularProgressIndicator(modifier = Modifier.size(18.dp))
                        } else {
                            Text(L.text(context, localePrefs, R.string.mobile_admin_orgBranding_aiProvider_save))
                        }
                    }
                    OutlinedButton(
                        onClick = {
                            val token = accessToken ?: return@OutlinedButton
                            scope.launch {
                                testing = true
                                testMessage = null
                                try {
                                    val response = LmsApi.testAiProviderConnection(token)
                                    testMessage = L.format(
                                        context,
                                        localePrefs,
                                        R.string.mobile_admin_orgBranding_aiProvider_testSuccess,
                                        response.provider ?: provider,
                                        response.latencyMs?.toString() ?: "?",
                                        response.responsePreview ?: "OK",
                                    )
                                } catch (error: Throwable) {
                                    testMessage = OrgBrandingAdminLogic.userFacingError(
                                        error,
                                        L.text(context, localePrefs, R.string.mobile_admin_orgBranding_aiProvider_testError),
                                    )
                                } finally {
                                    testing = false
                                }
                            }
                        },
                        enabled = !saving && !testing,
                    ) {
                        if (testing) {
                            CircularProgressIndicator(modifier = Modifier.size(18.dp))
                        } else {
                            Text(L.text(context, localePrefs, R.string.mobile_admin_orgBranding_aiProvider_test))
                        }
                    }
                }

                statusMessage?.let {
                    Text(text = it, fontSize = 12.sp, color = androidx.compose.ui.graphics.Color(0xFF059669))
                }
                errorMessage?.let {
                    Text(text = it, fontSize = 12.sp, color = androidx.compose.ui.graphics.Color(0xFFDC2626))
                }
                testMessage?.let {
                    Text(text = it, fontSize = 12.sp, color = textSecondary())
                }
            }
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun DropdownField(
    label: String,
    value: String,
    options: List<Pair<String, String>>,
    onSelect: (String) -> Unit,
) {
    var expanded by remember { mutableStateOf(false) }
    ExposedDropdownMenuBox(expanded = expanded, onExpandedChange = { expanded = !expanded }) {
        OutlinedTextField(
            value = value,
            onValueChange = {},
            readOnly = true,
            label = { Text(label) },
            trailingIcon = { ExposedDropdownMenuDefaults.TrailingIcon(expanded = expanded) },
            modifier = Modifier.menuAnchor().fillMaxWidth(),
        )
        ExposedDropdownMenu(expanded = expanded, onDismissRequest = { expanded = false }) {
            options.forEach { (key, display) ->
                DropdownMenuItem(
                    text = { Text(display) },
                    onClick = {
                        onSelect(key)
                        expanded = false
                    },
                )
            }
        }
    }
}
