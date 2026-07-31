import SwiftUI

/// Mounts one tool fence inside the page host.
struct ContentToolHostView: View {
    @Environment(AuthSession.self) private var session
    @Environment(OfflineService.self) private var offline
    @Environment(\.contentToolsPage) private var page
    @Environment(\.contentToolsInstances) private var instances
    @Environment(\.contentToolsLoading) private var loading
    @Environment(\.contentToolsStudentResetAllowed) private var studentResetAllowed
    @Environment(\.openURL) private var openURL

    let instanceId: String
    let toolId: String

    @State private var crashed = false
    @State private var showResetConfirm = false
    @State private var envelope: ToolStateEnvelope
    @State private var syncStatus: ContentToolHostLogic.SyncStatus = .idle
    @State private var errorMessage: String?
    @State private var pending: JSONValue?
    @State private var dirty = false
    @State private var saving = false
    @State private var saveTask: Task<Void, Never>?
    @State private var actionKeys: [String: String] = [:]

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
        let reason = ContentToolHostLogic.readOnlyReason(
            instance: instance,
            pastDue: page.pastDue,
            respectsDueDate: false,
            observer: page.observer
        )
        let readOnly = reason != nil
        let readOnlyMessage = reason.map { L.text(String.LocalizationValue(ContentToolHostLogic.readOnlyMessageKey($0))) }

        if instance.tombstone || instance.breakerOpen {
            ToolPlaceholderView(
                reason: instance.breakerOpen ? .maintenance : .unavailable,
                toolName: ContentToolHostLogic.displayTitle(instance: instance, toolId: instance.toolId)
            )
        } else {
            let path = ContentToolSandboxLogic.resolveRenderPath(
                toolId: instance.toolId,
                contract: instance.contract,
                sandboxMode: instance.sandboxMode,
                sandboxEnabled: page.mobileContentToolsSandboxEnabled,
                registered: ToolRendererRegistry.registeredIds(),
                tombstone: instance.tombstone,
                breakerOpen: instance.breakerOpen,
                deprecated: instance.deprecated,
                killed: false
            )
            switch path {
            case .placeholder:
                ToolPlaceholderView(
                    reason: .openInBrowser,
                    toolName: ContentToolHostLogic.displayTitle(instance: instance, toolId: instance.toolId),
                    onOpenInBrowser: { openWeb(page: page, instanceId: instance.id) }
                )
            case .native:
                if crashed {
                    ToolErrorCardView(
                        toolName: ContentToolHostLogic.displayTitle(instance: instance, toolId: instance.toolId),
                        onRetry: { crashed = false }
                    )
                } else {
                    toolFrame(instance: instance, page: page, readOnly: readOnly, readOnlyMessage: readOnlyMessage) {
                        ToolRendererRegistry.view(
                            for: instance.toolId,
                            props: rendererProps(instance: instance, page: page, readOnly: readOnly)
                        )
                    }
                }
            case .sandbox:
                toolFrame(
                    instance: instance,
                    page: page,
                    readOnly: readOnly,
                    readOnlyMessage: readOnlyMessage,
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
                        capabilities: instance.capabilities,
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
        showSandboxBadge: Bool = false,
        @ViewBuilder content: () -> Content
    ) -> some View {
        ToolFrameView(
            title: ContentToolHostLogic.displayTitle(instance: instance, toolId: instance.toolId),
            status: envelope.status,
            syncStatus: syncStatus,
            score: envelope.score,
            readOnly: readOnly,
            readOnlyMessage: readOnlyMessage ?? errorMessage,
            studentResetAllowed: studentResetAllowed,
            showSandboxBadge: showSandboxBadge,
            onReset: { showResetConfirm = true }
        ) {
            content()
        }
    }

    private func openWeb(page: ContentToolsPageContext, instanceId: String) {
        let path = ContentToolHostLogic.webActivityPath(
            courseCode: page.courseCode,
            itemId: page.itemId,
            instanceId: instanceId
        )
        openURL(AppConfiguration.webURL(path: path))
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
        } catch is ContentToolHostPersistCancel {
            // State write was queued offline; status already updated.
        } catch let LMSAPI.ContentToolAPIError.revisionConflict(current) {
            await applyConflict(current, client: nextState, page: page, toolId: toolId, token: token)
        } catch LMSAPI.ContentToolAPIError.stateTooLarge {
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
            return res.result
        } catch let APIError.transport {
            ToolLiveRegion.announce(L.text("mobile.contentTools.runtime.needsConnection"), assertive: true)
            throw APIError.transport(URLError(.notConnectedToInternet))
        }
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
