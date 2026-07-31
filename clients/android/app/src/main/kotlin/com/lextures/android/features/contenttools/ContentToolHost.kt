package com.lextures.android.features.contenttools

import android.net.Uri
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateMapOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.runtime.staticCompositionLocalOf
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.semantics.liveRegion
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.unit.dp
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.ui.semantics.LiveRegionMode
import com.lextures.android.R
import com.lextures.android.core.config.AppConfiguration
import com.lextures.android.core.i18n.L
import com.lextures.android.core.lms.ContentToolHostLogic
import com.lextures.android.core.lms.ContentToolsApi
import com.lextures.android.core.lms.ToolInstance
import com.lextures.android.core.lms.ToolStateEnvelope
import com.lextures.android.core.lms.emptyToolState
import com.lextures.android.core.network.ApiError
import com.lextures.android.core.offline.OfflineCacheKey
import com.lextures.android.core.offline.OfflineService
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.buildJsonObject

data class ContentToolsPageContext(
    val courseCode: String,
    val itemId: String,
    val contentToolsEnabled: Boolean,
    val mobileContentToolsEnabled: Boolean,
    val accessToken: String?,
    val observer: Boolean = false,
    val pastDue: Boolean = false,
    val loading: Boolean = false,
    val instances: Map<String, ToolInstance> = emptyMap(),
    val studentResetAllowed: Boolean = false,
    val announce: (message: String, assertive: Boolean) -> Unit = { _, _ -> },
    val onOpenBrowser: (Uri) -> Unit = {},
)

val LocalContentToolsPage = staticCompositionLocalOf<ContentToolsPageContext?> { null }

@Composable
fun ContentToolsPageProvider(
    courseCode: String,
    itemId: String,
    contentToolsEnabled: Boolean,
    mobileContentToolsEnabled: Boolean,
    accessToken: String?,
    observer: Boolean = false,
    pastDue: Boolean = false,
    onOpenBrowser: (Uri) -> Unit = {},
    content: @Composable () -> Unit,
) {
    val context = LocalContext.current
    val offline = remember { OfflineService.get(context.applicationContext) }
    var loading by remember(courseCode, itemId) { mutableStateOf(false) }
    var instances by remember(courseCode, itemId) { mutableStateOf<Map<String, ToolInstance>>(emptyMap()) }
    var studentResetAllowed by remember { mutableStateOf(false) }
    var liveMessage by remember { mutableStateOf("") }
    var liveAssertive by remember { mutableStateOf(false) }

    val announce: (String, Boolean) -> Unit = { message, assertive ->
        liveMessage = message
        liveAssertive = assertive
    }

    LaunchedEffect(courseCode, itemId, contentToolsEnabled, mobileContentToolsEnabled, accessToken) {
        val shouldFetch = ContentToolHostLogic.shouldFetchInstances(
            mobileContentToolsEnabled = mobileContentToolsEnabled,
            contentToolsEnabled = contentToolsEnabled,
            courseCode = courseCode,
            itemId = itemId,
        )
        if (!shouldFetch || accessToken.isNullOrBlank()) {
            instances = emptyMap()
            loading = false
            return@LaunchedEffect
        }
        loading = true
        try {
            val list = offline.cachedFetch(
                key = OfflineCacheKey.contentToolInstances(courseCode, itemId),
                accessToken = accessToken,
                serializer = com.lextures.android.core.lms.ToolInstancesListResponse.serializer(),
            ) {
                com.lextures.android.core.lms.ToolInstancesListResponse(
                    ContentToolsApi.fetchInstances(
                        courseCode = courseCode,
                        accessToken = accessToken,
                        itemId = itemId,
                        withState = true,
                    ),
                )
            }.first.instances
            instances = ContentToolHostLogic.instanceMap(list)
            studentResetAllowed = runCatching {
                ContentToolsApi.fetchSettings(courseCode, accessToken).studentResetAllowed
            }.getOrDefault(false)
        } catch (_: Exception) {
            instances = emptyMap()
        } finally {
            loading = false
        }
    }

    CompositionLocalProvider(
        LocalContentToolsPage provides ContentToolsPageContext(
            courseCode = courseCode,
            itemId = itemId,
            contentToolsEnabled = contentToolsEnabled,
            mobileContentToolsEnabled = mobileContentToolsEnabled,
            accessToken = accessToken,
            observer = observer,
            pastDue = pastDue,
            loading = loading,
            instances = instances,
            studentResetAllowed = studentResetAllowed,
            announce = announce,
            onOpenBrowser = onOpenBrowser,
        ),
    ) {
        content()
        // Shared polite/assertive live region for the screen (FR-15).
        Text(
            text = liveMessage,
            modifier = Modifier
                .semantics {
                    liveRegion = if (liveAssertive) LiveRegionMode.Assertive else LiveRegionMode.Polite
                }
                .height(0.dp),
        )
    }
}

@Composable
fun ContentToolHost(
    instanceId: String,
    toolId: String,
    modifier: Modifier = Modifier,
) {
    val page = LocalContentToolsPage.current
    if (page == null) {
        // No page context — keep CT.M1 placeholder.
        com.lextures.android.features.courses.markdown.MarkdownToolPlaceholder(toolId = toolId, modifier = modifier)
        return
    }

    when (
        ContentToolHostLogic.fenceRenderMode(
            mobileContentToolsEnabled = page.mobileContentToolsEnabled,
            contentToolsEnabled = page.contentToolsEnabled,
        )
    ) {
        ContentToolHostLogic.FenceRenderMode.LEGACY_PLACEHOLDER -> {
            com.lextures.android.features.courses.markdown.MarkdownToolPlaceholder(toolId = toolId, modifier = modifier)
            return
        }
        ContentToolHostLogic.FenceRenderMode.HIDDEN -> {
            Box(modifier)
            return
        }
        ContentToolHostLogic.FenceRenderMode.HOST -> Unit
    }

    if (page.loading && page.instances.isEmpty()) {
        ToolPlaceholder(reason = ToolPlaceholderReason.LOADING, modifier = modifier)
        return
    }

    val instance = page.instances[instanceId]
    if (instance == null) {
        // FR-3: missing / invisible → render nothing.
        Box(modifier)
        return
    }

    ContentToolHostMounted(
        instance = instance,
        page = page,
        modifier = modifier,
    )
}

@Composable
private fun ContentToolHostMounted(
    instance: ToolInstance,
    page: ContentToolsPageContext,
    modifier: Modifier = Modifier,
) {
    val scope = rememberCoroutineScope()
    val context = LocalContext.current
    val offline = remember { OfflineService.get(context.applicationContext) }
    var crashed by remember(instance.id) { mutableStateOf(false) }
    var showResetConfirm by remember { mutableStateOf(false) }
    var envelope by remember(instance.id) {
        mutableStateOf(instance.state ?: emptyToolState(instance.id))
    }
    var syncStatus by remember { mutableStateOf(ContentToolHostLogic.SyncStatus.IDLE) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var pending by remember { mutableStateOf<JsonObject?>(null) }
    var dirty by remember { mutableStateOf(false) }
    var saving by remember { mutableStateOf(false) }
    var debounceJob by remember { mutableStateOf<Job?>(null) }
    val actionKeys = remember { mutableStateMapOf<String, String>() }

    // Resolve copy once in composition — L.text is @Composable and cannot run inside suspend/try.
    val savedLabel = L.text(R.string.mobile_contentTools_runtime_saved)
    val unsyncedLabel = L.text(R.string.mobile_contentTools_runtime_unsynced)
    val retryLabel = L.text(R.string.mobile_contentTools_runtime_retry)
    val stateTooLargeLabel = L.text(R.string.mobile_contentTools_runtime_stateTooLarge)
    val schemaInvalidLabel = L.text(R.string.mobile_contentTools_runtime_schemaInvalid)
    val unavailableLabel = L.text(R.string.mobile_contentTools_runtime_unavailable)
    val needsConnectionLabel = L.text(R.string.mobile_contentTools_runtime_needsConnection)
    val resetLabel = L.text(R.string.mobile_contentTools_runtime_reset)
    val resetConfirmLabel = L.text(R.string.mobile_contentTools_runtime_resetConfirm)

    val readOnlyReason = ContentToolHostLogic.readOnlyReason(
        instance = instance,
        pastDue = page.pastDue,
        respectsDueDate = false,
        observer = page.observer,
    )
    val readOnly = readOnlyReason != null
    val readOnlyMessage = readOnlyReason?.let {
        L.text(
            when (it) {
                ContentToolHostLogic.ReadOnlyReason.ARCHIVED -> R.string.mobile_contentTools_runtime_readOnlyArchived
                ContentToolHostLogic.ReadOnlyReason.PAST_DUE -> R.string.mobile_contentTools_runtime_readOnlyPastDue
                ContentToolHostLogic.ReadOnlyReason.OBSERVER -> R.string.mobile_contentTools_runtime_readOnlyPreview
                else -> R.string.mobile_contentTools_runtime_unavailable
            },
        )
    }

    if (instance.tombstone || instance.breakerOpen) {
        ToolPlaceholder(
            reason = if (instance.breakerOpen) ToolPlaceholderReason.MAINTENANCE else ToolPlaceholderReason.UNAVAILABLE,
            toolName = ContentToolHostLogic.displayTitle(instance, instance.toolId),
            modifier = modifier,
        )
        return
    }

    if (ContentToolHostLogic.shouldShowUnsupportedPlaceholder(
            toolId = instance.toolId,
            contract = instance.contract,
            registered = ToolRegistry.registeredIds(),
        )
    ) {
        val path = ContentToolHostLogic.webActivityPath(page.courseCode, page.itemId, instance.id)
        ToolPlaceholder(
            reason = ToolPlaceholderReason.OPEN_IN_BROWSER,
            toolName = ContentToolHostLogic.displayTitle(instance, instance.toolId),
            onOpenInBrowser = {
                page.onOpenBrowser(Uri.parse(AppConfiguration.webUrl(path)))
            },
            modifier = modifier,
        )
        return
    }

    if (crashed) {
        ToolErrorCard(
            toolName = ContentToolHostLogic.displayTitle(instance, instance.toolId),
            onRetry = { crashed = false },
            modifier = modifier,
        )
        return
    }

    val renderer = ToolRegistry.resolve(instance.toolId)
    if (renderer == null) {
        ToolPlaceholder(
            reason = ToolPlaceholderReason.OPEN_IN_BROWSER,
            toolName = ContentToolHostLogic.displayTitle(instance, instance.toolId),
            onOpenInBrowser = {
                val path = ContentToolHostLogic.webActivityPath(page.courseCode, page.itemId, instance.id)
                page.onOpenBrowser(Uri.parse(AppConfiguration.webUrl(path)))
            },
            modifier = modifier,
        )
        return
    }

    fun applyEnvelope(next: ToolStateEnvelope) {
        envelope = next
    }

    suspend fun persist(nextState: JsonObject, mode: String) {
        val token = page.accessToken ?: return
        if (readOnly || saving) {
            if (!readOnly) {
                pending = nextState
                dirty = true
            }
            return
        }
        saving = true
        syncStatus = ContentToolHostLogic.SyncStatus.SAVING
        errorMessage = null
        val revision = envelope.revision
        try {
            val result = if (mode == "submit") {
                ContentToolsApi.submit(page.courseCode, instance.id, revision, nextState, token)
            } else {
                try {
                    ContentToolsApi.putState(page.courseCode, instance.id, revision, nextState, token)
                } catch (e: ApiError.Transport) {
                    if (ContentToolHostLogic.canQueueStateWriteOffline()) {
                        offline.enqueueMutation(
                            method = "PUT",
                            path = ContentToolsApi.statePutPath(page.courseCode, instance.id),
                            bodyJson = ContentToolsApi.encodeStateBody(revision, nextState),
                            label = "content-tool-state:${instance.id}",
                            accessToken = token,
                            preferQueue = true,
                        )
                        syncStatus = ContentToolHostLogic.SyncStatus.UNSYNCED
                        dirty = true
                        pending = nextState
                        page.announce(unsyncedLabel, false)
                        return
                    }
                    throw e
                }
            }
            applyEnvelope(result)
            dirty = false
            pending = null
            syncStatus = ContentToolHostLogic.SyncStatus.SAVED
            page.announce(savedLabel, false)
        } catch (e: ContentToolsApi.RevisionConflictException) {
            val policy = ContentToolHostLogic.conflictPolicyForTool(instance.toolId)
            val resolved = ContentToolHostLogic.resolveConflictJson(policy, nextState, e.current.document())
            applyEnvelope(e.current.copy(state = resolved, stateJson = resolved))
            if (policy == ContentToolHostLogic.ConflictPolicy.SERVER_WINS) {
                dirty = false
                pending = null
                syncStatus = ContentToolHostLogic.SyncStatus.SAVED
                page.announce(savedLabel, false)
            } else {
                try {
                    val retry = ContentToolsApi.putState(
                        page.courseCode,
                        instance.id,
                        e.current.revision,
                        resolved,
                        token,
                    )
                    applyEnvelope(retry)
                    dirty = false
                    pending = null
                    syncStatus = ContentToolHostLogic.SyncStatus.SAVED
                    page.announce(savedLabel, false)
                } catch (_: Exception) {
                    dirty = true
                    syncStatus = ContentToolHostLogic.SyncStatus.UNSYNCED
                    errorMessage = retryLabel
                }
            }
        } catch (_: ContentToolsApi.StateTooLargeException) {
            dirty = true
            syncStatus = ContentToolHostLogic.SyncStatus.ERROR
            errorMessage = stateTooLargeLabel
            page.announce(stateTooLargeLabel, true)
        } catch (_: ContentToolsApi.SchemaInvalidException) {
            dirty = true
            syncStatus = ContentToolHostLogic.SyncStatus.ERROR
            errorMessage = schemaInvalidLabel
            page.announce(schemaInvalidLabel, true)
        } catch (_: ApiError.HttpStatus) {
            dirty = true
            syncStatus = ContentToolHostLogic.SyncStatus.UNSYNCED
            errorMessage = unavailableLabel
            // AC-13: flag off mid-session → read-only without discarding work.
        } catch (_: Exception) {
            dirty = true
            syncStatus = ContentToolHostLogic.SyncStatus.UNSYNCED
            errorMessage = retryLabel
        } finally {
            saving = false
            if (dirty && pending != null) {
                val queued = pending
                pending = null
                if (queued != null) persist(queued, "save")
            }
        }
    }

    fun scheduleSave(nextState: JsonObject) {
        pending = nextState
        dirty = true
        syncStatus = ContentToolHostLogic.syncStatusAfterEdit(syncStatus)
        debounceJob?.cancel()
        debounceJob = scope.launch {
            delay(ContentToolHostLogic.DEFAULT_DEBOUNCE_MS.toLong())
            val pendingState = pending ?: return@launch
            persist(pendingState, "save")
        }
    }

    DisposableEffect(instance.id) {
        onDispose {
            debounceJob?.cancel()
            val pendingState = pending
            if (pendingState != null && dirty && !readOnly) {
                scope.launch { persist(pendingState, "save") }
            }
        }
    }

    if (showResetConfirm) {
        AlertDialog(
            onDismissRequest = { showResetConfirm = false },
            title = { Text(resetLabel) },
            text = { Text(resetConfirmLabel) },
            confirmButton = {
                TextButton(onClick = {
                    showResetConfirm = false
                    val token = page.accessToken ?: return@TextButton
                    scope.launch {
                        runCatching {
                            applyEnvelope(ContentToolsApi.selfReset(page.courseCode, instance.id, token))
                            syncStatus = ContentToolHostLogic.SyncStatus.SAVED
                        }
                    }
                }) { Text(resetLabel) }
            },
            dismissButton = {
                TextButton(onClick = { showResetConfirm = false }) {
                    Text(retryLabel)
                }
            },
        )
    }

    ToolFrame(
        title = ContentToolHostLogic.displayTitle(instance, instance.toolId),
        status = envelope.status,
        syncStatus = syncStatus,
        score = envelope.score,
        readOnly = readOnly,
        readOnlyMessage = readOnlyMessage ?: errorMessage,
        studentResetAllowed = page.studentResetAllowed,
        onReset = { showResetConfirm = true },
        frameModifier = modifier,
    ) {
        renderer(
            ContentToolRendererProps(
                instanceId = instance.id,
                toolId = instance.toolId,
                config = instance.config,
                state = envelope.document(),
                status = envelope.status,
                readOnly = readOnly,
                save = { patch ->
                    val next = ContentToolHostLogic.mergeStatePatch(envelope.document(), patch)
                    scheduleSave(next)
                },
                submit = { patch ->
                    val next = ContentToolHostLogic.mergeStatePatch(envelope.document(), patch)
                    scope.launch { persist(next, "submit") }
                },
                runAction = { name, input ->
                    val token = page.accessToken
                    if (token.isNullOrBlank()) {
                        page.announce(needsConnectionLabel, true)
                        throw IllegalStateException("offline")
                    }
                    val key = actionKeys.getOrPut(name) { ContentToolHostLogic.newIdempotencyKey() }
                    try {
                        val res = ContentToolsApi.runAction(
                            courseCode = page.courseCode,
                            instanceId = instance.id,
                            action = name,
                            input = input,
                            accessToken = token,
                            idempotencyKey = key,
                        )
                        res.state?.let { applyEnvelope(it) }
                        actionKeys.remove(name)
                        res.result
                    } catch (e: ApiError.Transport) {
                        page.announce(needsConnectionLabel, true)
                        throw e
                    }
                },
                announce = page.announce,
            ),
        )
    }
}
