package com.lextures.android.features.courses

import android.annotation.SuppressLint
import android.net.Uri
import android.view.ViewGroup
import android.webkit.WebResourceRequest
import android.webkit.WebView
import android.webkit.WebViewClient
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.remember
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp
import androidx.compose.ui.viewinterop.AndroidView
import com.lextures.android.core.config.AppConfiguration
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.i18n.L

/** In-app browser for external links and textbook resources (M3.1). */
@SuppressLint("SetJavaScriptEnabled")
@Composable
fun WebItemScreen(
    title: String,
    urlString: String,
    accessToken: String?,
    onOpenExternal: (Uri) -> Unit,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    val resolvedUrl = remember(urlString) {
        when {
            urlString.startsWith("/") -> AppConfiguration.apiUrl(urlString).toString()
            else -> urlString
        }
    }

    Column(modifier = modifier.fillMaxSize()) {
        AndroidView(
            modifier = Modifier.weight(1f),
            factory = {
                WebView(context).apply {
                    layoutParams = ViewGroup.LayoutParams(
                        ViewGroup.LayoutParams.MATCH_PARENT,
                        ViewGroup.LayoutParams.MATCH_PARENT,
                    )
                    settings.javaScriptEnabled = true
                    webViewClient = object : WebViewClient() {
                        override fun shouldOverrideUrlLoading(view: WebView?, request: WebResourceRequest?): Boolean {
                            return false
                        }
                    }
                    val headers = buildMap {
                        if (accessToken != null && resolvedUrl.startsWith(AppConfiguration.apiBaseUrl)) {
                            put("Authorization", "Bearer $accessToken")
                        }
                    }
                    loadUrl(resolvedUrl, headers)
                }
            },
            update = { webView ->
                val headers = buildMap {
                    if (accessToken != null && resolvedUrl.startsWith(AppConfiguration.apiBaseUrl)) {
                        put("Authorization", "Bearer $accessToken")
                    }
                }
                if (webView.url != resolvedUrl) {
                    webView.loadUrl(resolvedUrl, headers)
                }
            },
        )
        TextButton(
            onClick = { runCatching { Uri.parse(resolvedUrl) }.getOrNull()?.let(onOpenExternal) },
            modifier = Modifier.fillMaxWidth().padding(8.dp),
        ) {
            Text(L.text("mobile.modules.openExternal"), color = textPrimary())
        }
    }
}
