package com.lextures.android.features.introcourse

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
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
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import com.lextures.android.R
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.i18n.L
import com.lextures.android.core.i18n.LocalLocalePreferences
import com.lextures.android.core.lms.IntroCourseLogic
import com.lextures.android.core.lms.IntroCourseObservability
import com.lextures.android.core.lms.IntroCourseProgress
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.offline.OfflineCacheKey
import com.lextures.android.core.offline.OfflineService
import com.lextures.android.core.routing.DeepLinkDestination
import com.lextures.android.features.home.HomeShellState
import kotlinx.coroutines.launch
import kotlinx.serialization.serializer

@Composable
fun IntroCelebrationWatcher(
    session: AuthSession,
    shell: HomeShellState,
) {
    if (!IntroCourseLogic.introCourseEnabled(shell.platformFeatures)) return

    val context = LocalContext.current
    val accessToken by session.accessToken.collectAsState()
    val offline = remember { OfflineService.get(context) }
    val scope = rememberCoroutineScope()
    var progress by remember { mutableStateOf<IntroCourseProgress?>(null) }
    var visible by remember { mutableStateOf(false) }

    LaunchedEffect(accessToken) {
        val token = accessToken ?: return@LaunchedEffect
        runCatching {
            val (value, _) = offline.cachedFetch(
                key = OfflineCacheKey.introCourseProgress(),
                accessToken = token,
                serializer = serializer<IntroCourseProgress>(),
            ) { LmsApi.fetchIntroCourseProgress(token) }
            if (IntroCourseLogic.shouldShowCelebration(value)) {
                progress = value
                visible = true
            }
        }
    }

    val current = progress
    if (visible && current != null) {
        IntroCompletionCelebrationDialog(
            progress = current,
            credentialAvailable = shell.platformFeatures.ffCompletionCredentials && current.credentialId != null,
            onDismiss = { openCredentials ->
                visible = false
                scope.launch {
                    accessToken?.let { token ->
                        runCatching { LmsApi.markIntroCelebrationSeen(token) }
                        runCatching {
                            offline.cachedFetch(
                                key = OfflineCacheKey.introCourseProgress(),
                                accessToken = token,
                                serializer = serializer<IntroCourseProgress>(),
                            ) { LmsApi.fetchIntroCourseProgress(token) }
                        }
                    }
                    if (openCredentials) {
                        shell.openDeepLink(DeepLinkDestination.Credentials)
                    }
                }
            },
        )
    }
}

@Composable
private fun IntroCompletionCelebrationDialog(
    progress: IntroCourseProgress,
    credentialAvailable: Boolean,
    onDismiss: (openCredentials: Boolean) -> Unit,
) {
    val context = LocalContext.current
    val localePrefs = LocalLocalePreferences.current
    LaunchedEffect(Unit) { IntroCourseObservability.recordCelebrationView(context) }

    AlertDialog(
        onDismissRequest = { onDismiss(false) },
        title = {
            Text(
                L.text(context, localePrefs, R.string.mobile_introCourse_celebration_title),
                textAlign = TextAlign.Center,
            )
        },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
                Text(
                    L.text(
                        context,
                        localePrefs,
                        if (credentialAvailable) {
                            R.string.mobile_introCourse_celebration_bodyWithCredential
                        } else {
                            R.string.mobile_introCourse_celebration_body
                        },
                    ),
                )
                if (credentialAvailable) {
                    Text(
                        L.text(context, localePrefs, R.string.mobile_introCourse_celebration_badgeLabel),
                        fontWeight = FontWeight.SemiBold,
                    )
                }
                TextButton(onClick = { onDismiss(true) }, modifier = Modifier.fillMaxWidth()) {
                    Text(L.text(context, localePrefs, R.string.mobile_introCourse_celebration_credentialsLink))
                }
            }
        },
        confirmButton = {
            Button(onClick = { onDismiss(false) }) {
                Text(L.text(context, localePrefs, R.string.mobile_introCourse_celebration_close))
            }
        },
    )
}