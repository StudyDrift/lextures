import SwiftUI

/// Page-scoped context for one batched instances fetch (FR-2).
struct ContentToolsPageContext: Equatable {
    var courseCode: String
    var itemId: String
    var contentToolsEnabled: Bool
    var mobileContentToolsEnabled: Bool
    /// CT.M4 sandbox WebView host capability (independent of CT.M3).
    var mobileContentToolsSandboxEnabled: Bool = false
    var observer: Bool = false
    var pastDue: Bool = false
}

/// CT.M9 — cached course settings + governance snapshot for mount gating.
struct ContentToolsGovernanceContext: Equatable {
    var settings: ContentToolSettings?
    var fetchedAtMs: Int64 = 0
    var fetchSucceeded = false
    var nonConformantToolIds: Set<String> = []

    var policy: ToolGovernancePolicy? { settings?.policy }

    var ageMs: Int64 {
        guard fetchedAtMs > 0 else { return Int64.max }
        let now = Int64(Date().timeIntervalSince1970 * 1000)
        return max(0, now - fetchedAtMs)
    }
}

private struct ContentToolsPageContextKey: EnvironmentKey {
    static let defaultValue: ContentToolsPageContext? = nil
}

private struct ContentToolsInstancesKey: EnvironmentKey {
    static let defaultValue: [String: ToolInstance] = [:]
}

private struct ContentToolsLoadingKey: EnvironmentKey {
    static let defaultValue: Bool = false
}

private struct ContentToolsStudentResetKey: EnvironmentKey {
    static let defaultValue: Bool = false
}

private struct ContentToolsGovernanceKey: EnvironmentKey {
    static let defaultValue = ContentToolsGovernanceContext()
}

private struct ContentToolsCanModerateKey: EnvironmentKey {
    static let defaultValue: Bool = false
}

private struct ContentToolsRefreshGovernanceKey: EnvironmentKey {
    static let defaultValue: (() -> Void)? = nil
}

extension EnvironmentValues {
    var contentToolsPage: ContentToolsPageContext? {
        get { self[ContentToolsPageContextKey.self] }
        set { self[ContentToolsPageContextKey.self] = newValue }
    }

    var contentToolsInstances: [String: ToolInstance] {
        get { self[ContentToolsInstancesKey.self] }
        set { self[ContentToolsInstancesKey.self] = newValue }
    }

    var contentToolsLoading: Bool {
        get { self[ContentToolsLoadingKey.self] }
        set { self[ContentToolsLoadingKey.self] = newValue }
    }

    var contentToolsStudentResetAllowed: Bool {
        get { self[ContentToolsStudentResetKey.self] }
        set { self[ContentToolsStudentResetKey.self] = newValue }
    }

    var contentToolsGovernance: ContentToolsGovernanceContext {
        get { self[ContentToolsGovernanceKey.self] }
        set { self[ContentToolsGovernanceKey.self] = newValue }
    }

    var contentToolsCanModerate: Bool {
        get { self[ContentToolsCanModerateKey.self] }
        set { self[ContentToolsCanModerateKey.self] = newValue }
    }

    var contentToolsRefreshGovernance: (() -> Void)? {
        get { self[ContentToolsRefreshGovernanceKey.self] }
        set { self[ContentToolsRefreshGovernanceKey.self] = newValue }
    }
}

/// Loads instances once per item and injects them for fence hosts.
struct ContentToolsPageProvider<Content: View>: View {
    @Environment(AuthSession.self) private var session
    @Environment(OfflineService.self) private var offline
    @Environment(\.scenePhase) private var scenePhase

    let context: ContentToolsPageContext
    @ViewBuilder var content: () -> Content

    @State private var instances: [String: ToolInstance] = [:]
    @State private var loading = false
    @State private var studentResetAllowed = false
    @State private var governance = ContentToolsGovernanceContext()
    @State private var canModerate = false
    @State private var refreshTick = 0

    var body: some View {
        content()
            .environment(\.contentToolsPage, context)
            .environment(\.contentToolsInstances, instances)
            .environment(\.contentToolsLoading, loading)
            .environment(\.contentToolsStudentResetAllowed, studentResetAllowed)
            .environment(\.contentToolsGovernance, governance)
            .environment(\.contentToolsCanModerate, canModerate)
            .environment(\.contentToolsRefreshGovernance, { refreshTick += 1 })
            .task(id: "\(context.courseCode)|\(context.itemId)|\(context.contentToolsEnabled)|\(context.mobileContentToolsEnabled)|\(refreshTick)") {
                await load()
            }
            .onChange(of: scenePhase) { _, phase in
                // FR-4: re-evaluate policy on foreground so kills apply without an app release.
                if phase == .active {
                    refreshTick += 1
                }
            }
            .onChange(of: context.courseCode) { _, _ in
                refreshTick += 1
            }
    }

    private func load() async {
        let shouldFetch = ContentToolHostLogic.shouldFetchInstances(
            mobileContentToolsEnabled: context.mobileContentToolsEnabled,
            contentToolsEnabled: context.contentToolsEnabled,
            courseCode: context.courseCode,
            itemId: context.itemId
        )
        guard shouldFetch, let token = session.accessToken else {
            instances = [:]
            loading = false
            return
        }
        loading = true
        defer { loading = false }
        do {
            let list = try await offline.cachedFetch(
                key: OfflineCacheKey.contentToolInstances(
                    courseCode: context.courseCode,
                    itemId: context.itemId
                ),
                accessToken: token
            ) {
                try await LMSAPI.fetchContentToolInstances(
                    courseCode: context.courseCode,
                    accessToken: token,
                    itemId: context.itemId,
                    withState: true
                )
            }.value
            instances = ContentToolHostLogic.instanceMap(list)

            // Settings + governance snapshot (CT.M9) — fail closed for AI/third-party when missing.
            let priorNonConformant = governance.nonConformantToolIds
            let priorSettings = governance.settings
            let priorFetchedAt = governance.fetchedAtMs
            if let settings = try? await LMSAPI.fetchContentToolSettings(
                courseCode: context.courseCode,
                accessToken: token
            ) {
                studentResetAllowed = settings.studentResetAllowed
                governance = ContentToolsGovernanceContext(
                    settings: settings,
                    fetchedAtMs: Int64(Date().timeIntervalSince1970 * 1000),
                    fetchSucceeded: true,
                    nonConformantToolIds: priorNonConformant
                )
            } else if let priorSettings {
                // Keep cached policy; mark fetch failed so staleness rules apply.
                governance = ContentToolsGovernanceContext(
                    settings: priorSettings,
                    fetchedAtMs: priorFetchedAt,
                    fetchSucceeded: false,
                    nonConformantToolIds: priorNonConformant
                )
            } else {
                governance = ContentToolsGovernanceContext(
                    settings: nil,
                    fetchedAtMs: 0,
                    fetchSucceeded: false,
                    nonConformantToolIds: []
                )
            }

            // Staff moderation: non-observers may attempt; 403 handled in sheet (FR-11).
            canModerate = !context.observer

            if let conf = try? await LMSAPI.fetchContentToolConformance(accessToken: token) {
                let bad = Set(conf.tools.filter { !$0.ok }.map(\.toolId))
                governance = ContentToolsGovernanceContext(
                    settings: governance.settings,
                    fetchedAtMs: governance.fetchedAtMs,
                    fetchSucceeded: governance.fetchSucceeded,
                    nonConformantToolIds: bad
                )
            }
        } catch {
            instances = [:]
        }
    }
}
