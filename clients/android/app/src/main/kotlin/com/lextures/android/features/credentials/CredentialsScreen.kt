package com.lextures.android.features.credentials

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Verified
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalePreferences
import com.lextures.android.core.lms.CredentialsLogic
import com.lextures.android.core.lms.IssuedCredentialSummary
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.offline.OfflineCacheKey
import com.lextures.android.core.offline.OfflineService
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsEmptyState
import com.lextures.android.features.home.LmsSkeletonList
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import kotlinx.serialization.builtins.ListSerializer
import kotlinx.serialization.serializer

@Composable
fun CredentialsScreen(
    session: AuthSession,
    localePrefs: LocalePreferences,
    onOpenCredential: (IssuedCredentialSummary) -> Unit,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    val offline = remember { OfflineService.get(context) }
    val accessToken = session.accessToken.value

    var credentials by remember { mutableStateOf<List<IssuedCredentialSummary>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var errorMessage by remember { mutableStateOf<String?>(null) }

    LaunchedEffect(accessToken) {
        val token = accessToken ?: return@LaunchedEffect
        loading = credentials.isEmpty()
        errorMessage = null
        try {
            val result = offline.cachedFetch(
                key = OfflineCacheKey.credentialsList(),
                accessToken = token,
                serializer = ListSerializer(IssuedCredentialSummary.serializer()),
            ) {
                LmsApi.fetchMyCredentials(token)
            }
            credentials = result.first
        } catch (_: Exception) {
            errorMessage = L.text(context, localePrefs, R.string.mobile_credentials_loadError)
        } finally {
            loading = false
        }
    }

    Column(
        modifier = modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(16.dp),
    ) {
        when {
            loading -> LmsSkeletonList(count = 3)
            errorMessage != null && credentials.isEmpty() -> LmsEmptyState(
                icon = Icons.Default.Verified,
                title = L.text(context, localePrefs, R.string.mobile_credentials_errorTitle),
                message = errorMessage!!,
            )
            credentials.isEmpty() -> LmsEmptyState(
                icon = Icons.Default.Verified,
                title = L.text(context, localePrefs, R.string.mobile_credentials_emptyTitle),
                message = L.text(context, localePrefs, R.string.mobile_credentials_emptyMessage),
            )
            else -> credentials.forEach { credential ->
                LmsCard(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(bottom = 12.dp)
                        .clickable { onOpenCredential(credential) },
                ) {
                    Text(credential.title, fontWeight = FontWeight.SemiBold, color = textPrimary())
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
                        fontSize = 12.sp,
                        color = textSecondary(),
                    )
                }
            }
        }
    }
}