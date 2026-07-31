import Foundation

/// Pure CT.M4 sandbox decisions — bridge validation, rate/size guards, height clamp,
/// opaque participant ids, and native → sandbox → placeholder resolution. No networking.
enum ContentToolSandboxLogic {
    static let bridgeVersion = 1
    static let bridgeMaxMessageBytes = 64 * 1024
    static let bridgeMaxMessagesPerSec = 20
    static let readyTimeoutMs = 10_000
    static let minHeightPt = 80.0
    static let maxHeightPt = 2000.0
    static let maxLiveWebViews = 3
    static let announceMaxPerSec = 5
    static let supportedReadyContract = "1"

    private static let fromToolTypes: Set<String> = ["ready", "save", "runAction", "resize", "announce"]
    private static let toToolTypes: Set<String> = ["init", "stateAccepted", "actionResult", "error"]

    enum RenderPath: String, Equatable {
        case native
        case sandbox
        case placeholder
    }

    enum RejectionReason: String, Equatable {
        case oversized
        case rateLimited = "rate_limited"
        case malformed
        case unknownType = "unknown_type"
    }

    struct ResolutionInput: Equatable {
        var toolId: String
        var contract: Int
        var sandboxMode: String?
        var sandboxEnabled: Bool
        var registered: Set<String>
        var tombstone: Bool = false
        var breakerOpen: Bool = false
        var deprecated: Bool = false
        var killed: Bool = false
    }

    // MARK: - Resolution (FR-1)

    static func hasNativeRenderer(_ toolId: String, registered: Set<String>) -> Bool {
        registered.contains(toolId)
    }

    static func isSandboxable(_ input: ResolutionInput) -> Bool {
        guard input.sandboxEnabled else { return false }
        if input.tombstone || input.breakerOpen || input.deprecated || input.killed { return false }
        return ContentToolHostLogic.contractSupported(input.contract)
    }

    /// Pure resolution: native → sandbox → placeholder.
    static func resolveRenderPath(_ input: ResolutionInput) -> RenderPath {
        if input.tombstone || input.breakerOpen || input.killed || input.deprecated {
            return .placeholder
        }
        if hasNativeRenderer(input.toolId, registered: input.registered),
           ContentToolHostLogic.contractSupported(input.contract) {
            return .native
        }
        if isSandboxable(input) {
            return .sandbox
        }
        return .placeholder
    }

    static func resolveRenderPath(
        toolId: String,
        contract: Int,
        sandboxMode: String?,
        sandboxEnabled: Bool,
        registered: Set<String>,
        tombstone: Bool = false,
        breakerOpen: Bool = false,
        deprecated: Bool = false,
        killed: Bool = false
    ) -> RenderPath {
        resolveRenderPath(
            ResolutionInput(
                toolId: toolId,
                contract: contract,
                sandboxMode: sandboxMode,
                sandboxEnabled: sandboxEnabled,
                registered: registered,
                tombstone: tombstone,
                breakerOpen: breakerOpen,
                deprecated: deprecated,
                killed: killed
            )
        )
    }

    // MARK: - Protocol validation (FR-4, FR-6)

    private static func bridgeVersionMatches(_ value: Any?) -> Bool {
        if let intValue = value as? Int { return intValue == bridgeVersion }
        if let num = value as? NSNumber { return num.intValue == bridgeVersion }
        return false
    }

    static func isBridgeFromTool(_ msg: Any?) -> Bool {
        guard let dict = msg as? [String: Any],
              bridgeVersionMatches(dict["v"]),
              let messageType = dict["t"] as? String else { return false }
        return fromToolTypes.contains(messageType)
    }

    static func isBridgeToTool(_ msg: Any?) -> Bool {
        guard let dict = msg as? [String: Any],
              bridgeVersionMatches(dict["v"]),
              let messageType = dict["t"] as? String else { return false }
        return toToolTypes.contains(messageType)
    }

    static func measureMessageBytes(_ msg: Any?) -> Int {
        guard let msg else { return Int.max }
        guard JSONSerialization.isValidJSONObject(msg),
              let data = try? JSONSerialization.data(withJSONObject: msg, options: []) else {
            return Int.max
        }
        return data.count
    }

    static func measureMessageBytes(jsonString: String) -> Int {
        jsonString.utf8.count
    }

    /// Classify a raw ingress payload before dispatch. Returns nil when accepted.
    static func rejectIngress(
        rawJSON: String,
        limiter: BridgeRateLimiter,
        nowMs: Int64 = Int64(Date().timeIntervalSince1970 * 1000)
    ) -> RejectionReason? {
        if !limiter.allow(nowMs: nowMs) { return .rateLimited }
        if measureMessageBytes(jsonString: rawJSON) > bridgeMaxMessageBytes { return .oversized }
        guard let data = rawJSON.data(using: .utf8),
              let obj = try? JSONSerialization.jsonObject(with: data) else {
            return .malformed
        }
        if !isBridgeFromTool(obj) { return .malformed }
        return nil
    }

    static func clampHeight(_ height: Double) -> Double {
        guard height.isFinite else { return minHeightPt }
        return min(maxHeightPt, max(minHeightPt, height))
    }

    static func contractInSupportedRange(_ contract: String) -> Bool {
        contract == supportedReadyContract
    }

    /// Opaque per-instance participant id — not email, not user id (FR-7).
    /// Matches web `opaqueParticipantId` (Java-style 32-bit string hash).
    static func opaqueParticipantId(_ instanceId: String, enrollmentHint: String? = nil) -> String {
        let raw = "\(instanceId):\(enrollmentHint ?? "anon")"
        var hash: Int32 = 0
        for scalar in raw.unicodeScalars {
            // Math.imul(31, h) + charCode | 0
            let multiplied = Int32(truncatingIfNeeded: Int(hash) &* 31)
            hash = multiplied &+ Int32(scalar.value)
        }
        let unsigned = UInt32(bitPattern: hash)
        return "p_\(String(unsigned, radix: 16))"
    }

    static func documentPath(toolId: String, version: String? = nil) -> String {
        let encoded = toolId.addingPercentEncoding(withAllowedCharacters: .urlPathAllowed) ?? toolId
        var path = "/tool-sandbox/\(encoded).html"
        if let version, !version.isEmpty {
            let sep = path.contains("?") ? "&" : "?"
            path += "\(sep)v=\(version.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) ?? version)"
        }
        return path
    }

    static func poolShouldEvict(aliveCount: Int, maxAlive: Int = maxLiveWebViews) -> Bool {
        aliveCount > maxAlive
    }

    // MARK: - Rate limiter

    final class BridgeRateLimiter {
        private var timestamps: [Int64] = []
        private let maxPerSec: Int

        init(maxPerSec: Int = ContentToolSandboxLogic.bridgeMaxMessagesPerSec) {
            self.maxPerSec = maxPerSec
        }

        func allow(nowMs: Int64) -> Bool {
            let cut = nowMs - 1000
            timestamps = timestamps.filter { $0 >= cut }
            if timestamps.count >= maxPerSec { return false }
            timestamps.append(nowMs)
            return true
        }
    }
}
