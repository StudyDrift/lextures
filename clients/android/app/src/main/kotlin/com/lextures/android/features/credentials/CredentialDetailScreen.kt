package com.lextures.android.features.credentials

import android.content.Intent
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Button
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.core.net.toUri
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.config.AppConfiguration
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalePreferences
import com.lextures.android.core.lms.CredentialsLogic
import com.lextures.android.core.lms.IssuedCredentialSummary
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsErrorBanner
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import kotlinx.coroutines.launch

@Composable
fun CredentialDetailScreen(
    session: AuthSession,
    localePrefs: LocalePreferences,
    credential: IssuedCredentialSummary,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    val accessToken = session.accessToken.value
    var actionError by remember { mutableStateOf<String?>(null) }
    var linkedInLoading by remember { mutableStateOf(false) }
    var badgeLoading by remember { mutableStateOf(false) }

    Column(
        modifier = modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(16.dp),
    ) {
        LmsCard {
            Text(credential.title, fontWeight = FontWeight.Bold, fontSize = 18.sp, color = textPrimary())
            Text(
                CredentialsLogic.sourceTypeLabel(credential.sourceType),
                fontSize = 12.sp,
                color = textSecondary(),
            )
            Text(
                context.getString(
                    R.string.mobile_credentials_issued,
                    CredentialsLogic.issuedDateLabel(credential.issuedAt),
                ),
                fontSize = 14.sp,
                color = textSecondary(),
            )
        }

        actionError?.let { LmsErrorBanner(message = it, modifier = Modifier.padding(top = 12.dp)) }

        Button(
            onClick = {
                val intent = Intent(Intent.ACTION_SEND).apply {
                    type = "text/plain"
                    putExtra(Intent.EXTRA_TEXT, CredentialsLogic.shareText(credential.title, credential.verificationUrl))
                }
                context.startActivity(Intent.createChooser(intent, null))
            },
            modifier = Modifier.fillMaxWidth().padding(top = 12.dp),
        ) {
            Text(L.text(context, localePrefs, R.string.mobile_credentials_shareVerify))
        }

        Button(
            onClick = {
                context.startActivity(Intent(Intent.ACTION_VIEW, credential.verificationUrl.toUri()))
            },
            modifier = Modifier.fillMaxWidth().padding(top = 8.dp),
        ) {
            Text(L.text(context, localePrefs, R.string.mobile_credentials_openVerify))
        }

        Button(
            onClick = {
                val token = accessToken ?: return@Button
                linkedInLoading = true
                actionError = null
                scope.launch {
                    try {
                        val params = LmsApi.fetchCredentialLinkedInParams(credential.id, token)
                        LmsApi.recordCredentialShare(credential.id, "linkedin", token)
                        context.startActivity(Intent(Intent.ACTION_VIEW, params.url.toUri()))
                    } catch (_: Exception) {
                        actionError = L.text(context, localePrefs, R.string.mobile_credentials_linkedInError)
                    } finally {
                        linkedInLoading = false
                    }
                }
            },
            enabled = !linkedInLoading && !credential.revoked,
            modifier = Modifier.fillMaxWidth().padding(top = 8.dp),
        ) {
            Text(
                if (linkedInLoading) {
                    L.text(context, localePrefs, R.string.mobile_credentials_openingLinkedIn)
                } else {
                    L.text(context, localePrefs, R.string.mobile_credentials_addLinkedIn)
                },
            )
        }

        Button(
            onClick = {
                val token = accessToken ?: return@Button
                badgeLoading = true
                actionError = null
                scope.launch {
                    try {
                        val export = LmsApi.fetchCredentialBadgeExportUrl(credential.id, token)
                        LmsApi.recordCredentialShare(credential.id, "badge_export", token)
                        context.startActivity(Intent(Intent.ACTION_VIEW, export.downloadUrl.toUri()))
                    } catch (_: Exception) {
                        actionError = L.text(context, localePrefs, R.string.mobile_credentials_badgeExportError)
                    } finally {
                        badgeLoading = false
                    }
                }
            },
            enabled = !badgeLoading && !credential.revoked,
            modifier = Modifier.fillMaxWidth().padding(top = 8.dp),
        ) {
            Text(
                if (badgeLoading) {
                    L.text(context, localePrefs, R.string.mobile_credentials_exportingBadge)
                } else {
                    L.text(context, localePrefs, R.string.mobile_credentials_exportBadge)
                },
            )
        }
    }
}