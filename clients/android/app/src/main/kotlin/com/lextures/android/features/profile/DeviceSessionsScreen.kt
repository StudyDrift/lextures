package com.lextures.android.features.profile

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
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
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.R
import com.lextures.android.core.auth.ActiveSession
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.auth.SessionsApi
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.LexturesType
import com.lextures.android.core.design.accentColor
import com.lextures.android.core.design.sceneBackground
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.DateFormatting
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalLocalePreferences
import com.lextures.android.features.home.LmsCard
import kotlinx.coroutines.launch
import java.util.Locale

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun DeviceSessionsScreen(
    session: AuthSession,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    val localePreferences = LocalLocalePreferences.current
    val accessToken by session.accessToken.collectAsState()
    val scope = rememberCoroutineScope()
    var sessions by remember { mutableStateOf<List<ActiveSession>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var confirmingRevokeOthers by remember { mutableStateOf(false) }
    var revokingSessionId by remember { mutableStateOf<String?>(null) }

    val locale = localePreferences.effectiveLocale

    fun loadSessions() {
        val token = accessToken ?: return
        scope.launch {
            loading = true
            errorMessage = null
            try {
                sessions = SessionsApi.fetchSessions(token)
            } catch (_: Exception) {
                errorMessage = L.text(context, localePreferences, R.string.mobile_sessions_loadError)
            } finally {
                loading = false
            }
        }
    }

    LaunchedEffect(accessToken) {
        loadSessions()
    }

    if (confirmingRevokeOthers) {
        androidx.compose.material3.AlertDialog(
            onDismissRequest = { confirmingRevokeOthers = false },
            title = { Text(L.text(context, localePreferences, R.string.mobile_sessions_signOutOthersConfirm)) },
            confirmButton = {
                TextButton(onClick = {
                    confirmingRevokeOthers = false
                    val token = accessToken ?: return@TextButton
                    scope.launch {
                        try {
                            SessionsApi.revokeOtherSessions(token)
                            sessions = sessions.filter { it.isCurrent }
                        } catch (_: Exception) {
                            errorMessage = L.text(context, localePreferences, R.string.mobile_sessions_revokeOthersError)
                        }
                    }
                }) {
                    Text(L.text(context, localePreferences, R.string.mobile_sessions_signOutOthers))
                }
            },
            dismissButton = {
                TextButton(onClick = { confirmingRevokeOthers = false }) {
                    Text(L.text(context, localePreferences, android.R.string.cancel))
                }
            },
        )
    }

    Column(
        modifier = modifier
            .fillMaxSize()
            .background(sceneBackground()),
    ) {
        TopAppBar(
            title = { Text(L.text(context, localePreferences, R.string.mobile_sessions_title)) },
            navigationIcon = {
                IconButton(onClick = onBack) {
                    Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = null)
                }
            },
        )

        Column(
            modifier = Modifier
                .fillMaxSize()
                .verticalScroll(rememberScrollState())
                .padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            Text(
                text = L.text(context, localePreferences, R.string.mobile_sessions_description),
                fontSize = 14.sp,
                color = textSecondary(),
            )

            errorMessage?.let {
                Text(text = it, color = LexturesColors.Error, fontSize = 13.sp)
            }

            if (loading && sessions.isEmpty()) {
                CircularProgressIndicator(color = accentColor(), modifier = Modifier.align(Alignment.CenterHorizontally))
            } else if (sessions.isEmpty()) {
                Text(
                    text = L.text(context, localePreferences, R.string.mobile_sessions_emptyMessage),
                    color = textSecondary(),
                )
            } else {
                sessions.forEach { row ->
                    SessionRow(
                        row = row,
                        locale = locale,
                        revoking = revokingSessionId == row.id,
                        onRevoke = {
                            val token = accessToken ?: return@SessionRow
                            scope.launch {
                                revokingSessionId = row.id
                                try {
                                    SessionsApi.revokeSession(row.id, token)
                                    sessions = sessions.filter { it.id != row.id }
                                } catch (_: Exception) {
                                    errorMessage = L.text(context, localePreferences, R.string.mobile_sessions_revokeError)
                                } finally {
                                    revokingSessionId = null
                                }
                            }
                        },
                    )
                }

                if (sessions.any { !it.isCurrent }) {
                    TextButton(
                        onClick = { confirmingRevokeOthers = true },
                        modifier = Modifier.fillMaxWidth(),
                    ) {
                        Text(
                            text = L.text(context, localePreferences, R.string.mobile_sessions_signOutOthers),
                            color = LexturesColors.Error,
                            fontWeight = FontWeight.SemiBold,
                        )
                    }
                }
            }
        }
    }
}

@Composable
private fun SessionRow(
    row: ActiveSession,
    locale: Locale,
    revoking: Boolean,
    onRevoke: () -> Unit,
) {
    val context = LocalContext.current
    val localePreferences = LocalLocalePreferences.current

    LmsCard {
        Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Text(
                    text = row.deviceLabel,
                    fontSize = 14.sp,
                    fontWeight = FontWeight.SemiBold,
                    color = textPrimary(),
                )
                if (row.isCurrent) {
                    Text(
                        text = L.text(context, localePreferences, R.string.mobile_sessions_currentDevice),
                        fontSize = 12.sp,
                        fontWeight = FontWeight.SemiBold,
                        color = accentColor(),
                        modifier = Modifier
                            .clip(RoundedCornerShape(999.dp))
                            .background(LexturesColors.BrandTeal.copy(alpha = 0.14f))
                            .padding(horizontal = 8.dp, vertical = 3.dp),
                    )
                }
            }
            SessionMeta(
                L.text(context, localePreferences, R.string.mobile_sessions_lastActive),
                DateFormatting.formatAbsoluteShort(row.lastUsedAt, locale),
            )
            SessionMeta(
                L.text(context, localePreferences, R.string.mobile_sessions_location),
                row.location,
            )
            SessionMeta(
                L.text(context, localePreferences, R.string.mobile_sessions_authMethod),
                row.authMethod,
            )
            if (!row.isCurrent) {
                if (revoking) {
                    CircularProgressIndicator(strokeWidth = 2.dp, modifier = Modifier.padding(top = 4.dp))
                } else {
                    TextButton(onClick = onRevoke) {
                        Text(
                            text = L.text(context, localePreferences, R.string.mobile_sessions_signOutDevice),
                            color = LexturesColors.Error,
                            fontSize = 12.sp,
                            fontWeight = FontWeight.SemiBold,
                        )
                    }
                }
            }
        }
    }
}

@Composable
private fun SessionMeta(label: String, value: String) {
    Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
        Text(text = label, fontSize = 12.sp, color = textSecondary(), modifier = Modifier.fillMaxWidth(0.35f))
        Text(text = value, fontSize = 12.sp, color = textPrimary())
    }
}
