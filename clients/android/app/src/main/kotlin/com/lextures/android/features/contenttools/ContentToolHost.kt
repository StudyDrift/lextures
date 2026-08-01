package com.lextures.android.features.contenttools

import android.net.Uri
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableLongStateOf
import androidx.compose.runtime.mutableStateMapOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.runtime.staticCompositionLocalOf
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.semantics.LiveRegionMode
import androidx.compose.ui.semantics.liveRegion
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.unit.dp
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.LifecycleEventObserver
import androidx.lifecycle.compose.LocalLifecycleOwner
import com.lextures.android.R
import com.lextures.android.core.config.AppConfiguration
import com.lextures.android.core.i18n.L
import com.lextures.android.core.lms.ContentToolAIConsent
import com.lextures.android.core.lms.ContentToolGovernanceLogic
import com.lextures.android.core.lms.ContentToolHostLogic
import com.lextures.android.core.lms.ContentToolModerationAction
import com.lextures.android.core.lms.ContentToolSandboxLogic
import com.lextures.android.core.lms.ContentToolSettings
import com.lextures.android.core.lms.ContentToolsApi
import com.lextures.android.core.lms.ContentToolsObservability
import com.lextures.android.core.lms.ToolInstance
import com.lextures.android.core.lms.ToolStateEnvelope
import com.lextures.android.core.lms.emptyToolState
import com.lextures.android.core.network.ApiError
import com.lextures.android.core.offline.OfflineCacheKey
import com.lextures.android.core.offline.OfflineService
import com.lextures.android.features.contenttools.governance.AIDisclosureBanner
import com.lextures.android.features.contenttools.governance.ConsentGateView
import com.lextures.android.features.contenttools.governance.CrisisResourcesView
import com.lextures.android.features.contenttools.governance.ModerationResult
import com.lextures.android.features.contenttools.governance.ModerationSheet
import com.lextures.android.features.contenttools.governance.PolicyBlockedPlaceholder
import com.lextures.android.features.contenttools.governance.ReportSheet
import com.lextures.android.features.contenttools.sandbox.SandboxWebViewHost
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.booleanOrNull
import kotlinx.serialization.json.contentOrNull

data class ContentToolsPageContext(
    val courseCode: String,
    val itemId: String,
    val contentToolsEnabled: Boolean,
    val mobileContentToolsEnabled: Boolean,
    val accessToken: String?,
    /** CT.M4 sandbox WebView host capability (independent of CT.M3). */
    val mobileContentToolsSandboxEnabled: Boolean = false,
    val observer: Boolean = false,
    val pastDue: Boolean = false,
    val loading: Boolean = false,
    val instances: Map<String, ToolInstance> = emptyMap(),
    val studentResetAllowed: Boolean = false,
    /** CT.M9 — settings/policy snapshot for mount gating. */
    val settings: ContentToolSettings? = null,
    val fetchedAtMs: Long = 0L,
    val fetchSucceeded: Boolean = false,
    val nonConformantToolIds: Set<String> = emptySet(),
    val canModerate: Boolean = false,
    val onRefreshGovernance: () -> Unit = {},
    val announce: (message: String, assertive: Boolean) -> Unit = { _, _ -> },
    val onOpenBrowser: (Uri) -> Unit = {},
) {
    val policy get() = settings?.policy

    val ageMs: Long
        get() {
            if (fetchedAtMs <= 0L) return Long.MAX_VALUE
            return maxOf(0L, System.currentTimeMillis() - fetchedAtMs)
        }
}

val LocalContentToolsPage = staticCompositionLocalOf<ContentToolsPageContext?> { null }

private val REPORT_CATEGORIES = listOf("harassment", "hate", "self_harm", "spam", "other")

@Composable
fun ContentToolsPageProvider(
    courseCode: String,
    itemId: String,
    contentToolsEnabled: Boolean,
    mobileContentToolsEnabled: Boolean,
    accessToken: String?,
    mobileContentToolsSandboxEnabled: Boolean = false,
    observer: Boolean = false,
    pastDue: Boolean = false,
    onOpenBrowser: (Uri) -> Unit = {},
    content: @Composable () -> Unit,
) {
    val context = LocalContext.current
    val offline = remember { OfflineService.get(context.applicationContext) }
    val lifecycleOwner = LocalLifecycleOwner.current
    var loading by remember(courseCode, itemId) { mutableStateOf(false) }
    var instances by remember(courseCode, itemId) { mutableStateOf<Map<String, ToolInstance>>(emptyMap()) }
    var studentResetAllowed by remember { mutableStateOf(false) }
    var settings by remember { mutableStateOf<ContentToolSettings?>(null) }
    var fetchedAtMs by remember { mutableLongStateOf(0L) }
    var fetchSucceeded by remember { mutableStateOf(false) }
    var nonConformantToolIds by remember { mutableStateOf<Set<String>>(emptySet()) }
    var canModerate by remember { mutableStateOf(false) }
    var refreshTick by remember { mutableIntStateOf(0) }
    var liveMessage by remember { mutableStateOf("") }
    var liveAssertive by remember { mutableStateOf(false) }

    val announce: (String, Boolean) -> Unit = { message, assertive ->
        liveMessage = message
        liveAssertive = assertive
    }

    // FR-4: re-evaluate policy on foreground so kills apply without an app release.
    // Skip the initial RESUMED dispatch when the observer is attached.
    var wasStopped by remember { mutableStateOf(false) }
    DisposableEffect(lifecycleOwner) {
        val observer = LifecycleEventObserver { _, event ->
            when (event) {
                Lifecycle.Event.ON_STOP -> wasStopped = true
                Lifecycle.Event.ON_RESUME -> {
                    if (wasStopped) {
                        wasStopped = false
                        refreshTick += 1
                    }
                }
                else -> Unit
            }
        }
        lifecycleOwner.lifecycle.addObserver(observer)
        onDispose { lifecycleOwner.lifecycle.removeObserver(observer) }
    }

    LaunchedEffect(
        courseCode,
        itemId,
        contentToolsEnabled,
        mobileContentToolsEnabled,
        accessToken,
        refreshTick,
    ) {
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

            // Settings + governance snapshot (CT.M9) — fail closed for AI/third-party when missing.
            val priorNonConformant = nonConformantToolIds
            val priorSettings = settings
            val priorFetchedAt = fetchedAtMs
            val fetched = runCatching {
                ContentToolsApi.fetchSettings(courseCode, accessToken)
            }.getOrNull()
            if (fetched != null) {
                studentResetAllowed = fetched.studentResetAllowed
                settings = fetched
                fetchedAtMs = System.currentTimeMillis()
                fetchSucceeded = true
                nonConformantToolIds = priorNonConformant
            } else if (priorSettings != null) {
                // Keep cached policy; mark fetch failed so staleness rules apply.
                settings = priorSettings
                fetchedAtMs = priorFetchedAt
                fetchSucceeded = false
                nonConformantToolIds = priorNonConformant
            } else {
                settings = null
                fetchedAtMs = 0L
                fetchSucceeded = false
                nonConformantToolIds = emptySet()
                studentResetAllowed = false
            }

            // Staff moderation: non-observers may attempt; 403 handled in sheet (FR-11).
            canModerate = !observer

            runCatching { ContentToolsApi.fetchConformance(accessToken) }.getOrNull()?.let { conf ->
                nonConformantToolIds = conf.tools.filter { !it.ok }.map { it.toolId }.toSet()
            }
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
            mobileContentToolsSandboxEnabled = mobileContentToolsSandboxEnabled,
            accessToken = accessToken,
            observer = observer,
            pastDue = pastDue,
            loading = loading,
            instances = instances,
            studentResetAllowed = studentResetAllowed,
            settings = settings,
            fetchedAtMs = fetchedAtMs,
            fetchSucceeded = fetchSucceeded,
            nonConformantToolIds = nonConformantToolIds,
            canModerate = canModerate,
            onRefreshGovernance = { refreshTick += 1 },
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
    val settings = page.settings
    val policy = settings?.policy
    val killed = ContentToolGovernanceLogic.toolIsKilled(
        toolId = instance.toolId,
        capabilities = instance.capabilities,
        killedToolIds = settings?.killedToolIds.orEmpty(),
        killedCapabilities = settings?.killedCapabilities.orEmpty(),
        killAllAI = settings?.killAllAI ?: false,
    )
    // Merge course allowlist with org policy allow/deny lists.
    val allowed = ContentToolGovernanceLogic.effectiveAllowedToolIds(
        courseAllowed = settings?.allowedToolIds.orEmpty(),
        orgAllowed = policy?.allowedToolIds.orEmpty(),
    )
    val decision = ContentToolGovernanceLogic.mountDecision(
        ContentToolGovernanceLogic.MountInput(
            toolId = instance.toolId,
            capabilities = instance.capabilities,
            sandboxMode = instance.sandboxMode,
            tombstone = instance.tombstone,
            breakerOpen = instance.breakerOpen,
            deprecated = instance.deprecated,
            killed = killed,
            allowedToolIds = allowed,
            deniedToolIds = policy?.deniedToolIds.orEmpty(),
            deniedCapabilities = policy?.deniedCapabilities.orEmpty(),
            policyFetched = page.fetchSucceeded,
            policyAgeMs = page.ageMs,
            staleWindowMs = ContentToolGovernanceLogic.DEFAULT_STALE_WINDOW_MS,
            unknownGovernanceState = false,
            hasCachedPolicy = settings != null,
        ),
    )

    if (decision != ContentToolGovernanceLogic.MountDecision.MOUNT) {
        ContentToolsObservability.record(
            "policy_blocked",
            toolId = instance.toolId,
            attributes = mapOf("reason" to decision.wire),
        )
        PolicyBlockedPlaceholder(
            decision = decision,
            toolName = ContentToolHostLogic.displayTitle(instance, instance.toolId),
            onRefresh = page.onRefreshGovernance,
            modifier = modifier,
        )
        return
    }

    ContentToolHostAllowed(
        instance = instance,
        page = page,
        killed = killed,
        modifier = modifier,
    )
}

@Composable
private fun ContentToolHostAllowed(
    instance: ToolInstance,
    page: ContentToolsPageContext,
    killed: Boolean,
    modifier: Modifier = Modifier,
) {
    val scope = rememberCoroutineScope()
    val context = LocalContext.current
    val offline = remember { OfflineService.get(context.applicationContext) }
    var crashed by remember(instance.id) { mutableStateOf(false) }
    var showResetConfirm by remember { mutableStateOf(false) }
    var showReport by remember { mutableStateOf(false) }
    var showModerate by remember { mutableStateOf(false) }
    var moderationItems by remember { mutableStateOf<List<ContentToolModerationAction>>(emptyList()) }
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
    var consent by remember { mutableStateOf<ContentToolAIConsent?>(null) }
    var consentFetched by remember { mutableStateOf(false) }
    var consentBusy by remember { mutableStateOf(false) }
    var showCrisis by remember { mutableStateOf(false) }

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
    val cancelLabel = L.text(R.string.mobile_contentTools_runtime_cancel)
    val consentErrorLabel = L.text(R.string.mobile_contentTools_governance_consentError)
    val reportThanksLabel = L.text(R.string.mobile_contentTools_governance_reportThanks)
    val moderateForbiddenLabel = L.text(R.string.mobile_contentTools_governance_moderateForbidden)
    val moderateErrorLabel = L.text(R.string.mobile_contentTools_governance_moderateError)
    val crisisBodyLabel = L.text(R.string.mobile_contentTools_governance_crisisBody)
    val crisisTitleLabel = L.text(R.string.mobile_contentTools_governance_crisisTitle)
    val filteredLabel = L.text(R.string.mobile_contentTools_governance_filtered)

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
    val requiresAI = ContentToolGovernanceLogic.isAICapable(instance.capabilities)
    val nonConformant = page.nonConformantToolIds.contains(instance.toolId)

    val renderPath = ContentToolSandboxLogic.resolveRenderPath(
        toolId = instance.toolId,
        contract = instance.contract,
        sandboxMode = instance.sandboxMode,
        sandboxEnabled = page.mobileContentToolsSandboxEnabled,
        registered = ToolRegistry.registeredIds(),
        tombstone = instance.tombstone,
        breakerOpen = instance.breakerOpen,
        deprecated = instance.deprecated,
        killed = killed,
    )
    ContentToolsObservability.record("tool_mount", toolId = instance.toolId)

    if (renderPath == ContentToolSandboxLogic.RenderPath.PLACEHOLDER) {
        ContentToolsObservability.record("unsupported_placeholder", toolId = instance.toolId)
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

    if (crashed && renderPath == ContentToolSandboxLogic.RenderPath.NATIVE) {
        ContentToolsObservability.record(
            "render_error",
            toolId = instance.toolId,
            attributes = mapOf("error_class" to "crash"),
        )
        ToolErrorCard(
            toolName = ContentToolHostLogic.displayTitle(instance, instance.toolId),
            onRetry = { crashed = false },
            modifier = modifier,
        )
        return
    }

    val renderer = if (renderPath == ContentToolSandboxLogic.RenderPath.NATIVE) {
        ToolRegistry.resolve(instance.toolId)
    } else {
        null
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
                        ContentToolsObservability.record(
                            "offline_replay",
                            toolId = instance.toolId,
                            attributes = mapOf("outcome" to "queued"),
                        )
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
            ContentToolsObservability.record(
                "state_save",
                toolId = instance.toolId,
                attributes = mapOf("outcome" to "ok"),
            )
        } catch (e: ContentToolsApi.RevisionConflictException) {
            ContentToolsObservability.record(
                "revision_conflict",
                toolId = instance.toolId,
                attributes = mapOf("outcome" to "conflict"),
            )
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
            ContentToolsObservability.record(
                "state_save",
                toolId = instance.toolId,
                attributes = mapOf("outcome" to "error", "error_class" to "too_large"),
            )
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

    fun handleActionResult(result: JsonElement?) {
        val obj = result as? JsonObject ?: return
        val code = (obj["error"] as? JsonPrimitive)?.contentOrNull
            ?: (obj["code"] as? JsonPrimitive)?.contentOrNull
        val crisis = (obj["crisis"] as? JsonPrimitive)?.booleanOrNull ?: false
        val outcome = ContentToolGovernanceLogic.filterCrisisOutcome(
            ContentToolGovernanceLogic.FilterCrisisInput(errorCode = code, crisis = crisis),
        )
        when (outcome.kind) {
            ContentToolGovernanceLogic.FilterOutcomeKind.CRISIS -> {
                showCrisis = true
                errorMessage = crisisBodyLabel
                page.announce(crisisTitleLabel, true)
            }
            ContentToolGovernanceLogic.FilterOutcomeKind.FILTERED -> {
                // Plain language — do not echo blocked content.
                errorMessage = filteredLabel
                page.announce(filteredLabel, true)
            }
            ContentToolGovernanceLogic.FilterOutcomeKind.GENERIC -> Unit
        }
    }

    suspend fun runAction(name: String, input: JsonElement): JsonElement? {
        val token = page.accessToken
        if (token.isNullOrBlank()) {
            page.announce(needsConnectionLabel, true)
            ContentToolsObservability.record(
                "action_outcome",
                toolId = instance.toolId,
                attributes = mapOf("outcome" to "error", "error_class" to "offline"),
            )
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
            handleActionResult(res.result)
            ContentToolsObservability.record(
                "action_outcome",
                toolId = instance.toolId,
                attributes = mapOf("outcome" to "ok"),
            )
            return res.result
        } catch (e: ApiError.Transport) {
            page.announce(needsConnectionLabel, true)
            ContentToolsObservability.record(
                "action_outcome",
                toolId = instance.toolId,
                attributes = mapOf("outcome" to "error", "error_class" to "offline"),
            )
            throw e
        } catch (e: Exception) {
            ContentToolsObservability.record(
                "action_outcome",
                toolId = instance.toolId,
                attributes = mapOf("outcome" to "error", "error_class" to "unknown"),
            )
            throw e
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
                    Text(cancelLabel)
                }
            },
        )
    }

    if (showReport) {
        ReportSheet(
            categories = REPORT_CATEGORIES,
            onSubmit = { category, note ->
                val token = page.accessToken ?: return@ReportSheet false
                try {
                    ContentToolsApi.reportContent(
                        courseCode = page.courseCode,
                        instanceId = instance.id,
                        category = category,
                        reason = note,
                        contentPath = null,
                        accessToken = token,
                    )
                    ContentToolsObservability.record(
                        "report_submitted",
                        toolId = instance.toolId,
                        attributes = mapOf("outcome" to "ok"),
                    )
                    page.announce(reportThanksLabel, false)
                    true
                } catch (_: Exception) {
                    ContentToolsObservability.record(
                        "report_submitted",
                        toolId = instance.toolId,
                        attributes = mapOf("outcome" to "error"),
                    )
                    false
                }
            },
            onDismiss = { showReport = false },
        )
    }

    if (showModerate) {
        ModerationSheet(
            items = moderationItems,
            onModerate = { action ->
                val token = page.accessToken ?: return@ModerationSheet ModerationResult.ERROR
                try {
                    ContentToolsApi.moderateContent(
                        courseCode = page.courseCode,
                        instanceId = instance.id,
                        action = action,
                        category = null,
                        reason = null,
                        contentPath = null,
                        accessToken = token,
                    )
                    ModerationResult.OK
                } catch (e: ApiError.HttpStatus) {
                    if (e.code == 403) {
                        errorMessage = moderateForbiddenLabel
                        ModerationResult.FORBIDDEN
                    } else {
                        ModerationResult.ERROR
                    }
                } catch (_: Exception) {
                    ModerationResult.ERROR
                }
            },
            onDismiss = { showModerate = false },
        )
    }

    val showDisclosure = requiresAI && ContentToolGovernanceLogic.shouldShowAIDisclosure(
        disclosureMode = consent?.aiDisclosureMode ?: page.policy?.aiDisclosureMode,
        decision = consent?.decision,
        consentFetched = consentFetched,
    )
    val aiAllowed = !requiresAI || ContentToolGovernanceLogic.aiActionsAllowed(
        disclosureMode = consent?.aiDisclosureMode ?: page.policy?.aiDisclosureMode,
        decision = consent?.decision,
        consentFetched = consentFetched,
    )

    // Capability denial: never solicit OS permissions for denied caps (FR-2).
    val denied = (page.settings?.policy?.deniedCapabilities.orEmpty()).map { it.lowercase() }
    val filteredCaps = instance.capabilities.filter {
        ContentToolGovernanceLogic.canObtainDeniedCapability(it, denied)
    }

    ToolFrame(
        title = ContentToolHostLogic.displayTitle(instance, instance.toolId),
        status = envelope.status,
        syncStatus = syncStatus,
        score = envelope.score,
        readOnly = readOnly,
        readOnlyMessage = readOnlyMessage ?: errorMessage,
        studentResetAllowed = page.studentResetAllowed,
        showSandboxBadge = renderPath == ContentToolSandboxLogic.RenderPath.SANDBOX,
        showNonConformantNote = nonConformant,
        canReport = true,
        canModerate = page.canModerate,
        onReset = { showResetConfirm = true },
        onReport = { showReport = true },
        onModerate = {
            scope.launch {
                val token = page.accessToken ?: return@launch
                try {
                    moderationItems = ContentToolsApi.fetchModeration(
                        page.courseCode,
                        instance.id,
                        token,
                    )
                    showModerate = true
                } catch (e: ApiError.HttpStatus) {
                    if (e.code == 403) {
                        errorMessage = moderateForbiddenLabel
                        page.announce(moderateForbiddenLabel, true)
                    } else {
                        errorMessage = moderateErrorLabel
                    }
                } catch (_: Exception) {
                    errorMessage = moderateErrorLabel
                }
            }
        },
        frameModifier = modifier,
        disclosure = {
            Column(
                Modifier.fillMaxWidth(),
            ) {
                if (requiresAI) {
                    LaunchedEffect(instance.toolId, page.courseCode, page.accessToken) {
                        val token = page.accessToken
                        if (token.isNullOrBlank()) {
                            consentFetched = false
                            return@LaunchedEffect
                        }
                        runCatching {
                            ContentToolsApi.fetchAIConsent(page.courseCode, instance.toolId, token)
                        }.onSuccess {
                            consent = it
                            consentFetched = true
                        }.onFailure {
                            consent = null
                            consentFetched = false
                        }
                    }
                }
                if (showDisclosure) {
                    AIDisclosureBanner(
                        mode = consent?.aiDisclosureMode ?: "acknowledge",
                        busy = consentBusy,
                        onAcknowledge = {
                            scope.launch {
                                val token = page.accessToken ?: return@launch
                                consentBusy = true
                                runCatching {
                                    ContentToolsApi.postAIConsent(
                                        page.courseCode,
                                        instance.toolId,
                                        "acknowledged",
                                        token,
                                    )
                                }.onSuccess {
                                    consent = it
                                    consentFetched = true
                                }.onFailure {
                                    errorMessage = consentErrorLabel
                                }
                                consentBusy = false
                            }
                        },
                        onOptOut = {
                            scope.launch {
                                val token = page.accessToken ?: return@launch
                                consentBusy = true
                                runCatching {
                                    ContentToolsApi.postAIConsent(
                                        page.courseCode,
                                        instance.toolId,
                                        "opted_out",
                                        token,
                                    )
                                }.onSuccess {
                                    consent = it
                                    consentFetched = true
                                }.onFailure {
                                    errorMessage = consentErrorLabel
                                }
                                consentBusy = false
                            }
                        },
                    )
                } else if (requiresAI && !aiAllowed) {
                    ContentToolsObservability.record(
                        "ai_blocked_by_consent",
                        toolId = instance.toolId,
                    )
                    ConsentGateView(
                        busy = consentBusy,
                        onGrant = {
                            scope.launch {
                                val token = page.accessToken ?: return@launch
                                consentBusy = true
                                runCatching {
                                    ContentToolsApi.postAIConsent(
                                        page.courseCode,
                                        instance.toolId,
                                        "acknowledged",
                                        token,
                                    )
                                }.onSuccess {
                                    consent = it
                                    consentFetched = true
                                }.onFailure {
                                    errorMessage = consentErrorLabel
                                }
                                consentBusy = false
                            }
                        },
                    )
                }
                if (showCrisis) {
                    CrisisResourcesView()
                }
            }
        },
    ) {
        when (renderPath) {
            ContentToolSandboxLogic.RenderPath.SANDBOX -> {
                SandboxWebViewHost(
                    toolId = instance.toolId,
                    instanceId = instance.id,
                    toolVersion = instance.toolVersion,
                    title = ContentToolHostLogic.displayTitle(instance, instance.toolId),
                    config = instance.config,
                    state = envelope.document(),
                    revision = envelope.revision,
                    readOnly = readOnly,
                    capabilities = filteredCaps,
                    accessToken = page.accessToken,
                    save = { next -> scheduleSave(next) },
                    runAction = { name, input -> runAction(name, input) },
                    announce = page.announce,
                    onOpenUrl = { page.onOpenBrowser(it) },
                )
            }
            ContentToolSandboxLogic.RenderPath.NATIVE -> {
                val native = renderer
                if (native == null) {
                    val path = ContentToolHostLogic.webActivityPath(page.courseCode, page.itemId, instance.id)
                    ToolPlaceholder(
                        reason = ToolPlaceholderReason.OPEN_IN_BROWSER,
                        toolName = ContentToolHostLogic.displayTitle(instance, instance.toolId),
                        onOpenInBrowser = {
                            page.onOpenBrowser(Uri.parse(AppConfiguration.webUrl(path)))
                        },
                    )
                } else {
                    native(
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
                            runAction = { name, input -> runAction(name, input) },
                            announce = page.announce,
                        ),
                    )
                }
            }
            ContentToolSandboxLogic.RenderPath.PLACEHOLDER -> Unit
        }
    }
}
