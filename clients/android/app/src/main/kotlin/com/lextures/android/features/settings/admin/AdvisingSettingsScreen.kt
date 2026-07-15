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
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.ExposedDropdownMenuBox
import androidx.compose.material3.ExposedDropdownMenuDefaults
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
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
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.config.AppConfiguration
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalePreferences
import com.lextures.android.core.lms.AdminAdvisingConfig
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.TranscriptsAdvisingAdminLogic
import com.lextures.android.features.home.HomeShellState
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsEmptyState
import com.lextures.android.features.home.LmsErrorBanner
import com.lextures.android.features.home.LmsSkeletonList
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun AdvisingSettingsScreen(
    session: AuthSession,
    shell: HomeShellState,
    localePrefs: LocalePreferences,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    val accessToken by session.accessToken.collectAsState()
    val canView = TranscriptsAdvisingAdminLogic.canViewAdvising(
        shell.platformFeatures,
        shell.permissions,
    )

    var appointmentUrl by remember { mutableStateOf("") }
    var provider by remember { mutableStateOf(TranscriptsAdvisingAdminLogic.DegreeAuditProvider.NONE) }
    var baseUrl by remember { mutableStateOf("") }
    var credentialsRef by remember { mutableStateOf("") }
    var atRiskBanner by remember { mutableStateOf(false) }
    var providerExpanded by remember { mutableStateOf(false) }
    var loading by remember { mutableStateOf(true) }
    var saving by remember { mutableStateOf(false) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var savedMessage by remember { mutableStateOf<String?>(null) }

    fun apply(config: AdminAdvisingConfig) {
        appointmentUrl = config.appointmentUrl
        provider = TranscriptsAdvisingAdminLogic.DegreeAuditProvider.normalized(config.degreeAuditProvider)
        baseUrl = config.degreeAuditBaseUrl
        credentialsRef = config.apiCredentialsRef
        atRiskBanner = config.atRiskBannerEnabled
    }

    suspend fun load(token: String) {
        loading = true
        errorMessage = null
        try {
            apply(LmsApi.fetchAdminAdvisingConfig(token))
        } catch (e: Exception) {
            errorMessage = TranscriptsAdvisingAdminLogic.userFacingError(
                e,
                L.text(context, localePrefs, R.string.mobile_admin_advising_loadError),
            )
        }
        loading = false
    }

    LaunchedEffect(accessToken, canView) {
        val token = accessToken ?: return@LaunchedEffect
        if (canView) load(token)
    }

    if (!canView) {
        Scaffold(
            modifier = modifier.fillMaxSize(),
            topBar = {
                TopAppBar(
                    title = { Text(L.text(context, localePrefs, R.string.mobile_admin_advising_title)) },
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
                title = L.text(context, localePrefs, R.string.mobile_admin_transcriptsAdvising_accessDenied_title),
                message = L.text(context, localePrefs, R.string.mobile_admin_advising_flagOff),
            )
        }
        return
    }

    val saveDisabled = TranscriptsAdvisingAdminLogic.isAdvisingSaveDisabled(saving, appointmentUrl)

    fun providerLabel(p: TranscriptsAdvisingAdminLogic.DegreeAuditProvider): String = when (p) {
        TranscriptsAdvisingAdminLogic.DegreeAuditProvider.NONE ->
            L.text(context, localePrefs, R.string.mobile_admin_advising_provider_none)
        TranscriptsAdvisingAdminLogic.DegreeAuditProvider.DEGREEWORKS ->
            L.text(context, localePrefs, R.string.mobile_admin_advising_provider_degreeworks)
        TranscriptsAdvisingAdminLogic.DegreeAuditProvider.STELLIC ->
            L.text(context, localePrefs, R.string.mobile_admin_advising_provider_stellic)
    }

    Scaffold(
        modifier = modifier.fillMaxSize(),
        topBar = {
            TopAppBar(
                title = { Text(L.text(context, localePrefs, R.string.mobile_admin_advising_title)) },
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
            Text(
                text = L.text(context, localePrefs, R.string.mobile_admin_advising_description),
                fontSize = 14.sp,
                color = textSecondary(),
            )
            errorMessage?.let { LmsErrorBanner(message = it) }
            savedMessage?.let {
                Text(it, fontSize = 12.sp, color = LexturesColors.BrandTeal)
            }
            if (loading) {
                LmsSkeletonList(count = 4)
            } else {
                LmsCard {
                    Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
                        OutlinedTextField(
                            value = appointmentUrl,
                            onValueChange = { appointmentUrl = it },
                            modifier = Modifier.fillMaxWidth(),
                            label = {
                                Text(L.text(context, localePrefs, R.string.mobile_admin_advising_appointmentUrl))
                            },
                            placeholder = {
                                Text(
                                    L.text(
                                        context,
                                        localePrefs,
                                        R.string.mobile_admin_advising_appointmentUrl_placeholder,
                                    ),
                                )
                            },
                            singleLine = true,
                        )
                        Text(
                            L.text(context, localePrefs, R.string.mobile_admin_advising_appointmentUrl_hint),
                            fontSize = 12.sp,
                            color = textSecondary(),
                        )
                        ExposedDropdownMenuBox(
                            expanded = providerExpanded,
                            onExpandedChange = { providerExpanded = it },
                        ) {
                            OutlinedTextField(
                                value = providerLabel(provider),
                                onValueChange = {},
                                readOnly = true,
                                modifier = Modifier
                                    .menuAnchor()
                                    .fillMaxWidth(),
                                label = {
                                    Text(L.text(context, localePrefs, R.string.mobile_admin_advising_provider))
                                },
                                trailingIcon = {
                                    ExposedDropdownMenuDefaults.TrailingIcon(expanded = providerExpanded)
                                },
                            )
                            ExposedDropdownMenu(
                                expanded = providerExpanded,
                                onDismissRequest = { providerExpanded = false },
                            ) {
                                TranscriptsAdvisingAdminLogic.DegreeAuditProvider.entries.forEach { option ->
                                    DropdownMenuItem(
                                        text = { Text(providerLabel(option)) },
                                        onClick = {
                                            provider = option
                                            providerExpanded = false
                                        },
                                    )
                                }
                            }
                        }
                        if (provider != TranscriptsAdvisingAdminLogic.DegreeAuditProvider.NONE) {
                            OutlinedTextField(
                                value = baseUrl,
                                onValueChange = { baseUrl = it },
                                modifier = Modifier.fillMaxWidth(),
                                label = {
                                    Text(L.text(context, localePrefs, R.string.mobile_admin_advising_baseUrl))
                                },
                                placeholder = {
                                    Text(
                                        L.text(
                                            context,
                                            localePrefs,
                                            R.string.mobile_admin_advising_baseUrl_placeholder,
                                        ),
                                    )
                                },
                                singleLine = true,
                            )
                            OutlinedTextField(
                                value = credentialsRef,
                                onValueChange = { credentialsRef = it },
                                modifier = Modifier.fillMaxWidth(),
                                label = {
                                    Text(
                                        L.text(
                                            context,
                                            localePrefs,
                                            R.string.mobile_admin_advising_credentialsRef,
                                        ),
                                    )
                                },
                                placeholder = {
                                    Text(
                                        L.text(
                                            context,
                                            localePrefs,
                                            R.string.mobile_admin_advising_credentialsRef_placeholder,
                                        ),
                                    )
                                },
                                singleLine = true,
                            )
                            Row(
                                modifier = Modifier.fillMaxWidth(),
                                verticalAlignment = Alignment.CenterVertically,
                                horizontalArrangement = Arrangement.SpaceBetween,
                            ) {
                                Text(
                                    L.text(context, localePrefs, R.string.mobile_admin_advising_atRiskBanner),
                                    modifier = Modifier.weight(1f).padding(end = 12.dp),
                                    fontSize = 14.sp,
                                )
                                Switch(checked = atRiskBanner, onCheckedChange = { atRiskBanner = it })
                            }
                        }
                    }
                }
                Button(
                    onClick = {
                        val token = accessToken ?: return@Button
                        scope.launch {
                            saving = true
                            errorMessage = null
                            savedMessage = null
                            try {
                                val body = TranscriptsAdvisingAdminLogic.buildAdvisingSaveRequest(
                                    appointmentUrl = appointmentUrl,
                                    provider = provider,
                                    baseUrl = baseUrl,
                                    credentialsRef = credentialsRef,
                                    atRiskBannerEnabled = atRiskBanner,
                                )
                                apply(LmsApi.postAdminAdvisingConfig(body, token))
                                savedMessage = L.text(
                                    context,
                                    localePrefs,
                                    R.string.mobile_admin_advising_saved,
                                )
                            } catch (e: Exception) {
                                errorMessage = TranscriptsAdvisingAdminLogic.userFacingError(
                                    e,
                                    L.text(context, localePrefs, R.string.mobile_admin_advising_saveError),
                                )
                            }
                            saving = false
                        }
                    },
                    enabled = !saveDisabled,
                    modifier = Modifier.fillMaxWidth(),
                ) {
                    if (saving) {
                        CircularProgressIndicator(modifier = Modifier.padding(4.dp))
                    } else {
                        Text(L.text(context, localePrefs, R.string.mobile_admin_advising_save))
                    }
                }
                OutlinedButton(
                    onClick = {
                        val url = AppConfiguration.webUrl(
                            TranscriptsAdvisingAdminLogic.Section.ADVISING.webPath,
                        )
                        context.startActivity(Intent(Intent.ACTION_VIEW, Uri.parse(url)))
                    },
                    modifier = Modifier.fillMaxWidth(),
                ) {
                    Text(L.text(context, localePrefs, R.string.mobile_admin_advising_configureOnWeb))
                }
            }
        }
    }
}
