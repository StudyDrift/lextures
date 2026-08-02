package com.lextures.android.features.courses

import android.net.Uri
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.runtime.Composable
import androidx.compose.runtime.remember
import androidx.compose.ui.Modifier
import com.lextures.android.core.config.AppConfiguration
import com.lextures.android.core.lms.MobileLinkPolicy
import com.lextures.android.core.navigation.MobilePlatformFeatures
import com.lextures.android.core.routing.LinkOpener
import com.lextures.android.features.browser.InAppBrowserScreen

/** In-app browser for external links and textbook resources (MB.1 / M3.1). */
@Composable
fun WebItemScreen(
    title: String,
    urlString: String,
    accessToken: String?,
    onOpenExternal: (Uri) -> Unit,
    modifier: Modifier = Modifier,
    platformFeatures: MobilePlatformFeatures = MobilePlatformFeatures(),
) {
    val resolvedUrl = remember(urlString) {
        when {
            urlString.startsWith("/") -> AppConfiguration.apiUrl(urlString).toString()
            else -> urlString
        }
    }
    val state = LinkOpener.policyState(platformFeatures)
    val classification = MobileLinkPolicy.classify(urlString, state)

    if (classification == MobileLinkPolicy.Classification.BLOCKED) {
        // Policy blocked — no blank web view.
        androidx.compose.material3.Text(
            text = com.lextures.android.core.i18n.L.text(
                com.lextures.android.R.string.mobile_browser_blockedByPolicy,
            ),
            modifier = modifier.fillMaxSize(),
        )
        return
    }

    InAppBrowserScreen(
        session = LinkOpener.BrowserSession(
            initialUrl = resolvedUrl,
            accessToken = accessToken,
            source = "web_item",
            allowReport = false,
        ),
        onDismiss = {
            // Nav-hosted: open external as escape if user closes chrome.
            runCatching { Uri.parse(resolvedUrl) }.getOrNull()?.let(onOpenExternal)
        },
        modifier = modifier.fillMaxSize(),
    )
}
