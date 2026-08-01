import SwiftUI

/// Mounts one tool fence inside the page host.
struct ContentToolHostView: View {
    @Environment(AuthSession.self) private var session
    @Environment(OfflineService.self) private var offline
    @Environment(\.contentToolsPage) private var page
    @Environment(\.contentToolsInstances) private var instances
    @Environment(\.contentToolsLoading) private var loading
    @Environment(\.contentToolsStudentResetAllowed) private var studentResetAllowed
    @Environment(\.contentToolsGovernance) private var governance
    @Environment(\.contentToolsCanModerate) private var canModerate
    @Environment(\.contentToolsRefreshGovernance) private var refreshGovernance
    @Environment(\.openURL) private var openURL

    let instanceId: String
    let toolId: String

    @State private var crashed = false
    @State private var showResetConfirm = false
    @State private var showReport = false
    @State private var showModerate = false
    @State private var moderationItems: [ContentToolModerationAction] = []
    @State private var envelope: ToolStateEnvelope
    @State private var syncStatus: ContentToolHostLogic.SyncStatus = .idle
    @State private var errorMessage: String?
    @State private var pending: JSONValue?
    @State private var dirty = false
    @State private var saving = false
    @State private var saveTask: Task<Void, Never>?
    @State private var actionKeys: [String: String] = [:]
    @State private var consent: ContentToolAIConsent?
    @State private var consentFetched = false
    @State private var consentBusy = false
    @State private var showCrisis = false

    private let reportCategories = [
        "harassment", "hate", "self_harm", "spam", "other",
    ]

    init(instanceId: String, toolId: String) {
        self.instanceId = instanceId
        self.toolId = toolId
        _envelope = State(initialValue: .empty(instanceId: instanceId))
    }

    var body: some View {
        Group {
            if let page {
                switch ContentToolHostLogic.fenceRenderMode(
                    mobileContentToolsEnabled: page.mobileContentToolsEnabled,
                    contentToolsEnabled: page.contentToolsEnabled
                ) {
                case .legacyPlaceholder:
                    legacyPlaceholder
                case .hidden:
                    EmptyView()
                case .host:
                    hostBody(page: page)
                }
            } else {
                legacyPlaceholder
            }
        }
        .onAppear {
            if let instance = instances[instanceId], let state = instance.state {
                envelope = state
            }
        }
        .onChange(of: instances[instanceId]?.state) { _, newState in
            if let newState { envelope = newState }
        }
        .confirmationDialog(
            L.text("mobile.contentTools.runtime.reset"),
            isPresented: $showResetConfirm,
            titleVisibility: .visible
        ) {
            Button(L.text("mobile.contentTools.runtime.reset"), role: .destructive) {
                Task { await selfReset() }
            }
        } message: {
            Text(L.text("mobile.contentTools.runtime.resetConfirm"))
        }
        .sheet(isPresented: $showReport) {
            ReportSheet(categories: reportCategories) { category, note in
                await submitReport(category: category, note: note)
            }
        }
        .sheet(isPresented: $showModerate) {
            ModerationSheet(items: moderationItems) { action in
                await submitModerate(action: action)
            }
        }
    }

    private var legacyPlaceholder: some View {
        Label {
            Text(L.format("mobile.markdown.tool.placeholder", toolId))
                .font(.subheadline)
        } icon: {
            Image(systemName: "puzzlepiece.extension")
        }
        .accessibilityLabel(L.format("mobile.markdown.tool.placeholder", toolId))
    }

    @ViewBuilder
    private func hostBody(page: ContentToolsPageContext) -> some View {
        if loading && instances.isEmpty {
            ToolPlaceholderView(reason: .loading)
        } else if let instance = instances[instanceId] {
            mounted(instance: instance, page: page)
        } else {
            EmptyView()
        }
    }

    @ViewBuilder
    private func mounted(instance: ToolInstance, page: ContentToolsPageContext) -> some View {
        let settings = governance.settings
        let policy = settings?.policy
        let killed = ContentToolGovernanceLogic.toolIsKilled(
            toolId: instance.toolId,
            capabilities: instance.capabilities,
            killedToolIds: settings?.killedToolIds ?? [],
            killedCapabilities: settings?.killedCapabilities ?? [],
            killAllAI: settings?.killAllAI ?? false
        )
        let allowed = ContentToolGovernanceLogic.effectiveAllowedToolIds(
            courseAllowed: settings?.allowedToolIds ?? [],
            orgAllowed: policy?.allowedToolIds ?? []
        )
        let decision = ContentToolGovernanceLogic.mountDecision(
            ContentToolGovernanceLogic.MountInput(
                toolId: instance.toolId,
                capabilities: instance.capabilities,
                sandboxMode: instance.sandboxMode,
                tombstone: instance.tombstone,
                breakerOpen: instance.breakerOpen,
                deprecated: instance.deprecated,
                killed: killed,
                allowedToolIds: allowed,
                deniedToolIds: policy?.deniedToolIds ?? [],
                deniedCapabilities: policy?.deniedCapabilities ?? [],
                policyFetched: governance.fetchSucceeded,
                policyAgeMs: governance.ageMs,
                staleWindowMs: ContentToolGovernanceLogic.defaultStaleWindowMs,
                unknownGovernanceState: false,
                hasCachedPolicy: settings != nil
            )
        )

        if decision != .mount {
            PolicyBlockedPlaceholder(
                decision: decision,
                toolName: ContentToolHostLogic.displayTitle(instance: instance, toolId: instance.toolId),
                onRefresh: refreshGovernance
            )
            .onAppear {
                ContentToolsObservability.record(
                    "policy_blocked",
                    toolId: instance.toolId,
                    attributes: ["reason": decision.rawValue]
                )
            }
        } else {
            mountedAllowed(instance: instance, page: page, killed: killed)
        }
    }

    @ViewBuilder
    private func mountedAllowed(instance: ToolInstance, page: ContentToolsPageContext, killed: Bool) -> some View {
        let reason = ContentToolHostLogic.readOnlyReason(
            instance: instance,
            pastDue: page.pastDue,
            respectsDueDate: false,
            observer: page.observer
        )
        let readOnly = reason != nil
        let readOnlyMessage = reason.map {
            L.text(String.LocalizationValue(ContentToolHostLogic.readOnlyMessageKey($0)))
        }
        let requiresAI = ContentToolGovernanceLogic.isAICapable(instance.capabilities)
        let nonConformant = governance.nonConformantToolIds.contains(instance.toolId)
        let path = ContentToolSandboxLogic.resolveRenderPath(
            toolId: instance.toolId,
            contract: instance.contract,
            sandboxMode: instance.sandboxMode,
            sandboxEnabled: page.mobileContentToolsSandboxEnabled,
            registered: ToolRendererRegistry.registeredIds(),
            tombstone: instance.tombstone,
            breakerOpen: instance.breakerOpen,
            deprecated: instance.deprecated,
            killed: killed
        )
        Group {
            switch path {
            case .placeholder:
                unsupportedPlaceholder(instance: instance, page: page)
            case .native:
                nativeMounted(
                    instance: instance,
                    page: page,
                    readOnly: readOnly,
                    readOnlyMessage: readOnlyMessage,
                    requiresAI: requiresAI,
                    nonConformant: nonConformant
                )
            case .sandbox:
                sandboxMounted(
                    instance: instance,
                    page: page,
                    readOnly: readOnly,
                    readOnlyMessage: readOnlyMessage,
                    requiresAI: requiresAI,
                    nonConformant: nonConformant
                )
            }
        }
        .onAppear {
            ContentToolsObservability.record("tool_mount", toolId: instance.toolId)
        }
    }

    @ViewBuilder
    private func unsupportedPlaceholder(instance: ToolInstance, page: ContentToolsPageContext) -> some View {
        ToolPlaceholderView(
            reason: .openInBrowser,
            toolName: ContentToolHostLogic.displayTitle(instance: instance, toolId: instance.toolId),
            onOpenInBrowser: { openWeb(page: page, instanceId: instance.id) }
        )
        .onAppear {
            ContentToolsObservability.record("unsupported_placeholder", toolId: instance.toolId)
        }
    }

    @ViewBuilder
    private func nativeMounted(
        instance: ToolInstance,
        page: ContentToolsPageContext,
        readOnly: Bool,
        readOnlyMessage: String?,
        requiresAI: Bool,
        nonConformant: Bool
    ) -> some View {
        if crashed {
            ToolErrorCardView(
                toolName: ContentToolHostLogic.displayTitle(instance: instance, toolId: instance.toolId),
                onRetry: { crashed = false }
            )
            .onAppear {
                ContentToolsObservability.record(
                    "render_error",
                    toolId: instance.toolId,
                    attributes: ["error_class": "crash"]
                )
            }
        } else {
            toolFrame(
                instance: instance,
                page: page,
                readOnly: readOnly,
                readOnlyMessage: readOnlyMessage,
                requiresAI: requiresAI,
                nonConformant: nonConformant
            ) {
                ToolRendererRegistry.view(
                    for: instance.toolId,
                    props: rendererProps(instance: instance, page: page, readOnly: readOnly)
                )
            }
        }
    }

    @ViewBuilder
    private func sandboxMounted(
        instance: ToolInstance,
        page: ContentToolsPageContext,
        readOnly: Bool,
        readOnlyMessage: String?,
        requiresAI: Bool,
        nonConformant: Bool
    ) -> some View {
        // Capability denial: never solicit OS permissions for denied caps (FR-2).
        let denied = Set((governance.settings?.policy?.deniedCapabilities ?? []).map { $0.lowercased() })
        let filteredCaps = instance.capabilities.filter {
            ContentToolGovernanceLogic.canObtainDeniedCapability(
                capability: $0,
                deniedCapabilities: Array(denied)
            )
        }
        toolFrame(
            instance: instance,
            page: page,
            readOnly: readOnly,
            readOnlyMessage: readOnlyMessage,
            requiresAI: requiresAI,
            nonConformant: nonConformant,
            showSandboxBadge: true
        ) {
            SandboxWebViewHost(
                toolId: instance.toolId,
                instanceId: instance.id,
                toolVersion: instance.toolVersion,
                title: ContentToolHostLogic.displayTitle(instance: instance, toolId: instance.toolId),
                config: instance.config,
                state: envelope.document,
                revision: envelope.revision,
                readOnly: readOnly,
                capabilities: filteredCaps,
                save: { next in
                    scheduleSave(next, page: page, toolId: instance.toolId, readOnly: readOnly)
                },
                runAction: { name, input in
                    try await runAction(name: name, input: input, page: page, instanceId: instance.id)
                },
                announce: { message, assertive in
                    ToolLiveRegion.announce(message, assertive: assertive)
                }
            )
        }
    }

    private func rendererProps(
        instance: ToolInstance,
        page: ContentToolsPageContext,
        readOnly: Bool
    ) -> ContentToolRendererProps {
        ContentToolRendererProps(
            instanceId: instance.id,
            toolId: instance.toolId,
            config: instance.config,
            state: envelope.document,
            status: envelope.status,
            readOnly: readOnly,
            save: { patch in
                let next = ContentToolHostLogic.mergeStatePatch(base: envelope.document, patch: patch)
                scheduleSave(next, page: page, toolId: instance.toolId, readOnly: readOnly)
            },
            submit: { patch in
                let next = ContentToolHostLogic.mergeStatePatch(base: envelope.document, patch: patch)
                Task { await persist(next, mode: "submit", page: page, toolId: instance.toolId, readOnly: readOnly) }
            },
            runAction: { name, input in
                try await runAction(name: name, input: input, page: page, instanceId: instance.id)
            },
            announce: { message, assertive in
                ToolLiveRegion.announce(message, assertive: assertive)
            }
        )
    }

    private func toolFrame<Content: View>(
        instance: ToolInstance,
        page: ContentToolsPageContext,
        readOnly: Bool,
        readOnlyMessage: String?,
        requiresAI: Bool,
        nonConformant: Bool,
        showSandboxBadge: Bool = false,
        @ViewBuilder content: @escaping () -> Content
    ) -> some View {
        let showDisclosure = requiresAI && ContentToolGovernanceLogic.shouldShowAIDisclosure(
            disclosureMode: consent?.aiDisclosureMode ?? governance.policy?.aiDisclosureMode,
            decision: consent?.decision,
            consentFetched: consentFetched
        )
        let aiAllowed = !requiresAI || ContentToolGovernanceLogic.aiActionsAllowed(
            disclosureMode: consent?.aiDisclosureMode ?? governance.policy?.aiDisclosureMode,
            decision: consent?.decision,
            consentFetched: consentFetched
        )

        return ToolFrameView(
            title: ContentToolHostLogic.displayTitle(instance: instance, toolId: instance.toolId),
            status: envelope.status,
            syncStatus: syncStatus,
            score: envelope.score,
            readOnly: readOnly,
            readOnlyMessage: readOnlyMessage ?? errorMessage,
            studentResetAllowed: studentResetAllowed,
            showSandboxBadge: showSandboxBadge,
            showNonConformantNote: nonConformant,
            canReport: true,
            canModerate: canModerate,
            onReset: { showResetConfirm = true },
            onReport: { showReport = true },
            onModerate: {
                Task { await openModeration() }
            },
            disclosure: {
                Group {
                    if requiresAI {
                        Color.clear.frame(height: 0).task(id: instance.toolId) {
                            await loadConsent(toolId: instance.toolId, page: page)
                        }
                    }
                    if showDisclosure {
                        AIDisclosureBanner(
                            mode: consent?.aiDisclosureMode ?? "acknowledge",
                            busy: consentBusy,
                            onAcknowledge: {
                                Task { await postConsent(decision: "acknowledged", toolId: instance.toolId, page: page) }
                            },
                            onOptOut: {
                                Task { await postConsent(decision: "opted_out", toolId: instance.toolId, page: page) }
                            }
                        )
                    } else if requiresAI && !aiAllowed {
                        ConsentGateView(busy: consentBusy) {
                            Task { await postConsent(decision: "acknowledged", toolId: instance.toolId, page: page) }
                        }
                        .onAppear {
                            ContentToolsObservability.record(
                                "ai_blocked_by_consent",
                                toolId: instance.toolId
                            )
                        }
                    }
                    if showCrisis {
                        CrisisResourcesView()
                    }
                }
            },
            content: content
        )
    }

    private func openWeb(page: ContentToolsPageContext, instanceId: String) {
        let path = ContentToolHostLogic.webActivityPath(
            courseCode: page.courseCode,
            itemId: page.itemId,
            instanceId: instanceId
        )
        openURL(AppConfiguration.webURL(path: path))
    }

    private func loadConsent(toolId: String, page: ContentToolsPageContext) async {
        guard let token = session.accessToken else {
            consentFetched = false
            return
        }
        do {
            consent = try await LMSAPI.fetchContentToolAIConsent(
                courseCode: page.courseCode,
                toolId: toolId,
                accessToken: token
            )
            consentFetched = true
        } catch {
            consent = nil
            consentFetched = false
        }
    }

    private func postConsent(decision: String, toolId: String, page: ContentToolsPageContext) async {
        guard let token = session.accessToken else { return }
        consentBusy = true
        defer { consentBusy = false }
        do {
            consent = try await LMSAPI.postContentToolAIConsent(
                courseCode: page.courseCode,
                toolId: toolId,
                decision: decision,
                accessToken: token
            )
            consentFetched = true
        } catch {
            errorMessage = L.text("mobile.contentTools.governance.consentError")
        }
    }

    private func submitReport(category: String, note: String?) async -> Bool {
        guard let page, let token = session.accessToken else { return false }
        do {
            _ = try await LMSAPI.reportContentToolContent(
                courseCode: page.courseCode,
                instanceId: instanceId,
                category: category,
                reason: note,
                contentPath: nil,
                accessToken: token
            )
            ContentToolsObservability.record(
                "report_submitted",
                toolId: toolId,
                attributes: ["outcome": "ok"]
            )
            ToolLiveRegion.announce(L.text("mobile.contentTools.governance.reportThanks"))
            return true
        } catch {
            ContentToolsObservability.record(
                "report_submitted",
                toolId: toolId,
                attributes: ["outcome": "error"]
            )
            return false
        }
    }

    private func openModeration() async {
        guard let page, let token = session.accessToken else { return }
        do {
            moderationItems = try await LMSAPI.fetchContentToolModeration(
                courseCode: page.courseCode,
                instanceId: instanceId,
                accessToken: token
            )
            showModerate = true
        } catch let APIError.httpStatus(status, _) where status == 403 {
            errorMessage = L.text("mobile.contentTools.governance.moderateForbidden")
            ToolLiveRegion.announce(L.text("mobile.contentTools.governance.moderateForbidden"), assertive: true)
        } catch {
            errorMessage = L.text("mobile.contentTools.governance.moderateError")
        }
    }

    private func submitModerate(action: String) async -> Bool {
        guard let page, let token = session.accessToken else { return false }
        do {
            _ = try await LMSAPI.moderateContentToolContent(
                courseCode: page.courseCode,
                instanceId: instanceId,
                action: action,
                category: nil,
                reason: nil,
                contentPath: nil,
                accessToken: token
            )
            return true
        } catch let APIError.httpStatus(status, _) where status == 403 {
            errorMessage = L.text("mobile.contentTools.governance.moderateForbidden")
            return false
        } catch {
            return false
        }
    }

    private func scheduleSave(_ next: JSONValue, page: ContentToolsPageContext, toolId: String, readOnly: Bool) {
        pending = next
        dirty = true
        syncStatus = ContentToolHostLogic.syncStatusAfterEdit(syncStatus)
        saveTask?.cancel()
        saveTask = Task {
            try? await Task.sleep(nanoseconds: UInt64(ContentToolHostLogic.defaultDebounceMs) * 1_000_000)
            guard !Task.isCancelled, let pending else { return }
            await persist(pending, mode: "save", page: page, toolId: toolId, readOnly: readOnly)
        }
    }

    private func persist(
        _ nextState: JSONValue,
        mode: String,
        page: ContentToolsPageContext,
        toolId: String,
        readOnly: Bool
    ) async {
        guard let token = session.accessToken, !readOnly else { return }
        if saving {
            pending = nextState
            dirty = true
            return
        }
        saving = true
        syncStatus = .saving
        errorMessage = nil
        let revision = envelope.revision
        defer {
            saving = false
            if dirty, let pending {
                self.pending = nil
                Task { await persist(pending, mode: "save", page: page, toolId: toolId, readOnly: readOnly) }
            }
        }
        do {
            let result = try await writeState(
                nextState,
                mode: mode,
                page: page,
                revision: revision,
                token: token
            )
            applySaved(result)
            ContentToolsObservability.record(
                "state_save",
                toolId: toolId,
                attributes: ["outcome": "ok"]
            )
        } catch is ContentToolHostPersistCancel {
            ContentToolsObservability.record(
                "offline_replay",
                toolId: toolId,
                attributes: ["outcome": "queued"]
            )
        } catch let LMSAPI.ContentToolAPIError.revisionConflict(current) {
            ContentToolsObservability.record(
                "revision_conflict",
                toolId: toolId,
                attributes: ["outcome": "conflict"]
            )
            await applyConflict(current, client: nextState, page: page, toolId: toolId, token: token)
        } catch LMSAPI.ContentToolAPIError.stateTooLarge {
            ContentToolsObservability.record(
                "state_save",
                toolId: toolId,
                attributes: ["outcome": "error", "error_class": "too_large"]
            )
            applyHardError(L.text("mobile.contentTools.runtime.stateTooLarge"))
        } catch LMSAPI.ContentToolAPIError.schemaInvalid {
            applyHardError(L.text("mobile.contentTools.runtime.schemaInvalid"))
        } catch {
            dirty = true
            syncStatus = .unsynced
            errorMessage = L.text("mobile.contentTools.runtime.retry")
        }
    }

    private func writeState(
        _ nextState: JSONValue,
        mode: String,
        page: ContentToolsPageContext,
        revision: Int64,
        token: String
    ) async throws -> ToolStateEnvelope {
        if mode == "submit" {
            return try await LMSAPI.submitContentToolState(
                courseCode: page.courseCode,
                instanceId: instanceId,
                revision: revision,
                state: nextState,
                accessToken: token
            )
        }
        do {
            return try await LMSAPI.putContentToolState(
                courseCode: page.courseCode,
                instanceId: instanceId,
                revision: revision,
                state: nextState,
                accessToken: token
            )
        } catch let error as APIError {
            if case .transport = error, ContentToolHostLogic.canQueueStateWriteOffline() {
                _ = try await offline.enqueueMutation(
                    method: "PUT",
                    path: LMSAPI.contentToolStatePutPath(courseCode: page.courseCode, instanceId: instanceId),
                    body: SaveToolStateBody(revision: revision, state: nextState, stateJson: nextState),
                    label: "content-tool-state:\(instanceId)",
                    accessToken: token,
                    preferQueue: true
                )
                syncStatus = .unsynced
                dirty = true
                pending = nextState
                ToolLiveRegion.announce(L.text("mobile.contentTools.runtime.unsynced"))
                throw ContentToolHostPersistCancel()
            }
            throw error
        }
    }

    private func applySaved(_ result: ToolStateEnvelope) {
        envelope = result
        dirty = false
        pending = nil
        syncStatus = .saved
        ToolLiveRegion.announce(L.text("mobile.contentTools.runtime.saved"))
    }

    private func applyHardError(_ message: String) {
        dirty = true
        syncStatus = .error
        errorMessage = message
        ToolLiveRegion.announce(message, assertive: true)
    }

    private func applyConflict(
        _ current: ToolStateEnvelope,
        client: JSONValue,
        page: ContentToolsPageContext,
        toolId: String,
        token: String
    ) async {
        let policy = ContentToolHostLogic.conflictPolicyForTool(toolId)
        let resolved = ContentToolHostLogic.resolveConflictJSON(
            policy: policy,
            client: client,
            server: current.document
        )
        envelope = ToolStateEnvelope(
            instanceId: current.instanceId,
            revision: current.revision,
            status: current.status,
            state: resolved,
            stateJson: resolved,
            score: current.score,
            updatedAt: current.updatedAt,
            resetCount: current.resetCount,
            lastResetAt: current.lastResetAt,
            scope: current.scope,
            stateSchemaVersion: current.stateSchemaVersion,
            quarantined: current.quarantined
        )
        if policy == .serverWins {
            applySaved(envelope)
            return
        }
        do {
            let retry = try await LMSAPI.putContentToolState(
                courseCode: page.courseCode,
                instanceId: instanceId,
                revision: current.revision,
                state: resolved,
                accessToken: token
            )
            applySaved(retry)
        } catch {
            dirty = true
            syncStatus = .unsynced
            errorMessage = L.text("mobile.contentTools.runtime.retry")
        }
    }

    private func runAction(
        name: String,
        input: JSONValue,
        page: ContentToolsPageContext,
        instanceId: String
    ) async throws -> JSONValue? {
        guard let token = session.accessToken else {
            ToolLiveRegion.announce(L.text("mobile.contentTools.runtime.needsConnection"), assertive: true)
            throw APIError.transport(URLError(.notConnectedToInternet))
        }
        let key = actionKeys[name] ?? ContentToolHostLogic.newIdempotencyKey()
        actionKeys[name] = key
        do {
            let res = try await LMSAPI.runContentToolAction(
                courseCode: page.courseCode,
                instanceId: instanceId,
                action: name,
                input: input,
                accessToken: token,
                idempotencyKey: key
            )
            if let state = res.state { envelope = state }
            actionKeys[name] = nil
            if let result = res.result {
                handleActionResult(result, toolId: toolId)
            }
            ContentToolsObservability.record(
                "action_outcome",
                toolId: toolId,
                attributes: ["outcome": "ok"]
            )
            return res.result
        } catch APIError.transport {
            ToolLiveRegion.announce(L.text("mobile.contentTools.runtime.needsConnection"), assertive: true)
            ContentToolsObservability.record(
                "action_outcome",
                toolId: toolId,
                attributes: ["outcome": "error", "error_class": "offline"]
            )
            throw APIError.transport(URLError(.notConnectedToInternet))
        } catch {
            ContentToolsObservability.record(
                "action_outcome",
                toolId: toolId,
                attributes: ["outcome": "error", "error_class": "unknown"]
            )
            throw error
        }
    }

    private func handleActionResult(_ result: JSONValue, toolId: String) {
        guard case .object(let obj) = result else { return }
        let code: String? = {
            if case .string(let errorCode) = obj["error"] { return errorCode }
            if case .string(let statusCode) = obj["code"] { return statusCode }
            return nil
        }()
        let crisis: Bool = {
            if case .bool(let flagged) = obj["crisis"] { return flagged }
            return false
        }()
        let outcome = ContentToolGovernanceLogic.filterCrisisOutcome(
            ContentToolGovernanceLogic.FilterCrisisInput(errorCode: code, crisis: crisis)
        )
        switch outcome.kind {
        case .crisis:
            showCrisis = true
            errorMessage = L.text("mobile.contentTools.governance.crisisBody")
            ToolLiveRegion.announce(L.text("mobile.contentTools.governance.crisisTitle"), assertive: true)
        case .filtered:
            // Plain language — do not echo blocked content.
            errorMessage = L.text("mobile.contentTools.governance.filtered")
            ToolLiveRegion.announce(L.text("mobile.contentTools.governance.filtered"), assertive: true)
        case .generic:
            break
        }
        _ = toolId
    }

    private func selfReset() async {
        guard let page, let token = session.accessToken else { return }
        do {
            envelope = try await LMSAPI.selfResetContentTool(
                courseCode: page.courseCode,
                instanceId: instanceId,
                accessToken: token
            )
            syncStatus = .saved
        } catch {
            errorMessage = L.text("mobile.contentTools.runtime.retry")
        }
    }
}

/// Thrown when a state write was intentionally queued offline (not a user-facing failure).
private struct ContentToolHostPersistCancel: Error {}
