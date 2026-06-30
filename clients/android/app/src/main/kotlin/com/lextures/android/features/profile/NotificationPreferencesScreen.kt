package com.lextures.android.features.profile

import androidx.activity.compose.BackHandler
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.Divider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
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
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.design.LexturesType
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.lms.NotificationLogic
import com.lextures.android.core.lms.NotificationPreference
import com.lextures.android.core.lms.NotificationPreferencePatch
import com.lextures.android.core.lms.NotificationPreferencesCache
import com.lextures.android.core.lms.NotificationPreferencesUpdate
import com.lextures.android.core.notebook.NotebookStore
import com.lextures.android.core.offline.OfflineCacheKey
import com.lextures.android.core.offline.OfflineService
import com.lextures.android.features.home.LmsCard
import com.lextures.android.features.home.LmsErrorBanner
import com.lextures.android.features.home.LmsSkeletonList
import kotlinx.coroutines.launch
import kotlinx.serialization.builtins.ListSerializer
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json

/** Per-category notification preferences (push + email) persisted via the preferences API. */
@Composable
fun NotificationPreferencesScreen(
    session: AuthSession,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val accessToken by session.accessToken.collectAsState()
    val context = LocalContext.current
    val offline = remember { OfflineService.get(context) }
    val scope = rememberCoroutineScope()
    val json = remember { Json { ignoreUnknownKeys = true } }

    var preferences by remember { mutableStateOf<List<NotificationPreference>>(emptyList()) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var saveMessage by remember { mutableStateOf<String?>(null) }
    var loading by remember { mutableStateOf(true) }

    val preferencesTitle = notificationsPreferencesTitle()
    val preferencesDescription = notificationsPreferencesDescription()
    val preferencesSavedMessage = notificationsPreferencesSavedMessage()
    val preferencesPushLabel = notificationsPreferencesPushLabel()
    val preferencesEmailLabel = notificationsPreferencesEmailLabel()
    val preferencesSaveMutationLabel = notificationsPreferencesSaveMutationLabel()

    BackHandler(onBack = onBack)

    LaunchedEffect(accessToken) {
        val token = accessToken ?: return@LaunchedEffect
        loading = true
        errorMessage = null
        val ownerKey = NotebookStore.jwtSubject(token) ?: "anonymous"
        try {
            val (rows, _) = offline.cachedFetch(
                key = OfflineCacheKey.notificationPreferences(),
                accessToken = token,
                serializer = ListSerializer(NotificationPreference.serializer()),
            ) {
                LmsApi.fetchNotificationPreferences(token)
            }
            preferences = rows
            NotificationPreferencesCache.save(context, ownerKey, rows)
        } catch (e: Exception) {
            val cached = NotificationPreferencesCache.load(context, ownerKey)
            if (cached.isNotEmpty()) {
                preferences = cached
            } else {
                errorMessage = session.mapError(e)
            }
        } finally {
            loading = false
        }
    }

    Column(modifier = modifier.fillMaxSize()) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(top = 8.dp, end = 16.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            IconButton(onClick = onBack) {
                Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back", tint = textPrimary())
            }
            Text(
                text = preferencesTitle,
                fontSize = 18.sp,
                fontWeight = FontWeight.SemiBold,
                color = textPrimary(),
            )
        }

        LazyColumn(
            modifier = Modifier.fillMaxSize(),
            contentPadding = PaddingValues(16.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            item {
                Text(
                    text = preferencesDescription,
                    fontSize = 12.sp,
                    color = textSecondary(),
                )
            }

            errorMessage?.let { message ->
                item { LmsErrorBanner(message) }
            }

            saveMessage?.let { message ->
                item {
                    Text(text = message, fontSize = 12.sp, color = textSecondary())
                }
            }

            if (loading && preferences.isEmpty()) {
                item { LmsSkeletonList(count = 4) }
            } else {
                NotificationLogic.groupedPreferences(preferences).forEach { (category, rows) ->
                    item {
                        LmsCard {
                            Text(
                                text = categoryLabel(category),
                                style = LexturesType.display(17),
                                color = textPrimary(),
                            )
                            rows.forEachIndexed { index, row ->
                                if (index > 0) {
                                    Divider(modifier = Modifier.padding(vertical = 8.dp))
                                }
                                PreferenceRow(
                                    row = row,
                                    pushLabel = preferencesPushLabel,
                                    emailLabel = preferencesEmailLabel,
                                    onPushChange = { enabled ->
                                        preferences = preferences.map {
                                            if (it.eventType == row.eventType) it.copy(pushEnabled = enabled) else it
                                        }
                                        persistPreference(
                                            session = session,
                                            offline = offline,
                                            context = context,
                                            json = json,
                                            scope = scope,
                                            preferences = preferences,
                                            eventType = row.eventType,
                                            saveMutationLabel = preferencesSaveMutationLabel,
                                            onError = { errorMessage = it },
                                            onSaved = { saveMessage = preferencesSavedMessage },
                                        )
                                    },
                                    onEmailChange = { enabled ->
                                        preferences = preferences.map {
                                            if (it.eventType == row.eventType) it.copy(emailEnabled = enabled) else it
                                        }
                                        persistPreference(
                                            session = session,
                                            offline = offline,
                                            context = context,
                                            json = json,
                                            scope = scope,
                                            preferences = preferences,
                                            eventType = row.eventType,
                                            saveMutationLabel = preferencesSaveMutationLabel,
                                            onError = { errorMessage = it },
                                            onSaved = { saveMessage = preferencesSavedMessage },
                                        )
                                    },
                                )
                            }
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun PreferenceRow(
    row: NotificationPreference,
    pushLabel: String,
    emailLabel: String,
    onPushChange: (Boolean) -> Unit,
    onEmailChange: (Boolean) -> Unit,
) {
    val eventLabel = eventTypeLabel(row.eventType)
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .padding(vertical = 4.dp),
        verticalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        Text(text = eventLabel, fontSize = 14.sp, fontWeight = FontWeight.SemiBold, color = textPrimary())
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(text = pushLabel, fontSize = 14.sp, color = textSecondary())
            Switch(
                checked = row.pushEnabled,
                onCheckedChange = onPushChange,
                modifier = Modifier.semantics {
                    contentDescription = "$eventLabel, $pushLabel"
                },
            )
        }
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(text = emailLabel, fontSize = 14.sp, color = textSecondary())
            Switch(
                checked = row.emailEnabled,
                onCheckedChange = onEmailChange,
                modifier = Modifier.semantics {
                    contentDescription = "$eventLabel, $emailLabel"
                },
            )
        }
    }
}

private fun persistPreference(
    session: AuthSession,
    offline: OfflineService,
    context: android.content.Context,
    json: Json,
    scope: kotlinx.coroutines.CoroutineScope,
    preferences: List<NotificationPreference>,
    eventType: String,
    saveMutationLabel: String,
    onError: (String) -> Unit,
    onSaved: () -> Unit,
) {
    scope.launch {
        val token = session.accessToken.value ?: return@launch
        val row = preferences.firstOrNull { it.eventType == eventType } ?: return@launch
        val ownerKey = NotebookStore.jwtSubject(token) ?: "anonymous"
        val body = json.encodeToString(
            NotificationPreferencesUpdate.serializer(),
            NotificationPreferencesUpdate(
                preferences = listOf(
                    NotificationPreferencePatch(
                        eventType = row.eventType,
                        emailEnabled = row.emailEnabled,
                        pushEnabled = row.pushEnabled,
                    ),
                ),
            ),
        )
        runCatching {
            offline.enqueueMutation(
                method = "PUT",
                path = "/api/v1/me/notification-preferences",
                bodyJson = body,
                label = saveMutationLabel,
                accessToken = token,
            )
            NotificationPreferencesCache.save(context, ownerKey, preferences)
            onSaved()
        }.onFailure {
            onError(session.mapError(it))
        }
    }
}
