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
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
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
import androidx.compose.ui.text.font.FontFamily
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
import com.lextures.android.core.lms.AdminAuditEvent
import com.lextures.android.core.lms.AuditLogAdminLogic
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.features.home.HomeShellState
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsEmptyState
import com.lextures.android.features.home.LmsErrorBanner
import com.lextures.android.features.home.LmsSkeletonList
import kotlinx.coroutines.launch
import java.time.Instant
import java.time.ZoneId
import java.time.format.DateTimeFormatter
import java.time.format.FormatStyle

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun AuditLogAdminScreen(
    session: AuthSession,
    shell: HomeShellState,
    localePrefs: LocalePreferences,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    val accessToken by session.accessToken.collectAsState()
    var actionFilter by remember { mutableStateOf("") }
    var events by remember { mutableStateOf<List<AdminAuditEvent>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    val canView = AuditLogAdminLogic.canView(shell.platformFeatures, shell.permissions)

    fun load() {
        val token = accessToken ?: return
        scope.launch {
            loading = true
            errorMessage = null
            try {
                events = LmsApi.fetchAdminAuditLog(accessToken = token, action = actionFilter)
            } catch (_: Exception) {
                errorMessage = L.text(R.string.mobile_admin_auditLog_error)
                events = emptyList()
            } finally {
                loading = false
            }
        }
    }

    LaunchedEffect(canView, accessToken) {
        if (canView) load()
    }

    Scaffold(
        modifier = modifier.fillMaxSize(),
        topBar = {
            TopAppBar(
                title = { Text(L.text(R.string.mobile_admin_auditLog_title)) },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = null)
                    }
                },
            )
        },
    ) { padding ->
        if (!canView) {
            Column(Modifier = Modifier.fillMaxSize().padding(padding).padding(16.dp)) {
                LmsEmptyState(
                    icon = Icons.Default.Lock,
                    title = L.text(R.string.mobile_admin_auditLog_accessDenied_title),
                    message = L.text(R.string.mobile_admin_auditLog_accessDenied_message),
                )
            }
            return@Scaffold
        }

        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
                .verticalScroll(rememberScrollState())
                .padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(16.dp),
        ) {
            Text(
                L.text(R.string.mobile_admin_auditLog_description),
                color = textSecondary(),
                fontSize = 14.sp,
            )
            LmsCard {
                Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    OutlinedTextField(
                        value = actionFilter,
                        onValueChange = { actionFilter = it },
                        modifier = Modifier.fillMaxWidth(),
                        singleLine = true,
                        label = { Text(L.text(R.string.mobile_admin_auditLog_filter_label)) },
                        placeholder = { Text(L.text(R.string.mobile_admin_auditLog_filter_placeholder)) },
                    )
                    Button(onClick = { load() }, modifier = Modifier.fillMaxWidth()) {
                        Text(L.text(R.string.mobile_admin_auditLog_filter_apply))
                    }
                }
            }
            errorMessage?.let { LmsErrorBanner(message = it) }
            when {
                loading && events.isEmpty() -> LmsSkeletonList(count = 4)
                events.isEmpty() -> LmsEmptyState(
                    icon = Icons.Default.Lock,
                    title = L.text(R.string.mobile_admin_auditLog_empty_title),
                    message = L.text(R.string.mobile_admin_auditLog_empty_message),
                )
                else -> events.forEach { event ->
                    AuditEventCard(event)
                }
            }
            Button(
                onClick = {
                    val uri = Uri.parse(AppConfiguration.webUrl(AuditLogAdminLogic.webPath()))
                    context.startActivity(Intent(Intent.ACTION_VIEW, uri))
                },
                modifier = Modifier.fillMaxWidth(),
            ) {
                Text(L.text(R.string.mobile_admin_auditLog_openOnWeb))
            }
        }
    }
}

@Composable
private fun AuditEventCard(event: AdminAuditEvent) {
    LmsCard {
        Column(verticalArrangement = Arrangement.spacedBy(4.dp)) {
            Text(formatTimestamp(event.timestamp), fontSize = 12.sp, color = textSecondary())
            Text(event.eventType, fontWeight = FontWeight.SemiBold, color = textPrimary())
            Text(
                L.format(R.string.mobile_admin_auditLog_actor, event.actorId),
                fontSize = 12.sp,
                fontFamily = FontFamily.Monospace,
                color = textSecondary(),
            )
            Text(
                L.format(
                    R.string.mobile_admin_auditLog_target,
                    AuditLogAdminLogic.targetLabel(event.targetType, event.targetId),
                ),
                fontSize = 12.sp,
                color = textSecondary(),
            )
        }
    }
}

private fun formatTimestamp(raw: String): String {
    return try {
        val instant = Instant.parse(raw)
        DateTimeFormatter.ofLocalizedDateTime(FormatStyle.MEDIUM)
            .withZone(ZoneId.systemDefault())
            .format(instant)
    } catch (_: Exception) {
        raw
    }
}
