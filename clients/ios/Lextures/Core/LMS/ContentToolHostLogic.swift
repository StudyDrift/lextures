import Foundation

/// Pure CT.M3 host decisions — fence mapping, debounce, conflict, read-only reasons,
/// contract gating, and render gates. No networking.
enum ContentToolHostLogic {
    static let defaultDebounceMs = 1500
    static let minDebounceMs = 500
    static let maxDebounceMs = 10_000
    static let runtimeContractVersion = 1

    enum ConflictPolicy: String, Equatable {
        case serverWins = "server_wins"
        case clientWins = "client_wins"
        case merge = "merge"

        static func from(_ raw: String?) -> ConflictPolicy {
            switch raw?.lowercased() {
            case "client_wins": return .clientWins
            case "merge": return .merge
            default: return .serverWins
            }
        }
    }

    enum ReadOnlyReason: String, Equatable {
        case tombstone
        case breaker
        case archived
        case pastDue = "past_due"
        case observer
        case quarantined
        case flagOff = "flag_off"
    }

    enum SyncStatus: String, Equatable {
        case idle
        case saving
        case saved
        case unsynced
        case error
    }

    enum FenceRenderMode: String, Equatable {
        /// Client dark-launch flag off — keep CT.M1 placeholder.
        case legacyPlaceholder = "legacy_placeholder"
        /// Course contentToolsEnabled false — render nothing (parity with web).
        case hidden
        /// Mount the CT.M3 host.
        case host
    }

    struct ReadOnlyInput: Equatable {
        var tombstone = false
        var breakerOpen = false
        var status = "published"
        var pastDue = false
        var respectsDueDate = false
        var observer = false
        var quarantined = false
        var courseFlagOffAfterLoad = false
    }

    static func clampDebounceMs(_ ms: Int?) -> Int {
        guard let ms else { return defaultDebounceMs }
        return min(maxDebounceMs, max(minDebounceMs, ms))
    }

    static func clampDebounceMs(_ ms: Double?) -> Int {
        guard let ms, ms.isFinite else { return defaultDebounceMs }
        return clampDebounceMs(Int(ms.rounded()))
    }

    static func instanceMap(_ instances: [ToolInstance]) -> [String: ToolInstance] {
        Dictionary(uniqueKeysWithValues: instances.map { ($0.id, $0) })
    }

    static func resolveInstance(_ instances: [String: ToolInstance], instanceId: String) -> ToolInstance? {
        instances[instanceId]
    }

    /// Missing / invisible fences render nothing — never an error (FR-3).
    static func shouldMountFence(_ instance: ToolInstance?) -> Bool {
        instance != nil
    }

    static func fenceRenderMode(
        mobileContentToolsEnabled: Bool,
        contentToolsEnabled: Bool
    ) -> FenceRenderMode {
        if !mobileContentToolsEnabled { return .legacyPlaceholder }
        if !contentToolsEnabled { return .hidden }
        return .host
    }

    static func shouldFetchInstances(
        mobileContentToolsEnabled: Bool,
        contentToolsEnabled: Bool,
        courseCode: String?,
        itemId: String?
    ) -> Bool {
        mobileContentToolsEnabled
            && contentToolsEnabled
            && !(courseCode?.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty ?? true)
            && !(itemId?.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty ?? true)
    }

    static func contractSupported(_ contract: Int, supported: Int = runtimeContractVersion) -> Bool {
        contract == supported
    }

    static func conflictPolicyForTool(_ toolId: String, manifestPolicy: String? = nil) -> ConflictPolicy {
        if ContentToolPack1Logic.pack1ToolIds.contains(toolId) {
            return ContentToolPack1Logic.conflictPolicy(for: toolId)
        }
        if ContentToolPack2Logic.pack2ToolIds.contains(toolId) {
            return ContentToolPack2Logic.conflictPolicy(for: toolId)
        }
        if ContentToolPack3Logic.pack3ToolIds.contains(toolId) {
            return ContentToolPack3Logic.conflictPolicy(for: toolId)
        }
        return .from(manifestPolicy)
    }

    static func defaultMerge(
        client: [String: JSONValue],
        server: [String: JSONValue]
    ) -> [String: JSONValue] {
        var out = server
        for (k, v) in client { out[k] = v }
        return out
    }

    static func resolveConflictState(
        policy: ConflictPolicy,
        client: [String: JSONValue],
        server: [String: JSONValue],
        mergeFn: ([String: JSONValue], [String: JSONValue]) -> [String: JSONValue] = defaultMerge
    ) -> [String: JSONValue] {
        switch policy {
        case .serverWins: return server
        case .clientWins: return client
        case .merge: return mergeFn(client, server)
        }
    }

    static func resolveConflictJSON(
        policy: ConflictPolicy,
        client: JSONValue,
        server: JSONValue
    ) -> JSONValue {
        let c: [String: JSONValue]
        let s: [String: JSONValue]
        if case .object(let obj) = client { c = obj } else { c = [:] }
        if case .object(let obj) = server { s = obj } else { s = [:] }
        return .object(resolveConflictState(policy: policy, client: c, server: s))
    }

    static func readOnlyReason(_ input: ReadOnlyInput) -> ReadOnlyReason? {
        if input.tombstone { return .tombstone }
        if input.breakerOpen { return .breaker }
        if input.status.lowercased() == "archived" { return .archived }
        if input.pastDue && input.respectsDueDate { return .pastDue }
        if input.observer { return .observer }
        if input.quarantined { return .quarantined }
        if input.courseFlagOffAfterLoad { return .flagOff }
        return nil
    }

    static func readOnlyReason(
        instance: ToolInstance,
        pastDue: Bool = false,
        respectsDueDate: Bool = false,
        observer: Bool = false,
        courseFlagOffAfterLoad: Bool = false
    ) -> ReadOnlyReason? {
        readOnlyReason(
            ReadOnlyInput(
                tombstone: instance.tombstone,
                breakerOpen: instance.breakerOpen,
                status: instance.status,
                pastDue: pastDue,
                respectsDueDate: respectsDueDate,
                observer: observer,
                quarantined: instance.state?.quarantined == true,
                courseFlagOffAfterLoad: courseFlagOffAfterLoad
            )
        )
    }

    static func readOnlyMessageKey(_ reason: ReadOnlyReason) -> String {
        switch reason {
        case .tombstone, .breaker, .quarantined, .flagOff:
            return "mobile.contentTools.runtime.unavailable"
        case .archived:
            return "mobile.contentTools.runtime.readOnlyArchived"
        case .pastDue:
            return "mobile.contentTools.runtime.readOnlyPastDue"
        case .observer:
            return "mobile.contentTools.runtime.readOnlyPreview"
        }
    }

    static func syncStatusAfterEdit(_ current: SyncStatus) -> SyncStatus {
        current == .saving ? .saving : .unsynced
    }

    static func newIdempotencyKey() -> String {
        UUID().uuidString
    }

    /// Actions must not be queued offline (FR-11).
    static func canQueueActionOffline() -> Bool { false }

    static func canQueueStateWriteOffline() -> Bool { true }

    /// Per-instance ordered replay helper.
    static func orderOutboxByInstance(_ items: [(String, Int64)]) -> [(String, Int64)] {
        items.sorted {
            if $0.0 != $1.0 { return $0.0 < $1.0 }
            return $0.1 < $1.1
        }
    }

    static func webActivityPath(courseCode: String, itemId: String, instanceId: String) -> String {
        "/courses/\(courseCode.trimmingCharacters(in: .whitespacesAndNewlines))/modules/items/\(itemId.trimmingCharacters(in: .whitespacesAndNewlines))#lex-tool-\(instanceId)"
    }

    static func displayTitle(instance: ToolInstance?, toolId: String) -> String {
        let title = instance?.title?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        return title.isEmpty ? toolId : title
    }

    static func statusChip(_ status: String) -> String {
        status.isEmpty ? "not_started" : status
    }

    static func accessibleName(title: String, status: String) -> String {
        "\(title), \(statusChip(status).replacingOccurrences(of: "_", with: " "))"
    }

    static func registeredNativeToolIds() -> Set<String> {
        var ids: Set<String> = ["noop_probe"]
        ids.formUnion(ContentToolPack1Logic.allowlistedToolIds())
        ids.formUnion(ContentToolPack2Logic.allowlistedToolIds())
        ids.formUnion(ContentToolPack3Logic.allowlistedToolIds())
        return ids
    }

    static func hasNativeRenderer(_ toolId: String, registered: Set<String> = registeredNativeToolIds()) -> Bool {
        registered.contains(toolId)
    }

    static func shouldShowUnsupportedPlaceholder(
        toolId: String,
        contract: Int,
        registered: Set<String> = registeredNativeToolIds()
    ) -> Bool {
        !hasNativeRenderer(toolId, registered: registered) || !contractSupported(contract)
    }

    static func mergeStatePatch(base: JSONValue, patch: [String: JSONValue]) -> JSONValue {
        var map: [String: JSONValue]
        if case .object(let obj) = base { map = obj } else { map = [:] }
        for (k, v) in patch { map[k] = v }
        return .object(map)
    }

    static func stringField(_ value: JSONValue?, key: String) -> String? {
        guard case .object(let obj) = value, let field = obj[key] else { return nil }
        if case .string(let s) = field { return s }
        return nil
    }
}
