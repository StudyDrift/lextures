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
}

/// Loads instances once per item and injects them for fence hosts.
struct ContentToolsPageProvider<Content: View>: View {
    @Environment(AuthSession.self) private var session
    @Environment(OfflineService.self) private var offline

    let context: ContentToolsPageContext
    @ViewBuilder var content: () -> Content

    @State private var instances: [String: ToolInstance] = [:]
    @State private var loading = false
    @State private var studentResetAllowed = false

    var body: some View {
        content()
            .environment(\.contentToolsPage, context)
            .environment(\.contentToolsInstances, instances)
            .environment(\.contentToolsLoading, loading)
            .environment(\.contentToolsStudentResetAllowed, studentResetAllowed)
            .task(id: "\(context.courseCode)|\(context.itemId)|\(context.contentToolsEnabled)|\(context.mobileContentToolsEnabled)") {
                await load()
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
            if let settings = try? await LMSAPI.fetchContentToolSettings(
                courseCode: context.courseCode,
                accessToken: token
            ) {
                studentResetAllowed = settings.studentResetAllowed
            }
        } catch {
            instances = [:]
        }
    }
}
