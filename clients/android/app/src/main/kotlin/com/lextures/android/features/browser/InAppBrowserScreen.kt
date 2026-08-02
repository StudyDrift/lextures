package com.lextures.android.features.browser

import android.annotation.SuppressLint
import android.content.ClipData
import android.content.ClipboardManager
import android.content.Context
import android.content.Intent
import android.graphics.Bitmap
import android.net.Uri
import android.view.ViewGroup
import android.webkit.WebChromeClient
import android.webkit.WebResourceError
import android.webkit.WebResourceRequest
import android.webkit.WebView
import android.webkit.WebViewClient
import androidx.activity.compose.BackHandler
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.gestures.detectVerticalDragGestures
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Close
import androidx.compose.material.icons.filled.Lock
import androidx.compose.material.icons.filled.MoreVert
import androidx.compose.material.icons.filled.Warning
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableFloatStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.IntOffset
import androidx.compose.ui.unit.dp
import androidx.compose.ui.viewinterop.AndroidView
import com.lextures.android.R
import com.lextures.android.core.config.AppConfiguration
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.lms.MobileLinkPolicy
import com.lextures.android.core.routing.InAppBrowserTelemetry
import com.lextures.android.core.routing.LinkOpener
import kotlin.math.roundToInt

private enum class LoadKind { Error, Offline, Blocked }

@SuppressLint("SetJavaScriptEnabled")
@Composable
fun InAppBrowserScreen(
    session: LinkOpener.BrowserSession,
    onDismiss: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    var currentUrl by remember { mutableStateOf(session.initialUrl) }
    var pageTitle by remember { mutableStateOf("") }
    var progress by remember { mutableFloatStateOf(0f) }
    var canGoBack by remember { mutableStateOf(false) }
    var loadError by remember { mutableStateOf<LoadKind?>(null) }
    var showMenu by remember { mutableStateOf(false) }
    var showCopied by remember { mutableStateOf(false) }
    var dragOffset by remember { mutableFloatStateOf(0f) }
    var webViewRef by remember { mutableStateOf<WebView?>(null) }

    val host = MobileLinkPolicy.displayHost(currentUrl)
    val secure = MobileLinkPolicy.isSecure(currentUrl)
    val copyLabel = L.text(R.string.mobile_browser_copyLink)

    BackHandler {
        val wv = webViewRef
        if (wv != null && wv.canGoBack()) {
            wv.goBack()
        } else {
            recordDismiss(session.source, "back")
            onDismiss()
        }
    }

    Column(
        modifier = modifier
            .fillMaxSize()
            .offset { IntOffset(0, dragOffset.roundToInt()) }
            .background(MaterialTheme.colorScheme.background)
            .semantics { contentDescription = "$copyLabel, $host" },
    ) {
        BrowserChromeHeader(
            host = host,
            title = pageTitle,
            isSecure = secure,
            progress = progress,
            canGoBack = canGoBack,
            onClose = {
                recordDismiss(session.source, "close")
                onDismiss()
            },
            onBack = { webViewRef?.goBack() },
            onCopy = {
                copyLink(context, currentUrl)
                showCopied = true
                InAppBrowserTelemetry.record(
                    MobileLinkPolicy.sanitizeTelemetry(
                        mapOf(
                            "source" to session.source,
                            "classification" to "external",
                            "outcome" to "copy",
                        ),
                    ),
                )
            },
            onOverflow = { showMenu = true },
            modifier = Modifier.pointerInput(Unit) {
                detectVerticalDragGestures(
                    onVerticalDrag = { _, dragAmount ->
                        if (dragAmount > 0) dragOffset += dragAmount
                    },
                    onDragEnd = {
                        if (dragOffset > 240f) {
                            recordDismiss(session.source, "drag")
                            onDismiss()
                        } else {
                            dragOffset = 0f
                        }
                    },
                )
            },
        )

        Box(modifier = Modifier.weight(1f).fillMaxWidth()) {
            AndroidView(
                modifier = Modifier.fillMaxSize(),
                factory = { ctx ->
                    createBrowserWebView(
                        ctx = ctx,
                        session = session,
                        onProgress = { progress = it },
                        onTitle = { pageTitle = it },
                        onUrl = { currentUrl = it },
                        onCanGoBack = { canGoBack = it },
                        onError = { loadError = it },
                        onReady = { webViewRef = it },
                    )
                },
                update = { /* fixed session */ },
            )

            val err = loadError
            if (err != null) {
                BrowserErrorState(
                    kind = err,
                    onRetry = { webViewRef?.reload(); loadError = null },
                    onOpenExternal = if (err != LoadKind.Blocked) {
                        { openExternal(context, currentUrl, session.source) }
                    } else {
                        null
                    },
                )
            }

            if (showCopied) {
                Text(
                    text = L.text(R.string.mobile_browser_linkCopied),
                    modifier = Modifier
                        .align(Alignment.BottomCenter)
                        .padding(bottom = 32.dp)
                        .clip(RoundedCornerShape(24.dp))
                        .background(MaterialTheme.colorScheme.surfaceVariant)
                        .padding(horizontal = 16.dp, vertical = 10.dp),
                    color = textPrimary(),
                    style = MaterialTheme.typography.labelLarge,
                )
            }

            Box(modifier = Modifier.align(Alignment.TopEnd)) {
                DropdownMenu(expanded = showMenu, onDismissRequest = { showMenu = false }) {
                    DropdownMenuItem(
                        text = { Text(L.text(R.string.mobile_browser_copyLink)) },
                        onClick = {
                            showMenu = false
                            copyLink(context, currentUrl)
                            showCopied = true
                        },
                    )
                    DropdownMenuItem(
                        text = { Text(L.text(R.string.mobile_browser_share)) },
                        onClick = {
                            showMenu = false
                            val send = Intent(Intent.ACTION_SEND).apply {
                                type = "text/plain"
                                putExtra(Intent.EXTRA_TEXT, currentUrl)
                                putExtra(Intent.EXTRA_TITLE, pageTitle)
                            }
                            context.startActivity(Intent.createChooser(send, null))
                            InAppBrowserTelemetry.record(
                                MobileLinkPolicy.sanitizeTelemetry(
                                    mapOf(
                                        "source" to session.source,
                                        "classification" to "external",
                                        "outcome" to "share",
                                    ),
                                ),
                            )
                        },
                    )
                    DropdownMenuItem(
                        text = { Text(L.text(R.string.mobile_browser_openInBrowser)) },
                        onClick = {
                            showMenu = false
                            openExternal(context, currentUrl, session.source)
                        },
                    )
                    DropdownMenuItem(
                        text = { Text(L.text(R.string.mobile_browser_reload)) },
                        onClick = {
                            showMenu = false
                            webViewRef?.reload()
                        },
                    )
                    if (session.allowReport) {
                        DropdownMenuItem(
                            text = { Text(L.text(R.string.mobile_browser_report)) },
                            onClick = {
                                showMenu = false
                                InAppBrowserTelemetry.record(
                                    MobileLinkPolicy.sanitizeTelemetry(
                                        mapOf(
                                            "source" to session.source,
                                            "classification" to "external",
                                            "outcome" to "report",
                                        ),
                                    ),
                                )
                            },
                        )
                    }
                }
            }
        }
    }

    DisposableEffect(Unit) {
        onDispose {
            webViewRef?.apply {
                stopLoading()
                destroy()
            }
            webViewRef = null
        }
    }
}

@SuppressLint("SetJavaScriptEnabled")
private fun createBrowserWebView(
    ctx: Context,
    session: LinkOpener.BrowserSession,
    onProgress: (Float) -> Unit,
    onTitle: (String) -> Unit,
    onUrl: (String) -> Unit,
    onCanGoBack: (Boolean) -> Unit,
    onError: (LoadKind?) -> Unit,
    onReady: (WebView) -> Unit,
): WebView {
    return WebView(ctx).apply {
        layoutParams = ViewGroup.LayoutParams(
            ViewGroup.LayoutParams.MATCH_PARENT,
            ViewGroup.LayoutParams.MATCH_PARENT,
        )
        settings.javaScriptEnabled = true
        settings.domStorageEnabled = true
        settings.mediaPlaybackRequiresUserGesture = false
        webChromeClient = object : WebChromeClient() {
            override fun onProgressChanged(view: WebView?, newProgress: Int) {
                onProgress(if (newProgress >= 100) 0f else newProgress / 100f)
            }

            override fun onReceivedTitle(view: WebView?, title: String?) {
                onTitle(title.orEmpty())
            }
        }
        webViewClient = object : WebViewClient() {
            override fun shouldOverrideUrlLoading(
                view: WebView?,
                request: WebResourceRequest?,
            ): Boolean {
                val uri = request?.url ?: return false
                val scheme = uri.scheme?.lowercase().orEmpty()
                if (scheme == "http" || scheme == "https") return false
                if (scheme in setOf("mailto", "tel", "sms", "market")) {
                    runCatching { ctx.startActivity(Intent(Intent.ACTION_VIEW, uri)) }
                    return true
                }
                return true
            }

            override fun onPageStarted(view: WebView?, url: String?, favicon: Bitmap?) {
                onError(null)
                if (url != null) onUrl(url)
                onCanGoBack(view?.canGoBack() == true)
            }

            override fun onPageFinished(view: WebView?, url: String?) {
                if (url != null) onUrl(url)
                onCanGoBack(view?.canGoBack() == true)
                onProgress(0f)
            }

            override fun onReceivedError(
                view: WebView?,
                request: WebResourceRequest?,
                error: WebResourceError?,
            ) {
                if (request?.isForMainFrame != true) return
                val code = error?.errorCode
                onError(
                    if (code == ERROR_HOST_LOOKUP ||
                        code == ERROR_CONNECT ||
                        code == ERROR_TIMEOUT ||
                        code == ERROR_IO
                    ) {
                        LoadKind.Offline
                    } else {
                        LoadKind.Error
                    },
                )
            }
        }
        val headers = buildMap {
            val host = Uri.parse(session.initialUrl).host
            if (session.accessToken != null &&
                MobileLinkPolicy.shouldAttachBearer(
                    host,
                    Uri.parse(AppConfiguration.apiBaseUrl).host.orEmpty(),
                )
            ) {
                put("Authorization", "Bearer ${session.accessToken}")
            }
        }
        loadUrl(session.initialUrl, headers)
        onReady(this)
    }
}

@Composable
private fun BrowserChromeHeader(
    host: String,
    title: String,
    isSecure: Boolean,
    progress: Float,
    canGoBack: Boolean,
    onClose: () -> Unit,
    onBack: () -> Unit,
    onCopy: () -> Unit,
    onOverflow: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val backCd = L.text(R.string.mobile_browser_back)
    val closeCd = L.text(R.string.mobile_browser_close)
    val copyCd = L.text(R.string.mobile_browser_copyLink)
    val overflowCd = L.text(R.string.mobile_browser_overflowTitle)
    val notSecure = L.text(R.string.mobile_browser_notSecure)
    Column(modifier.fillMaxWidth().background(MaterialTheme.colorScheme.surface)) {
        Row(
            Modifier
                .fillMaxWidth()
                .padding(horizontal = 4.dp, vertical = 4.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            if (canGoBack) {
                IconButton(onClick = onBack) {
                    Icon(
                        Icons.AutoMirrored.Filled.ArrowBack,
                        contentDescription = backCd,
                    )
                }
            }
            IconButton(onClick = onClose) {
                Icon(Icons.Default.Close, contentDescription = closeCd)
            }
            Row(
                modifier = Modifier
                    .weight(1f)
                    .clip(RoundedCornerShape(20.dp))
                    .background(MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.6f))
                    .clickable(onClick = onCopy)
                    .padding(horizontal = 10.dp, vertical = 6.dp)
                    .semantics {
                        contentDescription = "$copyCd, $host"
                    },
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Icon(
                    imageVector = if (isSecure) Icons.Default.Lock else Icons.Default.Warning,
                    contentDescription = null,
                    modifier = Modifier.size(14.dp),
                    tint = if (isSecure) textSecondary() else MaterialTheme.colorScheme.error,
                )
                Spacer(Modifier.width(4.dp))
                Text(
                    text = if (isSecure) host else notSecure,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                    style = MaterialTheme.typography.labelLarge,
                    color = textPrimary(),
                )
            }
            IconButton(onClick = onOverflow) {
                Icon(Icons.Default.MoreVert, contentDescription = overflowCd)
            }
        }
        if (title.isNotBlank()) {
            Text(
                text = title,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
                style = MaterialTheme.typography.bodySmall,
                color = textSecondary(),
                modifier = Modifier.padding(horizontal = 16.dp, vertical = 2.dp),
            )
        }
        if (progress > 0f && progress < 1f) {
            LinearProgressIndicator(
                progress = { progress },
                modifier = Modifier.fillMaxWidth().height(2.dp),
            )
        } else {
            Spacer(Modifier.height(2.dp))
        }
    }
}

@Composable
private fun BrowserErrorState(
    kind: LoadKind,
    onRetry: (() -> Unit)?,
    onOpenExternal: (() -> Unit)?,
) {
    Column(
        Modifier
            .fillMaxSize()
            .background(MaterialTheme.colorScheme.background)
            .padding(24.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Spacer(Modifier.weight(1f))
        Text(
            text = when (kind) {
                LoadKind.Error -> L.text(R.string.mobile_browser_errorTitle)
                LoadKind.Offline -> L.text(R.string.mobile_browser_offlineTitle)
                LoadKind.Blocked -> L.text(R.string.mobile_browser_blockedByPolicy)
            },
            style = MaterialTheme.typography.titleMedium,
            color = textPrimary(),
        )
        Spacer(Modifier.height(8.dp))
        Text(
            text = when (kind) {
                LoadKind.Error -> L.text(R.string.mobile_browser_errorBody)
                LoadKind.Offline -> L.text(R.string.mobile_browser_offlineBody)
                LoadKind.Blocked -> L.text(R.string.mobile_browser_blockedByPolicy)
            },
            style = MaterialTheme.typography.bodyMedium,
            color = textSecondary(),
        )
        Spacer(Modifier.height(16.dp))
        if (onRetry != null) {
            TextButton(onClick = onRetry) {
                Text(L.text(R.string.mobile_browser_retry))
            }
        }
        if (onOpenExternal != null) {
            TextButton(onClick = onOpenExternal) {
                Text(L.text(R.string.mobile_browser_openInBrowser))
            }
        }
        Spacer(Modifier.weight(1f))
    }
}

private fun copyLink(context: Context, url: String) {
    val cm = context.getSystemService(Context.CLIPBOARD_SERVICE) as ClipboardManager
    cm.setPrimaryClip(ClipData.newPlainText("link", url))
}

private fun openExternal(context: Context, url: String, source: String) {
    runCatching {
        context.startActivity(Intent(Intent.ACTION_VIEW, Uri.parse(url)))
    }
    InAppBrowserTelemetry.record(
        MobileLinkPolicy.sanitizeTelemetry(
            mapOf(
                "source" to source,
                "classification" to "external",
                "outcome" to "escape_system",
            ),
        ),
    )
}

private fun recordDismiss(source: String, method: String) {
    InAppBrowserTelemetry.record(
        MobileLinkPolicy.sanitizeTelemetry(
            mapOf(
                "source" to source,
                "classification" to "external",
                "outcome" to "dismiss_$method",
            ),
        ),
    )
}

/** Purge WebView cookies / storage (FR-23). Call on sign-out and Settings clear. */
object InAppBrowserDataStore {
    fun purgeAll(context: Context) {
        runCatching {
            val cookieManager = android.webkit.CookieManager.getInstance()
            cookieManager.removeAllCookies(null)
            cookieManager.flush()
            WebView(context.applicationContext).apply {
                clearCache(true)
                clearFormData()
                clearHistory()
                destroy()
            }
            context.applicationContext.deleteDatabase("webview.db")
            context.applicationContext.deleteDatabase("webviewCache.db")
        }
    }
}
