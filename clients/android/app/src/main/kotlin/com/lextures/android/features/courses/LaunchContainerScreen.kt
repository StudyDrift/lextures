package com.lextures.android.features.courses

import android.annotation.SuppressLint
import android.content.Intent
import android.net.Uri
import android.view.ViewGroup
import android.webkit.JavascriptInterface
import android.webkit.WebResourceRequest
import android.webkit.WebView
import android.webkit.WebViewClient
import androidx.activity.compose.BackHandler
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.compose.ui.viewinterop.AndroidView
import com.lextures.android.core.auth.AuthSession
import com.lextures.android.core.config.AppConfiguration
import com.lextures.android.core.design.LexturesColors
import com.lextures.android.core.design.textPrimary
import com.lextures.android.core.design.textSecondary
import com.lextures.android.core.lms.CourseStructureItem
import com.lextures.android.core.lms.CourseSummary
import com.lextures.android.core.lms.InteractiveLaunchContent
import com.lextures.android.core.lms.InteractiveLaunchKind
import com.lextures.android.core.lms.InteractiveLaunchLogic
import com.lextures.android.core.lms.InteractiveLaunchTarget
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.network.ApiError
import com.lextures.android.core.offline.OfflineService
import com.lextures.android.features.home.LmsEmptyState
import kotlinx.coroutines.launch
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonObject

/** Secured web container for H5P, SCORM, LTI, and vibe activities (M3.3). */
@Composable
fun LaunchContainerScreen(
    session: AuthSession,
    course: CourseSummary,
    item: CourseStructureItem,
    onBack: () -> Unit,
    onProgressChanged: suspend () -> Unit = {},
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    val accessToken by session.accessToken.collectAsState()
    val offline = remember { OfflineService.get(context) }
    val isOnline by offline.networkMonitor.isOnline.collectAsState()
    val scope = rememberCoroutineScope()

    var target by remember { mutableStateOf<InteractiveLaunchTarget?>(null) }
    var loading by remember { mutableStateOf(true) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var loadGeneration by remember { mutableIntStateOf(0) }

    val offlineLabel = moduleInteractiveOfflineLabel()
    val retryLabel = moduleInteractiveRetryLabel()
    val resumeLabel = moduleInteractiveResumeLabel()
    val loadErrorFallback = moduleLoadErrorLabel()
    val preparingFallback = moduleInteractivePreparingLabel()
    val ltiErrorLabel = moduleInteractiveLtiErrorLabel()
    val webLoadErrorLabel = moduleWebLoadErrorLabel()
    val openExternalLabel = moduleOpenExternalLabel()

    BackHandler(onBack = onBack)

    DisposableEffect(Unit) {
        onDispose { scope.launch { onProgressChanged() } }
    }

    LaunchedEffect(accessToken, item.id, loadGeneration, isOnline) {
        val token = accessToken ?: return@LaunchedEffect
        if (!isOnline) {
            loading = false
            target = null
            return@LaunchedEffect
        }
        loading = true
        errorMessage = null
        try {
            target = LaunchContainerLoader.resolveLaunchTarget(course.courseCode, item, token)
        } catch (e: Exception) {
            errorMessage = when (e) {
                is ApiError.HttpStatus -> when (e.code) {
                    503 -> preparingFallback
                    500 if e.apiMessage == "lti_error" -> ltiErrorLabel
                    else -> e.apiMessage ?: loadErrorFallback
                }
                else -> session.mapError(e)
            }
            target = null
        } finally {
            loading = false
        }
    }

    Column(modifier = modifier) {
        RowHeader(title = target?.title ?: item.title, onBack = onBack)

        when {
            !isOnline -> Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                LmsEmptyState(
                    icon = ItemKind.icon(item.kind),
                    title = item.title,
                    message = offlineLabel,
                )
            }
            loading -> Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                CircularProgressIndicator(color = LexturesColors.Primary)
            }
            target != null -> {
                val launchTarget = target!!
                if (launchTarget.hasResume) {
                    Text(
                        text = resumeLabel,
                        fontSize = 12.sp,
                        color = textSecondary(),
                        modifier = Modifier.padding(horizontal = 16.dp, vertical = 8.dp),
                    )
                }
                AuthenticatedLaunchWebView(
                    target = launchTarget,
                    accessToken = accessToken,
                    onWebError = { errorMessage = webLoadErrorLabel },
                    onH5pXapi = { statement ->
                        val packageId = launchTarget.packageId ?: return@AuthenticatedLaunchWebView
                        val token = accessToken ?: return@AuthenticatedLaunchWebView
                        runCatching {
                            LmsApi.postXapiStatement(course.courseCode, packageId, statement, token)
                        }
                        onProgressChanged()
                    },
                    onActivityEvent = { scope.launch { onProgressChanged() } },
                    modifier = Modifier.weight(1f),
                )
                externalUrlFor(launchTarget)?.let { externalUrl ->
                    TextButton(
                        onClick = {
                            context.startActivity(Intent(Intent.ACTION_VIEW, Uri.parse(externalUrl)))
                        },
                        modifier = Modifier.fillMaxWidth().padding(8.dp),
                    ) {
                        Text(openExternalLabel, color = textPrimary())
                    }
                }
            }
            else -> Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                Column(horizontalAlignment = Alignment.CenterHorizontally) {
                    LmsEmptyState(
                        icon = ItemKind.icon(item.kind),
                        title = item.title,
                        message = errorMessage ?: loadErrorFallback,
                    )
                    Button(
                        onClick = {
                            errorMessage = null
                            loading = true
                            loadGeneration += 1
                        },
                        modifier = Modifier.padding(top = 16.dp),
                    ) {
                        Text(retryLabel)
                    }
                }
            }
        }
    }
}

private fun externalUrlFor(target: InteractiveLaunchTarget): String? =
    when (val content = target.content) {
        is InteractiveLaunchContent.WebUrl -> content.url
        is InteractiveLaunchContent.Html -> null
    }

object LaunchContainerLoader {
    suspend fun resolveLaunchTarget(
        courseCode: String,
        item: CourseStructureItem,
        accessToken: String,
    ): InteractiveLaunchTarget {
        val kind = InteractiveLaunchLogic.kindFor(item.kind)
            ?: throw ApiError.HttpStatus(404, "Unsupported item kind")
        return when (kind) {
            InteractiveLaunchKind.H5p -> {
                val payload = LmsApi.fetchModuleH5P(courseCode, item.id, accessToken)
                if (payload.extractStatus != "ready" || payload.packageId.isBlank()) {
                    throw ApiError.HttpStatus(503, "preparing")
                }
                InteractiveLaunchTarget(
                    title = payload.title.ifBlank { item.title },
                    kind = kind,
                    content = InteractiveLaunchContent.WebUrl(
                        InteractiveLaunchLogic.resolveUrl(
                            InteractiveLaunchLogic.h5pRenderPath(courseCode, payload.packageId),
                        ),
                    ),
                    packageId = payload.packageId,
                )
            }
            InteractiveLaunchKind.Scorm -> {
                val payload = LmsApi.fetchModuleScorm(courseCode, item.id, accessToken)
                if (payload.extractStatus != "ready") throw ApiError.HttpStatus(503, "preparing")
                val scoId = payload.scos.firstOrNull()?.id.orEmpty()
                if (scoId.isBlank()) throw ApiError.HttpStatus(404, "missing sco")
                val launch = LmsApi.launchScorm(courseCode, scoId, accessToken)
                if (launch.renderUrl.isBlank()) throw ApiError.HttpStatus(500, "launch failed")
                InteractiveLaunchTarget(
                    title = payload.title.ifBlank { item.title },
                    kind = kind,
                    content = InteractiveLaunchContent.WebUrl(
                        InteractiveLaunchLogic.resolveUrl(launch.renderUrl),
                    ),
                    hasResume = InteractiveLaunchLogic.scormHasResume(launch.initialCmi),
                )
            }
            InteractiveLaunchKind.LtiLink -> {
                val meta = LmsApi.fetchModuleLtiLink(courseCode, item.id, accessToken)
                val ticket = LmsApi.postLtiEmbedTicket(courseCode, item.id, accessToken)
                if (ticket.ticket.isBlank()) throw ApiError.HttpStatus(500, "lti_error")
                InteractiveLaunchTarget(
                    title = meta.title.ifBlank { item.title },
                    kind = kind,
                    content = InteractiveLaunchContent.WebUrl(
                        InteractiveLaunchLogic.resolveUrl(
                            InteractiveLaunchLogic.ltiFramePath(ticket.ticket),
                        ),
                    ),
                )
            }
            InteractiveLaunchKind.VibeActivity -> {
                val payload = LmsApi.fetchModuleVibeActivity(courseCode, item.id, accessToken)
                InteractiveLaunchTarget(
                    title = payload.title.ifBlank { item.title },
                    kind = kind,
                    content = InteractiveLaunchContent.Html(
                        InteractiveLaunchLogic.vibeActivityHtml(payload.html),
                    ),
                )
            }
        }
    }
}

@SuppressLint("SetJavaScriptEnabled")
@Composable
private fun AuthenticatedLaunchWebView(
    target: InteractiveLaunchTarget,
    accessToken: String?,
    onWebError: () -> Unit,
    onH5pXapi: suspend (JsonElement) -> Unit,
    onActivityEvent: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    val json = remember { Json { ignoreUnknownKeys = true } }
    val apiBase = remember { AppConfiguration.apiBaseUrl }
    val authScript = remember(accessToken, apiBase) {
        accessToken?.let { InteractiveLaunchLogic.authInjectionScript(it, apiBase) }
    }

    AndroidView(
        modifier = modifier.fillMaxSize(),
        factory = {
            WebView(context).apply {
                layoutParams = ViewGroup.LayoutParams(
                    ViewGroup.LayoutParams.MATCH_PARENT,
                    ViewGroup.LayoutParams.MATCH_PARENT,
                )
                settings.javaScriptEnabled = true
                settings.domStorageEnabled = true
                settings.mediaPlaybackRequiresUserGesture = false
                settings.allowFileAccess = false

                addJavascriptInterface(
                    object {
                        @JavascriptInterface
                        fun postH5pXapi(raw: String) {
                            runCatching {
                                val element = json.parseToJsonElement(raw)
                                val statement = (element as? JsonObject)?.get("statement") ?: return
                                scope.launch {
                                    onH5pXapi(statement)
                                    onActivityEvent()
                                }
                            }
                        }
                    },
                    "LexturesInteractiveBridge",
                )

                webViewClient = object : WebViewClient() {
                    override fun shouldOverrideUrlLoading(view: WebView?, request: WebResourceRequest?): Boolean = false

                    override fun onPageStarted(view: WebView?, url: String?, favicon: android.graphics.Bitmap?) {
                        authScript?.let { script -> view?.evaluateJavascript(script, null) }
                    }

                    override fun onReceivedError(
                        view: WebView?,
                        request: WebResourceRequest?,
                        error: android.webkit.WebResourceError?,
                    ) {
                        if (request?.isForMainFrame == true) onWebError()
                    }
                }
            }
        },
        update = { webView ->
            val token = accessToken
            if (token.isNullOrBlank()) {
                onWebError()
                return@AndroidView
            }

            when (val content = target.content) {
                is InteractiveLaunchContent.WebUrl -> {
                    val headers = buildMap {
                        if (content.url.startsWith(AppConfiguration.apiBaseUrl)) {
                            put("Authorization", "Bearer $token")
                        }
                    }
                    if (webView.url != content.url) {
                        webView.loadUrl(content.url, headers)
                    }
                }
                is InteractiveLaunchContent.Html -> {
                    webView.loadDataWithBaseURL(
                        AppConfiguration.apiBaseUrl,
                        content.html,
                        "text/html",
                        "UTF-8",
                        null,
                    )
                }
            }
        },
    )
}
