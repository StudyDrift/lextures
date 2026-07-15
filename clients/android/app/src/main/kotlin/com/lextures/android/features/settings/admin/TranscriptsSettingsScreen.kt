package com.lextures.android.features.settings.admin

import android.content.Intent
import android.net.Uri
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
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
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
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
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.config.AppConfiguration
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalePreferences
import com.lextures.android.core.lms.AdminTranscriptRequestRow
import com.lextures.android.core.lms.AdminTranscriptsConfig
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
fun TranscriptsSettingsScreen(
    session: AuthSession,
    shell: HomeShellState,
    localePrefs: LocalePreferences,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    val accessToken by session.accessToken.collectAsState()
    val canView = TranscriptsAdvisingAdminLogic.canViewTranscripts(
        shell.platformFeatures,
        shell.permissions,
    )

    var webhookUrl by remember { mutableStateOf("") }
    var webhookSecret by remember { mutableStateOf("") }
    var pickupInstructions by remember { mutableStateOf("") }
    var hasWebhookSecret by remember { mutableStateOf(false) }
    var failures by remember { mutableStateOf<List<AdminTranscriptRequestRow>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var saving by remember { mutableStateOf(false) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var savedMessage by remember { mutableStateOf<String?>(null) }

    fun apply(config: AdminTranscriptsConfig) {
        webhookUrl = config.webhookUrl
        hasWebhookSecret = config.hasWebhookSecret
        webhookSecret = TranscriptsAdvisingAdminLogic.webhookSecretField(config)
        pickupInstructions = config.pickupInstructions.orEmpty()
    }

    suspend fun load(token: String) {
        loading = true
        errorMessage = null
        try {
            apply(LmsApi.fetchAdminTranscriptsConfig(token))
            failures = LmsApi.fetchAdminTranscriptRequests(token)
        } catch (e: Exception) {
            errorMessage = TranscriptsAdvisingAdminLogic.userFacingError(
                e,
                L.text(context, localePrefs, R.string.mobile_admin_transcripts_loadError),
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
                    title = { Text(L.text(context, localePrefs, R.string.mobile_admin_transcripts_title)) },
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
                message = L.text(context, localePrefs, R.string.mobile_admin_transcripts_flagOff),
            )
        }
        return
    }

    val saveDisabled = TranscriptsAdvisingAdminLogic.isTranscriptsSaveDisabled(saving, webhookUrl)

    Scaffold(
        modifier = modifier.fillMaxSize(),
        topBar = {
            TopAppBar(
                title = { Text(L.text(context, localePrefs, R.string.mobile_admin_transcripts_title)) },
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
                text = L.text(context, localePrefs, R.string.mobile_admin_transcripts_description),
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
                            value = webhookUrl,
                            onValueChange = { webhookUrl = it },
                            modifier = Modifier.fillMaxWidth(),
                            label = {
                                Text(L.text(context, localePrefs, R.string.mobile_admin_transcripts_webhookUrl))
                            },
                            placeholder = {
                                Text(
                                    L.text(
                                        context,
                                        localePrefs,
                                        R.string.mobile_admin_transcripts_webhookUrl_placeholder,
                                    ),
                                )
                            },
                            singleLine = true,
                        )
                        Text(
                            L.text(context, localePrefs, R.string.mobile_admin_transcripts_webhookUrl_hint),
                            fontSize = 12.sp,
                            color = textSecondary(),
                        )
                        OutlinedTextField(
                            value = pickupInstructions,
                            onValueChange = { pickupInstructions = it },
                            modifier = Modifier.fillMaxWidth(),
                            label = {
                                Text(L.text(context, localePrefs, R.string.mobile_admin_transcripts_pickup))
                            },
                            placeholder = {
                                Text(
                                    L.text(
                                        context,
                                        localePrefs,
                                        R.string.mobile_admin_transcripts_pickup_placeholder,
                                    ),
                                )
                            },
                            minLines = 3,
                        )
                        Text(
                            L.text(context, localePrefs, R.string.mobile_admin_transcripts_pickup_hint),
                            fontSize = 12.sp,
                            color = textSecondary(),
                        )
                        OutlinedTextField(
                            value = webhookSecret,
                            onValueChange = { webhookSecret = it },
                            modifier = Modifier.fillMaxWidth(),
                            label = {
                                Text(L.text(context, localePrefs, R.string.mobile_admin_transcripts_secret))
                            },
                            placeholder = {
                                Text(
                                    if (hasWebhookSecret) {
                                        TranscriptsAdvisingAdminLogic.SECRET_PLACEHOLDER
                                    } else {
                                        L.text(
                                            context,
                                            localePrefs,
                                            R.string.mobile_admin_transcripts_secret_placeholder,
                                        )
                                    },
                                )
                            },
                            singleLine = true,
                            visualTransformation = PasswordVisualTransformation(),
                        )
                        Text(
                            L.text(context, localePrefs, R.string.mobile_admin_transcripts_secret_hint),
                            fontSize = 12.sp,
                            color = textSecondary(),
                        )
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
                                val body = TranscriptsAdvisingAdminLogic.buildTranscriptsSaveRequest(
                                    webhookUrl = webhookUrl,
                                    webhookSecret = webhookSecret,
                                    pickupInstructions = pickupInstructions,
                                )
                                apply(LmsApi.putAdminTranscriptsConfig(body, token))
                                savedMessage = L.text(
                                    context,
                                    localePrefs,
                                    R.string.mobile_admin_transcripts_saved,
                                )
                            } catch (e: Exception) {
                                errorMessage = TranscriptsAdvisingAdminLogic.userFacingError(
                                    e,
                                    L.text(context, localePrefs, R.string.mobile_admin_transcripts_saveError),
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
                        Text(L.text(context, localePrefs, R.string.mobile_admin_transcripts_save))
                    }
                }
                if (failures.isNotEmpty()) {
                    Text(
                        L.text(context, localePrefs, R.string.mobile_admin_transcripts_failures_title),
                        fontWeight = FontWeight.Bold,
                        fontSize = 18.sp,
                    )
                    Text(
                        L.text(context, localePrefs, R.string.mobile_admin_transcripts_failures_description),
                        fontSize = 12.sp,
                        color = textSecondary(),
                    )
                    failures.forEach { row ->
                        LmsCard {
                            Column(verticalArrangement = Arrangement.spacedBy(4.dp)) {
                                Text(row.requestedAt.ifBlank { "—" }, fontWeight = FontWeight.SemiBold)
                                Text(
                                    row.errorMessage?.takeIf { it.isNotBlank() } ?: "—",
                                    fontSize = 12.sp,
                                    color = LexturesColors.Error,
                                )
                                Text(
                                    "${L.text(context, localePrefs, R.string.mobile_admin_transcripts_failures_http)}: ${TranscriptsAdvisingAdminLogic.httpStatusLabel(row.webhookResponseCode)}",
                                    fontSize = 11.sp,
                                    color = textSecondary(),
                                )
                            }
                        }
                    }
                }
                OutlinedButton(
                    onClick = {
                        val url = AppConfiguration.webUrl(
                            TranscriptsAdvisingAdminLogic.Section.TRANSCRIPTS.webPath,
                        )
                        context.startActivity(Intent(Intent.ACTION_VIEW, Uri.parse(url)))
                    },
                    modifier = Modifier.fillMaxWidth(),
                ) {
                    Text(L.text(context, localePrefs, R.string.mobile_admin_transcripts_configureOnWeb))
                }
            }
        }
    }
}
