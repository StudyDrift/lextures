package com.lextures.android.features.contenttools.sandbox

import android.annotation.SuppressLint
import android.net.Uri
import android.view.ViewGroup
import android.webkit.PermissionRequest
import android.webkit.WebChromeClient
import android.webkit.WebResourceRequest
import android.webkit.WebView
import android.webkit.WebViewClient
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalConfiguration
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.compose.ui.viewinterop.AndroidView
import com.lextures.android.R
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.i18n.L
import com.lextures.android.core.lms.ContentToolSandboxLogic
import com.lextures.android.core.network.ApiClient
import com.lextures.android.features.contenttools.ToolErrorCard
import com.lextures.android.features.contenttools.ToolPlaceholder
import com.lextures.android.features.contenttools.ToolPlaceholderReason
import java.util.Locale
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonObject

/**
 * CT.M4 sandboxed tool host: native document fetch → opaque-origin WebView → bridge.
 */
@SuppressLint("SetJavaScriptEnabled")
@Composable
fun SandboxWebViewHost(
    toolId: String,
    instanceId: String,
    toolVersion: String,
    title: String,
    config: JsonElement,
    state: JsonElement,
    revision: Long,
    readOnly: Boolean,
    capabilities: List<String>,
    accessToken: String?,
    save: (JsonObject) -> Unit,
    runAction: suspend (name: String, input: JsonObject) -> JsonElement?,
    announce: (String, Boolean) -> Unit,
    onOpenUrl: (Uri) -> Unit = {},
    modifier: Modifier = Modifier,
) {
    val scope = rememberCoroutineScope()
    val configuration = LocalConfiguration.current
    val localeTag = remember(configuration) {
        configuration.locales[0]?.toLanguageTag() ?: Locale.getDefault().toLanguageTag()
    }
    val dir = if (configuration.layoutDirection == android.util.LayoutDirection.RTL) "rtl" else "ltr"

    var heightDp by remember { mutableStateOf(160.0) }
    var ready by remember(instanceId) { mutableStateOf(false) }
    var timedOut by remember(instanceId) { mutableStateOf(false) }
    var crashed by remember(instanceId) { mutableStateOf(false) }
    var documentHtml by remember(instanceId) { mutableStateOf<String?>(null) }
    var fetchFailed by remember(instanceId) { mutableStateOf(false) }
    var reloadToken by remember(instanceId) { mutableStateOf(0) }
    var bridgeRef by remember { mutableStateOf<SandboxBridge?>(null) }
    var unsupportedBridge by remember { mutableStateOf(false) }

    val badge = L.text(R.string.mobile_contentTools_sandbox_badge)
    val timeoutMsg = L.text(R.string.mobile_contentTools_sandbox_timeout)
    val crashedMsg = L.text(R.string.mobile_contentTools_sandbox_crashed)
    val needsConnection = L.text(R.string.mobile_contentTools_sandbox_needsConnection)
    val a11y = "$title, $badge"

    fun resetAndReload() {
        crashed = false
        timedOut = false
        ready = false
        fetchFailed = false
        documentHtml = null
        unsupportedBridge = false
        bridgeRef?.dispose()
        bridgeRef = null
        reloadToken += 1
    }

    DisposableEffect(instanceId) {
        SandboxWebViewPool.retain(instanceId)
        onDispose {
            bridgeRef?.dispose()
            bridgeRef = null
            SandboxWebViewPool.release(instanceId)
        }
    }

    LaunchedEffect(toolId, instanceId, reloadToken, accessToken) {
        documentHtml = null
        fetchFailed = false
        ready = false
        timedOut = false
        val path = ContentToolSandboxLogic.documentPath(toolId, toolVersion)
        val html = runCatching {
            val (body, code) = ApiClient().request(
                path = path,
                method = "GET",
                accessToken = accessToken,
            )
            if (code in 200..299) body.takeIf { it.isNotBlank() } else null
        }.getOrNull()
        if (html == null) {
            fetchFailed = true
            return@LaunchedEffect
        }
        documentHtml = html
        delay(ContentToolSandboxLogic.READY_TIMEOUT_MS.toLong())
        if (!ready) timedOut = true
    }

    when {
        unsupportedBridge -> {
            ToolPlaceholder(
                reason = ToolPlaceholderReason.OPEN_IN_BROWSER,
                toolName = title,
                message = needsConnection,
                modifier = modifier,
            )
        }
        crashed -> {
            ToolErrorCard(
                toolName = title,
                message = crashedMsg,
                onRetry = { resetAndReload() },
                modifier = modifier,
            )
        }
        timedOut && !ready -> {
            ToolErrorCard(
                toolName = title,
                message = timeoutMsg,
                onRetry = { resetAndReload() },
                modifier = modifier,
            )
        }
        fetchFailed -> {
            ToolPlaceholder(
                reason = ToolPlaceholderReason.UNAVAILABLE,
                message = needsConnection,
                modifier = modifier,
            )
        }
        else -> {
            Column(modifier.fillMaxWidth()) {
                if (!ready) {
                    ToolPlaceholder(reason = ToolPlaceholderReason.LOADING)
                }
                val html = documentHtml
                if (html != null) {
                    AndroidView(
                        modifier = Modifier
                            .fillMaxWidth()
                            .height(if (ready) heightDp.dp else 0.dp)
                            .semantics { contentDescription = a11y },
                        factory = { ctx ->
                            WebView(ctx).apply {
                                layoutParams = ViewGroup.LayoutParams(
                                    ViewGroup.LayoutParams.MATCH_PARENT,
                                    ViewGroup.LayoutParams.WRAP_CONTENT,
                                )
                                settings.javaScriptEnabled = true
                                settings.domStorageEnabled = false
                                settings.allowFileAccess = false
                                settings.allowContentAccess = false
                                // Opaque origin (FR-2): loadDataWithBaseURL(null, …)
                                webViewClient = object : WebViewClient() {
                                    override fun shouldOverrideUrlLoading(
                                        view: WebView?,
                                        request: WebResourceRequest?,
                                    ): Boolean {
                                        val uri = request?.url
                                        if (uri != null) onOpenUrl(uri)
                                        return true // block all navigation (FR-13)
                                    }

                                    override fun onPageFinished(view: WebView?, url: String?) {
                                        val bridge = SandboxBridge.create(
                                            this@apply,
                                            SandboxBridge.Handlers(
                                                onReady = { contract ->
                                                    if (ContentToolSandboxLogic.contractInSupportedRange(contract)) {
                                                        ready = true
                                                    } else {
                                                        crashed = true
                                                    }
                                                },
                                                onSave = { nextState, _ ->
                                                    if (!readOnly) {
                                                        val obj = nextState as? JsonObject
                                                            ?: JsonObject(emptyMap())
                                                        save(obj)
                                                        bridgeRef?.postStateAccepted(revision + 1)
                                                    }
                                                },
                                                onRunAction = { id, action, input ->
                                                    scope.launch {
                                                        try {
                                                            val result = runAction(
                                                                action,
                                                                input as? JsonObject ?: JsonObject(emptyMap()),
                                                            )
                                                            bridgeRef?.postActionResult(id, result)
                                                        } catch (e: Exception) {
                                                            bridgeRef?.postError(
                                                                id,
                                                                "action_failed",
                                                                e.message ?: "action failed",
                                                            )
                                                        }
                                                    }
                                                },
                                                onResize = { h -> heightDp = h },
                                                onAnnounce = announce,
                                            ),
                                        )
                                        if (bridge == null) {
                                            unsupportedBridge = true
                                            return
                                        }
                                        bridgeRef = bridge
                                        bridge.postInit(
                                            instanceId = instanceId,
                                            config = config,
                                            state = state,
                                            revision = revision,
                                            locale = localeTag,
                                            dir = dir,
                                            readOnly = readOnly,
                                            participantId = ContentToolSandboxLogic.opaqueParticipantId(instanceId),
                                        )
                                    }
                                }
                                webChromeClient = object : WebChromeClient() {
                                    override fun onPermissionRequest(request: PermissionRequest?) {
                                        val req = request ?: return
                                        val granted = req.resources.filter { resource ->
                                            when (resource) {
                                                PermissionRequest.RESOURCE_VIDEO_CAPTURE ->
                                                    "camera" in capabilities
                                                PermissionRequest.RESOURCE_AUDIO_CAPTURE ->
                                                    "microphone" in capabilities
                                                else -> false
                                            }
                                        }.toTypedArray()
                                        if (granted.isEmpty()) req.deny() else req.grant(granted)
                                    }

                                    override fun onCreateWindow(
                                        view: WebView?,
                                        isDialog: Boolean,
                                        isUserGesture: Boolean,
                                        resultMsg: android.os.Message?,
                                    ): Boolean = false
                                }
                                loadDataWithBaseURL(null, html, "text/html", "UTF-8", null)
                            }
                        },
                        update = { /* bridge owns messaging */ },
                    )
                }
                Text(text = badge, fontSize = 11.sp, color = textSecondary())
            }
        }
    }
}
