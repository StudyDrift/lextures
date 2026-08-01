import Foundation

/// CT.M9 FR-17/18 — content-free mobile content-tools counters.
enum ContentToolsObservability {
    private static var counters: [String: Int] = [:]
    private static let lock = NSLock()
    static let platform = "ios"

    static func record(_ event: String, toolId: String? = nil, attributes: [String: String] = [:]) {
        var attrs = attributes
        attrs["platform"] = platform
        if let toolId, !toolId.isEmpty {
            attrs["tool_id"] = toolId
        }
        attrs["event"] = event
        guard ContentToolGovernanceLogic.telemetryAttributesAreContentFree(attrs) else {
            // Drop payloads that would violate FR-18 rather than emit learner content.
            return
        }
        lock.lock()
        defer { lock.unlock() }
        let key = attrs.isEmpty
            ? event
            : event + "|" + attrs.keys.sorted().map { "\($0)=\(attrs[$0] ?? "")" }.joined(separator: ",")
        counters[key, default: 0] += 1
    }

    static func count(for event: String) -> Int {
        lock.lock()
        defer { lock.unlock() }
        return counters.filter { $0.key == event || $0.key.hasPrefix(event + "|") }.values.reduce(0, +)
    }

    #if DEBUG
    static func resetForTests() {
        lock.lock()
        counters.removeAll()
        lock.unlock()
    }

    static func lastAttributes(for event: String) -> [String: String]? {
        lock.lock()
        defer { lock.unlock() }
        guard let key = counters.keys.first(where: { $0 == event || $0.hasPrefix(event + "|") }) else {
            return nil
        }
        guard let pipe = key.firstIndex(of: "|") else { return ["event": event] }
        let rest = key[key.index(after: pipe)...]
        var out: [String: String] = ["event": event]
        for part in rest.split(separator: ",") {
            let bits = part.split(separator: "=", maxSplits: 1).map(String.init)
            if bits.count == 2 { out[bits[0]] = bits[1] }
        }
        return out
    }
    #endif
}
