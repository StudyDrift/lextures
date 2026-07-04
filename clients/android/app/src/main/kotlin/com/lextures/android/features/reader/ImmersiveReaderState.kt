package com.lextures.android.features.reader

import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import com.lextures.android.core.lms.ImmersiveReaderCapabilities
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.navigation.MobilePlatformFeatures

class ImmersiveReaderState(
    val capabilities: ImmersiveReaderCapabilities,
    val showPreferences: Boolean,
    val onShowPreferences: () -> Unit,
    val onDismissPreferences: () -> Unit,
    val store: ReadingPreferencesStore,
)

@Composable
fun rememberImmersiveReaderState(accessToken: String?): ImmersiveReaderState {
    var capabilities by remember { mutableStateOf(ImmersiveReaderCapabilities()) }
    var showPreferences by remember { mutableStateOf(false) }
    val store = LocalReadingPreferencesStore.current

    LaunchedEffect(accessToken) {
        val token = accessToken ?: return@LaunchedEffect
        val features = runCatching { LmsApi.fetchPlatformFeatures(token) }.getOrNull()
        capabilities = MobilePlatformFeatures.from(features).immersiveReader
        store.loadFromServer(token, capabilities.preferencesEnabled)
    }

    return remember(capabilities, showPreferences, store) {
        ImmersiveReaderState(
            capabilities = capabilities,
            showPreferences = showPreferences,
            onShowPreferences = { showPreferences = true },
            onDismissPreferences = { showPreferences = false },
            store = store,
        )
    }
}

@Composable
fun ImmersiveReaderPreferencesSheet(state: ImmersiveReaderState, accessToken: String?) {
    ReadingPreferencesSheet(
        visible = state.showPreferences,
        store = state.store,
        accessToken = accessToken,
        onDismiss = state.onDismissPreferences,
    )
}